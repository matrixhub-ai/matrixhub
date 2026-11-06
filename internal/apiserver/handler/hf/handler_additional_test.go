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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	authzmocks "github.com/matrixhub-ai/matrixhub/internal/domain/authz/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
)

func doHFRequest(t *testing.T, client *http.Client, method, url, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, url, strings.NewReader(body))
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	return response
}

func TestHuggingFaceTree(t *testing.T) {
	server, _ := setupTestServer(t)
	createRepoAndCommit(t, server.URL, "model", "test-user", "tree-model")

	response := doHFRequest(t, server.Client(), http.MethodGet,
		server.URL+"/api/models/test-user/tree-model/tree/main?recursive=true&expand=true", "")
	require.Equal(t, http.StatusOK, response.StatusCode)

	var entries []treeEntry
	require.NoError(t, json.NewDecoder(response.Body).Decode(&entries))
	require.NotEmpty(t, entries)
	var readme *treeEntry
	for i := range entries {
		if entries[i].Path == "README.md" {
			readme = &entries[i]
			break
		}
	}
	require.NotNil(t, readme)
	require.NotEmpty(t, readme.OID)
	require.Positive(t, readme.Size)
	require.NotNil(t, readme.LastCommit)
}

func TestHuggingFaceValidateYAML(t *testing.T) {
	server, _ := setupTestServer(t)
	response := doHFRequest(t, server.Client(), http.MethodPost, server.URL+"/api/validate-yaml", "---\nlicense: apache-2.0\n---")
	require.Equal(t, http.StatusOK, response.StatusCode)
	var result struct {
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	require.Empty(t, result.Errors)
	require.Empty(t, result.Warnings)
}

func TestHuggingFaceHandlerOptionsAndFallback(t *testing.T) {
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := NewHandler(
		WithMiddlewares(middleware),
		WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
		WithNext(next),
		WithPermissionHookFunc(nil),
		WithPreReceiveHookFunc(nil),
		WithPostReceiveHookFunc(nil),
		WithLFSStorage(nil),
		WithMirror(nil),
		WithServices(nil, nil, nil),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/not-hugging-face", nil))
	require.Equal(t, http.StatusTeapot, response.Code)

	response = httptest.NewRecorder()
	handler.Router().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/validate-yaml", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "true", response.Header().Get("X-Test-Middleware"))
}

func TestHuggingFaceReadPermissionFailures(t *testing.T) {
	tests := []struct {
		name       string
		allowed    bool
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "denied", wantStatus: http.StatusForbidden, wantBody: "permission denied"},
		{name: "hook error", err: errors.New("permission backend unavailable"), wantStatus: http.StatusInternalServerError, wantBody: "permission backend unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(
				WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
				WithPermissionHookFunc(func(_ context.Context, operation permission.Operation, repo string, _ permission.Context) (bool, error) {
					require.Equal(t, permission.OperationReadRepo, operation)
					require.Equal(t, "org/model", repo)
					return tt.allowed, tt.err
				}),
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/models/org/model/tree/main", nil))
			require.Equal(t, tt.wantStatus, response.Code)
			require.Contains(t, response.Body.String(), tt.wantBody)
		})
	}
}

func TestHuggingFaceRejectsMalformedRequests(t *testing.T) {
	server, _ := setupTestServer(t)
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "create repository", method: http.MethodPost, path: "/api/repos/create", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "delete repository", method: http.MethodDelete, path: "/api/repos/delete", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "move repository", method: http.MethodPost, path: "/api/repos/move", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "preupload", method: http.MethodPost, path: "/api/models/org/missing/preupload/main", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "commit ndjson", method: http.MethodPost, path: "/api/models/org/missing/commit/main", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing tree", method: http.MethodGet, path: "/api/models/org/missing/tree/main", wantStatus: http.StatusNotFound},
		{name: "missing resolve", method: http.MethodGet, path: "/org/missing/resolve/main/README.md", wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := doHFRequest(t, server.Client(), tt.method, server.URL+tt.path, tt.body)
			require.Equal(t, tt.wantStatus, response.StatusCode)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NotEmpty(t, body)
		})
	}
}

func TestHuggingFaceRejectsInvalidRefsAndSettings(t *testing.T) {
	server, _ := setupTestServer(t)
	createRepoAndCommit(t, server.URL, "model", "test-user", "invalid-ref-model")

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "malformed settings", method: http.MethodPut, path: "/api/models/test-user/invalid-ref-model/settings", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "existing branch", method: http.MethodPost, path: "/api/models/test-user/invalid-ref-model/branch/main", body: `{}`, wantStatus: http.StatusConflict},
		{name: "missing branch", method: http.MethodDelete, path: "/api/models/test-user/invalid-ref-model/branch/missing", body: `{}`, wantStatus: http.StatusNotFound},
		{name: "tag without name", method: http.MethodPost, path: "/api/models/test-user/invalid-ref-model/tag/main", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "missing tag", method: http.MethodDelete, path: "/api/models/test-user/invalid-ref-model/tag/missing", body: `{}`, wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := doHFRequest(t, server.Client(), tt.method, server.URL+tt.path, tt.body)
			require.Equal(t, tt.wantStatus, response.StatusCode)
		})
	}
}

func TestHuggingFaceRejectsWritesToProxyProject(t *testing.T) {
	proxyReadOnly := func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
		return false, project.ErrProxyProjectReadOnly
	}

	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "preupload", method: http.MethodPost, target: "/api/models/proxy/model/preupload/main", body: `{"files":[]}`},
		{name: "commit", method: http.MethodPost, target: "/api/models/proxy/model/commit/main", body: `{}`},
		{name: "super squash", method: http.MethodPost, target: "/api/models/proxy/model/super-squash/main", body: `{}`},
		{name: "settings", method: http.MethodPut, target: "/api/models/proxy/model/settings", body: `{}`},
		{name: "create branch", method: http.MethodPost, target: "/api/models/proxy/model/branch/feature", body: `{}`},
		{name: "delete branch", method: http.MethodDelete, target: "/api/models/proxy/model/branch/feature"},
		{name: "create tag", method: http.MethodPost, target: "/api/models/proxy/model/tag/main", body: `{"tag":"v1"}`},
		{name: "delete tag", method: http.MethodDelete, target: "/api/models/proxy/model/tag/v1"},
		{name: "move", method: http.MethodPost, target: "/api/repos/move", body: `{"fromRepo":"local/model","toRepo":"proxy/model"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(
				WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
				WithPermissionHookFunc(proxyReadOnly),
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body)))
			require.Equal(t, http.StatusForbidden, response.Code)
			require.Contains(t, response.Body.String(), project.ErrProxyProjectReadOnly.Error())
		})
	}

	// create_repo(exist_ok=True) answers a 403 by reading the repo, which starts
	// an upstream sync, so create-repo has to fail with another status.
	t.Run("create repo", func(t *testing.T) {
		authzService := authzmocks.NewMockIAuthzService(gomock.NewController(t))
		authzService.EXPECT().VerifyProjectPermissionByName(gomock.Any(), "proxy", gomock.Any()).Return(true, nil)
		handler := NewHandler(
			WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
			WithServices(nil, nil, authzService),
			WithPermissionHookFunc(proxyReadOnly),
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/repos/create", strings.NewReader(`{"name":"model","organization":"proxy"}`)))
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.Contains(t, response.Body.String(), project.ErrProxyProjectReadOnly.Error())
	})
}
