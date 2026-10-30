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
	"runtime/debug"
	"sync"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// Errors of an object. A status shows them to everyone who can read the
// model, so they never carry errors of other layers, which may name server
// paths; see publicError.
var (
	errNotQueued      = errors.New("object is not queued for download after refreshing the model from upstream")
	errDownloadFailed = errors.New("download failed; see server logs for details")
	errStalled        = errors.New("download made no progress")
	errNotStored      = errors.New("download finished but was not stored; a client may still be reading it")
	errModelGone      = errors.New("the model was deleted")
	errResyncFailed   = errors.New("refreshing the model from upstream failed; see server logs for details")
)

// errPanicked reports a panic that a background goroutine recovered from.
var errPanicked = errors.New("recovered from a panic")

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
	id      string
	key     string
	project string
	name    string
	// commit is the commit that the run started for. Runs are keyed by their
	// objects, so pulls of later commits with the same objects attach.
	commit    string
	startedAt time.Time

	mu         sync.Mutex
	state      TaskState
	finishedAt *time.Time
	err        string
	objects    map[string]*objectState
	// queue holds the objects that wait for a worker, and active counts the
	// workers: the run finishes when both are empty. wake tells the
	// dispatcher that one of them changed.
	queue  []string
	active int
	wake   chan struct{}
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
		queue:     oids,
		objects:   make(map[string]*objectState, len(oids)),
		wake:      make(chan struct{}, 1),
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

func (t *task) failed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state == TaskFailed
}

func (t *task) finished() *time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finishedAt
}

// within reports whether every object that the run pulls is in oids.
func (t *task) within(oids map[string]struct{}) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	for oid := range t.objects {
		if _, ok := oids[oid]; !ok {
			return false
		}
	}
	return true
}

// failedObjects returns the objects that the run failed.
func (t *task) failedObjects() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var failed []string
	for oid, o := range t.objects {
		if o.state == FileFailed {
			failed = append(failed, oid)
		}
	}
	return failed
}

func (t *task) update(oid string, fn func(o *objectState)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if o, ok := t.objects[oid]; ok {
		fn(o)
	}
}

// retry queues the failed objects among missing again while the run goes on,
// and reports whether it does.
func (t *task) retry(missing []string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state != TaskRunning {
		return false
	}
	queued := false
	for _, oid := range missing {
		if o, ok := t.objects[oid]; ok && o.state == FileFailed {
			o.state, o.downloaded, o.attempts, o.err = FileQueued, 0, 0, ""
			t.queue = append(t.queue, oid)
			queued = true
		}
	}
	if queued {
		t.signal()
	}
	return true
}

// signal wakes the dispatcher. Signals that it has not seen yet merge.
func (t *task) signal() {
	select {
	case t.wake <- struct{}{}:
	default:
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

// run hands the task's objects to at most PerTaskConcurrency workers and
// records the outcome. It runs on the service context, not on a request
// context.
func (s *Service) run(t *task) {
	defer s.wg.Done()
	for {
		oid, done := s.next(t)
		if done {
			return
		}
		if oid != "" {
			go s.work(t, oid)
			continue
		}
		// Wait for a worker to end or for retried objects. Once the service
		// closes, only the workers' ends matter.
		if s.ctx.Err() != nil {
			<-t.wake
			continue
		}
		select {
		case <-t.wake:
		case <-s.ctx.Done():
		}
	}
}

// next returns the object to hand to a new worker, "" when the dispatcher
// must wait, or done once the run finished. It finishes the run under the
// lock that retry takes, so no object is queued into a finished run.
func (s *Service) next(t *task) (oid string, done bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	closing := s.ctx.Err() != nil
	if !closing && len(t.queue) > 0 && t.active < s.opts.PerTaskConcurrency {
		oid, t.queue = t.queue[0], t.queue[1:]
		t.active++
		return oid, false
	}
	if t.active == 0 && (closing || len(t.queue) == 0) {
		s.finishLocked(t)
		return "", true
	}
	return "", false
}

// work fetches one object. It runs on its own goroutine, where no request
// handler recovers a panic, so it recovers one itself and fails the object
// instead of ending the process.
func (s *Service) work(t *task, oid string) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("model cache: object %s of %s/%s: %v: %v\n%s", oid, t.project, t.name, errPanicked, r, debug.Stack())
			t.update(oid, func(o *objectState) { o.state, o.err = FileFailed, errDownloadFailed.Error() })
		}
		t.mu.Lock()
		t.active--
		t.mu.Unlock()
		t.signal()
	}()
	s.fetchWithRetry(t, oid)
}

// finishLocked records the outcome of the run; t.mu is held.
func (s *Service) finishLocked(t *task) {
	now := s.opts.Now()
	interrupted := s.ctx.Err() != nil
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
	case failed == 0:
		t.state = TaskSucceeded
	case interrupted:
		t.state, t.err = TaskFailed, "interrupted: server shutting down"
	default:
		t.state, t.err = TaskFailed, fmt.Sprintf("%d of %d LFS objects failed", failed, len(t.objects))
	}
	t.finishedAt = &now
	log.Infof("model cache: task %s (%s) finished: state=%d failed=%d", t.id, t.key, t.state, failed)
}

// slot is one of the GlobalConcurrency download slots of the server, held by
// one worker.
type slot struct {
	ch   chan struct{}
	held bool
}

func (sl *slot) acquire(ctx context.Context) error {
	if sl.held {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case sl.ch <- struct{}{}:
		sl.held = true
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (sl *slot) release() {
	if sl.held {
		<-sl.ch
		sl.held = false
	}
}

func (s *Service) fetchWithRetry(t *task, oid string) {
	sl := &slot{ch: s.global}
	defer sl.release()
	for attempt := 1; attempt <= s.opts.MaxAttempts; attempt++ {
		t.update(oid, func(o *objectState) { o.attempts = attempt })
		err := s.fetchOnce(t, oid, sl)
		sl.release()
		if err == nil {
			t.update(oid, func(o *objectState) {
				o.state, o.downloaded, o.err = FileCached, o.size, ""
			})
			return
		}
		// The mirror cannot cancel a download, and a retry would attach to
		// the same download, so a stalled or unstored download ends the
		// object; so does a deleted model.
		if s.ctx.Err() != nil || attempt == s.opts.MaxAttempts ||
			errors.Is(err, errStalled) || errors.Is(err, errNotStored) || errors.Is(err, errModelGone) {
			log.Warnf("model cache: object %s of %s/%s failed after %d attempts: %v", oid, t.project, t.name, attempt, err)
			t.update(oid, func(o *objectState) {
				o.state, o.err = FileFailed, publicError(err)
			})
			return
		}
		backoff := s.opts.RetryBackoff << (attempt - 1)
		t.update(oid, func(o *objectState) {
			o.state = FileQueued
			o.err = fmt.Sprintf("attempt %d/%d failed: %s; retrying in %s", attempt, s.opts.MaxAttempts, publicError(err), backoff)
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
// is stored. It holds a download slot only while the object downloads, and
// the object shows as downloading only from then on: while it waits for a
// slot or a re-sync, it stays queued.
func (s *Service) fetchOnce(t *task, oid string, sl *slot) error {
	if s.git.LFSObjectExists(s.ctx, oid) {
		return nil
	}
	if err := sl.acquire(s.ctx); err != nil {
		return err
	}
	if s.git.FetchLFSObject(s.ctx, oid) == nil {
		if s.git.LFSObjectExists(s.ctx, oid) {
			return nil
		}
		// A sync is no download: the slot is free meanwhile.
		sl.release()
		if err := s.requeue(t.project, t.name); err != nil {
			return err
		}
		if err := sl.acquire(s.ctx); err != nil {
			return err
		}
		if s.git.FetchLFSObject(s.ctx, oid) == nil {
			if s.git.LFSObjectExists(s.ctx, oid) {
				return nil
			}
			return errNotQueued
		}
	}
	t.update(oid, func(o *objectState) { o.state = FileDownloading })
	return s.waitStored(t, oid, sl)
}

// waitStored polls until the object is stored. The mirror moves a finished
// download into storage before it forgets the download, so storage is checked
// first and again whenever the download disappears.
func (s *Service) waitStored(t *task, oid string, sl *slot) error {
	ticker := time.NewTicker(s.opts.PollInterval)
	defer ticker.Stop()

	last := int64(-1)
	lastChange := s.opts.Now()
	var complete time.Time
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
		now := s.opts.Now()
		switch {
		case downloaded >= d.Total():
			// A complete download is stored once its readers close. That
			// is not a stall, and it downloads nothing, so the slot is
			// free; but a reader that never closes must not keep the task
			// running forever.
			sl.release()
			if complete.IsZero() {
				complete = now
			}
			if now.Sub(complete) > s.opts.StoreTimeout {
				return errNotStored
			}
			lastChange = now
		case downloaded != last:
			last = downloaded
			lastChange = now
		}
		if s.opts.StallTimeout > 0 && now.Sub(lastChange) > s.opts.StallTimeout {
			return fmt.Errorf("%w for %s", errStalled, s.opts.StallTimeout)
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-ticker.C:
		}
	}
}

// refreshResult is the outcome of the last forced sync of a model.
type refreshResult struct {
	at  time.Time
	err error
}

// refresh forces a sync of the model, which also re-queues its missing
// objects. Calls for one model are merged and rate limited, because one sync
// re-queues every missing object of the repository: a call within
// RequeueMinInterval of the last sync gets that sync's result. The sync runs
// on the service context for at most RefreshTimeout, and the caller stops
// waiting when its own context ends.
func (s *Service) refresh(ctx context.Context, project, name string) error {
	key := project + "/" + name
	ch := s.refreshGroup.DoChan(key, func() (any, error) {
		s.mu.Lock()
		last, ok := s.lastRefresh[key]
		s.mu.Unlock()
		if ok && s.opts.Now().Sub(last.at) < s.opts.RequeueMinInterval {
			return nil, last.err
		}
		start := s.opts.Now()
		rctx, cancel := context.WithTimeout(s.ctx, s.opts.RefreshTimeout)
		defer cancel()
		// RefreshFromRemote never creates the model, so a deleted model
		// stays deleted.
		err := untilDone(rctx, func(ctx context.Context) error {
			return s.models.RefreshFromRemote(ctx, project, name)
		})
		if err != nil && !errors.Is(err, model.ErrModelNotFound) {
			log.Warnf("model cache: refresh of %s/%s from upstream failed: %v", project, name, err)
		}
		s.mu.Lock()
		s.lastRefresh[key] = refreshResult{at: start, err: err}
		s.mu.Unlock()
		return nil, err
	})
	select {
	case r := <-ch:
		return r.Err
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// requeue forces a sync of the model so the mirror queues its missing objects
// again; see refresh.
func (s *Service) requeue(project, name string) error {
	switch err := s.refresh(s.ctx, project, name); {
	case err == nil:
		return nil
	case errors.Is(err, model.ErrModelNotFound):
		return errModelGone
	default:
		return fmt.Errorf("%w: %w", errResyncFailed, err)
	}
}

// untilDone runs fn and returns its error, or the cause of ctx's end once ctx
// ends. The mirror runs git without a wait delay, so a git command can outlive
// its context until upstream answers; the caller then stops waiting, and fn
// ends in the background.
//
// fn runs on its own goroutine, where no request handler recovers a panic, so
// a panic becomes an error here instead of ending the process.
func untilDone(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("model cache: %v: %v\n%s", errPanicked, r, debug.Stack())
				done <- fmt.Errorf("%w: %v", errPanicked, r)
			}
		}()
		done <- fn(ctx)
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		select {
		case err := <-done:
			return err
		default:
			return context.Cause(ctx)
		}
	}
}

// publicError returns the message that a status shows for err.
func publicError(err error) string {
	switch {
	case errors.Is(err, errStalled):
		return err.Error()
	case errors.Is(err, context.Canceled):
		return "interrupted: server shutting down"
	}
	for _, known := range []error{errResyncFailed, errModelGone, errNotQueued, errNotStored, errDownloadFailed} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return errDownloadFailed.Error()
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
