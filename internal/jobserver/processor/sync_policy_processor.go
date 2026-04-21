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

// syncPolicyProcessor is the Adapter implementation for sync policies (thin wrapper over processor).
type syncPolicyProcessor struct {
	*processor
}

func NewSyncPolicyProcessor(cfg config.SyncPolicyConfig, svc syncpolicy.ISyncPolicyService) Adapter {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 10 * time.Second
	}
	if cfg.TaskMaxDuration <= 0 {
		cfg.TaskMaxDuration = 2 * time.Hour
	}
	execute := func(ctx context.Context, policyID int, triggerType int) error {
		return svc.CreatePendingSyncTask(ctx, policyID, triggerType)
	}
	pollDueFn := func(ctx context.Context, nowMs int64) ([]job.DueJob, error) {
		return svc.ClaimDueSyncPolicies(ctx, nowMs)
	}
	p := newProcessor(ProcessorSyncPolicy, cfg.PollInterval, cfg.MaxConcurrent, cfg.TaskMaxDuration, execute, pollDueFn)
	return &syncPolicyProcessor{processor: p}
}
