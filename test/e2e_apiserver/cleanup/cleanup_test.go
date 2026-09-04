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

package cleanup_test

import (
	"context"
	"crypto/sha256"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	v1alpha1cleanup "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/cleanup"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const hfCommandTimeout = 2 * time.Minute

var _ = Describe("Cleanup", Label("cleanup"), func() {
	var (
		ctx        context.Context
		cleanupApi *v1alpha1cleanup.CleanupApiService
	)

	BeforeEach(func() {
		ctx = context.Background()
		cleanupApi = tools.GetV1alpha1CleanupApi()
	})

	It("should report storage stats", Label("CL00001", "smoke"), func() {
		stats, resp, err := cleanupApi.CleanupGetStorageStats(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		for _, size := range []string{stats.TotalSizeBytes, stats.RepositoriesSizeBytes, stats.LfsSizeBytes, stats.OrphanedSizeBytes} {
			n, err := strconv.ParseInt(size, 10, 64)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(BeNumerically(">=", 0))
		}
	})

	It("should preview and dry-run cleanup", Label("CL00002", "smoke"), func() {
		_, resp, err := cleanupApi.CleanupPreviewCleanup(ctx, v1alpha1cleanup.V1alpha1PreviewCleanupRequest{
			IncludeOrphanedRepos: true,
			IncludeOrphanedLfs:   true,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		res, resp, err := cleanupApi.CleanupExecuteCleanup(ctx, v1alpha1cleanup.V1alpha1ExecuteCleanupRequest{
			CleanOrphanedRepos: true,
			CleanOrphanedLfs:   true,
			DryRun:             true,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(res.Errors).To(BeEmpty())
	})

	It("should reclaim a deleted model's LFS objects and keep live ones", Label("CL00003", "smoke", "lfs"), func() {
		root := GinkgoT().TempDir()
		fixture, err := tools.CreateProjectUserFixture(ctx, "cleanup", v1alpha1project.EDITOR_V1alpha1ProjectRoleType)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			if err := fixture.Cleanup(context.Background()); err != nil {
				GinkgoWriter.Printf("protocol fixture cleanup failed: %v\n", err)
			}
		})
		token, err := fixture.CreateAccessToken(ctx)
		Expect(err).NotTo(HaveOccurred())
		env := tools.HFCLIEnvironment(root, token)

		modelA := tools.GenerateTestModelName("gc-a")
		modelB := tools.GenerateTestModelName("gc-b")
		DeferCleanup(func() {
			_, _, _ = tools.GetV1alpha1ModelsApi().ModelsDeleteModel(context.Background(), fixture.Project.Name, modelB)
		})

		var hashes [2][32]byte
		for i, model := range []string{modelA, modelB} {
			repoID := fixture.Project.Name + "/" + model
			create := runHF(root, env, "repos", "create", repoID, "--repo-type", "model")
			Expect(create.Err).NotTo(HaveOccurred(), create.FailureMessage())

			// Distinct pseudo-random content per model so xet cannot dedup A against B.
			r := rand.New(rand.NewPCG(uint64(i+1), 0))
			payload := make([]byte, 1<<20)
			for j := range payload {
				payload[j] = byte(r.Uint32())
			}
			hashes[i] = sha256.Sum256(payload)
			source := filepath.Join(root, model, "weights.safetensors")
			Expect(os.MkdirAll(filepath.Dir(source), 0755)).To(Succeed())
			Expect(os.WriteFile(source, payload, 0600)).To(Succeed())

			GinkgoWriter.Printf("uploading LFS file to %s\n", repoID)
			upload := runHF(root, env, "upload", repoID, source, "weights.safetensors", "--commit-message", "cleanup e2e upload")
			Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())
		}

		GinkgoWriter.Printf("deleting model %s\n", modelA)
		_, _, err = tools.GetV1alpha1ModelsApi().ModelsDeleteModel(ctx, fixture.Project.Name, modelA)
		Expect(err).NotTo(HaveOccurred())

		res, resp, err := cleanupApi.CleanupExecuteCleanup(ctx, v1alpha1cleanup.V1alpha1ExecuteCleanupRequest{CleanOrphanedLfs: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(res.Errors).To(BeEmpty())
		Expect(res.LfsObjectsDeleted).To(BeNumerically(">=", 1))
		reclaimed, err := strconv.ParseInt(res.SpaceReclaimedBytes, 10, 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(reclaimed).To(BeNumerically(">", 0))
		GinkgoWriter.Printf("cleanup unlinked %d LFS objects and reclaimed %d bytes\n", res.LfsObjectsDeleted, reclaimed)

		downloadDir := filepath.Join(root, "download-b")
		download := runHF(root, env, "download", fixture.Project.Name+"/"+modelB, "weights.safetensors", "--local-dir", downloadDir, "--force-download")
		Expect(download.Err).NotTo(HaveOccurred(), download.FailureMessage())
		downloaded, err := os.ReadFile(filepath.Join(downloadDir, "weights.safetensors"))
		Expect(err).NotTo(HaveOccurred())
		Expect(sha256.Sum256(downloaded)).To(Equal(hashes[1]))

		again, resp, err := cleanupApi.CleanupExecuteCleanup(ctx, v1alpha1cleanup.V1alpha1ExecuteCleanupRequest{CleanOrphanedLfs: true})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(again.LfsObjectsDeleted).To(BeZero())
	})
})

func runHF(dir string, env map[string]string, args ...string) tools.CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), hfCommandTimeout)
	defer cancel()
	return tools.RunCommand(ctx, dir, env, "hf", args...)
}
