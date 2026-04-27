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
	"sync"
	"testing"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
)

func TestProcessor_ExecuteOnFirstPoll(t *testing.T) {
	var mu sync.Mutex
	var executed []int
	execute := func(ctx context.Context, policyID int, triggerType int) error {
		mu.Lock()
		executed = append(executed, policyID)
		mu.Unlock()
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
	time.Sleep(150 * time.Millisecond)
	cancel()
	base.Wait()
	mu.Lock()
	defer mu.Unlock()
	if len(executed) != 1 || executed[0] != 99 {
		t.Fatalf("expected single execute for policy 99, got %v (pollCalls=%d)", executed, pollCalls)
	}
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
