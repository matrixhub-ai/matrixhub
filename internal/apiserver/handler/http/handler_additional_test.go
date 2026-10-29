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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"

	backend "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/http"
)

func TestHTTPHandlerOptionsAndFallback(t *testing.T) {
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := backend.NewHandler(
		backend.WithMiddlewares(middleware),
		backend.WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
		backend.WithNext(next),
		backend.WithPermissionHookFunc(nil),
		backend.WithPreReceiveHookFunc(nil),
		backend.WithPostReceiveHookFunc(nil),
		backend.WithMirror(nil),
		backend.WithServices(nil, nil, nil),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/not-git", nil))
	require.Equal(t, http.StatusTeapot, response.Code)

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/org/repo.git/info/refs", nil))
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, "true", response.Header().Get("X-Test-Middleware"))
}

func TestHTTPHandlerRejectsInvalidDiscoveryRequests(t *testing.T) {
	handler := backend.NewHandler(
		backend.WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
	)
	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
	}{
		{name: "missing service", method: http.MethodGet, target: "/org/repo.git/info/refs", wantStatus: http.StatusBadRequest, wantBody: "service parameter is required"},
		{name: "unsupported service", method: http.MethodGet, target: "/org/repo.git/info/refs?service=git-archive", wantStatus: http.StatusForbidden, wantBody: "unsupported service"},
		{name: "missing repository info refs", method: http.MethodGet, target: "/org/repo.git/info/refs?service=git-upload-pack", wantStatus: http.StatusNotFound, wantBody: "not found"},
		{name: "missing repository upload", method: http.MethodPost, target: "/org/repo.git/git-upload-pack", wantStatus: http.StatusNotFound, wantBody: "not found"},
		{name: "missing repository receive", method: http.MethodPost, target: "/org/repo.git/git-receive-pack", wantStatus: http.StatusNotFound, wantBody: "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.target, nil))
			require.Equal(t, tt.wantStatus, response.Code)
			require.Contains(t, response.Body.String(), tt.wantBody)
		})
	}
}

func TestHTTPHandlerPermissionFailures(t *testing.T) {
	hookError := errors.New("permission backend unavailable")
	tests := []struct {
		name       string
		user       string
		allowed    bool
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "authenticated denied", user: "alice", wantStatus: http.StatusForbidden, wantBody: "permission denied"},
		{name: "anonymous denied", user: authenticate.Anonymous, wantStatus: http.StatusUnauthorized, wantBody: "Unauthorized"},
		{name: "hook error", user: "alice", err: hookError, wantStatus: http.StatusInternalServerError, wantBody: hookError.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := backend.NewHandler(
				backend.WithStorage(storage.NewStorage(storage.WithRootDir(t.TempDir()))),
				backend.WithPermissionHookFunc(func(_ context.Context, operation permission.Operation, repo string, _ permission.Context) (bool, error) {
					require.Equal(t, permission.OperationReadRepo, operation)
					require.Equal(t, "datasets/org/repo", repo)
					return tt.allowed, tt.err
				}),
			)
			request := httptest.NewRequest(http.MethodGet, "/datasets/org/repo.git/info/refs?service=git-upload-pack", nil)
			request = request.WithContext(authenticate.WithContext(request.Context(), authenticate.UserInfo{User: tt.user}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tt.wantStatus, response.Code)
			require.Contains(t, response.Body.String(), tt.wantBody)
		})
	}
}
