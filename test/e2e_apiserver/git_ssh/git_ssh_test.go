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

package git_ssh_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const gitCommandTimeout = 2 * time.Minute

var _ = Describe("Git over SSH", Label("git-ssh"), func() {
	var (
		ctx       context.Context
		fixture   *tools.ProjectUserFixture
		sshClient *tools.SSHClientFixture
		root      string
	)

	BeforeEach(func() {
		ctx = context.Background()
		root = GinkgoT().TempDir()

		var err error
		fixture, err = tools.CreateProjectUserFixture(ctx, "git-ssh", v1alpha1project.EDITOR_V1alpha1ProjectRoleType)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			if err := fixture.Cleanup(context.Background()); err != nil {
				GinkgoWriter.Printf("protocol fixture cleanup failed: %v\n", err)
			}
		})

		sshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		sshClient, err = tools.NewSSHClientFixture(sshCtx, filepath.Join(root, "ssh"))
		Expect(err).NotTo(HaveOccurred())
		Expect(fixture.RegisterSSHKey(ctx, sshClient.PublicKey)).To(Succeed())
	})

	It("should push and clone a model repository over SSH", Label("GS00001", "smoke"), func() {
		model := tools.GenerateTestModelName("git-model")
		remote, source := createAndCloneModelRepository(ctx, fixture, sshClient, root, model, "source")

		const content = "MatrixHub Git SSH e2e\n"
		Expect(os.WriteFile(filepath.Join(source, "README.md"), []byte(content), 0600)).To(Succeed())
		runGitSuccessfully(source, sshClient.GitEnvironment(), "add", "README.md")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "commit", "-m", "initial SSH push")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "push", "-u", "origin", "main")

		clone := filepath.Join(root, "clone")
		runGitSuccessfully(root, sshClient.GitEnvironment(), "clone", remote, clone)
		cloned, err := os.ReadFile(filepath.Join(clone, "README.md"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(cloned)).To(Equal(content))

		branch := runGit(clone, sshClient.GitEnvironment(), "rev-parse", "--abbrev-ref", "HEAD")
		Expect(branch.Err).NotTo(HaveOccurred(), branch.FailureMessage())
		Expect(branch.Stdout).To(Equal("main\n"))
	})

	It("should reject an unregistered SSH key", Label("GS00002"), func() {
		model := tools.GenerateTestModelName("git-bad-key")
		remote, source := createAndCloneModelRepository(ctx, fixture, sshClient, root, model, "source")
		Expect(os.WriteFile(filepath.Join(source, "README.md"), []byte("seed\n"), 0600)).To(Succeed())
		runGitSuccessfully(source, sshClient.GitEnvironment(), "add", "README.md")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "commit", "-m", "seed repository")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "push", "-u", "origin", "main")

		badKeyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		unregistered, err := tools.NewSSHClientFixture(badKeyCtx, filepath.Join(root, "unregistered-ssh"))
		Expect(err).NotTo(HaveOccurred())
		clone := runGit(root, unregistered.GitEnvironment(), "clone", remote, filepath.Join(root, "bad-clone"))
		Expect(clone.Err).To(HaveOccurred(), "clone unexpectedly succeeded with an unregistered key")
	})

	It("should allow a viewer to clone but reject pushes", Label("GS00003"), func() {
		model := tools.GenerateTestModelName("git-viewer")
		remote, source := createAndCloneModelRepository(ctx, fixture, sshClient, root, model, "source")
		Expect(os.WriteFile(filepath.Join(source, "README.md"), []byte("viewer seed\n"), 0600)).To(Succeed())
		runGitSuccessfully(source, sshClient.GitEnvironment(), "add", "README.md")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "commit", "-m", "viewer seed")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "push", "-u", "origin", "main")

		viewer := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
		memberType := v1alpha1project.USER_V1alpha1MemberType
		_, _, err := tools.GetV1alpha1ProjectsApi().ProjectsUpdateProjectMemberRole(ctx, fixture.Project.Name, fixture.UserID, v1alpha1project.ProjectsUpdateProjectMemberRoleBody{
			MemberType: &memberType,
			Role:       &viewer,
		})
		Expect(err).NotTo(HaveOccurred())

		cloneDir := filepath.Join(root, "viewer-clone")
		runGitSuccessfully(root, sshClient.GitEnvironment(), "clone", remote, cloneDir)
		configureGitIdentity(cloneDir, sshClient.GitEnvironment())
		Expect(os.WriteFile(filepath.Join(cloneDir, "viewer.txt"), []byte("denied\n"), 0600)).To(Succeed())
		runGitSuccessfully(cloneDir, sshClient.GitEnvironment(), "add", "viewer.txt")
		runGitSuccessfully(cloneDir, sshClient.GitEnvironment(), "commit", "-m", "viewer write")
		push := runGit(cloneDir, sshClient.GitEnvironment(), "push", "origin", "main")
		Expect(push.Err).To(HaveOccurred(), "viewer push unexpectedly succeeded")
	})

	It("should push branches and tags over SSH", Label("GS00005"), func() {
		model := tools.GenerateTestModelName("git-refs")
		remote, source := createAndCloneModelRepository(ctx, fixture, sshClient, root, model, "refs-source")
		Expect(os.WriteFile(filepath.Join(source, "README.md"), []byte("main\n"), 0600)).To(Succeed())
		runGitSuccessfully(source, sshClient.GitEnvironment(), "add", "README.md")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "commit", "-m", "main commit")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "push", "-u", "origin", "main")

		runGitSuccessfully(source, sshClient.GitEnvironment(), "switch", "-c", "feature/e2e")
		Expect(os.WriteFile(filepath.Join(source, "feature.txt"), []byte("feature\n"), 0600)).To(Succeed())
		runGitSuccessfully(source, sshClient.GitEnvironment(), "add", "feature.txt")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "commit", "-m", "feature commit")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "tag", "v0.0.1-e2e")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "push", "origin", "feature/e2e", "v0.0.1-e2e")

		refs := runGit(root, sshClient.GitEnvironment(), "ls-remote", remote)
		Expect(refs.Err).NotTo(HaveOccurred(), refs.FailureMessage())
		Expect(refs.Stdout).To(ContainSubstring("refs/heads/feature/e2e"))
		Expect(refs.Stdout).To(ContainSubstring("refs/tags/v0.0.1-e2e"))
	})

	It("should reject new SSH connections after the key is deleted", Label("GS00006"), func() {
		model := tools.GenerateTestModelName("git-revoked-key")
		remote, source := createAndCloneModelRepository(ctx, fixture, sshClient, root, model, "revoked-source")
		Expect(os.WriteFile(filepath.Join(source, "README.md"), []byte("seed\n"), 0600)).To(Succeed())
		runGitSuccessfully(source, sshClient.GitEnvironment(), "add", "README.md")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "commit", "-m", "seed repository")
		runGitSuccessfully(source, sshClient.GitEnvironment(), "push", "-u", "origin", "main")

		keys, _, err := fixture.CurrentUser.CurrentUserListSSHKeys(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(keys.Items).To(HaveLen(1))
		_, _, err = fixture.CurrentUser.CurrentUserDeleteSSHKey(ctx, keys.Items[0].Id)
		Expect(err).NotTo(HaveOccurred())

		lsRemote := runGit(root, sshClient.GitEnvironment(), "ls-remote", remote)
		Expect(lsRemote.Err).To(HaveOccurred(), "SSH access unexpectedly succeeded after key deletion")
	})
})

func createAndCloneModelRepository(
	ctx context.Context,
	fixture *tools.ProjectUserFixture,
	sshClient *tools.SSHClientFixture,
	root, model, dirName string,
) (string, string) {
	modelsAPI := tools.CreateModelClientWithCookie(fixture.Cookie)
	_, _, err := modelsAPI.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
		Project: fixture.Project.Name,
		Name:    model,
	})
	Expect(err).NotTo(HaveOccurred())

	remote := sshClient.RepositoryURL(fixture.Project.Name, model)
	dir := filepath.Join(root, dirName)
	env := sshClient.GitEnvironment()
	runGitSuccessfully(root, env, "clone", remote, dir)
	configureGitIdentity(dir, env)
	return remote, dir
}

func configureGitIdentity(dir string, env map[string]string) {
	runGitSuccessfully(dir, env, "config", "user.name", "MatrixHub E2E")
	runGitSuccessfully(dir, env, "config", "user.email", "e2e@matrixhub.invalid")
}

func runGitSuccessfully(dir string, env map[string]string, args ...string) {
	result := runGit(dir, env, args...)
	ExpectWithOffset(1, result.Err).NotTo(HaveOccurred(), result.FailureMessage())
}

func runGit(dir string, env map[string]string, args ...string) tools.CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	result := tools.RunCommand(ctx, dir, env, "git", args...)
	if result.Err != nil {
		GinkgoWriter.Printf("%s\n", result.FailureMessage())
	}
	return result
}
