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

package protocol_interop_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HF and Git interoperability", Label("protocol-interop"), func() {
	It("should expose HF uploads to Git and Git pushes to HF", Label("PI00001", "smoke"), func() {
		ctx := context.Background()
		root := GinkgoT().TempDir()

		fixture, err := tools.CreateProjectUserFixture(ctx, "interop", v1alpha1project.EDITOR_V1alpha1ProjectRoleType)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			if err := fixture.Cleanup(context.Background()); err != nil {
				GinkgoWriter.Printf("protocol fixture cleanup failed: %v\n", err)
			}
		})

		token, err := fixture.CreateAccessToken(ctx)
		Expect(err).NotTo(HaveOccurred())
		hfEnv := tools.HFCLIEnvironment(root, token)

		sshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		sshClient, err := tools.NewSSHClientFixture(sshCtx, filepath.Join(root, "ssh"))
		Expect(err).NotTo(HaveOccurred())
		Expect(fixture.RegisterSSHKey(ctx, sshClient.PublicKey)).To(Succeed())

		model := tools.GenerateTestModelName("interop-model")
		repoID := fixture.Project.Name + "/" + model
		remote := sshClient.RepositoryURL(fixture.Project.Name, model)

		create := runInteropCommand(root, hfEnv, "hf", "repos", "create", repoID, "--repo-type", "model")
		Expect(create.Err).NotTo(HaveOccurred(), create.FailureMessage())
		fromHF := filepath.Join(root, "from-hf.txt")
		Expect(os.WriteFile(fromHF, []byte("written by hf\n"), 0600)).To(Succeed())
		upload := runInteropCommand(root, hfEnv, "hf", "upload", repoID, fromHF, "from-hf.txt", "--commit-message", "upload from hf")
		Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())

		cloneDir := filepath.Join(root, "clone")
		clone := runInteropCommand(root, sshClient.GitEnvironment(), "git", "clone", remote, cloneDir)
		Expect(clone.Err).NotTo(HaveOccurred(), clone.FailureMessage())
		content, err := os.ReadFile(filepath.Join(cloneDir, "from-hf.txt"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("written by hf\n"))

		gitEnv := sshClient.GitEnvironment()
		for _, command := range [][]string{
			{"config", "user.name", "MatrixHub E2E"},
			{"config", "user.email", "e2e@matrixhub.invalid"},
		} {
			result := runInteropCommand(cloneDir, gitEnv, "git", command...)
			Expect(result.Err).NotTo(HaveOccurred(), result.FailureMessage())
		}
		Expect(os.WriteFile(filepath.Join(cloneDir, "from-git.txt"), []byte("written by git\n"), 0600)).To(Succeed())
		for _, command := range [][]string{
			{"add", "from-git.txt"},
			{"commit", "-m", "push from git"},
			{"push", "origin", "main"},
		} {
			result := runInteropCommand(cloneDir, gitEnv, "git", command...)
			Expect(result.Err).NotTo(HaveOccurred(), result.FailureMessage())
		}

		downloadDir := filepath.Join(root, "hf-download")
		download := runInteropCommand(root, hfEnv, "hf", "download", repoID, "from-git.txt", "--local-dir", downloadDir, "--force-download")
		Expect(download.Err).NotTo(HaveOccurred(), download.FailureMessage())
		content, err = os.ReadFile(filepath.Join(downloadDir, "from-git.txt"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("written by git\n"))
	})
})

func runInteropCommand(dir string, env map[string]string, name string, args ...string) tools.CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return tools.RunCommand(ctx, dir, env, name, args...)
}
