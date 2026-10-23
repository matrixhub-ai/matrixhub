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

package handler

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	modelv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	authzmocks "github.com/matrixhub-ai/matrixhub/internal/domain/authz/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/modelcache"
	modelcachemocks "github.com/matrixhub-ai/matrixhub/internal/domain/modelcache/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

// cacheHandlerFixture holds strict mocks: a permission check or a cache call
// that a test does not expect fails it.
type cacheHandlerFixture struct {
	handler *ModelHandler
	authz   *authzmocks.MockIAuthzService
	cache   *modelcachemocks.MockIModelCacheService
}

func newCacheHandler(t *testing.T) *cacheHandlerFixture {
	ctrl := gomock.NewController(t)
	f := &cacheHandlerFixture{
		authz: authzmocks.NewMockIAuthzService(ctrl),
		cache: modelcachemocks.NewMockIModelCacheService(ctrl),
	}
	f.handler = &ModelHandler{cache: f.cache, authzService: f.authz}
	return f
}

// expectPermission expects one check of perm on project "proj".
func (f *cacheHandlerFixture) expectPermission(perm role.Permission, allowed bool) {
	f.authz.EXPECT().VerifyProjectPermissionByName(gomock.Any(), "proj", perm).Return(allowed, nil)
}

var missingStatus = &modelcache.CacheStatus{Project: "proj", Name: "model", State: modelcache.CacheStateMissing}

func TestCacheModelNeedsPushPermission(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			f := newCacheHandler(t)
			f.expectPermission(role.ModelPush, false)

			_, err := f.handler.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{Project: "proj", Name: "model", DryRun: dryRun})

			require.Equal(t, codes.PermissionDenied, status.Code(err))
		})
	}
}

func TestCacheModelPassesTheRequest(t *testing.T) {
	f := newCacheHandler(t)
	f.expectPermission(role.ModelPush, true)
	f.cache.EXPECT().CacheModel(gomock.Any(), &modelcache.Request{
		Project:  "proj",
		Name:     "model",
		Revision: "main",
		Selection: modelcache.Selection{
			AllowPatterns:  []string{"*.safetensors"},
			IgnorePatterns: []string{"original/"},
		},
		DryRun: true,
	}).Return(missingStatus, nil)

	st, err := f.handler.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{
		Project:        "proj",
		Name:           "model",
		Revision:       "main",
		AllowPatterns:  []string{"*.safetensors"},
		IgnorePatterns: []string{"original/"},
		DryRun:         true,
	})

	require.NoError(t, err)
	require.Equal(t, modelv1alpha1.CacheState_CACHE_STATE_MISSING, st.GetState())
}

func TestGetModelCacheNeedsGetPermission(t *testing.T) {
	f := newCacheHandler(t)
	f.expectPermission(role.ModelGet, false)

	_, err := f.handler.GetModelCache(t.Context(), &modelv1alpha1.GetModelCacheRequest{Project: "proj", Name: "model"})

	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetModelCachePassesTheRequest(t *testing.T) {
	f := newCacheHandler(t)
	f.expectPermission(role.ModelGet, true)
	f.cache.EXPECT().GetModelCache(gomock.Any(), "proj", "model", "v1").Return(missingStatus, nil)

	_, err := f.handler.GetModelCache(t.Context(), &modelv1alpha1.GetModelCacheRequest{Project: "proj", Name: "model", Revision: "v1"})

	require.NoError(t, err)
}

func TestModelCacheHandlersRejectInvalidRequests(t *testing.T) {
	// Validation comes first: no permission check and no cache call.
	cacheRequests := map[string]*modelv1alpha1.CacheModelRequest{
		"too many patterns": {Project: "proj", Name: "model", IgnorePatterns: slices.Repeat([]string{"*.bin"}, 17)},
		"pattern too long":  {Project: "proj", Name: "model", AllowPatterns: []string{strings.Repeat("a", 257)}},
		"invalid name":      {Project: "proj", Name: "../model"},
	}
	for name, req := range cacheRequests {
		t.Run("CacheModel "+name, func(t *testing.T) {
			_, err := newCacheHandler(t).handler.CacheModel(t.Context(), req)

			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
	getRequests := map[string]*modelv1alpha1.GetModelCacheRequest{
		"invalid name":      {Project: "proj", Name: "../model"},
		"revision too long": {Project: "proj", Name: "model", Revision: strings.Repeat("a", 256)},
	}
	for name, req := range getRequests {
		t.Run("GetModelCache "+name, func(t *testing.T) {
			_, err := newCacheHandler(t).handler.GetModelCache(t.Context(), req)

			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

// Hugging Face repository names may start with '_', unlike the names that
// CreateModel accepts.
func TestModelCacheHandlersAcceptNamesStartingWithAnUnderscore(t *testing.T) {
	f := newCacheHandler(t)
	f.expectPermission(role.ModelPush, true)
	f.expectPermission(role.ModelGet, true)
	f.cache.EXPECT().CacheModel(gomock.Any(), gomock.Any()).Return(missingStatus, nil)
	f.cache.EXPECT().GetModelCache(gomock.Any(), "proj", "_tiny", "").Return(missingStatus, nil)

	_, err := f.handler.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{Project: "proj", Name: "_tiny", DryRun: true})
	require.NoError(t, err)
	_, err = f.handler.GetModelCache(t.Context(), &modelv1alpha1.GetModelCacheRequest{Project: "proj", Name: "_tiny"})
	require.NoError(t, err)
}

func TestModelCacheHandlersMapErrors(t *testing.T) {
	// Errors from local storage and syncs can name server paths.
	storageErr := errors.New("repository does not exist at /data/repositories/proj/model.git")
	tests := []struct {
		err  error
		want codes.Code
	}{
		{fmt.Errorf("%w: bad pattern", modelcache.ErrInvalidSelection), codes.InvalidArgument},
		{fmt.Errorf("%w: refs/pr/1", modelcache.ErrUnsupportedRevision), codes.InvalidArgument},
		{fmt.Errorf("%w: proj/model", modelcache.ErrModelNotFound), codes.NotFound},
		{fmt.Errorf("%w: %w", modelcache.ErrRevisionNotFound, storageErr), codes.NotFound},
		{fmt.Errorf("%w: %w", modelcache.ErrUpstreamSync, storageErr), codes.Unavailable},
		{fmt.Errorf("%w: c9 resolves to c9 upstream", modelcache.ErrRevisionNotMirrored), codes.FailedPrecondition},
		{fmt.Errorf("%w: proj", modelcache.ErrNotProxyProject), codes.FailedPrecondition},
		{fmt.Errorf("%w: gated", modelcache.ErrUpstreamAccessDenied), codes.FailedPrecondition},
		{fmt.Errorf("%w: rate limited", modelcache.ErrUpstreamUnavailable), codes.Unavailable},
		{fmt.Errorf("%w: more than 100 pages", modelcache.ErrTooManyFiles), codes.FailedPrecondition},
		// An upstream timeout is not the request's deadline.
		{fmt.Errorf("%w: %w", modelcache.ErrUpstreamUnavailable, context.DeadlineExceeded), codes.Unavailable},
		{modelcache.ErrClosed, codes.Unavailable},
		{fmt.Errorf("%w: 8 running", modelcache.ErrTooManyTasks), codes.ResourceExhausted},
		{storageErr, codes.Internal},
	}
	for _, tt := range tests {
		t.Run("CacheModel "+tt.err.Error(), func(t *testing.T) {
			f := newCacheHandler(t)
			f.expectPermission(role.ModelPush, true)
			f.cache.EXPECT().CacheModel(gomock.Any(), gomock.Any()).Return(nil, tt.err)

			_, err := f.handler.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{Project: "proj", Name: "model"})

			requireStatus(t, err, tt.want)
		})
		t.Run("GetModelCache "+tt.err.Error(), func(t *testing.T) {
			f := newCacheHandler(t)
			f.expectPermission(role.ModelGet, true)
			f.cache.EXPECT().GetModelCache(gomock.Any(), "proj", "model", "").Return(nil, tt.err)

			_, err := f.handler.GetModelCache(t.Context(), &modelv1alpha1.GetModelCacheRequest{Project: "proj", Name: "model"})

			requireStatus(t, err, tt.want)
		})
	}
}

func TestCacheModelReportsAnEndedRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	f := newCacheHandler(t)
	f.expectPermission(role.ModelPush, true)
	f.cache.EXPECT().CacheModel(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("%w: %w", modelcache.ErrUpstreamUnavailable, context.Canceled))

	_, err := f.handler.CacheModel(ctx, &modelv1alpha1.CacheModelRequest{Project: "proj", Name: "model"})

	require.Equal(t, codes.Canceled, status.Code(err))
}

// Upstream failures name upstream URLs and network details, which a client
// does not see; the reasons a user can act on stay.
func TestModelCacheHandlersHideUpstreamDetails(t *testing.T) {
	const detail = `Get "http://10.0.0.7/api/models/Org/m": dial tcp 10.0.0.7:80: connect: connection refused`
	tests := []struct {
		err  error
		want string
	}{
		{
			fmt.Errorf("%w: %w", modelcache.ErrUpstreamUnavailable, fmt.Errorf("%w: %s", registrydiscovery.ErrRemoteUnavailable, detail)),
			"upstream registry unavailable",
		},
		{
			fmt.Errorf("%w: %w", modelcache.ErrUpstreamUnavailable,
				fmt.Errorf("%w: %w; retry after 30s (http://10.0.0.7/api)", registrydiscovery.ErrRemoteUnavailable, registrydiscovery.ErrRemoteRateLimited)),
			"upstream registry unavailable: rate limited",
		},
		{
			fmt.Errorf("%w: %w", modelcache.ErrUpstreamAccessDenied,
				fmt.Errorf("%w: %w (http://10.0.0.7/api)", registrydiscovery.ErrRemoteAccessDenied, registrydiscovery.ErrRemoteGated)),
			"upstream registry denied access: gated repository needs an authorized credential",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			f := newCacheHandler(t)
			f.expectPermission(role.ModelPush, true)
			f.cache.EXPECT().CacheModel(gomock.Any(), gomock.Any()).Return(nil, tt.err)

			_, err := f.handler.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{Project: "proj", Name: "model", DryRun: true})

			require.Equal(t, tt.want, status.Convert(err).Message())
		})
	}
}

func requireStatus(t *testing.T, err error, want codes.Code) {
	t.Helper()
	require.Equal(t, want, status.Code(err), "err: %v", err)
	require.NotContains(t, status.Convert(err).Message(), "/data/")
}

func TestModelCacheStatusToProto(t *testing.T) {
	started := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	finished := started.Add(time.Minute)

	got := modelCacheStatusToProto(&modelcache.CacheStatus{
		Project:           "proj",
		Name:              "model",
		RequestedRevision: "main",
		Revision:          "main",
		Commit:            "c1",
		State:             modelcache.CacheStatePartial,
		Totals: modelcache.Totals{
			Files: 9, Bytes: 8, CachedFiles: 7, CachedBytes: 6, DownloadingFiles: 5,
			DownloadingBytes: 4, QueuedFiles: 3, MissingFiles: 2, FailedFiles: 1,
		},
		Files: []modelcache.FileStatus{
			{Path: "config.json", Size: 10, State: modelcache.FileCached},
			{Path: "a.safetensors", Size: 100, IsLFS: true, OID: "oa", State: modelcache.FileDownloading, DownloadedBytes: 40},
			{Path: "b.safetensors", Size: 100, IsLFS: true, OID: "ob", State: modelcache.FileQueued},
			{Path: "c.safetensors", Size: 100, IsLFS: true, OID: "oc", State: modelcache.FileMissing},
			{Path: "d.safetensors", Size: 100, IsLFS: true, OID: "od", State: modelcache.FileFailed, Error: "download failed"},
		},
		Task:     &modelcache.TaskStatus{ID: "run-1", State: modelcache.TaskFailed, StartedAt: started, FinishedAt: &finished, Error: "1 of 4 files failed"},
		Warnings: []string{"the selection matched no files"},
	})

	want := &modelv1alpha1.ModelCacheStatus{
		Project:           "proj",
		Name:              "model",
		RequestedRevision: "main",
		Revision:          "main",
		Commit:            "c1",
		State:             modelv1alpha1.CacheState_CACHE_STATE_PARTIAL,
		Totals: &modelv1alpha1.ModelCacheTotals{
			Files: 9, Bytes: 8, CachedFiles: 7, CachedBytes: 6, DownloadingFiles: 5,
			DownloadingBytes: 4, QueuedFiles: 3, MissingFiles: 2, FailedFiles: 1,
		},
		Files: []*modelv1alpha1.ModelCacheFile{
			{Path: "config.json", Size: 10, State: modelv1alpha1.CacheFileState_CACHE_FILE_STATE_CACHED},
			{Path: "a.safetensors", Size: 100, Lfs: true, Oid: "oa", State: modelv1alpha1.CacheFileState_CACHE_FILE_STATE_DOWNLOADING, DownloadedBytes: 40},
			{Path: "b.safetensors", Size: 100, Lfs: true, Oid: "ob", State: modelv1alpha1.CacheFileState_CACHE_FILE_STATE_QUEUED},
			{Path: "c.safetensors", Size: 100, Lfs: true, Oid: "oc", State: modelv1alpha1.CacheFileState_CACHE_FILE_STATE_MISSING},
			{Path: "d.safetensors", Size: 100, Lfs: true, Oid: "od", State: modelv1alpha1.CacheFileState_CACHE_FILE_STATE_FAILED, Error: "download failed"},
		},
		Task: &modelv1alpha1.ModelCacheTask{
			Id:         "run-1",
			State:      modelv1alpha1.CacheTaskState_CACHE_TASK_STATE_FAILED,
			StartedAt:  "2026-10-15T08:00:00Z",
			FinishedAt: "2026-10-15T08:01:00Z",
			Error:      "1 of 4 files failed",
		},
		Warnings: []string{"the selection matched no files"},
	}
	require.True(t, proto.Equal(want, got), "got %v", got)

	// Without a run the task is unset.
	require.Nil(t, modelCacheStatusToProto(&modelcache.CacheStatus{State: modelcache.CacheStateCached}).GetTask())
}
