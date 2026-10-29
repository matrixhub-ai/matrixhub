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

package lfs_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	hfdlfs "github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"

	backend "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/lfs"
)

const lfsMetaMediaType = "application/vnd.git-lfs+json"

func newLFSHandler(t *testing.T, opts ...backend.Option) *backend.Handler {
	t.Helper()
	dataDir := t.TempDir()
	base := []backend.Option{
		backend.WithStorage(storage.NewStorage(storage.WithRootDir(dataDir))),
		backend.WithLFSStorage(hfdlfs.NewLocal(dataDir)),
	}
	return backend.NewHandler(append(base, opts...)...)
}

func lfsRequest(t *testing.T, handler http.Handler, method, target, body, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Accept", lfsMetaMediaType)
	if user != "" {
		req = req.WithContext(authenticate.WithContext(req.Context(), authenticate.UserInfo{User: user}))
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func objectOID(content []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func TestLFSObjectLifecycle(t *testing.T) {
	handler := newLFSHandler(t)
	content := []byte("matrixhub-lfs-content")
	oid := objectOID(content)

	batchBody := fmt.Sprintf(`{"operation":"upload","objects":[{"oid":%q,"size":%d}]}`, oid, len(content))
	response := lfsRequest(t, handler, http.MethodPost, "http://example.com/org/repo.git/info/lfs/objects/batch", batchBody, "")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, lfsMetaMediaType, response.Header().Get("Content-Type"))
	var batch struct {
		Objects []struct {
			OID     string `json:"oid"`
			Actions map[string]struct {
				Href string `json:"href"`
			} `json:"actions"`
		} `json:"objects"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &batch))
	require.Len(t, batch.Objects, 1)
	require.Equal(t, oid, batch.Objects[0].OID)
	require.Equal(t, "http://example.com/objects/"+oid, batch.Objects[0].Actions["upload"].Href)
	require.Equal(t, "http://example.com/objects/"+oid+"/verify", batch.Objects[0].Actions["verify"].Href)

	put := httptest.NewRequest(http.MethodPut, "/objects/"+oid, bytes.NewReader(content))
	put.ContentLength = int64(len(content))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, put)
	require.Equal(t, http.StatusOK, response.Code)

	verifyBody := fmt.Sprintf(`{"oid":%q,"size":%d}`, oid, len(content))
	response = lfsRequest(t, handler, http.MethodPost, "/objects/"+oid+"/verify", verifyBody, "")
	require.Equal(t, http.StatusOK, response.Code)

	response = lfsRequest(t, handler, http.MethodGet, "/objects/"+oid, "", "")
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, content, response.Body.Bytes())
	require.Equal(t, `"`+oid+`"`, response.Header().Get("ETag"))

	batchBody = fmt.Sprintf(`{"operation":"download","objects":[{"oid":%q,"size":%d}]}`, oid, len(content))
	response = lfsRequest(t, handler, http.MethodPost, "https://example.com/org/repo.git/info/lfs/objects/batch", batchBody, "")
	require.Equal(t, http.StatusOK, response.Code)
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &batch))
	require.Equal(t, "https://example.com/objects/"+oid, batch.Objects[0].Actions["download"].Href)
}

func TestLFSObjectErrorsAndSignedActions(t *testing.T) {
	signer := authenticate.NewTokenSignValidator([]byte("unit-test-key"))
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "true")
			next.ServeHTTP(w, r)
		})
	}
	handler := newLFSHandler(t,
		backend.WithMiddlewares(middleware),
		backend.WithNext(next),
		backend.WithTokenSignValidator(signer),
		backend.WithMirror(nil),
	)

	response := lfsRequest(t, handler, http.MethodGet, "/not-lfs", "", "")
	require.Equal(t, http.StatusTeapot, response.Code)

	missingOID := strings.Repeat("a", 64)
	response = lfsRequest(t, handler, http.MethodGet, "/objects/"+missingOID, "", "")
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, "true", response.Header().Get("X-Test-Middleware"))
	require.Contains(t, response.Body.String(), "not found")

	response = lfsRequest(t, handler, http.MethodPost, "/objects/"+missingOID+"/verify", `{"oid":"`+missingOID+`","size":1}`, "")
	require.Equal(t, http.StatusNotFound, response.Code)

	batchBody := `{"operation":"upload","objects":[{"oid":"` + missingOID + `","size":1}]}`
	response = lfsRequest(t, handler, http.MethodPost, "http://example.com/org/repo.git/info/lfs/objects/batch", batchBody, "alice")
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"Authorization":"Bearer `)

	batchBody = `{"operation":"download","objects":[{"oid":"` + missingOID + `","size":1}]}`
	response = lfsRequest(t, handler, http.MethodPost, "http://example.com/org/repo.git/info/lfs/objects/batch", batchBody, "alice")
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"code":404`)
}

func TestLFSPermissionChecks(t *testing.T) {
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
			var gotOperation permission.Operation
			handler := newLFSHandler(t, backend.WithPermissionHookFunc(func(_ context.Context, op permission.Operation, repo string, _ permission.Context) (bool, error) {
				gotOperation = op
				require.Equal(t, "org/repo", repo)
				return tt.allowed, tt.err
			}))
			body := `{"operation":"upload","objects":[{"oid":"abc","size":1}]}`
			response := lfsRequest(t, handler, http.MethodPost, "http://example.com/org/repo.git/info/lfs/objects/batch", body, "")
			require.Equal(t, tt.wantStatus, response.Code)
			require.Contains(t, response.Body.String(), tt.wantBody)
			require.Equal(t, permission.OperationUpdateRepo, gotOperation)
		})
	}
}

func TestLFSLockLifecycle(t *testing.T) {
	handler := newLFSHandler(t)
	create := lfsRequest(t, handler, http.MethodPost, "/org/repo.git/locks", `{"path":"weights.bin"}`, "alice")
	require.Equal(t, http.StatusCreated, create.Code)
	var created struct {
		Lock struct {
			ID    string `json:"id"`
			Path  string `json:"path"`
			Owner struct {
				Name string `json:"name"`
			} `json:"owner"`
		} `json:"lock"`
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	require.Len(t, created.Lock.ID, 40)
	require.Equal(t, "weights.bin", created.Lock.Path)
	require.Equal(t, "alice", created.Lock.Owner.Name)

	duplicate := lfsRequest(t, handler, http.MethodPost, "/org/repo.git/locks", `{"path":"weights.bin"}`, "alice")
	require.Equal(t, http.StatusConflict, duplicate.Code)

	list := lfsRequest(t, handler, http.MethodGet, "/org/repo.git/locks?path=weights.bin&limit=1", "", "alice")
	require.Equal(t, http.StatusOK, list.Code)
	require.Contains(t, list.Body.String(), created.Lock.ID)

	invalidLimit := lfsRequest(t, handler, http.MethodGet, "/org/repo.git/locks?limit=-1", "", "alice")
	require.Equal(t, http.StatusBadRequest, invalidLimit.Code)

	verify := lfsRequest(t, handler, http.MethodPost, "/org/repo.git/locks/verify", `{}`, "alice")
	require.Equal(t, http.StatusOK, verify.Code)
	require.Contains(t, verify.Body.String(), `"ours"`)

	denied := lfsRequest(t, handler, http.MethodPost, "/org/repo.git/locks/"+created.Lock.ID+"/unlock", `{"force":false}`, "bob")
	require.Equal(t, http.StatusForbidden, denied.Code)

	deleted := lfsRequest(t, handler, http.MethodPost, "/org/repo.git/locks/"+created.Lock.ID+"/unlock", `{"force":false}`, "alice")
	require.Equal(t, http.StatusOK, deleted.Code)
	require.Contains(t, deleted.Body.String(), created.Lock.ID)
}

func TestLFSRejectsMalformedLockRequests(t *testing.T) {
	handler := newLFSHandler(t)
	for _, target := range []string{"/org/repo.git/locks", "/org/repo.git/locks/verify"} {
		response := lfsRequest(t, handler, http.MethodPost, target, `{`, "alice")
		require.Equal(t, http.StatusBadRequest, response.Code)
	}
}
