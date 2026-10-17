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

package modelcache

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

var (
	errNotQueued      = errors.New("object is not queued for download after refreshing the model from upstream")
	errDownloadFailed = errors.New("download failed; see server logs for details")
	errStalled        = errors.New("download made no progress")
)

type objectState struct {
	size       int64
	state      FileState
	downloaded int64
	attempts   int
	err        string
}

// task is one run that pulls the missing LFS objects of one selection.
type task struct {
	// id identifies the run; key identifies the selection, so that requests
	// for the same files attach to a running task.
	id        string
	key       string
	project   string
	name      string
	commit    string
	startedAt time.Time

	mu         sync.Mutex
	state      TaskState
	finishedAt *time.Time
	err        string
	order      []string
	objects    map[string]*objectState
}

func newTask(key, project, name, commit string, oids []string, sizes map[string]int64, now time.Time) *task {
	t := &task{
		id:        rand.Text(),
		key:       key,
		project:   project,
		name:      name,
		commit:    commit,
		startedAt: now,
		state:     TaskRunning,
		order:     oids,
		objects:   make(map[string]*objectState, len(oids)),
	}
	for _, oid := range oids {
		t.objects[oid] = &objectState{size: sizes[oid], state: FileQueued}
	}
	return t
}

func (t *task) running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state == TaskRunning
}

func (t *task) finished() *time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finishedAt
}

func (t *task) update(oid string, fn func(o *objectState)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if o, ok := t.objects[oid]; ok {
		fn(o)
	}
}

func (t *task) snapshotObjects() map[string]objectState {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]objectState, len(t.objects))
	for oid, o := range t.objects {
		out[oid] = *o
	}
	return out
}

// snapshot copies the objects and the status under one lock, so a finished
// status never comes with objects from before the end.
func (t *task) snapshot() (map[string]objectState, TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	objects := make(map[string]objectState, len(t.objects))
	for oid, o := range t.objects {
		objects[oid] = *o
	}
	st := TaskStatus{ID: t.id, State: t.state, StartedAt: t.startedAt, Error: t.err}
	if t.finishedAt != nil {
		finished := *t.finishedAt
		st.FinishedAt = &finished
	}
	return objects, st
}

// run downloads the task's objects with bounded concurrency and records the
// outcome. It runs on the service context, not on a request context.
func (s *Service) run(t *task) {
	defer s.wg.Done()

	perTask := make(chan struct{}, s.opts.PerTaskConcurrency)
	var workers sync.WaitGroup
dispatch:
	for _, oid := range t.order {
		if s.git.LFSObjectExists(s.ctx, oid) {
			t.update(oid, func(o *objectState) { o.state = FileCached })
			continue
		}
		select {
		case perTask <- struct{}{}:
		case <-s.ctx.Done():
			break dispatch
		}
		select {
		case s.global <- struct{}{}:
		case <-s.ctx.Done():
			<-perTask
			break dispatch
		}
		workers.Add(1)
		go func(oid string) {
			defer workers.Done()
			defer func() {
				<-s.global
				<-perTask
			}()
			s.fetchWithRetry(t, oid)
		}(oid)
	}
	workers.Wait()
	s.finish(t)
}

func (s *Service) finish(t *task) {
	now := s.opts.Now()
	interrupted := s.ctx.Err() != nil

	t.mu.Lock()
	defer t.mu.Unlock()
	failed := 0
	for _, o := range t.objects {
		if o.state == FileCached {
			continue
		}
		failed++
		if interrupted && o.state != FileFailed {
			o.state, o.err = FileFailed, "interrupted: server shutting down"
		}
	}
	switch {
	case interrupted:
		t.state, t.err = TaskFailed, "interrupted: server shutting down"
	case failed > 0:
		t.state, t.err = TaskFailed, fmt.Sprintf("%d of %d files failed", failed, len(t.objects))
	default:
		t.state = TaskSucceeded
	}
	t.finishedAt = &now
	log.Infof("model cache: task %s (%s) finished: state=%d failed=%d", t.id, t.key, t.state, failed)
}

func (s *Service) fetchWithRetry(t *task, oid string) {
	for attempt := 1; attempt <= s.opts.MaxAttempts; attempt++ {
		t.update(oid, func(o *objectState) {
			o.state, o.attempts = FileDownloading, attempt
		})
		err := s.fetchOnce(t, oid)
		if err == nil {
			t.update(oid, func(o *objectState) {
				o.state, o.downloaded, o.err = FileCached, o.size, ""
			})
			return
		}
		// The mirror cannot cancel a stalled download, and a retry would
		// attach to the same download, so a stall ends the object.
		if s.ctx.Err() != nil || attempt == s.opts.MaxAttempts || errors.Is(err, errStalled) {
			log.Warnf("model cache: object %s of %s/%s failed after %d attempts: %v", oid, t.project, t.name, attempt, err)
			t.update(oid, func(o *objectState) {
				o.state, o.err = FileFailed, err.Error()
			})
			return
		}
		backoff := s.opts.RetryBackoff << (attempt - 1)
		t.update(oid, func(o *objectState) {
			o.err = fmt.Sprintf("attempt %d/%d failed: %v; retrying in %s", attempt, s.opts.MaxAttempts, err, backoff)
		})
		if !sleepCtx(s.ctx, backoff) {
			t.update(oid, func(o *objectState) {
				o.state, o.err = FileFailed, "interrupted: server shutting down"
			})
			return
		}
	}
}

// fetchOnce makes sure the object is downloading, re-queuing it through a
// forced model sync when the mirror no longer tracks it, and waits until it
// is stored.
func (s *Service) fetchOnce(t *task, oid string) error {
	if s.git.LFSObjectExists(s.ctx, oid) {
		return nil
	}
	if s.git.FetchLFSObject(s.ctx, oid) == nil {
		if s.git.LFSObjectExists(s.ctx, oid) {
			return nil
		}
		if err := s.requeue(t.project, t.name); err != nil {
			return err
		}
		if s.git.FetchLFSObject(s.ctx, oid) == nil {
			if s.git.LFSObjectExists(s.ctx, oid) {
				return nil
			}
			return errNotQueued
		}
	}
	return s.waitStored(t, oid)
}

// waitStored polls until the object is stored. The mirror moves a finished
// download into storage before it forgets the download, so storage is checked
// first and again whenever the download disappears.
func (s *Service) waitStored(t *task, oid string) error {
	ticker := time.NewTicker(s.opts.PollInterval)
	defer ticker.Stop()

	last := int64(-1)
	lastChange := s.opts.Now()
	for {
		if s.git.LFSObjectExists(s.ctx, oid) {
			return nil
		}
		d := s.git.FetchLFSObject(s.ctx, oid)
		if d == nil {
			if s.git.LFSObjectExists(s.ctx, oid) {
				return nil
			}
			return errDownloadFailed
		}
		downloaded := d.Downloaded()
		t.update(oid, func(o *objectState) { o.downloaded = downloaded })
		// A complete download waits for readers to close before it is
		// stored; that is not a stall.
		if downloaded != last || downloaded >= d.Total() {
			last = downloaded
			lastChange = s.opts.Now()
		}
		if s.opts.StallTimeout > 0 && s.opts.Now().Sub(lastChange) > s.opts.StallTimeout {
			return fmt.Errorf("%w for %s", errStalled, s.opts.StallTimeout)
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-ticker.C:
		}
	}
}

// requeue forces a sync of the model so the mirror queues its missing objects
// again. Concurrent calls for one model are merged and rate limited, because
// one sync re-queues every missing object of the repository.
func (s *Service) requeue(project, name string) error {
	key := project + "/" + name
	_, err, _ := s.requeueGroup.Do(key, func() (any, error) {
		s.mu.Lock()
		last, ok := s.lastRequeue[key]
		s.mu.Unlock()
		if ok && s.opts.Now().Sub(last) < s.opts.RequeueMinInterval {
			return nil, nil
		}
		// A deleted model must not be re-created by the sync.
		if _, err := s.models.GetModel(s.ctx, project, name); err != nil {
			return nil, fmt.Errorf("model %s/%s is no longer available: %w", project, name, err)
		}
		err := s.models.RefreshFromRemote(s.ctx, project, name)
		s.mu.Lock()
		s.lastRequeue[key] = s.opts.Now()
		s.mu.Unlock()
		return nil, err
	})
	return err
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
