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

package jobserver

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	syncjobmocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncjob/mocks"
	syncpolicymocks "github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/canceller"
	logstoremocks "github.com/matrixhub-ai/matrixhub/internal/jobserver/logstore/mocks"
)

// TestJobServer_RunInvokesExecuteForClaimedJob verifies that a sync policy
// claimed by the poll loop is forwarded to CreatePendingSyncTask.
//
// Dependencies are gomock mocks (go.uber.org/mock). The sync-task and sync-job
// processors also poll on their own loops, so their claim methods are stubbed
// with AnyTimes() to tolerate the nondeterministic number of background polls.
func TestJobServer_RunInvokesExecuteForClaimedJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	syncSvc := syncpolicymocks.NewMockISyncPolicyService(ctrl)
	syncJobSvc := syncjobmocks.NewMockISyncJobService(ctrl)
	logStore := logstoremocks.NewMockLogStore(ctrl)

	const (
		wantPolicyID    = 42
		wantTriggerType = int(2)
	)

	// First poll yields one due policy; subsequent polls yield nothing.
	var claims atomic.Int32
	syncSvc.EXPECT().
		ClaimDueSyncPolicies(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, nowMs int64) ([]job.DueJob, error) {
			if claims.Add(1) == 1 {
				return []job.DueJob{{
					ID:          wantPolicyID,
					PolicyID:    wantPolicyID,
					TriggerType: wantTriggerType,
					FireAtMs:    nowMs,
				}}, nil
			}
			return nil, nil
		}).
		MinTimes(1)

	// The claimed policy must be turned into a pending sync task at least once.
	var created atomic.Int32
	syncSvc.EXPECT().
		CreatePendingSyncTask(gomock.Any(), wantPolicyID, wantTriggerType).
		DoAndReturn(func(_ context.Context, _ int, _ int) error {
			created.Add(1)
			return nil
		}).
		MinTimes(1)

	// Other processors keep polling on their own intervals; accept any number
	// of empty polls and never return work for them in this test.
	syncSvc.EXPECT().ClaimPendingSyncTasks(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	syncJobSvc.EXPECT().ClaimPendingSyncJobs(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	cfg := &config.JobServerConfig{
		Enabled:       true,
		ShutdownGrace: 5 * time.Second,
		SyncPolicy: config.SyncPolicyConfig{
			PollInterval:    100 * time.Millisecond,
			MaxConcurrent:   2,
			TaskMaxDuration: time.Hour,
		},
	}

	js := New(cfg, syncSvc, syncJobSvc, logStore, canceller.NewMemCanceller())

	ctx, cancel := context.WithCancel(context.Background())
	go js.Run(ctx)
	time.Sleep(350 * time.Millisecond)
	cancel()
	js.Shutdown(2 * time.Second)

	require.GreaterOrEqual(t, claims.Load(), int32(1), "expected at least one ClaimDueSyncPolicies poll")
	require.GreaterOrEqual(t, created.Load(), int32(1), "expected at least one CreatePendingSyncTask for the claimed policy")
}
