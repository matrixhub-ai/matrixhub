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

// This file is the reference example for unit-testing a domain service with
// gomock (go.uber.org/mock) generated mocks + testify/require assertions.
//
// Pattern:
//   - generate mocks for the repo/port interfaces via `go generate ./...`
//     (see the //go:generate directives next to each interface);
//   - construct a gomock.Controller bound to *testing.T;
//   - inject the mocks into the service, leaving unused dependencies nil;
//   - set call expectations with EXPECT(), and assert results with require.
package syncpolicy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	syncjobmocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncjob/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy/mocks"
)

func TestSyncPolicyService_CreateSyncPolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("manual policy is persisted with no schedule", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockISyncPolicyRepo(ctrl)

		// Expect exactly one persistence call; capture the policy to assert the
		// service applied the schedule (manual => NextRunAt cleared to 0).
		repo.EXPECT().
			CreateSyncPolicy(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, p *syncpolicy.SyncPolicy) error {
				require.EqualValues(t, 0, p.NextRunAt)
				return nil
			})

		svc := syncpolicy.NewSyncPolicyService(repo, nil, nil, nil, nil)
		err := svc.CreateSyncPolicy(ctx, &syncpolicy.SyncPolicy{
			Name:        "manual",
			TriggerType: syncpolicy.TriggerTypeManual,
		})
		require.NoError(t, err)
	})

	t.Run("invalid cron is rejected before touching the repo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockISyncPolicyRepo(ctrl)
		// No EXPECT on the repo: gomock fails the test if CreateSyncPolicy is called.

		svc := syncpolicy.NewSyncPolicyService(repo, nil, nil, nil, nil)
		err := svc.CreateSyncPolicy(ctx, &syncpolicy.SyncPolicy{
			Name:        "bad-cron",
			TriggerType: syncpolicy.TriggerTypeScheduled,
			Cron:        "not-a-cron",
		})
		require.Error(t, err)
	})
}

func TestSyncPolicyService_ClaimDueSyncPolicies(t *testing.T) {
	ctx := context.Background()
	const nowMs = int64(1_700_000_000_000)

	t.Run("only CAS-claimed policies are returned as due jobs", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockISyncPolicyRepo(ctrl)

		due := []*syncpolicy.SyncPolicy{
			{ID: 1, TriggerType: syncpolicy.TriggerTypeManual},
			{ID: 2, TriggerType: syncpolicy.TriggerTypeManual},
		}
		repo.EXPECT().
			SelectDuePolicies(gomock.Any(), nowMs, gomock.Any()).
			Return(due, nil)

		// Policy 1 wins the CAS; policy 2 loses the race and must be skipped.
		repo.EXPECT().
			AdvanceNextRunAtCAS(gomock.Any(), 1, gomock.Any(), gomock.Any(), nowMs).
			Return(true, nil)
		repo.EXPECT().
			AdvanceNextRunAtCAS(gomock.Any(), 2, gomock.Any(), gomock.Any(), nowMs).
			Return(false, nil)

		svc := syncpolicy.NewSyncPolicyService(repo, nil, nil, nil, nil)
		jobs, err := svc.ClaimDueSyncPolicies(ctx, nowMs)

		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.Equal(t, job.DueJob{
			ID:          1,
			PolicyID:    1,
			TriggerType: int(syncpolicy.TriggerTypeManual),
			FireAtMs:    nowMs,
		}, jobs[0])
	})

	t.Run("repo error is propagated and stops claiming", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockISyncPolicyRepo(ctrl)

		wantErr := errors.New("db down")
		repo.EXPECT().
			SelectDuePolicies(gomock.Any(), nowMs, gomock.Any()).
			Return(nil, wantErr)
		// AdvanceNextRunAtCAS must not be called when selection fails.

		svc := syncpolicy.NewSyncPolicyService(repo, nil, nil, nil, nil)
		jobs, err := svc.ClaimDueSyncPolicies(ctx, nowMs)

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, jobs)
	})
}

func TestSyncPolicyService_CreateSyncTaskAsync(t *testing.T) {
	ctx := context.Background()

	// A user-initiated sync must be recorded as manually triggered even when the
	// policy itself is configured to run on a schedule.
	t.Run("task on a scheduled policy is recorded as manually triggered", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		taskRepo := mocks.NewMockISyncTaskRepo(ctrl)
		var triggeredTaskID int

		taskRepo.EXPECT().
			CreateSyncTask(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, task *syncpolicy.SyncTask) (*syncpolicy.SyncTask, error) {
				require.Equal(t, syncpolicy.TriggerTypeManual, task.TriggerType)
				require.Equal(t, syncpolicy.SyncTaskStatusPending, task.Status)
				require.Equal(t, 7, task.SyncPolicyID)
				task.ID = 42
				return task, nil
			})

		svc := syncpolicy.NewSyncPolicyService(nil, taskRepo, nil, nil, nil)
		svc.(interface{ SetOnTaskCreated(func(int) bool) }).SetOnTaskCreated(func(id int) bool {
			triggeredTaskID = id
			return true
		})
		task, err := svc.CreateSyncTaskAsync(ctx, &syncpolicy.SyncPolicy{
			ID:          7,
			TriggerType: syncpolicy.TriggerTypeScheduled,
			Cron:        "0 * * * *",
		})

		require.NoError(t, err)
		require.Equal(t, syncpolicy.TriggerTypeManual, task.TriggerType)
		require.Equal(t, 42, triggeredTaskID)
	})
}

func TestSyncPolicyService_ClaimPendingSyncTask(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	taskRepo := mocks.NewMockISyncTaskRepo(ctrl)
	taskRepo.EXPECT().GetSyncTask(ctx, 42).Return(&syncpolicy.SyncTask{
		ID:           42,
		SyncPolicyID: 7,
		TriggerType:  syncpolicy.TriggerTypeManual,
		Status:       syncpolicy.SyncTaskStatusPending,
	}, nil)
	taskRepo.EXPECT().UpdateTaskStatusCAS(ctx, 42, syncpolicy.SyncTaskStatusPending, syncpolicy.SyncTaskStatusRunning).Return(true, nil)

	svc := syncpolicy.NewSyncPolicyService(nil, taskRepo, nil, nil, nil)
	due, claimed, err := svc.(interface {
		ClaimPendingSyncTask(context.Context, int) (job.DueJob, bool, error)
	}).ClaimPendingSyncTask(ctx, 42)

	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, 42, due.ID)
	require.Equal(t, 7, due.PolicyID)
	require.Equal(t, int(syncpolicy.TriggerTypeManual), due.TriggerType)
}

func TestSyncPolicyService_ExecuteSyncTaskMarksFailureWhenJobBatchCreationFails(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	taskRepo := mocks.NewMockISyncTaskRepo(ctrl)
	policyRepo := mocks.NewMockISyncPolicyRepo(ctrl)
	jobService := syncjobmocks.NewMockISyncJobService(ctrl)
	generator := mocks.NewMockSyncJobGenerator(ctrl)
	task := &syncpolicy.SyncTask{ID: 42, SyncPolicyID: 7, Status: syncpolicy.SyncTaskStatusRunning}
	policy := &syncpolicy.SyncPolicy{ID: 7}
	jobs := []*syncjob.SyncJob{{}}
	createErr := errors.New("database unavailable")

	taskRepo.EXPECT().GetSyncTask(ctx, 42).Return(task, nil)
	policyRepo.EXPECT().GetSyncPolicy(ctx, 7).Return(policy, nil)
	generator.EXPECT().Generate(ctx, policy).Return(nil, jobs, nil)
	taskRepo.EXPECT().UpdateSyncTask(ctx, task).DoAndReturn(func(_ context.Context, updated *syncpolicy.SyncTask) error {
		require.Equal(t, 1, updated.TotalItems)
		return nil
	})
	jobService.EXPECT().CreateSyncJobs(ctx, jobs).Return(createErr)
	taskRepo.EXPECT().UpdateSyncTask(ctx, task).DoAndReturn(func(_ context.Context, updated *syncpolicy.SyncTask) error {
		require.Equal(t, syncpolicy.SyncTaskStatusFailed, updated.Status)
		require.NotZero(t, updated.CompletedTimestamp)
		return nil
	})

	svc := syncpolicy.NewSyncPolicyService(policyRepo, taskRepo, jobService, nil, generator)
	require.ErrorIs(t, svc.ExecuteSyncTask(ctx, task.ID), createErr)
}
