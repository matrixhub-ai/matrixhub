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

package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	domaingit "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/stretchr/testify/require"
)

func TestModelScopeGitAddresses(t *testing.T) {
	reg := &domaingit.GitRepository{RemoteRegistryURL: "https://www.modelscope.cn/", RemoteRegistryType: registry.TypeModelScope}
	require.Equal(t, "https://www.modelscope.cn/Qwen/demo.git", remoteRepositoryURL(reg, "Qwen/demo"))
	require.Equal(t, "https://www.modelscope.cn/datasets/team/demo.git", remoteRepositoryURL(reg, "datasets/team/demo"))
	reg.RemoteRegistryType = "REGISTRY_TYPE_HUGGINGFACE"
	require.Equal(t, "https://www.modelscope.cn/Qwen/demo", remoteRepositoryURL(reg, "Qwen/demo"))
}

func TestModelScopePingValidatesToken(t *testing.T) {
	for _, code := range []int{200, 401} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/v1/login", r.URL.Path)
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "secret", body["AccessToken"])
			require.NoError(t, json.NewEncoder(w).Encode(map[string]int{"Code": code}))
		}))
		reg := registry.Registry{Type: registry.TypeModelScope, URL: server.URL}
		reg.SetCredential(registry.NewBasicCredential("alice", "secret"))
		status, _, err := NewRegistryRepo(nil).PingRegistry(context.Background(), reg)
		require.NoError(t, err)
		require.Equal(t, code, status)
		server.Close()
	}
}

func TestModelScopePrivateGitPull(t *testing.T) {
	weights := "private-weights"
	sum := sha256.Sum256([]byte(weights))
	oid := hex.EncodeToString(sum[:])
	upstream := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	repo, err := repository.Init(t.Context(), upstream.ResolvePath("team/private"), "master")
	require.NoError(t, err)
	_, err = repo.CreateCommit(t.Context(), "master", "initial", "test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "config.json", Content: []byte(`{"private":true}`)},
		{Type: repository.CommitOperationAdd, Path: "weights.bin", Content: []byte(fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, len(weights)))},
	}, "")
	require.NoError(t, err)
	backend := backendhttp.NewHandler(backendhttp.WithStorage(upstream), backendhttp.WithNext(http.NotFoundHandler()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "secret", body["AccessToken"])
			_, _ = w.Write([]byte(`{"Code":200,"Data":{"Username":"alice","GitAccessToken":"git-secret"}}`))
			return
		}
		if r.URL.Path == "/api/v1/models/team/private/repo" {
			require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
			cookie, err := r.Cookie("m_session_id")
			require.NoError(t, err)
			require.Equal(t, "secret", cookie.Value)
			_, _ = w.Write([]byte(weights))
			return
		}
		name, password, ok := r.BasicAuth()
		if !ok || name != "alice" || password != "git-secret" {
			w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
			w.WriteHeader(401)
			return
		}
		backend.ServeHTTP(w, r)
	}))
	defer server.Close()
	local := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	m := mirror.NewMirror(mirror.WithLFSStorage(lfs.NewLocal(local.LFSDir())), mirror.WithCacheDir(t.TempDir()))
	g := NewGitDB(local, m).(*gitRepo)
	require.NoError(t, g.CreateRepository(t.Context(), "models", "local", "copy"))
	placeholder, err := repository.Open(local.ResolvePath("local/copy"))
	require.NoError(t, err)
	require.Equal(t, "main", placeholder.DefaultBranch()) // Exercise an already-cached default.

	err = g.PullFromRemote(t.Context(), &domaingit.GitRepository{RemoteRegistryURL: server.URL, RemoteRegistryType: registry.TypeModelScope, RemoteProjectName: "team", RemoteResourceName: "private", ProjectName: "local", ResourceName: "copy", ResourceType: "model", Credential: &domaingit.BasicCredential{Username: "alice", Password: "secret"}, LogWriter: io.Discard})
	require.NoError(t, err)
	cloned, err := repository.Open(local.ResolvePath("local/copy"))
	require.NoError(t, err)
	content, err := g.readBlobBytes(cloned, "master", "config.json")
	require.NoError(t, err)
	require.JSONEq(t, `{"private":true}`, string(content))
	metadata, err := g.ExtractMetadata(t.Context(), "models", "local", "copy")
	require.NoError(t, err)
	require.Equal(t, "master", metadata.DefaultBranch)
	require.JSONEq(t, `{"private":true}`, string(metadata.ConfigJSON))
	reader, err := g.openBlobContent(cloned, "master", "weights.bin")
	require.NoError(t, err)
	downloaded, err := io.ReadAll(reader)
	_ = reader.Close()
	require.NoError(t, err)
	require.Equal(t, weights, string(downloaded))
}

// Opt-in real upstream test: use a small public model to validate Git and LFS.
func TestModelScopeLiveUpstream(t *testing.T) {
	id := os.Getenv("MATRIXHUB_MODELSCOPE_MODEL")
	if id == "" {
		t.Skip("set MATRIXHUB_MODELSCOPE_MODEL for a real ModelScope pull")
	}
	parts := strings.Split(id, "/")
	require.Len(t, parts, 2)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	local := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	ls := lfs.NewLocal(local.LFSDir())
	m := mirror.NewMirror(mirror.WithLFSStorage(ls), mirror.WithCacheDir(t.TempDir()))
	g := NewGitDB(local, m).(*gitRepo)
	require.NoError(t, g.PullFromRemote(ctx, &domaingit.GitRepository{RemoteRegistryURL: "https://www.modelscope.cn", RemoteRegistryType: registry.TypeModelScope, RemoteProjectName: parts[0], RemoteResourceName: parts[1], ProjectName: "local", ResourceName: "copy", ResourceType: "model", LogWriter: io.Discard}))
	repo, err := repository.Open(local.ResolvePath("local/copy"))
	require.NoError(t, err)
	entries, err := repo.Tree("", "", &repository.TreeOptions{Recursive: true})
	require.NoError(t, err)
	files := 0
	lfsFiles := 0
	for _, entry := range entries {
		if entry.Type() != repository.EntryTypeFile {
			continue
		}
		files++
		blob, err := entry.Blob()
		require.NoError(t, err)
		pointer, _ := blob.LFSPointer()
		if pointer == nil {
			continue
		}
		lfsFiles++
		reader, err := g.openBlobContent(repo, "", entry.Path())
		require.NoError(t, err)
		hash := sha256.New()
		n, err := io.Copy(hash, reader)
		_ = reader.Close()
		require.NoError(t, err)
		require.Equal(t, pointer.Size(), n)
		require.Equal(t, pointer.OID(), hex.EncodeToString(hash.Sum(nil)))
	}
	require.Positive(t, files)
	require.Positive(t, lfsFiles)
	t.Logf("Pulled %s: %d Git files, %d LFS files with matching size and SHA256", id, files, lfsFiles)
}

func TestModelScopePullRejectsCorruptLFS(t *testing.T) {
	expected := "correct-weights-content!"
	for _, content := range []string{"truncated", strings.Repeat("x", len(expected))} {
		t.Run(content, func(t *testing.T) {
			sum := sha256.Sum256([]byte(expected))
			oid := hex.EncodeToString(sum[:])
			upstream := storage.NewStorage(storage.WithRootDir(t.TempDir()))
			repo, err := repository.Init(t.Context(), upstream.ResolvePath("team/model"), "master")
			require.NoError(t, err)
			_, err = repo.CreateCommit(t.Context(), "master", "fixture", "test", "test@example.com", []repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "weights.bin", Content: []byte(fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, len(expected)))}}, "")
			require.NoError(t, err)
			gitHandler := backendhttp.NewHandler(backendhttp.WithStorage(upstream), backendhttp.WithNext(http.NotFoundHandler()))
			fileRequested := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/models/team/model/repo" {
					select {
					case fileRequested <- struct{}{}:
					default:
					}
					require.Equal(t, http.MethodGet, r.Method)
					require.Len(t, r.URL.Query().Get("Revision"), 40)
					_, _ = w.Write([]byte(content))
					return
				}
				gitHandler.ServeHTTP(w, r)
			}))
			defer server.Close()
			local := storage.NewStorage(storage.WithRootDir(t.TempDir()))
			g := NewGitDB(local, nil)
			remote := &domaingit.GitRepository{RemoteRegistryURL: server.URL, RemoteRegistryType: registry.TypeModelScope, RemoteProjectName: "team", RemoteResourceName: "model", ProjectName: "local", ResourceName: "copy", ResourceType: "model"}
			refreshed, err := g.RefreshModelScopeRefs(t.Context(), remote)
			require.NoError(t, err)
			require.True(t, refreshed)
			refreshed, err = g.RefreshModelScopeRefs(t.Context(), remote)
			require.NoError(t, err)
			require.False(t, refreshed)
			require.Empty(t, fileRequested, "metadata refresh must not request weight bytes")
			err = g.PullFromRemote(t.Context(), remote)
			require.ErrorContains(t, err, "validate modelscope file weights.bin")
			require.False(t, lfs.NewLocal(local.LFSDir()).Exists(oid), "invalid weights must not be published to the content store")
		})
	}
}

func TestModelScopeLoginRejectsMissingGitToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"Code":200,"Data":{}}`)) }))
	defer server.Close()
	original := &domaingit.GitRepository{RemoteRegistryURL: server.URL, Credential: &domaingit.BasicCredential{Username: "alice", Password: "sdk-token"}}
	_, err := modelScopeGitConfig(t.Context(), original)
	require.ErrorContains(t, err, "did not return a Git token")
	require.Equal(t, "sdk-token", original.Credential.Password)
}
