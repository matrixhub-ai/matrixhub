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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	modelv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/modelcache"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// CacheModel pulls a model revision of a proxy project into MatrixHub storage.
func (mh *ModelHandler) CacheModel(ctx context.Context, request *modelv1alpha1.CacheModelRequest) (*modelv1alpha1.ModelCacheStatus, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Caching writes to the project's storage and may create the model.
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelPush); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	st, err := mh.cache.CacheModel(ctx, &modelcache.Request{
		Project:  request.Project,
		Name:     request.Name,
		Revision: request.Revision,
		Selection: modelcache.Selection{
			AllowPatterns:  request.AllowPatterns,
			IgnorePatterns: request.IgnorePatterns,
		},
		DryRun: request.DryRun,
	})
	if err != nil {
		return nil, modelCacheErrorToStatus(ctx, err)
	}
	return modelCacheStatusToProto(st), nil
}

// GetModelCache reports the local cache status of a model revision.
func (mh *ModelHandler) GetModelCache(ctx context.Context, request *modelv1alpha1.GetModelCacheRequest) (*modelv1alpha1.ModelCacheStatus, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	st, err := mh.cache.GetModelCache(ctx, request.Project, request.Name, request.Revision)
	if err != nil {
		return nil, modelCacheErrorToStatus(ctx, err)
	}
	return modelCacheStatusToProto(st), nil
}

// modelCacheErrorToStatus maps service errors to gRPC statuses. Errors that
// wrap local storage, sync or upstream failures can carry server paths,
// upstream URLs or network details, so they are logged and answered with the
// sentinel message only. A request that ended is reported as such first: its
// error is not a failure of upstream.
func modelCacheErrorToStatus(ctx context.Context, err error) error {
	switch ctxErr := ctx.Err(); {
	case errors.Is(ctxErr, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(ctxErr, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request timed out")
	}
	switch {
	case errors.Is(err, modelcache.ErrInvalidSelection), errors.Is(err, modelcache.ErrUnsupportedRevision):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, modelcache.ErrModelNotFound):
		return status.Error(codes.NotFound, modelcache.ErrModelNotFound.Error())
	case errors.Is(err, modelcache.ErrRevisionNotFound):
		log.Debugf("model cache: %v", err)
		return status.Error(codes.NotFound, modelcache.ErrRevisionNotFound.Error())
	case errors.Is(err, modelcache.ErrUpstreamSync):
		// Usually upstream is down, which a retry can fix.
		log.Warnf("model cache: %v", err)
		return status.Error(codes.Unavailable, modelcache.ErrUpstreamSync.Error())
	case errors.Is(err, modelcache.ErrNotProxyProject),
		errors.Is(err, modelcache.ErrRevisionNotMirrored):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, modelcache.ErrUpstreamAccessDenied):
		log.Infof("model cache: %v", err)
		return status.Error(codes.FailedPrecondition, upstreamMessage(modelcache.ErrUpstreamAccessDenied, err))
	case errors.Is(err, modelcache.ErrTooManyFiles):
		log.Infof("model cache: %v", err)
		return status.Error(codes.FailedPrecondition, modelcache.ErrTooManyFiles.Error())
	case errors.Is(err, modelcache.ErrUpstreamUnavailable):
		log.Warnf("model cache: %v", err)
		return status.Error(codes.Unavailable, upstreamMessage(modelcache.ErrUpstreamUnavailable, err))
	case errors.Is(err, modelcache.ErrClosed):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, modelcache.ErrTooManyTasks):
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		log.Errorf("model cache operation failed: %v", err)
		return status.Error(codes.Internal, "model cache operation failed")
	}
}

// upstreamMessage is the message of an upstream failure: its sentinel and,
// for the failures that a user can act on, the reason.
func upstreamMessage(sentinel, err error) string {
	for _, reason := range []error{registrydiscovery.ErrRemoteGated, registrydiscovery.ErrRemoteRateLimited} {
		if errors.Is(err, reason) {
			return sentinel.Error() + ": " + reason.Error()
		}
	}
	return sentinel.Error()
}

func cacheFileStateToProto(s modelcache.FileState) modelv1alpha1.CacheFileState {
	switch s {
	case modelcache.FileCached:
		return modelv1alpha1.CacheFileState_CACHE_FILE_STATE_CACHED
	case modelcache.FileDownloading:
		return modelv1alpha1.CacheFileState_CACHE_FILE_STATE_DOWNLOADING
	case modelcache.FileQueued:
		return modelv1alpha1.CacheFileState_CACHE_FILE_STATE_QUEUED
	case modelcache.FileFailed:
		return modelv1alpha1.CacheFileState_CACHE_FILE_STATE_FAILED
	default:
		return modelv1alpha1.CacheFileState_CACHE_FILE_STATE_MISSING
	}
}

func cacheTaskStateToProto(s modelcache.TaskState) modelv1alpha1.CacheTaskState {
	switch s {
	case modelcache.TaskRunning:
		return modelv1alpha1.CacheTaskState_CACHE_TASK_STATE_RUNNING
	case modelcache.TaskSucceeded:
		return modelv1alpha1.CacheTaskState_CACHE_TASK_STATE_SUCCEEDED
	case modelcache.TaskFailed:
		return modelv1alpha1.CacheTaskState_CACHE_TASK_STATE_FAILED
	default:
		return modelv1alpha1.CacheTaskState_CACHE_TASK_STATE_UNSPECIFIED
	}
}

func cacheStateToProto(s modelcache.CacheState) modelv1alpha1.CacheState {
	switch s {
	case modelcache.CacheStateCached:
		return modelv1alpha1.CacheState_CACHE_STATE_CACHED
	case modelcache.CacheStatePartial:
		return modelv1alpha1.CacheState_CACHE_STATE_PARTIAL
	case modelcache.CacheStateMissing:
		return modelv1alpha1.CacheState_CACHE_STATE_MISSING
	default:
		return modelv1alpha1.CacheState_CACHE_STATE_UNSPECIFIED
	}
}

func modelCacheTaskToProto(t *modelcache.TaskStatus) *modelv1alpha1.ModelCacheTask {
	if t == nil {
		return nil
	}
	return &modelv1alpha1.ModelCacheTask{
		Id:         t.ID,
		State:      cacheTaskStateToProto(t.State),
		StartedAt:  formatOptionalTime(&t.StartedAt),
		FinishedAt: formatOptionalTime(t.FinishedAt),
		Error:      t.Error,
	}
}

func modelCacheStatusToProto(st *modelcache.CacheStatus) *modelv1alpha1.ModelCacheStatus {
	files := make([]*modelv1alpha1.ModelCacheFile, len(st.Files))
	for i, f := range st.Files {
		files[i] = &modelv1alpha1.ModelCacheFile{
			Path:            f.Path,
			Size:            f.Size,
			Lfs:             f.IsLFS,
			Oid:             f.OID,
			State:           cacheFileStateToProto(f.State),
			DownloadedBytes: f.DownloadedBytes,
			Error:           f.Error,
		}
	}
	return &modelv1alpha1.ModelCacheStatus{
		Project:           st.Project,
		Name:              st.Name,
		RequestedRevision: st.RequestedRevision,
		Revision:          st.Revision,
		Commit:            st.Commit,
		State:             cacheStateToProto(st.State),
		Totals: &modelv1alpha1.ModelCacheTotals{
			Files:            st.Totals.Files,
			Bytes:            st.Totals.Bytes,
			CachedFiles:      st.Totals.CachedFiles,
			CachedBytes:      st.Totals.CachedBytes,
			DownloadingFiles: st.Totals.DownloadingFiles,
			DownloadingBytes: st.Totals.DownloadingBytes,
			QueuedFiles:      st.Totals.QueuedFiles,
			MissingFiles:     st.Totals.MissingFiles,
			FailedFiles:      st.Totals.FailedFiles,
		},
		Files:    files,
		Task:     modelCacheTaskToProto(st.Task),
		Warnings: st.Warnings,
	}
}
