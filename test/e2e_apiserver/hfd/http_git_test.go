// Copyright The MatrixHub Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hfd_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1alpha1current_user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/current_user"
	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testPassword = "Test@123456"
	modelName    = "gp-model"
)

// protoFixture is a per-spec user + access token + private project, torn down
// via DeferCleanup in reverse order (model+project first, then the user).
type protoFixture struct {
	username string
	token    string
	project  string
}

// setupFixture provisions the fixture through the public API: an admin-created
// user, a login session, an access token, and a private project owned by the
// user (project membership is what grants git push/pull permission).
func setupFixture(ctx context.Context) *protoFixture {
	GinkgoHelper()

	username := tools.GenerateTestUsername("gp")
	userID, cookie, err := tools.CreateUserAndLoginWithID(username, testPassword, false)
	Expect(err).NotTo(HaveOccurred(), "create and login test user")
	DeferCleanup(func() {
		_ = tools.DeleteUser(int64(userID))
	})

	currentUserApi := tools.CreateCurrentUserClientWithCookie(cookie)
	tokenResp, _, err := currentUserApi.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
		Name: "gitproto-e2e",
	})
	Expect(err).NotTo(HaveOccurred(), "create access token")
	Expect(tokenResp.Token).NotTo(BeEmpty())
	DeferCleanup(func() {
		// access_tokens rows are not cascaded by user deletion; remove them
		// via the API (best-effort) while the session is still valid.
		cleanupCtx := context.Background()
		if list, _, err := currentUserApi.CurrentUserListAccessTokens(cleanupCtx); err == nil {
			for _, item := range list.Items {
				_, _, _ = currentUserApi.CurrentUserDeleteAccessToken(cleanupCtx, item.Id)
			}
		}
	})

	projectsApi := tools.CreateProjectClientWithCookie(cookie)
	project := tools.GenerateTestProjectName("gp")
	// projects.name is varchar(64) in DB; keep room for suffixes.
	if len(project) > 60 {
		project = project[:60]
	}
	projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
	_, _, err = projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
		Name:  project,
		Type_: &projectType,
	})
	Expect(err).NotTo(HaveOccurred(), "create private project")
	DeferCleanup(func() {
		cleanupCtx := context.Background()
		_, _, _ = tools.GetV1alpha1ModelsApi().ModelsDeleteModel(cleanupCtx, project, modelName)
		_, _, _ = projectsApi.ProjectsDeleteProject(cleanupCtx, project)
	})
	_, _, err = tools.CreateModelClientWithCookie(cookie).ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
		Project: project,
		Name:    modelName,
	})
	Expect(err).NotTo(HaveOccurred(), "create model repository")

	return &protoFixture{
		username: username,
		token:    tokenResp.Token,
		project:  project,
	}
}

// httpRepoURL is the git-over-HTTP remote. Model repositories live at
// {base}/{project}/{name}.git — there is no "models/" prefix in the URL.
func (f *protoFixture) httpRepoURL() string {
	return fmt.Sprintf("%s/%s/%s.git", tools.GetBaseURL(), f.project, modelName)
}

func (f *protoFixture) authenticatedRepoURL() string {
	remote, err := url.Parse(f.httpRepoURL())
	Expect(err).NotTo(HaveOccurred())
	remote.User = url.UserPassword(f.username, f.token)
	return remote.String()
}

func runGit(dir string, extraEnv []string, args ...string) (string, error) {
	GinkgoHelper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "credential.helper="}, args...)...)
	cmd.Dir = dir
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_TRACE") || strings.HasPrefix(kv, "GIT_CURL_VERBOSE") || strings.HasPrefix(kv, "GIT_CONFIG") || strings.HasPrefix(kv, "GIT_ASKPASS=") || strings.HasPrefix(kv, "SSH_ASKPASS=") {
			continue
		}
		cmd.Env = append(cmd.Env, kv)
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	logged := make([]string, len(args))
	var redactions []string
	for i, arg := range args {
		if remote, parseErr := url.Parse(arg); parseErr == nil && remote.User != nil {
			redactions = append(redactions, remote.User.String(), "<redacted>")
			if password, ok := remote.User.Password(); ok && password != "" {
				redactions = append(redactions, password, "<redacted>")
			}
			arg = remote.Redacted()
		}
		logged[i] = arg
	}
	output := strings.NewReplacer(redactions...).Replace(string(out))
	GinkgoWriter.Printf("$ git %s\n%s", strings.Join(logged, " "), output)
	return output, err
}

// gitCommitFile writes content to path inside repoDir, stages it, and commits.
func gitCommitFile(repoDir, path, content, message string) {
	GinkgoHelper()
	Expect(os.MkdirAll(filepath.Dir(filepath.Join(repoDir, path)), 0o755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(repoDir, path), []byte(content), 0o644)).To(Succeed())
	_, err := runGit(repoDir, nil, "add", path)
	Expect(err).NotTo(HaveOccurred())
	_, err = runGit(repoDir, nil,
		"-c", "user.name=gitproto-e2e",
		"-c", "user.email=gitproto@e2e.test",
		"commit", "-m", message)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("GitProto over HTTP", Label("gitproto"), func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should push and clone a model repository over HTTP with basic auth", Label("GP00001", "smoke", "git"), func() {
		f := setupFixture(ctx)
		work := GinkgoT().TempDir()
		remote := f.authenticatedRepoURL()

		_, err := runGit(work, nil, "clone", remote, "repo")
		Expect(err).NotTo(HaveOccurred(), "clone freshly provisioned repository")
		repoDir := filepath.Join(work, "repo")

		content := fmt.Sprintf("gitproto e2e %s seed=%d\n", f.project, GinkgoRandomSeed())
		gitCommitFile(repoDir, "README.md", content, "add README")

		_, err = runGit(repoDir, nil, "push", remote, "HEAD")
		Expect(err).NotTo(HaveOccurred(), "push with basic auth")

		// Clone into a second directory and verify the content round-tripped.
		_, err = runGit(work, nil, "clone", remote, "verify")
		Expect(err).NotTo(HaveOccurred(), "clone for verification")

		got, err := os.ReadFile(filepath.Join(work, "verify", "README.md"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(got)).To(Equal(content), "cloned content must match pushed content")
	})

	It("should reject an anonymous push to a private project", Label("GP00002", "git"), func() {
		f := setupFixture(ctx)
		work := GinkgoT().TempDir()

		_, err := runGit(work, nil, "init", "-b", "main", "src")
		Expect(err).NotTo(HaveOccurred())
		srcDir := filepath.Join(work, "src")
		gitCommitFile(srcDir, "README.md", "anonymous push attempt\n", "init")

		status, _, headers := hfRequest(ctx, http.MethodGet, f.httpRepoURL()+"/info/refs?service=git-receive-pack", "", nil)
		Expect(status).To(Equal(http.StatusUnauthorized))
		Expect(headers.Get("WWW-Authenticate")).To(Equal(`Basic realm="hfd"`))
		_, err = runGit(srcDir, nil, "push", f.httpRepoURL(), "main")
		Expect(err).To(HaveOccurred(), "anonymous push must fail")
	})

	It("should not create models on anonymous reads of a public project", Label("GP00003", "git", "hf"), func() {
		project, err := tools.CreatePublicProjectFixture(ctx, "read-only")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_, _, _ = tools.GetV1alpha1ModelsApi().ModelsDeleteModel(context.Background(), project.Name, modelName)
			project.Cleanup(context.Background())
		})
		for _, endpoint := range []string{
			fmt.Sprintf("%s/api/models/%s/%s/tree/main", tools.GetBaseURL(), project.Name, modelName),
			fmt.Sprintf("%s/%s/%s.git/info/refs?service=git-upload-pack", tools.GetBaseURL(), project.Name, modelName),
		} {
			status, _, _ := hfRequest(ctx, http.MethodGet, endpoint, "", nil)
			Expect(status).To(Equal(http.StatusNotFound))
			_, response, err := tools.GetV1alpha1ModelsApi().ModelsGetModel(ctx, project.Name, modelName)
			Expect(err).To(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusNotFound))
		}
		_, _, err = tools.GetV1alpha1ModelsApi().ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{Project: project.Name, Name: modelName})
		Expect(err).NotTo(HaveOccurred())
		_, err = runGit(GinkgoT().TempDir(), nil, "clone", fmt.Sprintf("%s/%s/%s.git", tools.GetBaseURL(), project.Name, modelName), "repo")
		Expect(err).NotTo(HaveOccurred(), "existing public models remain readable anonymously")
	})
})
