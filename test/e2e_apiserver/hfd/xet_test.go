package hfd_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matrixhub-ai/matrixhub/test/tools"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const xetFilename = "weights.safetensors"

var _ = Describe("Xet client transfers", Label("gitproto", "xet"), func() {
	var (
		fixture  *protoFixture
		root     string
		payload  []byte
		observer *xetTransferObserver
	)

	BeforeEach(func() {
		Expect(strings.HasPrefix(tools.GetBaseURL(), "http://")).To(BeTrue(), "xet traffic assertions require the HTTP E2E endpoint")
		fixture = setupFixture(context.Background())
		root = GinkgoT().TempDir()
		payload = make([]byte, 2*1024*1024+127)
		_, err := rand.Read(payload)
		Expect(err).NotTo(HaveOccurred())
		observer = newXetTransferObserver()
		DeferCleanup(observer.Close)
	})

	It("should upload and download through xet with the real hf CLI", Label("GP00301", "hf", "smoke"), func() {
		source := filepath.Join(root, xetFilename)
		Expect(os.WriteFile(source, payload, 0600)).To(Succeed())
		environment := observer.environment(filepath.Join(root, "upload"), fixture.token)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		upload := tools.RunCommand(ctx, root, environment, "hf", "upload", fixture.project+"/"+modelName, source, xetFilename)
		Expect(upload.Err).NotTo(HaveOccurred(), upload.FailureMessage())
		Expect(observer.xorbUploads.Load()).To(BeNumerically(">", 0), "upload must reach xet CAS")
		Expect(observer.shardUploads.Load()).To(BeNumerically(">", 0), "upload must publish a xet shard")
		verifyXetDownload(root, fixture, observer, payload)
	})

	It("should upload through the real git-xet transfer agent", Label("GP00302", "git", "git-xet"), func() {
		environment := observer.environment(filepath.Join(root, "git-xet"), fixture.token)
		var gitEnvironment []string
		for key, value := range environment {
			gitEnvironment = append(gitEnvironment, key+"="+value)
		}
		gitEnvironment = append(gitEnvironment, "GIT_LFS_SKIP_PUSH=", "GIT_LFS_SKIP_SMUDGE=")
		remote := fixture.authenticatedRepoURL()
		_, err := runGit(root, gitEnvironment, "clone", remote, "repo")
		Expect(err).NotTo(HaveOccurred())
		repositoryDir := filepath.Join(root, "repo")
		for _, command := range [][]string{
			{"lfs", "install", "--local"},
			{"xet", "install", "--local"},
			{"config", "lfs.basictransfersonly", "false"},
			{"lfs", "track", xetFilename},
		} {
			_, err = runGit(repositoryDir, gitEnvironment, command...)
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(os.WriteFile(filepath.Join(repositoryDir, xetFilename), payload, 0600)).To(Succeed())
		_, err = runGit(repositoryDir, gitEnvironment, "add", ".gitattributes", xetFilename)
		Expect(err).NotTo(HaveOccurred())
		_, err = runGit(repositoryDir, gitEnvironment, "-c", "user.name=xet-e2e", "-c", "user.email=xet@e2e.test", "commit", "-m", "upload with git-xet")
		Expect(err).NotTo(HaveOccurred())
		pointer, err := runGit(repositoryDir, gitEnvironment, "show", "HEAD:"+xetFilename)
		Expect(err).NotTo(HaveOccurred())
		Expect(pointer).To(ContainSubstring(fmt.Sprintf("oid sha256:%x", sha256.Sum256(payload))))
		Expect(pointer).To(ContainSubstring(fmt.Sprintf("size %d", len(payload))))
		_, err = runGit(repositoryDir, gitEnvironment, "push", remote, "HEAD")
		Expect(err).NotTo(HaveOccurred())
		Expect(observer.xorbUploads.Load()).To(BeNumerically(">", 0), "git-xet must upload to CAS")
		Expect(observer.shardUploads.Load()).To(BeNumerically(">", 0), "git-xet must publish a shard")
		verifyXetDownload(root, fixture, observer, payload)
	})
})

func verifyXetDownload(root string, fixture *protoFixture, observer *xetTransferObserver, payload []byte) {
	GinkgoHelper()
	downloadRoot := filepath.Join(root, "fresh-download")
	environment := observer.environment(downloadRoot, fixture.token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	download := tools.RunCommand(ctx, root, environment, "hf", "download", fixture.project+"/"+modelName, xetFilename, "--local-dir", downloadRoot, "--force-download")
	Expect(download.Err).NotTo(HaveOccurred(), download.FailureMessage())
	downloaded, err := os.ReadFile(filepath.Join(downloadRoot, xetFilename))
	Expect(err).NotTo(HaveOccurred())
	Expect(len(downloaded)).To(Equal(len(payload)))
	Expect(sha256.Sum256(downloaded)).To(Equal(sha256.Sum256(payload)))
	Expect(observer.reconstructions.Load()).To(BeNumerically(">", 0), "download must use xet reconstruction")
	Expect(observer.xorbDownloads.Load()).To(BeNumerically(">", 0), "fresh cache must fetch xorb bytes")
	Expect(observer.basicTransfers.Load()).To(BeZero(), "LFS/basic fallback is not xet coverage")
}

type xetTransferObserver struct {
	server          *httptest.Server
	transport       *http.Transport
	xorbUploads     atomic.Int64
	shardUploads    atomic.Int64
	reconstructions atomic.Int64
	xorbDownloads   atomic.Int64
	basicTransfers  atomic.Int64
}

func newXetTransferObserver() *xetTransferObserver {
	observer := &xetTransferObserver{transport: http.DefaultTransport.(*http.Transport).Clone()}
	proxy := &httputil.ReverseProxy{
		Director:  func(*http.Request) {},
		Transport: observer.transport,
		ModifyResponse: func(response *http.Response) error {
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				return nil
			}
			request := response.Request
			if request.Method == http.MethodPost {
				if strings.HasPrefix(request.URL.Path, "/v1/xorbs/") {
					observer.xorbUploads.Add(1)
				}
				if request.URL.Path == "/v1/shards" || request.URL.Path == "/v2/shards" || request.URL.Path == "/shards" {
					observer.shardUploads.Add(1)
				}
			}
			if request.Method == http.MethodGet {
				if strings.Contains(request.URL.Path, "/reconstructions") {
					observer.reconstructions.Add(1)
				}
				if strings.HasPrefix(request.URL.Path, "/v1/xorbs/") {
					observer.xorbDownloads.Add(1)
				}
			}
			return nil
		},
	}
	observer.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		basic := strings.HasPrefix(request.URL.Path, "/objects/") || strings.HasPrefix(request.URL.Path, "/xet-bridge/")
		if basic && (request.Method == http.MethodGet || request.Method == http.MethodPut) {
			observer.basicTransfers.Add(1)
			http.Error(writer, "basic transfer disabled in xet test", http.StatusBadGateway)
			return
		}
		proxy.ServeHTTP(writer, request)
	}))
	return observer
}

func (observer *xetTransferObserver) environment(root, token string) map[string]string {
	environment := tools.HFCLIXetEnvironment(root, token)
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		environment[key] = observer.server.URL
	}
	environment["NO_PROXY"] = ""
	environment["no_proxy"] = ""
	environment["HF_HUB_VERBOSITY"] = "warning"
	return environment
}

func (observer *xetTransferObserver) Close() {
	observer.server.Close()
	observer.transport.CloseIdleConnections()
}

func TestXetTransferObserver(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)
	observer := newXetTransferObserver()
	t.Cleanup(observer.Close)
	proxyURL, err := url.Parse(observer.server.URL)
	require.NoError(t, err)
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}
	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodPut, "/objects/sha256", http.StatusBadGateway},
		{http.MethodGet, "/xet-bridge/sha256", http.StatusBadGateway},
		{http.MethodPost, "/v1/xorbs/default/hash", http.StatusOK},
		{http.MethodPost, "/v1/shards", http.StatusOK},
		{http.MethodGet, "/v2/reconstructions/hash", http.StatusOK},
		{http.MethodGet, "/v1/xorbs/default/hash", http.StatusOK},
	} {
		request, err := http.NewRequest(test.method, upstream.URL+test.path, nil)
		require.NoError(t, err)
		response, err := client.Do(request)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, test.status, response.StatusCode)
	}
	require.EqualValues(t, 2, observer.basicTransfers.Load())
	require.EqualValues(t, 1, observer.xorbUploads.Load())
	require.EqualValues(t, 1, observer.shardUploads.Load())
	require.EqualValues(t, 1, observer.reconstructions.Load())
	require.EqualValues(t, 1, observer.xorbDownloads.Load())
}
