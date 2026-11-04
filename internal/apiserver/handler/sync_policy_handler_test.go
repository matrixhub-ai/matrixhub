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
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	syncjobmocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncjob/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	syncpolicymocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/logstore"
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
	policyService := syncpolicymocks.NewMockISyncPolicyService(ctrl)
	jobService := syncjobmocks.NewMockISyncJobService(ctrl)
	h := &SyncPolicyHandler{syncPolicyService: policyService, syncJobService: jobService}

	policyService.EXPECT().GetSyncTask(gomock.Any(), 1).Return(&syncpolicy.SyncTask{ID: 1, SyncPolicyID: 3}, nil)
	jobService.EXPECT().
		ListSyncJobsByTaskID(gomock.Any(), 1, 1, 20, syncjob.SyncJobStatusUnspecified, "").
		Return([]*syncjob.SyncJob{}, int64(1<<31)+1, nil)

	response, err := h.ListSyncJobs(t.Context(), &v1alpha1.ListSyncJobsRequest{SyncPolicyId: 3, SyncTaskId: 1})
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

func TestListSyncPoliciesPaginationIncludesPageCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	policyService := syncpolicymocks.NewMockISyncPolicyService(ctrl)
	h := &SyncPolicyHandler{syncPolicyService: policyService}

	policyService.EXPECT().
		ListSyncPolicies(gomock.Any(), 2, 10, "").
		Return([]*syncpolicy.SyncPolicy{}, int64(21), nil)

	response, err := h.ListSyncPolicies(t.Context(), &v1alpha1.ListSyncPoliciesRequest{
		Page:     2,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListSyncPolicies() error = %v", err)
	}
	if got, want := response.Pagination.Pages, int32(3); got != want {
		t.Fatalf("Pagination.Pages = %d, want %d", got, want)
	}
	if got, want := response.Pagination.Total, int32(21); got != want {
		t.Fatalf("Pagination.Total = %d, want %d", got, want)
	}
	if got, want := response.Pagination.Page, int32(2); got != want {
		t.Fatalf("Pagination.Page = %d, want %d", got, want)
	}
	if got, want := response.Pagination.PageSize, int32(10); got != want {
		t.Fatalf("Pagination.PageSize = %d, want %d", got, want)
	}
}

// The task and job APIs address a task by its policy and a job by its task,
// so a path whose IDs do not belong together finds nothing.
func TestSyncTaskAPIsCheckThePath(t *testing.T) {
	type call func(h *SyncPolicyHandler) error
	listJobs := func(policyID, taskID int32) call {
		return func(h *SyncPolicyHandler) error {
			_, err := h.ListSyncJobs(t.Context(), &v1alpha1.ListSyncJobsRequest{SyncPolicyId: policyID, SyncTaskId: taskID})
			return err
		}
	}
	jobLog := func(policyID, taskID, jobID int32) call {
		return func(h *SyncPolicyHandler) error {
			_, err := h.GetSyncJobLog(t.Context(), &v1alpha1.GetSyncJobLogRequest{SyncPolicyId: policyID, SyncTaskId: taskID, SyncJobId: jobID})
			return err
		}
	}
	tests := []struct {
		name string
		call call
		code codes.Code
	}{
		{name: "jobs of a task of another policy", call: listJobs(4, 1), code: codes.InvalidArgument},
		{name: "jobs of a missing task", call: listJobs(3, 2), code: codes.NotFound},
		{name: "log under a task of another policy", call: jobLog(4, 1, 7), code: codes.InvalidArgument},
		{name: "log under a missing task", call: jobLog(3, 2, 7), code: codes.NotFound},
		{name: "log of a job of another task", call: jobLog(3, 1, 8), code: codes.InvalidArgument},
		{name: "log of a missing job", call: jobLog(3, 1, 9), code: codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			policyService := syncpolicymocks.NewMockISyncPolicyService(ctrl)
			jobService := syncjobmocks.NewMockISyncJobService(ctrl)
			h := &SyncPolicyHandler{
				syncPolicyService: policyService,
				syncJobService:    jobService,
				logStore:          logstore.NewFileLogStore(t.TempDir()),
			}
			// Task 1 belongs to policy 3; job 7 belongs to task 1, and job 8
			// to task 5.
			policyService.EXPECT().GetSyncTask(gomock.Any(), 1).Return(&syncpolicy.SyncTask{ID: 1, SyncPolicyID: 3}, nil).AnyTimes()
			policyService.EXPECT().GetSyncTask(gomock.Any(), 2).Return(nil, errors.New("record not found")).AnyTimes()
			jobService.EXPECT().GetSyncJob(gomock.Any(), 7).Return(&syncjob.SyncJob{ID: 7, SyncTaskID: 1}, nil).AnyTimes()
			jobService.EXPECT().GetSyncJob(gomock.Any(), 8).Return(&syncjob.SyncJob{ID: 8, SyncTaskID: 5}, nil).AnyTimes()
			jobService.EXPECT().GetSyncJob(gomock.Any(), 9).Return(nil, errors.New("record not found")).AnyTimes()

			if code := status.Code(tt.call(h)); code != tt.code {
				t.Fatalf("expected %s, got %s", tt.code, code)
			}
		})
	}
}

func TestGetSyncJobLogReturnsTheLogOfAJobOfTheTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	policyService := syncpolicymocks.NewMockISyncPolicyService(ctrl)
	jobService := syncjobmocks.NewMockISyncJobService(ctrl)
	logs := logstore.NewFileLogStore(t.TempDir())
	h := &SyncPolicyHandler{syncPolicyService: policyService, syncJobService: jobService, logStore: logs}

	w, err := logs.Writer(7)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("[INFO] job start\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	policyService.EXPECT().GetSyncTask(gomock.Any(), 1).Return(&syncpolicy.SyncTask{ID: 1, SyncPolicyID: 3}, nil)
	jobService.EXPECT().GetSyncJob(gomock.Any(), 7).Return(&syncjob.SyncJob{ID: 7, SyncTaskID: 1}, nil)

	response, err := h.GetSyncJobLog(t.Context(), &v1alpha1.GetSyncJobLogRequest{SyncPolicyId: 3, SyncTaskId: 1, SyncJobId: 7})
	if err != nil {
		t.Fatalf("GetSyncJobLog() error = %v", err)
	}
	if got, want := response.Log, "[INFO] job start\n"; got != want {
		t.Fatalf("Log = %q, want %q", got, want)
	}
}
