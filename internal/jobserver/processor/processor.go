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
	"sync"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// Processor identifies a processor implementation family (sync, future cleanup, ...).
type Processor string

const (
	ProcessorSyncPolicy Processor = "syncPolicy"
	ProcessorSyncTask   Processor = "syncTask"
	ProcessorSyncJob    Processor = "syncJob"
)

// ExecuteFn runs one claimed policy execution (insert task/jobs + git work happens inside service).
type ExecuteFn func(ctx context.Context, policyID int, triggerType int) error

// PollDueFn lists due work and performs CAS advance in the service / repo layer.
type PollDueFn func(ctx context.Context, nowMs int64) ([]job.DueJob, error)

// PolicyLocker serializes execution per logical policy key (single-replica in-memory).
type PolicyLocker interface {
	TryAcquire(key string, fireAt time.Time) bool
	Release(key string)
}

type memLocker struct {
	m sync.Map // key string -> struct{}
}

// NewMemLocker returns an in-process PolicyLocker (lost on restart).
func NewMemLocker() PolicyLocker {
	return &memLocker{}
}

func (l *memLocker) TryAcquire(key string, _ time.Time) bool {
	_, loaded := l.m.LoadOrStore(key, struct{}{})
	return !loaded
}

func (l *memLocker) Release(key string) {
	l.m.Delete(key)
}

// Adapter is the minimal lifecycle surface exposed to JobServer.
type Adapter interface {
	Processor() Processor
	Start(ctx context.Context)
	Wait()
}

// processor holds the shared poll loop, semaphore, per-policy lock, and execute path for one Processor value.
type processor struct {
	processor Processor
	interval  time.Duration
	sem       chan struct{}
	locker    PolicyLocker
	taskMax   time.Duration
	execute   ExecuteFn
	pollDueFn PollDueFn
	runOneFn  func(ctx context.Context, d job.DueJob) // optional override

	wg   sync.WaitGroup
	done chan struct{}
}

func newProcessor(
	proc Processor,
	interval time.Duration,
	maxConcurrent int,
	taskMax time.Duration,
	execute ExecuteFn,
	pollDueFn PollDueFn,
) *processor {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if taskMax <= 0 {
		taskMax = 2 * time.Hour
	}
	return &processor{
		processor: proc,
		interval:  interval,
		sem:       make(chan struct{}, maxConcurrent),
		locker:    NewMemLocker(),
		taskMax:   taskMax,
		execute:   execute,
		pollDueFn: pollDueFn,
	}
}

func (b *processor) Processor() Processor { return b.processor }

// String returns the string representation.
func (p Processor) String() string { return string(p) }

// Start launches the poll loop in a background goroutine (non-blocking).
func (b *processor) Start(ctx context.Context) {
	b.done = make(chan struct{})
	go func() {
		defer close(b.done)
		defer func() {
			if r := recover(); r != nil {
				log.Errorw("jobserver: processor loop panic recovered", "processor", b.processor, "recover", r)
			}
		}()
		b.runLoop(ctx)
	}()
}

// Wait blocks until the poll loop exits and all in-flight executions finish.
func (b *processor) Wait() {
	<-b.done
	b.wg.Wait()
}

func (b *processor) runLoop(ctx context.Context) {
	log.Infow("jobserver: processor loop start", "processor", b.processor, "pollInterval", b.interval)
	b.pollOnce(ctx)
	tick := time.NewTicker(b.interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			b.pollOnce(ctx)
		}
	}
}

func (b *processor) pollOnce(ctx context.Context) {
	nowMs := time.Now().UnixMilli()
	dues, err := b.pollDueFn(ctx, nowMs)
	if err != nil {
		log.Errorw("jobserver: poll due jobs failed", "processor", b.processor, "error", err)
		return
	}
	for _, d := range dues {
		d := d
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.runOne(ctx, d)
		}()
	}
}

func (b *processor) runOne(ctx context.Context, d job.DueJob) {
	if b.runOneFn != nil {
		b.runOneFn(ctx, d)
		return
	}
	b.runOneDefault(ctx, d)
}

func (b *processor) runOneDefault(ctx context.Context, d job.DueJob) {
	lockKey := string(b.processor) + ":" + strconv.Itoa(d.ID)

	qCtx, qCancel := context.WithTimeout(ctx, b.taskMax)
	defer qCancel()
	select {
	case b.sem <- struct{}{}:
	case <-qCtx.Done():
		log.Warnw("jobserver: queue timeout waiting for slot",
			"processor", b.processor, "id", d.ID, "error", qCtx.Err())
		return
	}
	defer func() { <-b.sem }()

	if !b.locker.TryAcquire(lockKey, time.UnixMilli(d.FireAtMs)) {
		log.Debugw("jobserver: skip, lock not acquired", "processor", b.processor, "id", d.ID)
		return
	}
	defer b.locker.Release(lockKey)

	runCtx, cancel := context.WithTimeout(ctx, b.taskMax)
	defer cancel()
	if err := b.execute(runCtx, d.ID, d.TriggerType); err != nil {
		log.Errorw("jobserver: execute failed", "processor", b.processor, "id", d.ID, "error", err)
	}
}
