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

package hf_cli_test

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"time"

	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const hfCommandTimeout = 2 * time.Minute

var _ = Describe("HF CLI", Label("hf-cli"), func() {
	var (
		ctx     context.Context
		fixture *tools.ProjectUserFixture
		root    string
		token   string
		env     map[string]string
	)

	BeforeEach(func() {
		ctx = context.Background()
		root = GinkgoT().TempDir()

		var err error
		fixture, err = tools.CreateProjectUserFixture(ctx, "hf-cli", v1alpha1project.EDITOR_V1alpha1ProjectRoleType)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			if err := fixture.Cleanup(context.Background()); err != nil {
				GinkgoWriter.Printf("protocol fixture cleanup failed: %v\n", err)
			}
		})

		token, err = fixture.CreateAccessToken(ctx)
		Expect(err).NotTo(HaveOccurred())
		env = tools.HFCLIEnvironment(root, token)
	})

	It("should authenticate, upload, and download a model with the real hf CLI", Label("HF00001", "smoke"), func() {
		model := tools.GenerateTestModelName("hf-model")
		repoID := fixture.Project.Name + "/" + model

		whoami := runHF(root, env, "auth", "whoami")
		Expect(whoami.Err).NotTo(HaveOccurred(), whoami.FailureMessage())
		Expect(whoami.Stdout).To(ContainSubstring(fixture.Username))

		create := runHF(root, env, "repos", "create", repoID, "--repo-type", "model")
		Expect(create.Err).NotTo(HaveOccurred(), create.FailureMessage())

		sourceDir := filepath.Join(root, "source")
		Expect(os.MkdirAll(sourceDir, 0755)).To(Succeed())
		const readme = "# MatrixHub HF CLI e2e\n"
		Expect(os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte(readme), 0600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(sourceDir, "config.json"), []byte(`{"model_type":"matrixhub-e2e"}`), 0600)).To(Succeed())

		upload := runHF(root, env, "upload", repoID, sourceDir, ".", "--commit-message", "hf cli e2e upload")
		Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())

		downloadDir := filepath.Join(root, "download")
		download := runHF(root, env, "download", repoID, "README.md", "--revision", "main", "--local-dir", downloadDir, "--force-download")
		Expect(download.Err).NotTo(HaveOccurred(), download.FailureMessage())
		content, err := os.ReadFile(filepath.Join(downloadDir, "README.md"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal(readme))
	})

	It("should reject an unauthenticated download from a private project", Label("HF00002"), func() {
		model := tools.GenerateTestModelName("hf-private")
		repoID := fixture.Project.Name + "/" + model

		create := runHF(root, env, "repos", "create", repoID, "--repo-type", "model")
		Expect(create.Err).NotTo(HaveOccurred(), create.FailureMessage())
		source := filepath.Join(root, "private.txt")
		Expect(os.WriteFile(source, []byte("private\n"), 0600)).To(Succeed())
		upload := runHF(root, env, "upload", repoID, source, "private.txt")
		Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())

		anonymousEnv := tools.HFCLIEnvironment(filepath.Join(root, "anonymous"), "")
		anonymousDownloadDir := filepath.Join(root, "anonymous-download")
		download := runHF(root, anonymousEnv, "download", repoID, "private.txt", "--local-dir", anonymousDownloadDir, "--force-download")
		Expect(download.Err).To(HaveOccurred(), "private repository download unexpectedly succeeded")
		_, err := os.Stat(filepath.Join(anonymousDownloadDir, "private.txt"))
		Expect(os.IsNotExist(err)).To(BeTrue(), "private file was written despite failed authentication")
	})

	It("should upload and download a large file through the HF LFS flow", Label("HF00003", "lfs", "slow"), func() {
		model := tools.GenerateTestModelName("hf-lfs")
		repoID := fixture.Project.Name + "/" + model

		create := runHF(root, env, "repos", "create", repoID, "--repo-type", "model")
		Expect(create.Err).NotTo(HaveOccurred(), create.FailureMessage())

		payload := make([]byte, 10*1024*1024+1)
		for i := range payload {
			payload[i] = byte(i % 251)
		}
		expectedHash := sha256.Sum256(payload)
		source := filepath.Join(root, "weights.safetensors")
		Expect(os.WriteFile(source, payload, 0600)).To(Succeed())

		upload := runHF(root, env, "upload", repoID, source, "weights.safetensors", "--commit-message", "hf lfs upload")
		Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())

		downloadDir := filepath.Join(root, "lfs-download")
		download := runHF(root, env, "download", repoID, "weights.safetensors", "--local-dir", downloadDir, "--force-download")
		Expect(download.Err).NotTo(HaveOccurred(), download.FailureMessage())
		downloaded, err := os.ReadFile(filepath.Join(downloadDir, "weights.safetensors"))
		Expect(err).NotTo(HaveOccurred())
		Expect(sha256.Sum256(downloaded)).To(Equal(expectedHash))
	})
})

func runHF(dir string, env map[string]string, args ...string) tools.CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), hfCommandTimeout)
	defer cancel()
	return tools.RunCommand(ctx, dir, env, "hf", args...)
}
