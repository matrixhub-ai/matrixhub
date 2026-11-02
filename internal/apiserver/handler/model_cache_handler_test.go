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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	authzmocks "github.com/matrixhub-ai/matrixhub/internal/domain/authz/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	syncjobmocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncjob/mocks"
)

func newModelCacheHandler(t *testing.T) (*ModelHandler, *authzmocks.MockIAuthzService, *syncjobmocks.MockISyncJobService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	authz := authzmocks.NewMockIAuthzService(ctrl)
	jobs := syncjobmocks.NewMockISyncJobService(ctrl)
	return &ModelHandler{authzService: authz, syncJobs: jobs}, authz, jobs
}

func TestCacheModel_CreatesAJob(t *testing.T) {
	h, authz, jobs := newModelCacheHandler(t)
	created := time.Unix(1700000000, 0)
	authz.EXPECT().VerifyProjectPermissionByName(gomock.Any(), "proxy", role.ModelPush).Return(true, nil)
	jobs.EXPECT().CreateModelCacheJob(gomock.Any(), "proxy", "gpt2", "v1.0").Return(&syncjob.SyncJob{
		ID:        21,
		JobType:   syncjob.JobTypeModelCache,
		Status:    syncjob.SyncJobStatusPending,
		CreatedAt: created,
	}, nil)

	job, err := h.CacheModel(t.Context(), &v1alpha1.CacheModelRequest{Project: "proxy", Name: "gpt2", Revision: "v1.0"})
	require.NoError(t, err)
	require.Equal(t, int32(21), job.Id)
	require.Equal(t, v1alpha1.SyncJobStatus_SYNC_JOB_STATUS_PENDING, job.Status)
	require.Equal(t, "v1.0", job.Revision)
	require.Equal(t, created.Unix(), job.CreatedTimestamp)
	require.Zero(t, job.CompletedTimestamp)
}

func TestCacheModel_RejectsAnInvalidRequestBeforeAuthorization(t *testing.T) {
	for _, req := range []*v1alpha1.CacheModelRequest{
		{Project: "", Name: "gpt2"},
		{Project: "proxy", Name: ""},
		{Project: "proxy", Name: "resolve"},
		{Project: "proxy", Name: "a/b"},
		{Project: "proxy", Name: "gpt2", Revision: string(make([]byte, 256))},
	} {
		h, _, _ := newModelCacheHandler(t)

		_, err := h.CacheModel(t.Context(), req)
		require.Equal(t, codes.InvalidArgument, status.Code(err), "%+v", req)
	}
}

func TestCacheModel_RequiresModelPush(t *testing.T) {
	for _, verify := range []struct {
		allowed bool
		err     error
	}{
		{allowed: false},
		{err: errors.New("project not found")},
	} {
		h, authz, _ := newModelCacheHandler(t)
		authz.EXPECT().VerifyProjectPermissionByName(gomock.Any(), "proxy", role.ModelPush).Return(verify.allowed, verify.err)

		_, err := h.CacheModel(t.Context(), &v1alpha1.CacheModelRequest{Project: "proxy", Name: "gpt2"})
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	}
}

func TestCacheModel_MapsErrors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code codes.Code
	}{
		{err: fmt.Errorf("%w: refs/pr/1", syncjob.ErrInvalidRevision), code: codes.InvalidArgument},
		{err: syncjob.ErrNotProxyProject, code: codes.FailedPrecondition},
		{err: gorm.ErrRecordNotFound, code: codes.NotFound},
		{err: errors.New("database is down"), code: codes.Internal},
	} {
		h, authz, jobs := newModelCacheHandler(t)
		authz.EXPECT().VerifyProjectPermissionByName(gomock.Any(), "proxy", role.ModelPush).Return(true, nil)
		jobs.EXPECT().CreateModelCacheJob(gomock.Any(), "proxy", "gpt2", "").Return(nil, tc.err)

		_, err := h.CacheModel(t.Context(), &v1alpha1.CacheModelRequest{Project: "proxy", Name: "gpt2"})
		require.Equal(t, tc.code, status.Code(err), tc.err.Error())
		if tc.code == codes.Internal {
			require.NotContains(t, err.Error(), "database is down")
		}
	}
}
