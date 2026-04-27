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
	"sync"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/canceller"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/logstore"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver/processor"
)

// JobServer owns one goroutine per processor (syncPolicy, ...). Processors do not share state with each other.
type JobServer struct {
	cfg        *config.JobServerConfig
	processors []processor.Adapter
}

// New builds a JobServer from config and domain services (cfg is copied; defaults applied without mutating the caller's struct).
func New(cfg *config.JobServerConfig, syncSvc syncpolicy.ISyncPolicyService, syncJobSvc syncjob.ISyncJobService, ls logstore.LogStore, canc canceller.Canceller) *JobServer {
	c := *cfg
	if c.ShutdownGrace == 0 {
		c.ShutdownGrace = 30 * time.Second
	}
	if c.SyncPolicy.PollInterval == 0 {
		c.SyncPolicy.PollInterval = 10 * time.Second
	}
	if c.SyncPolicy.MaxConcurrent == 0 {
		c.SyncPolicy.MaxConcurrent = 5
	}
	if c.SyncPolicy.TaskMaxDuration == 0 {
		c.SyncPolicy.TaskMaxDuration = 2 * time.Hour
	}
	if c.SyncTask.PollInterval == 0 {
		c.SyncTask.PollInterval = 5 * time.Second
	}
	if c.SyncTask.MaxConcurrent == 0 {
		c.SyncTask.MaxConcurrent = 5
	}
	if c.SyncTask.TaskMaxDuration == 0 {
		c.SyncTask.TaskMaxDuration = 2 * time.Hour
	}
	if c.SyncJob.PollInterval == 0 {
		c.SyncJob.PollInterval = 3 * time.Second
	}
	if c.SyncJob.MaxConcurrent == 0 {
		c.SyncJob.MaxConcurrent = 5
	}
	if c.SyncJob.TaskMaxDuration == 0 {
		c.SyncJob.TaskMaxDuration = 2 * time.Hour
	}
	return &JobServer{
		cfg: &c,
		processors: []processor.Adapter{
			processor.NewSyncPolicyProcessor(c.SyncPolicy, syncSvc),
			processor.NewSyncTaskProcessor(c.SyncTask, syncSvc),
			processor.NewSyncJobProcessor(c.SyncJob, syncJobSvc, canc),
		},
	}
}

// Run starts all processor loops and blocks until ctx is cancelled.
func (js *JobServer) Run(ctx context.Context) {
	for _, p := range js.processors {
		p.Start(ctx)
	}
	<-ctx.Done()
	log.Info("jobserver: context cancelled, processor loops stopping")
}

// Shutdown waits for each processor's Wait() with a global grace timeout.
func (js *JobServer) Shutdown(grace time.Duration) {
	if grace <= 0 {
		grace = 30 * time.Second
	}
	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for _, p := range js.processors {
			wg.Add(1)
			go func(p processor.Adapter) {
				defer wg.Done()
				p.Wait()
			}(p)
		}
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		log.Info("jobserver: all processors stopped")
	case <-time.After(grace):
		log.Warnw("jobserver: shutdown grace exceeded", "grace", grace)
	}
}
