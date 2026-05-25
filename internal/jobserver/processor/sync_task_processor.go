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
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

// syncTaskProcessor is the Adapter implementation for sync tasks.
type syncTaskProcessor struct {
	*processor
}

func NewSyncTaskProcessor(cfg config.SyncTaskConfig, svc syncpolicy.ISyncPolicyService) Adapter {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.TaskMaxDuration <= 0 {
		cfg.TaskMaxDuration = 2 * time.Hour
	}
	execute := func(ctx context.Context, taskID int, triggerType int) error {
		return svc.ExecuteSyncTask(ctx, taskID)
	}
	pollDueFn := func(ctx context.Context, nowMs int64) ([]job.DueJob, error) {
		return svc.ClaimPendingSyncTasks(ctx, nowMs)
	}
	p := newProcessor(ProcessorSyncTask, cfg.PollInterval, cfg.MaxConcurrent, cfg.TaskMaxDuration, execute, pollDueFn)
	return &syncTaskProcessor{processor: p}
}
