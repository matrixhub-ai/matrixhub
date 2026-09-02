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
	"testing"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	syncjobmocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncjob/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	syncpolicymocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy/mocks"
	"go.uber.org/mock/gomock"
)

// syncJobToProto and syncTaskToProto convert the domain status by a plain
// numeric cast. A domain state with no counterpart in the proto enum therefore
// travels the wire as a bare number — protojson only emits the symbolic name
// for values the enum declares — and every generated client fails to decode the
// whole response. Sync jobs are created in the pending state, so the gap is
// reachable through ListSyncJobs on any freshly created task.
func TestStatusEnumsAreRepresentableInProto(t *testing.T) {
	t.Run("sync job", func(t *testing.T) {
		for s := syncjob.SyncJobStatusUnspecified; s <= syncjob.SyncJobStatusPending; s++ {
			if _, ok := v1alpha1.SyncJobStatus_name[int32(s)]; !ok {
				t.Errorf("domain SyncJobStatus %d has no v1alpha1.SyncJobStatus counterpart; "+
					"it would serialise as a number and break generated clients", int32(s))
			}
		}
	})

	t.Run("sync task", func(t *testing.T) {
		for s := syncpolicy.SyncTaskStatusUnspecified; s <= syncpolicy.SyncTaskStatusPending; s++ {
			if _, ok := v1alpha1.SyncTaskStatus_name[int32(s)]; !ok {
				t.Errorf("domain SyncTaskStatus %d has no v1alpha1.SyncTaskStatus counterpart; "+
					"it would serialise as a number and break generated clients", int32(s))
			}
		}
	})
}

func TestListSyncTasksPaginationIncludesPageCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	policyService := syncpolicymocks.NewMockISyncPolicyService(ctrl)
	jobService := syncjobmocks.NewMockISyncJobService(ctrl)
	h := &SyncPolicyHandler{
		syncPolicyService: policyService,
		syncJobService:    jobService,
	}

	policyService.EXPECT().
		ListSyncTasksByPolicyID(gomock.Any(), 1, 2, 10, syncpolicy.SyncTaskStatusUnspecified).
		Return([]*syncpolicy.SyncTask{}, int64(21), nil)

	response, err := h.ListSyncTasks(t.Context(), &v1alpha1.ListSyncTasksRequest{
		SyncPolicyId: 1,
		Page:         2,
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("ListSyncTasks() error = %v", err)
	}
	if got, want := response.Pagination.Pages, int32(3); got != want {
		t.Fatalf("Pagination.Pages = %d, want %d", got, want)
	}
	if got, want := response.Pagination.Page, int32(2); got != want {
		t.Fatalf("Pagination.Page = %d, want %d", got, want)
	}
	if got, want := response.Pagination.PageSize, int32(10); got != want {
		t.Fatalf("Pagination.PageSize = %d, want %d", got, want)
	}
}

func TestListSyncJobsPaginationIncludesPageCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	jobService := syncjobmocks.NewMockISyncJobService(ctrl)
	h := &SyncPolicyHandler{syncJobService: jobService}

	jobService.EXPECT().
		ListSyncJobsByTaskID(gomock.Any(), 1, 1, 20, syncjob.SyncJobStatusUnspecified, "").
		Return([]*syncjob.SyncJob{}, int64(1<<31)+1, nil)

	response, err := h.ListSyncJobs(t.Context(), &v1alpha1.ListSyncJobsRequest{SyncTaskId: 1})
	if err != nil {
		t.Fatalf("ListSyncJobs() error = %v", err)
	}
	if got, want := response.Pagination.Pages, int32(107374183); got != want {
		t.Fatalf("Pagination.Pages = %d, want %d", got, want)
	}
	if got, want := response.Pagination.Total, int32(1<<31-1); got != want {
		t.Fatalf("Pagination.Total = %d, want %d", got, want)
	}
	if got, want := response.Pagination.Page, int32(1); got != want {
		t.Fatalf("Pagination.Page = %d, want %d", got, want)
	}
	if got, want := response.Pagination.PageSize, int32(20); got != want {
		t.Fatalf("Pagination.PageSize = %d, want %d", got, want)
	}
}
