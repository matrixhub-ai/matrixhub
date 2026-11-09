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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
)

func TestModelScopeColdDownloadsValidateUpstreamAndCache(t *testing.T) {
	fixture, sha, oid := modelScopeFixture(t)
	const content = "0123456789-modelscope-lfs"
	for _, tc := range []struct {
		name, content, byteRange, ifRange, upstreamRange, contentRange string
		length                                                         int64
		partial, cached, readError, openError                          bool
		status                                                         int
	}{
		{name: "matching If-Range", content: "2345", byteRange: "bytes=2-5", ifRange: "match", upstreamRange: "bytes=2-5", contentRange: "bytes 2-5/25", partial: true, length: 4, status: 206},
		{name: "stale If-Range gets full file", content: content, byteRange: "bytes=2-5", ifRange: `"old-etag"`, length: 25, cached: true, status: 200},
		{name: "upstream ignores Range", content: content, byteRange: "bytes=2-5", upstreamRange: "bytes=2-5", length: 25, cached: true, status: 200},
		{name: "unknown upstream length", content: content, length: -1, cached: true, status: 200},
		{name: "malformed range", partial: true, contentRange: "invalid", length: -1, status: 502},
		{name: "range exceeds file", partial: true, contentRange: "bytes 0-25/25", length: 26, status: 502},
		{name: "wrong range total", partial: true, contentRange: "bytes 2-5/100", length: 4, status: 502},
		{name: "wrong content length", content: content, length: 10, status: 502},
		{name: "corrupt weights", content: strings.Repeat("x", len(content)), length: 25, status: 200},
		{name: "truncated weights", content: "012", length: 25, status: 200},
		{name: "stream interrupted", content: "012", length: 25, readError: true, status: 200},
		{name: "upstream unavailable", openError: true, status: 502},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(WithStorage(fixture.storage), WithLFSStorage(lfs.NewLocal(t.TempDir())), WithNext(http.NotFoundHandler()))
			ctrl := gomock.NewController(t)
			models := modelmocks.NewMockIModelService(ctrl)
			gitRepo := gitmocks.NewMockIGitRepo(ctrl)
			h.modelService, h.gitRepo = models, gitRepo
			remote := &git.GitRepository{RemoteProjectName: "team", RemoteResourceName: "demo"}
			models.EXPECT().PrepareModelScopeRead(gomock.Any(), "team", "demo").Return(remote, nil)
			open := gitRepo.EXPECT().OpenModelScopeFile(gomock.Any(), remote, sha, "weights.bin", tc.upstreamRange)
			if tc.openError {
				open.Return(nil, errors.New("upstream unavailable"))
			} else {
				var reader io.Reader = strings.NewReader(tc.content)
				if tc.readError {
					reader = io.MultiReader(reader, iotest.ErrReader(errors.New("connection lost")))
				}
				open.Return(&git.RemoteFile{Body: io.NopCloser(reader), ContentLength: tc.length, ContentRange: tc.contentRange, Partial: tc.partial}, nil)
			}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo/repo?FilePath=weights.bin", nil)
			request.Header.Set("Range", tc.byteRange)
			if tc.ifRange == "match" {
				request.Header.Set("If-Range", `"`+oid+`"`)
			} else {
				request.Header.Set("If-Range", tc.ifRange)
			}
			r := httptest.NewRecorder()
			h.ServeHTTP(r, request)
			require.Equal(t, tc.status, r.Code, r.Body.String())
			require.Equal(t, tc.cached, h.lfsStorage.Exists(oid))
			if tc.status < 400 {
				require.Equal(t, tc.content, r.Body.String())
				require.Equal(t, `"`+oid+`"`, r.Header().Get("ETag"))
				require.Equal(t, sha, r.Header().Get("X-Repo-Commit"))
			} else {
				require.NotContains(t, r.Body.String(), content)
			}
		})
	}
}

func TestModelScopeRepositoryAndListingFailures(t *testing.T) {
	fixture, _, _ := modelScopeFixture(t)
	newHandler := func() *Handler {
		return NewHandler(WithStorage(fixture.storage), WithLFSStorage(fixture.lfsStorage), WithNext(http.NotFoundHandler()))
	}
	t.Run("authorization service fails", func(t *testing.T) {
		h := newHandler()
		h.permissionHookFunc = func(context.Context, permission.Operation, string, permission.Context) (bool, error) {
			return false, errors.New("private authorization details")
		}
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo", nil))
		require.Equal(t, 500, r.Code)
		require.NotContains(t, r.Body.String(), "private authorization details")
	})
	t.Run("upstream preparation fails", func(t *testing.T) {
		h := newHandler()
		ctrl := gomock.NewController(t)
		models := modelmocks.NewMockIModelService(ctrl)
		h.modelService = models
		models.EXPECT().PrepareModelScopeRead(gomock.Any(), "team", "demo").Return(nil, errors.New("private upstream details"))
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo", nil))
		require.Equal(t, 502, r.Code)
		require.NotContains(t, r.Body.String(), "private upstream details")
	})
	t.Run("missing repository", func(t *testing.T) {
		h := newHandler()
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/missing", nil))
		require.Equal(t, 404, r.Code)
	})
	for _, tc := range []struct {
		query  string
		status int
	}{
		{query: "Root=../secret", status: 400},
		{query: "Root=does-not-exist", status: 404},
	} {
		t.Run(tc.query, func(t *testing.T) {
			h := newHandler()
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo/repo/files?"+tc.query, nil))
			require.Equal(t, tc.status, r.Code)
		})
	}
}

func TestModelScopeInfoAndExistingMasterBranch(t *testing.T) {
	h, sha, _ := modelScopeFixture(t)
	repo, err := repository.Open(h.storage.ResolvePath("team/demo"))
	require.NoError(t, err)
	require.NoError(t, repo.CreateBranch("master", sha))
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo", nil))
	require.Equal(t, 200, r.Code)
	require.JSONEq(t, `{"Code":200,"Success":true,"Message":"","Data":{"Name":"demo","Path":"team","ModelId":"team/demo"}}`, r.Body.String())
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/models/team/demo/revisions", nil))
	require.Equal(t, 200, r.Code)
	require.Equal(t, 1, strings.Count(r.Body.String(), `"Revision":"master"`))
}
