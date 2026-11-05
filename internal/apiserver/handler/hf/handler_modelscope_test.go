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

package hf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
)

func modelScopeFixture(t *testing.T) (*Handler, string, string) {
	t.Helper()
	s := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	repo, err := repository.Init(context.Background(), s.ResolvePath("team/demo"), "main")
	require.NoError(t, err)
	weights := []byte("0123456789-modelscope-lfs")
	sum := sha256.Sum256(weights)
	oid := hex.EncodeToString(sum[:])
	ls := lfs.NewLocal(s.LFSDir())
	require.NoError(t, ls.Put(oid, bytes.NewReader(weights), int64(len(weights))))
	sha, err := repo.CreateCommit(context.Background(), "main", "fixture", "tester", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "README.md", Content: []byte("# ModelScope fixture\n")},
		{Type: repository.CommitOperationAdd, Path: "nested/config.json", Content: []byte(`{"model_type":"test"}`)},
		{Type: repository.CommitOperationAdd, Path: "weights.bin", Content: []byte(fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, len(weights)))},
	}, "")
	require.NoError(t, err)
	require.NoError(t, repo.CreateTag("v1", sha))
	h := NewHandler(WithStorage(s), WithLFSStorage(ls), WithNext(http.NotFoundHandler()))
	return h, sha, oid
}

func TestModelScopeSnapshotAndDownloads(t *testing.T) {
	h, sha, oid := modelScopeFixture(t)
	base := "/api/v1/models/team/demo"
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, base+"/repo/files?Revision=master&Recursive=True", nil))
	require.Equal(t, http.StatusOK, r.Code)
	var listing struct {
		Code int
		Data struct{ Files []msFile }
	}
	require.NoError(t, json.Unmarshal(r.Body.Bytes(), &listing))
	files := map[string]msFile{}
	for _, f := range listing.Data.Files {
		files[f.Path] = f
	}
	require.Equal(t, sha, files["README.md"].Revision)
	readmeHash := sha256.Sum256([]byte("# ModelScope fixture\n"))
	require.Equal(t, hex.EncodeToString(readmeHash[:]), files["README.md"].Sha256)
	require.Equal(t, oid, files["weights.bin"].Sha256)
	require.True(t, files["weights.bin"].IsLFS)
	require.Equal(t, int64(len("0123456789-modelscope-lfs")), files["weights.bin"].Size)
	for _, name := range []string{"README.md", "weights.bin"} {
		t.Run(name, func(t *testing.T) {
			endpoint := base + "/repo?Revision=" + sha + "&FilePath=" + url.QueryEscape(name)
			request := httptest.NewRequest(http.MethodGet, endpoint, nil)
			request.Header.Set("Range", "bytes=2-5")
			recorder := httptest.NewRecorder()
			h.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusPartialContent, recorder.Code)
			require.Len(t, recorder.Body.Bytes(), 4)
			recorder = httptest.NewRecorder()
			h.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, endpoint, nil))
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Empty(t, recorder.Body.String())
			require.NotEmpty(t, recorder.Header().Get("Content-Length"))
		})
	}
	for _, revision := range []string{"main", "master", "v1", sha} {
		r = httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, base+"/repo?Revision="+revision+"&FilePath=nested%2Fconfig.json", nil))
		require.Equal(t, http.StatusOK, r.Code)
		require.JSONEq(t, `{"model_type":"test"}`, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, base+"/revisions", nil))
	require.Equal(t, http.StatusOK, r.Code)
	require.Contains(t, r.Body.String(), `"Revision":"master"`)
	require.Contains(t, r.Body.String(), `"Revision":"v1"`)
}

func TestModelScopeRejectsUnauthorizedAndInvalidPaths(t *testing.T) {
	h, _, _ := modelScopeFixture(t)
	for _, endpoint := range []string{
		"/revisions", "/repo/files?Recursive=true", "/repo?FilePath=README.md", "",
	} {
		h.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
			return false, nil
		}
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo"+endpoint, nil))
		require.Equal(t, http.StatusForbidden, r.Code)
		require.NotContains(t, r.Body.String(), "fixture")
	}
	h.permissionHookFunc = nil
	for _, file := range []string{"../secret", "/etc/passwd", "nested/../README.md", "nested\\file", ""} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo/repo?FilePath="+url.QueryEscape(file), nil))
		require.Equal(t, http.StatusBadRequest, r.Code)
	}
	for _, suffix := range []string{"/repo/files?Revision=missing", "/repo?FilePath=README.md&Revision=missing", "/repo?FilePath=missing"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo"+suffix, nil))
		require.Equal(t, http.StatusNotFound, r.Code)
	}
}

// This opt-in integration test uses the unmodified official SDK and CLI.
// Set MATRIXHUB_MODELSCOPE_PYTHON to the Python in an environment containing
// modelscope (and, for newer releases, modelscope-hub).
func TestModelScopeOfficialSDK(t *testing.T) {
	python := os.Getenv("MATRIXHUB_MODELSCOPE_PYTHON")
	if python == "" {
		t.Skip("set MATRIXHUB_MODELSCOPE_PYTHON to run real SDK downloads")
	}
	h, _, _ := modelScopeFixture(t)
	server := httptest.NewServer(h)
	defer server.Close()
	directory := t.TempDir()
	script := `import pathlib,sys
from modelscope import snapshot_download
from modelscope.hub.file_download import model_file_download
endpoint, root = sys.argv[1:]
dest = snapshot_download('team/demo', endpoint=endpoint, local_dir=root+'/snapshot')
assert (pathlib.Path(dest)/'README.md').read_text() == '# ModelScope fixture\n'
assert (pathlib.Path(dest)/'weights.bin').read_bytes() == b'0123456789-modelscope-lfs'
f = model_file_download('team/demo', 'nested/config.json', revision='v1', endpoint=endpoint, local_dir=root+'/single')
assert pathlib.Path(f).read_text() == '{"model_type":"test"}'
print('Official ModelScope snapshot and single-file downloads passed')
`
	cmd := exec.CommandContext(t.Context(), python, "-c", script, server.URL, directory)
	cmd.Env = append(os.Environ(), "MODELSCOPE_CACHE="+filepath.Join(directory, "cache"), "MODELSCOPE_DOMAIN="+strings.TrimPrefix(server.URL, "http://"))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	t.Log(string(out))
	cli := exec.CommandContext(t.Context(), python, "-c", "from modelscope_hub.cli.main import run_cmd; run_cmd()", "--endpoint", server.URL, "download", "team/demo", "--revision", "v1", "--local-dir", filepath.Join(directory, "cli"))
	cli.Env = cmd.Env
	out, err = cli.CombinedOutput()
	require.NoError(t, err, "%s", out)
	cliContent, err := os.ReadFile(filepath.Join(directory, "cli", "weights.bin"))
	require.NoError(t, err)
	require.Equal(t, "0123456789-modelscope-lfs", string(cliContent))
	// Verify the content again outside the SDK's own cache/integrity machinery.
	file, err := os.Open(filepath.Join(directory, "snapshot", "weights.bin"))
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	contents, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, "0123456789-modelscope-lfs", string(contents))
}

func TestModelScopeColdProxyListsAndStreamsWithoutMaterializingWeights(t *testing.T) {
	h, sha, oid := modelScopeFixture(t)
	h.lfsStorage = lfs.NewLocal(t.TempDir())
	ctrl := gomock.NewController(t)
	modelService := modelmocks.NewMockIModelService(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	h.modelService, h.gitRepo = modelService, gitRepo
	remote := &git.GitRepository{RemoteProjectName: "upstream", RemoteResourceName: "demo"}
	modelService.EXPECT().PrepareModelScopeRead(gomock.Any(), "team", "demo").Return(remote, nil).Times(5)

	base := "/api/v1/models/team/demo"
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, base+"/repo/files?Recursive=True", nil))
	require.Equal(t, http.StatusOK, r.Code)
	require.Contains(t, r.Body.String(), oid)
	require.False(t, h.lfsStorage.Exists(oid))

	const content = "0123456789-modelscope-lfs"
	file := base + "/repo?Revision=master&FilePath=weights.bin"
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodHead, file, nil))
	require.Equal(t, http.StatusOK, r.Code)
	require.Equal(t, fmt.Sprint(len(content)), r.Header().Get("Content-Length"))
	require.Empty(t, r.Body.String())

	gitRepo.EXPECT().OpenModelScopeFile(gomock.Any(), remote, sha, "weights.bin", "bytes=2-5").
		Return(&git.RemoteFile{Body: io.NopCloser(strings.NewReader("2345")), ContentLength: 4, ContentRange: "bytes 2-5/25", Partial: true}, nil)
	req := httptest.NewRequest(http.MethodGet, file, nil)
	req.Header.Set("Range", "bytes=2-5")
	r = httptest.NewRecorder()
	h.ServeHTTP(r, req)
	require.Equal(t, http.StatusPartialContent, r.Code)
	require.Equal(t, "2345", r.Body.String())
	require.False(t, h.lfsStorage.Exists(oid), "a range must not be cached as a complete object")

	gitRepo.EXPECT().OpenModelScopeFile(gomock.Any(), remote, sha, "weights.bin", "").
		Return(&git.RemoteFile{Body: io.NopCloser(strings.NewReader(content)), ContentLength: int64(len(content))}, nil)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, file, nil))
	require.Equal(t, http.StatusOK, r.Code)
	require.Equal(t, content, r.Body.String())
	require.True(t, h.lfsStorage.Exists(oid))

	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, file, nil))
	require.Equal(t, http.StatusOK, r.Code)
	require.Equal(t, content, r.Body.String())
}
