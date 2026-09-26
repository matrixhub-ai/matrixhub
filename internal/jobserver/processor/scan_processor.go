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
	"os"
	"sync"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/scan"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// scanProcessor is the Adapter that drains the scan task queue. Each poll
// claims the oldest pending task and runs it with a bounded semaphore; stale
// claims (worker crash / restart) are re-queued on start.
type scanProcessor struct {
	cfg   config.ScanProcessorConfig
	svc   *scan.Service
	store scan.Store
	owner string
	sem   chan struct{}
	wg    sync.WaitGroup
	done  chan struct{}
}

// NewScanProcessor builds the security-scan processor. A nil svc disables the
// processor (scanning turned off).
func NewScanProcessor(cfg config.ScanProcessorConfig, svc *scan.Service, store scan.Store) Adapter {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 2
	}
	if cfg.TaskMaxDuration == 0 {
		cfg.TaskMaxDuration = 30 * time.Minute
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "scan-worker"
	}
	return &scanProcessor{
		cfg:   cfg,
		svc:   svc,
		store: store,
		owner: hostname,
		sem:   make(chan struct{}, cfg.MaxConcurrent),
		done:  make(chan struct{}),
	}
}

func (p *scanProcessor) Processor() Processor { return ProcessorScan }

func (p *scanProcessor) Start(ctx context.Context) {
	if p.svc == nil {
		close(p.done)
		return
	}
	// Crash recovery: re-queue tasks whose claim lease expired.
	if n, err := p.store.ResetStaleClaims(ctx, time.Now()); err != nil {
		log.Warnw("scan: reset stale claims failed", "error", err)
	} else if n > 0 {
		log.Infow("scan: re-queued stale tasks after restart", "count", n)
	}
	go p.loop(ctx)
}

func (p *scanProcessor) loop(ctx context.Context) {
	defer close(p.done)
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()
	// Claim leases must be re-checked periodically, not only at startup: a
	// worker goroutine can die without finishing its task (OOM-kill, panic
	// in a scanner) and the task would otherwise hang in "scanning" forever.
	sweep := time.NewTicker(time.Minute)
	defer sweep.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.drain(ctx)
		case <-sweep.C:
			if n, err := p.store.ResetStaleClaims(ctx, time.Now()); err != nil {
				log.Warnw("scan: reset stale claims failed", "error", err)
			} else if n > 0 {
				log.Infow("scan: re-queued stale tasks", "count", n)
			}
		}
	}
}

func (p *scanProcessor) drain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		select {
		case p.sem <- struct{}{}:
		default:
			return // concurrency saturated; next tick continues
		}
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			defer func() { <-p.sem }()
			task, err := p.svc.ExecuteNext(ctx, p.owner)
			if err != nil {
				log.Warnw("scan: execute failed", "error", err)
				return
			}
			if task != nil {
				log.Infow("scan: task finished",
					"task", task.ID, "status", task.Status,
					"verdict", task.Verdict, "files", task.FileCount, "error", task.Error)
			}
		}()
	}
}

func (p *scanProcessor) Wait() {
	p.wg.Wait()
	<-p.done
}
