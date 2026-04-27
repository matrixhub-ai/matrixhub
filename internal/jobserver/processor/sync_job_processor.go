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
	"strconv"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/canceller"
)

// syncJobProcessor is the Adapter implementation for sync jobs.
type syncJobProcessor struct {
	*processor
	canceller canceller.Canceller
}

func NewSyncJobProcessor(cfg config.SyncJobConfig, svc syncjob.ISyncJobService, canc canceller.Canceller) Adapter {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 3 * time.Second
	}
	if cfg.TaskMaxDuration <= 0 {
		cfg.TaskMaxDuration = 2 * time.Hour
	}
	p := &syncJobProcessor{
		canceller: canc,
	}
	execute := func(ctx context.Context, jobID int, triggerType int) error {
		return svc.ExecuteSyncJobWithLog(ctx, jobID)
	}
	pollDueFn := func(ctx context.Context, nowMs int64) ([]job.DueJob, error) {
		return svc.ClaimPendingSyncJobs(ctx, nowMs)
	}
	p.processor = newProcessor(ProcessorSyncJob, cfg.PollInterval, cfg.MaxConcurrent, cfg.TaskMaxDuration, execute, pollDueFn)
	p.runOneFn = p.runOneWithCanceller
	return p
}

// runOneWithCanceller overrides the default runOne to register canceller before execution.
func (p *syncJobProcessor) runOneWithCanceller(ctx context.Context, d job.DueJob) {
	lockKey := p.Processor().String() + ":" + strconv.Itoa(d.ID)

	qCtx, qCancel := context.WithTimeout(ctx, p.taskMax)
	defer qCancel()
	select {
	case p.sem <- struct{}{}:
	case <-qCtx.Done():
		log.Warnw("jobserver: queue timeout waiting for slot",
			"processor", p.Processor(), "id", d.ID, "error", qCtx.Err())
		return
	}
	defer func() { <-p.sem }()

	if !p.locker.TryAcquire(lockKey, time.UnixMilli(d.FireAtMs)) {
		log.Debugw("jobserver: skip, lock not acquired", "processor", p.Processor(), "id", d.ID)
		return
	}
	defer p.locker.Release(lockKey)

	runCtx, cancel := context.WithTimeout(ctx, p.taskMax)
	defer cancel()

	p.canceller.Register(d.ID, cancel)
	defer p.canceller.Unregister(d.ID)

	if err := p.execute(runCtx, d.ID, d.TriggerType); err != nil {
		log.Errorw("jobserver: execute failed", "processor", p.Processor(), "id", d.ID, "error", err)
	}
}
