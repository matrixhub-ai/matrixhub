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
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v1alpha1current_user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/current_user"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testPassword = "Test@123456"
	// modelName is the model repository name used by every spec; specs are
	// isolated by their per-spec project namespace, so a fixed name is safe.
	// The server auto-creates the model record and the git repository on the
	// first authenticated repository contact (EnsureModel pre-open hook).
	modelName = "gp-model"
)

// protoFixture is a per-spec user + access token + private project, torn down
// via DeferCleanup in reverse order (model+project first, then the user).
type protoFixture struct {
	username       string
	token          string
	project        string
	currentUserApi *v1alpha1current_user.CurrentUserApiService
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
		// The model record is auto-created server-side on first repo contact;
		// delete it (as admin) before the project. Both are best-effort.
		_, _, _ = tools.GetV1alpha1ModelsApi().ModelsDeleteModel(cleanupCtx, project, modelName)
		_, _, _ = projectsApi.ProjectsDeleteProject(cleanupCtx, project)
	})

	return &protoFixture{
		username:       username,
		token:          tokenResp.Token,
		project:        project,
		currentUserApi: currentUserApi,
	}
}

// httpRepoURL is the git-over-HTTP remote. Model repositories live at
// {base}/{project}/{name}.git — there is no "models/" prefix in the URL.
func (f *protoFixture) httpRepoURL() string {
	return fmt.Sprintf("%s/%s/%s.git", tools.GetBaseURL(), f.project, modelName)
}

// basicAuthArgs returns git config args that inject "Authorization: Basic
// user:token" on every request. URL-embedded credentials do not work against
// this server: git only sends them after a 401 challenge, but the server
// answers anonymous requests on private repos with 403 and no WWW-Authenticate
// header, so http.extraHeader is used to send credentials proactively.
func (f *protoFixture) basicAuthArgs() []string {
	cred := base64.StdEncoding.EncodeToString([]byte(f.username + ":" + f.token))
	return []string{"-c", "http.extraHeader=Authorization: Basic " + cred}
}

// runGit executes the git binary in dir with extraEnv appended to the
// environment, returning combined output. Terminal prompts are disabled so
// failed authentication surfaces as an error instead of hanging. Inherited
// GIT_TRACE*/GIT_CURL_VERBOSE variables are dropped and Authorization values
// are redacted from the logged command, so credentials never reach the logs.
func runGit(dir string, extraEnv []string, args ...string) (string, error) {
	GinkgoHelper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_TRACE") || strings.HasPrefix(kv, "GIT_CURL_VERBOSE") {
			continue // git tracing dumps argv and HTTP headers, leaking credentials
		}
		cmd.Env = append(cmd.Env, kv)
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	logged := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "http.extraHeader=Authorization:") {
			arg = "http.extraHeader=Authorization: <redacted>"
		}
		logged[i] = arg
	}
	GinkgoWriter.Printf("$ git %s\n%s", strings.Join(logged, " "), string(out))
	return string(out), err
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
		auth := f.basicAuthArgs()

		// Clone first: the server provisions the repository (with an initial
		// commit holding .gitattributes) on first authenticated contact, so a
		// push from a fresh local-only history would be non-fast-forward.
		_, err := runGit(work, nil, append(auth, "clone", f.httpRepoURL(), "repo")...)
		Expect(err).NotTo(HaveOccurred(), "clone freshly provisioned repository")
		repoDir := filepath.Join(work, "repo")

		content := fmt.Sprintf("gitproto e2e %s seed=%d\n", f.project, GinkgoRandomSeed())
		gitCommitFile(repoDir, "README.md", content, "add README")

		_, err = runGit(repoDir, nil, append(auth, "push", "origin", "HEAD")...)
		Expect(err).NotTo(HaveOccurred(), "push with basic auth")

		// Clone into a second directory and verify the content round-tripped.
		_, err = runGit(work, nil, append(auth, "clone", f.httpRepoURL(), "verify")...)
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

		// No credentials: the server denies the receive-pack advertisement
		// with 403 and git must exit non-zero.
		out, err := runGit(srcDir, nil, "push", f.httpRepoURL(), "main")
		Expect(err).To(HaveOccurred(), "anonymous push must fail")
		Expect(out).To(ContainSubstring("403"), "server must deny with 403")
	})
})
