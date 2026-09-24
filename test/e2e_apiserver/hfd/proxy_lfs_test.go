package hfd_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Proxy project LFS", Label("gitproto", "hf", "proxy"), func() {
	It("should serve a self-registry proxy model's LFS file with the source commit", Label("GP00401"), func() {
		// The registry URL is dereferenced by the server, so it must be routable from there (see SP0011).
		selfURL := tools.GetSelfURL()
		if selfURL == "" {
			Skip(fmt.Sprintf("%s not set: no server-reachable MatrixHub URL to use as a stand-in registry", tools.EnvMatrixHubSelfURL))
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		source := setupFixtureWithType(ctx, v1alpha1project.PUBLIC_V1alpha1ProjectType)
		root := GinkgoT().TempDir()
		payload := make([]byte, 2*1024*1024+127)
		_, err := rand.Read(payload)
		Expect(err).NotTo(HaveOccurred())
		sum := sha256.Sum256(payload)
		oid := fmt.Sprintf("%x", sum)

		localFile := filepath.Join(root, xetFilename)
		Expect(os.WriteFile(localFile, payload, 0600)).To(Succeed())
		upload := tools.RunCommand(ctx, root, tools.HFCLIXetEnvironment(filepath.Join(root, "upload"), source.token), "hf", "upload", source.project+"/"+modelName, localFile, xetFilename)
		Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())

		sourceResolve := fmt.Sprintf("%s/%s/%s/resolve/main/%s", tools.GetBaseURL(), source.project, modelName, xetFilename)
		status, _, headers := hfRequest(ctx, http.MethodHead, sourceResolve, "", nil)
		Expect(status).To(Equal(http.StatusOK))
		commit := headers.Get("X-Repo-Commit")
		Expect(commit).To(HaveLen(40), "source resolve must report the upload commit")
		Expect(headers.Get("ETag")).To(Equal(strconv.Quote(oid)), "source must serve the payload as an LFS object")

		registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-proxy", selfURL)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			registry.Cleanup(context.Background())
		})

		proxy := tools.GenerateTestProjectName("gp-proxy")
		if len(proxy) > 60 {
			proxy = proxy[:60]
		}
		projectType := v1alpha1project.PUBLIC_V1alpha1ProjectType
		projectsApi := tools.GetV1alpha1ProjectsApi()
		_, _, err = projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
			Name:         proxy,
			Type_:        &projectType,
			RegistryId:   int32(registry.ID),
			Organization: source.project,
		})
		Expect(err).NotTo(HaveOccurred(), "create proxy project")
		DeferCleanup(func() {
			cleanupCtx := context.Background()
			_, _, _ = tools.GetV1alpha1ModelsApi().ModelsDeleteModel(cleanupCtx, proxy, modelName)
			_, _, _ = projectsApi.ProjectsDeleteProject(cleanupCtx, proxy)
		})

		// The first read pulls the source repository; LFS bytes are already in the shared CAS.
		proxyResolve := fmt.Sprintf("%s/%s/%s/resolve/main/%s", tools.GetBaseURL(), proxy, modelName, xetFilename)
		Eventually(func(g Gomega) {
			status, body, _ := hfRequest(ctx, http.MethodGet, proxyResolve, "", nil)
			g.Expect(status).To(Equal(http.StatusOK))
			g.Expect(len(body)).To(Equal(len(payload)))
			g.Expect(sha256.Sum256(body)).To(Equal(sum))
		}).WithTimeout(2*time.Minute).WithPolling(2*time.Second).Should(Succeed(), "proxy resolve must serve the source LFS bytes")

		status, _, headers = hfRequest(ctx, http.MethodHead, proxyResolve, "", nil)
		Expect(status).To(Equal(http.StatusOK))
		Expect(headers.Get("X-Repo-Commit")).To(Equal(commit), "proxy must expose the source commit")
		Expect(headers.Get("ETag")).To(Equal(strconv.Quote(oid)))
		Expect(headers.Get("X-Linked-Size")).To(Equal(strconv.Itoa(len(payload))))

		pinnedResolve := fmt.Sprintf("%s/%s/%s/resolve/%s/%s", tools.GetBaseURL(), proxy, modelName, commit, xetFilename)
		status, body, _ := hfRequest(ctx, http.MethodGet, pinnedResolve, "", nil)
		Expect(status).To(Equal(http.StatusOK), "commit-pinned resolve must work on the proxy")
		Expect(sha256.Sum256(body)).To(Equal(sum))

		var entries []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			Size int64  `json:"size"`
			LFS  *struct {
				OID  string `json:"oid"`
				Size int64  `json:"size"`
			} `json:"lfs"`
		}
		status, body, _ = hfRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/models/%s/%s/tree/main", tools.GetBaseURL(), proxy, modelName), "", nil)
		Expect(status).To(Equal(http.StatusOK))
		Expect(json.Unmarshal(body, &entries)).To(Succeed())
		var found bool
		for _, entry := range entries {
			if entry.Path != xetFilename {
				continue
			}
			found = true
			Expect(entry.Type).To(Equal("file"))
			Expect(entry.Size).To(BeEquivalentTo(len(payload)))
			Expect(entry.LFS).NotTo(BeNil(), "tree entry must be an LFS pointer")
			Expect(entry.LFS.OID).To(Equal(oid))
			Expect(entry.LFS.Size).To(BeEquivalentTo(len(payload)))
		}
		Expect(found).To(BeTrue(), "proxy tree must list %s", xetFilename)

		_, _, err = tools.GetV1alpha1ModelsApi().ModelsGetModel(ctx, proxy, modelName)
		Expect(err).NotTo(HaveOccurred(), "proxy model record must exist in the proxy namespace")
		status, _, headers = hfRequest(ctx, http.MethodHead, sourceResolve, "", nil)
		Expect(status).To(Equal(http.StatusOK))
		Expect(headers.Get("X-Repo-Commit")).To(Equal(commit), "source must be unchanged by the proxy pull")

		downloadRoot := filepath.Join(root, "download")
		download := tools.RunCommand(ctx, root, tools.HFCLIXetEnvironment(downloadRoot, source.token), "hf", "download", proxy+"/"+modelName, xetFilename, "--local-dir", downloadRoot, "--force-download")
		Expect(download.Err).NotTo(HaveOccurred(), download.FailureMessage())
		downloaded, err := os.ReadFile(filepath.Join(downloadRoot, xetFilename))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(downloaded)).To(Equal(len(payload)))
		Expect(sha256.Sum256(downloaded)).To(Equal(sum))
	})
})
