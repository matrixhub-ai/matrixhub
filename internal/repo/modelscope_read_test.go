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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gitHTTP "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"

	domaingit "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
)

func modelScopeRemote(endpoint string) *domaingit.GitRepository {
	return &domaingit.GitRepository{
		RemoteRegistryURL: endpoint, RemoteRegistryType: registry.TypeModelScope, RemoteProjectName: "team", RemoteResourceName: "demo",
		ProjectName: "local", ResourceName: "demo", ResourceType: "model",
	}
}

func TestModelScopeFileReadsPreserveRangeAndAuthentication(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		byteRange string
		wantErr   string
	}{
		{name: "full file", status: 200},
		{name: "byte range", status: 206, byteRange: "bytes=2-5"},
		{name: "upstream ignores range", status: 200, byteRange: "bytes=2-5"},
		{name: "unsolicited partial", status: 206, wantErr: "HTTP 206"},
		{name: "missing file", status: 404, wantErr: "HTTP 404"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/v1/models/team/demo/repo", r.URL.Path)
				require.Equal(t, "nested/model.bin", r.URL.Query().Get("FilePath"))
				require.Equal(t, "v1", r.URL.Query().Get("Revision"))
				require.Equal(t, tc.byteRange, r.Header.Get("Range"))
				require.Equal(t, "identity", r.Header.Get("Accept-Encoding"))
				require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
				cookie, err := r.Cookie("m_session_id")
				require.NoError(t, err)
				require.Equal(t, "secret", cookie.Value)
				if tc.status == 206 {
					w.Header().Set("Content-Range", "bytes 2-5/10")
				}
				w.Header().Set("Content-Length", "4")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("2345"))
			}))
			defer server.Close()
			remote := modelScopeRemote(server.URL)
			remote.Credential = &domaingit.BasicCredential{Password: "secret"}
			g := NewGitDB(nil, nil)
			file, err := g.OpenModelScopeFile(t.Context(), remote, "v1", "nested/model.bin", tc.byteRange)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, file)
				return
			}
			require.NoError(t, err)
			defer func() { _ = file.Body.Close() }()
			body, err := io.ReadAll(file.Body)
			require.NoError(t, err)
			require.Equal(t, "2345", string(body))
			require.Equal(t, int64(4), file.ContentLength)
			require.Equal(t, tc.status == 206, file.Partial)
			if file.Partial {
				require.Equal(t, "bytes 2-5/10", file.ContentRange)
			}
		})
	}
}

func TestModelScopeRedirectsStripCredentialsAcrossOrigins(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("Cookie"))
		_, _ = w.Write([]byte("weights"))
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Contains(t, r.Header.Get("Cookie"), "m_session_id=secret")
		if r.URL.Path == "/same-origin" {
			http.Redirect(w, r, target.URL, http.StatusFound)
		} else {
			http.Redirect(w, r, "/same-origin", http.StatusFound)
		}
	}))
	defer source.Close()
	remote := modelScopeRemote(source.URL)
	remote.Credential = &domaingit.BasicCredential{Password: "secret"}
	file, err := NewGitDB(nil, nil).OpenModelScopeFile(t.Context(), remote, "master", "weights.bin", "")
	require.NoError(t, err)
	defer func() { _ = file.Body.Close() }()
	body, err := io.ReadAll(file.Body)
	require.NoError(t, err)
	require.Equal(t, "weights", string(body))
}

func TestModelScopeFileReadsFailOnCancellationAndRedirectLoops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer server.Close()
	g := NewGitDB(nil, nil)
	remote := modelScopeRemote(server.URL)
	_, err := g.OpenModelScopeFile(t.Context(), remote, "master", "weights.bin", "")
	require.ErrorContains(t, err, "too many modelscope download redirects")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = g.OpenModelScopeFile(ctx, remote, "master", "weights.bin", "")
	require.ErrorIs(t, err, context.Canceled)
}

func TestModelScopeRejectsInvalidEndpointsAndRepositoryPaths(t *testing.T) {
	for _, endpoint := range []string{"ssh://host", "file:///tmp/repo", "https://user:secret@host", "https://host?token=secret", "https://host/#fragment", "http://", "://invalid"} {
		remote := modelScopeRemote(endpoint)
		_, err := NewGitDB(nil, nil).OpenModelScopeFile(t.Context(), remote, "master", "config.json", "")
		require.ErrorContains(t, err, "HTTP(S) registry base URL", endpoint)
	}
	for _, name := range []string{"", ".", "..", "../escape", "nested/name", "nested\\name", "bad\x00name"} {
		for _, field := range []string{"local project", "local model", "remote project", "remote model"} {
			remote := modelScopeRemote("https://modelscope.example")
			switch field {
			case "local project":
				remote.ProjectName = name
			case "local model":
				remote.ResourceName = name
			case "remote project":
				remote.RemoteProjectName = name
			case "remote model":
				remote.RemoteResourceName = name
			}
			g := NewGitDB(nil, nil)
			require.ErrorContains(t, g.PullFromRemote(t.Context(), remote), "invalid modelscope repository path")
			_, err := g.RefreshModelScopeRefs(t.Context(), remote)
			require.ErrorContains(t, err, "invalid modelscope repository path")
		}
	}
	remote := modelScopeRemote("https://modelscope.example")
	remote.ResourceType = "dataset"
	_, err := NewGitDB(nil, nil).OpenModelScopeFile(t.Context(), remote, "master", "config.json", "")
	require.ErrorContains(t, err, "models only")
}

func TestModelScopeRepositoryWaitsRespectCancellation(t *testing.T) {
	g := NewGitDB(storage.NewStorage(storage.WithRootDir(t.TempDir())), nil).(*gitRepo)
	remote := modelScopeRemote("https://modelscope.example")
	path := g.gitPath(remote.ResourceType, remote.ProjectName, remote.ResourceName)
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	g.modelScopeLocks.Store(path, gate)
	g.modelScopeRefStates.Store(path, &modelScopeRefState{gate: gate})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, g.pullModelScope(ctx, remote), context.Canceled)
	_, err := g.RefreshModelScopeRefs(ctx, remote)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, g.pullModelScopeFiles(ctx, remote, path), context.Canceled)
}

func TestModelScopeGitLoginFailuresAndUsernameFallback(t *testing.T) {
	for _, tc := range []struct {
		name, body, wantErr string
		status              int
	}{
		{name: "unauthorized", status: 401, wantErr: "HTTP 401"},
		{name: "redirect", status: 302, wantErr: "HTTP 302"},
		{name: "invalid response", status: 200, body: "<html>login</html>", wantErr: "invalid modelscope authentication response"},
		{name: "API rejects token", status: 200, body: `{"Code":401,"Data":{"GitAccessToken":"unused"}}`, wantErr: "did not return a Git token"},
		{name: "username fallback", status: 200, body: `{"Code":200,"Data":{"GitAccessToken":"git-token"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Location", "/must-not-follow")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			remote := modelScopeRemote(server.URL)
			remote.Credential = &domaingit.BasicCredential{Username: "alice", Password: "sdk-token"}
			converted, err := modelScopeGitConfig(t.Context(), remote)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, &domaingit.BasicCredential{Username: "alice", Password: "git-token"}, converted.Credential)
			}
			require.Equal(t, "sdk-token", remote.Credential.Password, "HTTP reads must retain the SDK token")
		})
	}
	remote := modelScopeRemote("https://modelscope.example")
	remote.Credential = &domaingit.BasicCredential{Password: "git-token"}
	req := httptest.NewRequest(http.MethodGet, "https://modelscope.example", nil)
	modelScopeGitAuth(remote).(*gitHTTP.BasicAuth).SetAuth(req)
	name, token, ok := req.BasicAuth()
	require.True(t, ok)
	require.Equal(t, "oauth2", name)
	require.Equal(t, "git-token", token)
}
