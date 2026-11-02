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
	"gorm.io/gorm"

	modelv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// CacheModel creates a job that caches a model revision of a proxy project.
func (mh *ModelHandler) CacheModel(ctx context.Context, request *modelv1alpha1.CacheModelRequest) (*modelv1alpha1.ModelCacheJob, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Caching writes to the project's storage and may create the model.
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelPush); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	job, err := mh.syncJobs.CreateModelCacheJob(ctx, request.Project, request.Name, request.Revision)
	switch {
	case err == nil:
	case errors.Is(err, syncjob.ErrInvalidRevision):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, syncjob.ErrNotProxyProject):
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, status.Error(codes.NotFound, "project not found")
	default:
		log.Errorw("create model cache job failed", "project", request.Project, "model", request.Name, "error", err)
		return nil, status.Error(codes.Internal, "failed to create the model cache job")
	}
	return &modelv1alpha1.ModelCacheJob{
		Id:                 int32(job.ID),
		Status:             modelv1alpha1.SyncJobStatus(job.Status),
		Revision:           request.Revision,
		CreatedTimestamp:   job.CreatedAt.Unix(),
		CompletedTimestamp: job.CompletedTimestamp,
	}, nil
}
