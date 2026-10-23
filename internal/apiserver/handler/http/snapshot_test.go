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
package backend_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/storage"
	backend "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/http"
	domainGit "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
	"github.com/stretchr/testify/require"
)

func transportGit(t *testing.T, path string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", path}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	return strings.TrimSpace(string(output))
}

func TestGitSnapshotHTTPCloneDoesNotReadLaterPush(t *testing.T) {
	for _, version := range []string{"0", "2"} {
		t.Run("protocol-"+version, func(t *testing.T) { snapshotHTTPClone(t, version) })
	}
}
func snapshotHTTPClone(t *testing.T, version string) {
	store := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	original := store.ResolvePath("tester/model")
	require.NoError(t, os.MkdirAll(original, 0700))
	transportGit(t, original, "init", "-b", "main")
	transportGit(t, original, "config", "user.name", "Snapshot test")
	transportGit(t, original, "config", "user.email", "snapshot@example.invalid")
	require.NoError(t, os.WriteFile(filepath.Join(original, "config.json"), []byte("approved\n"), 0600))
	transportGit(t, original, "add", ".")
	transportGit(t, original, "commit", "-m", "approved")
	approved := transportGit(t, original, "rev-parse", "HEAD")
	transportGit(t, original, "tag", "-a", "v1", "-m", "approved tag")
	temporary := t.TempDir()
	var captures, releases atomic.Int32
	mutateAt := int32(2)
	if version == "2" {
		mutateAt = 3
	} // v2 has a separate ls-refs RPC.
	server := httptest.NewServer(backend.NewHandler(backend.WithStorage(store), backend.WithReadSnapshot(
		func(ctx context.Context, name, path string) (string, func(), error) {
			if name != "tester/model" {
				return "", nil, domainGit.ErrReadNotAdmitted
			}
			snapshot, release, err := utils.GitReadSnapshot(ctx, path, temporary)
			if err != nil {
				return "", nil, err
			}
			revisions, err := repo.ArtifactRevisionsAtPath(ctx, snapshot, 256)
			if err != nil {
				release()
				return "", nil, err
			}
			if len(revisions) != 1 || revisions[0] != approved {
				release()
				return "", nil, domainGit.ErrReadNotAdmitted
			}
			// The RPC snapshot is already admitted. A writer now advances live refs.
			if captures.Add(1) == mutateAt {
				require.NoError(t, os.WriteFile(filepath.Join(original, "config.json"), []byte("unscanned update\n"), 0600))
				transportGit(t, original, "add", ".")
				transportGit(t, original, "commit", "-m", "unscanned")
			}
			return snapshot, func() { release(); releases.Add(1) }, nil
		})))
	defer server.Close()
	client := filepath.Join(t.TempDir(), "client")
	transportGit(t, t.TempDir(), "-c", "protocol.version="+version, "clone", server.URL+"/tester/model.git", client)
	require.GreaterOrEqual(t, captures.Load(), mutateAt)
	require.Equal(t, approved, transportGit(t, client, "rev-parse", "HEAD"))
	contents, err := os.ReadFile(filepath.Join(client, "config.json"))
	require.NoError(t, err)
	require.Equal(t, "approved\n", string(contents))
	require.NotEqual(t, approved, transportGit(t, original, "rev-parse", "HEAD"))
	response, err := http.Get(server.URL + "/tester/model.git/info/refs?service=git-upload-pack")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.Equal(t, captures.Load(), releases.Load())
	entries, err := os.ReadDir(temporary)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestGitSnapshotHTTPBothReadEndpointsFailClosed(t *testing.T) {
	store := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	var calls atomic.Int32
	server := httptest.NewServer(backend.NewHandler(backend.WithStorage(store), backend.WithReadSnapshot(
		func(context.Context, string, string) (string, func(), error) {
			calls.Add(1)
			return "", nil, domainGit.ErrReadNotAdmitted
		})))
	defer server.Close()
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodGet, "/tester/model.git/info/refs?service=git-upload-pack"},
		{http.MethodPost, "/tester/model.git/git-upload-pack"},
	} {
		request, err := http.NewRequest(endpoint.method, server.URL+endpoint.path, strings.NewReader("0000"))
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, response.StatusCode)
		require.NotContains(t, string(data), "PACK")
	}
	require.Equal(t, int32(2), calls.Load())
}
