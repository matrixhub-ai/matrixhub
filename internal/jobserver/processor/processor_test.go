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

package processor

import (
	"context"
	"testing"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
)

func TestProcessor_ExecuteOnFirstPoll(t *testing.T) {
	executedCh := make(chan int, 1)
	execute := func(ctx context.Context, policyID int, triggerType int) error {
		executedCh <- policyID
		return nil
	}
	pollCalls := 0
	pollDueFn := func(ctx context.Context, nowMs int64) ([]job.DueJob, error) {
		pollCalls++
		if pollCalls == 1 {
			return []job.DueJob{{
				ID:          99,
				PolicyID:    99,
				TriggerType: 1,
				FireAtMs:    nowMs,
			}}, nil
		}
		return nil, nil
	}
	base := newProcessor(ProcessorSyncPolicy, 500*time.Millisecond, 2, 5*time.Second, execute, pollDueFn)
	ctx, cancel := context.WithCancel(context.Background())
	base.Start(ctx)
	select {
	case policyID := <-executedCh:
		if policyID != 99 {
			t.Fatalf("expected execute for policy 99, got %d (pollCalls=%d)", policyID, pollCalls)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("execute not called (pollCalls=%d)", pollCalls)
	}
	cancel()
	base.Wait()
}

func TestProcessor_WaitReturnsAfterCancel(t *testing.T) {
	execute := func(ctx context.Context, policyID int, triggerType int) error { return nil }
	pollDueFn := func(ctx context.Context, nowMs int64) ([]job.DueJob, error) { return nil, nil }
	base := newProcessor(ProcessorSyncPolicy, time.Hour, 1, time.Second, execute, pollDueFn)
	ctx, cancel := context.WithCancel(context.Background())
	base.Start(ctx)
	cancel()
	done := make(chan struct{})
	go func() {
		base.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Wait() did not return after context cancel")
	}
}

func TestProcessor_TriggerClaimsSpecificID(t *testing.T) {
	executedCh := make(chan int, 1)
	pollCalls := 0
	base := newProcessor(
		ProcessorSyncTask,
		time.Hour,
		1,
		time.Second,
		func(_ context.Context, id int, _ int) error {
			executedCh <- id
			return nil
		},
		func(context.Context, int64) ([]job.DueJob, error) {
			pollCalls++
			return nil, nil
		},
	)
	base.claimOne = func(_ context.Context, id int) (job.DueJob, bool, error) {
		if id != 42 {
			t.Fatalf("expected trigger for ID 42, got %d", id)
		}
		return job.DueJob{ID: id}, true, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	base.Start(ctx)
	if !base.Trigger(42) {
		t.Fatal("Trigger(42) returned false")
	}
	select {
	case id := <-executedCh:
		if id != 42 {
			t.Fatalf("expected execute for ID 42, got %d", id)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("triggered work was not executed")
	}
	cancel()
	base.Wait()
	if pollCalls != 1 {
		t.Fatalf("expected only the startup recovery poll, got %d polls", pollCalls)
	}
}

func TestProcessor_QueueTimeoutReleasesUnstartedClaim(t *testing.T) {
	base := newProcessor(
		ProcessorSyncTask,
		time.Hour,
		1,
		10*time.Millisecond,
		func(context.Context, int, int) error {
			t.Fatal("execute must not run without a concurrency slot")
			return nil
		},
		func(context.Context, int64) ([]job.DueJob, error) { return nil, nil },
	)
	base.sem <- struct{}{}
	released := make(chan int, 1)
	base.release = func(_ context.Context, id int) error {
		released <- id
		return nil
	}

	base.runOne(context.Background(), job.DueJob{ID: 42})
	select {
	case id := <-released:
		if id != 42 {
			t.Fatalf("expected claim for ID 42 to be released, got %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("unstarted claim was not released after queue timeout")
	}
}

func TestProcessor_ExecuteErrorReleasesClaimForRetry(t *testing.T) {
	released := make(chan int, 1)
	base := newProcessor(
		ProcessorSyncTask,
		time.Hour,
		1,
		time.Second,
		func(context.Context, int, int) error { return context.Canceled },
		func(context.Context, int64) ([]job.DueJob, error) { return nil, nil },
	)
	base.release = func(_ context.Context, id int) error {
		released <- id
		return nil
	}

	base.runOne(context.Background(), job.DueJob{ID: 42})
	select {
	case id := <-released:
		if id != 42 {
			t.Fatalf("expected claim for ID 42 to be released, got %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("claim was not released after execution failed")
	}
}
