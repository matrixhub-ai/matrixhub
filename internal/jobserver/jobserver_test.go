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
	"errors"
	"sync"
	"testing"
	"time"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

// fakeSyncPolicyService implements syncpolicy.ISyncPolicyService with only ClaimDueSyncPolicies /
// CreatePendingSyncTask wired for jobserver tests; other methods are stubs.
type fakeSyncPolicyService struct {
	mu sync.Mutex

	claimCalls int
	claimFn    func(ctx context.Context, nowMs int64) ([]job.DueJob, error)

	execCalls    int
	lastPolicyID int
	lastTrigger  int
	execFn       func(ctx context.Context, policyID int, triggerType int) error
}

func (f *fakeSyncPolicyService) ClaimDueSyncPolicies(ctx context.Context, nowMs int64) ([]job.DueJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimCalls++
	if f.claimFn != nil {
		return f.claimFn(ctx, nowMs)
	}
	if f.claimCalls == 1 {
		return []job.DueJob{{
			PolicyID:    42,
			TriggerType: 2,
			FireAtMs:    nowMs,
		}}, nil
	}
	return nil, nil
}

func (f *fakeSyncPolicyService) CreatePendingSyncTask(ctx context.Context, policyID int, triggerType int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.execCalls++
	f.lastPolicyID = policyID
	f.lastTrigger = triggerType
	if f.execFn != nil {
		return f.execFn(ctx, policyID, triggerType)
	}
	return nil
}

func (f *fakeSyncPolicyService) CreateExcecuteSyncTaskAndSyncJobs(context.Context, *syncpolicy.SyncPolicy) error {
	return errors.New("not used")
}

func (f *fakeSyncPolicyService) GetSyncPolicy(context.Context, int) (*syncpolicy.SyncPolicy, error) {
	return nil, errors.New("not used")
}
func (f *fakeSyncPolicyService) CreateSyncPolicy(context.Context, *syncpolicy.SyncPolicy) error {
	return errors.New("not used")
}
func (f *fakeSyncPolicyService) UpdateSyncPolicy(context.Context, *syncpolicy.SyncPolicy) error {
	return errors.New("not used")
}
func (f *fakeSyncPolicyService) DeleteSyncPolicy(context.Context, int) error {
	return errors.New("not used")
}
func (f *fakeSyncPolicyService) ListSyncPolicies(context.Context, int, int, string) ([]*syncpolicy.SyncPolicy, int64, error) {
	return nil, 0, errors.New("not used")
}
func (f *fakeSyncPolicyService) GetSyncTask(context.Context, int) (*syncpolicy.SyncTask, error) {
	return nil, errors.New("not used")
}
func (f *fakeSyncPolicyService) CreateSyncTask(context.Context, *syncpolicy.SyncTask) (*syncpolicy.SyncTask, error) {
	return nil, errors.New("not used")
}
func (f *fakeSyncPolicyService) ListSyncTasksByPolicyID(context.Context, int, int, int, v1alpha1.SyncTaskStatus) ([]*syncpolicy.SyncTask, int64, error) {
	return nil, 0, errors.New("not used")
}
func (f *fakeSyncPolicyService) CreateSyncTaskAndSyncJobs(context.Context, *syncpolicy.SyncPolicy) error {
	return errors.New("not used")
}

func TestJobServer_RunInvokesExecuteForClaimedJob(t *testing.T) {
	fake := &fakeSyncPolicyService{}
	cfg := &config.JobServerConfig{
		Enabled:       true,
		ShutdownGrace: 5 * time.Second,
		SyncPolicy: config.SyncPolicyConfig{
			PollInterval:    100 * time.Millisecond,
			MaxConcurrent:   2,
			TaskMaxDuration: time.Hour,
		},
	}
	js := New(cfg, fake)
	ctx, cancel := context.WithCancel(context.Background())
	go js.Run(ctx)
	time.Sleep(350 * time.Millisecond)
	cancel()
	js.Shutdown(2 * time.Second)

	fake.mu.Lock()
	ec, cc := fake.execCalls, fake.claimCalls
	fake.mu.Unlock()
	if ec < 1 || cc < 1 {
		t.Fatalf("expected CreatePendingSyncTask at least once, got execCalls=%d claimCalls=%d", ec, cc)
	}
}
