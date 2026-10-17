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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

const repoTypeModels = "models"

// Service is an in-memory implementation of IModelCacheService. Task state is
// lost on restart and is not shared between replicas.
type Service struct {
	models     model.IModelService
	projects   project.IProjectRepo
	registries registry.IRegistryRepo
	git        git.IGitRepo
	lister     registrydiscovery.RemoteFileLister
	opts       Options

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// mu guards tasks, lastRequeue and modelLocks. Lock order: s.mu, then
	// task.mu.
	mu           sync.Mutex
	tasks        map[string]*task
	lastRequeue  map[string]time.Time
	modelLocks   map[string]*modelLock
	global       chan struct{}
	requeueGroup singleflight.Group
}

// modelLock serializes the pulls of one model. A channel, unlike a mutex,
// lets a waiting request give up when its context ends.
type modelLock struct {
	ch   chan struct{}
	refs int
}

// NewModelCacheService creates the service. Call Close on shutdown.
func NewModelCacheService(
	models model.IModelService,
	projects project.IProjectRepo,
	registries registry.IRegistryRepo,
	gitRepo git.IGitRepo,
	lister registrydiscovery.RemoteFileLister,
	opts Options,
) IModelCacheService {
	opts = opts.withDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		models:      models,
		projects:    projects,
		registries:  registries,
		git:         gitRepo,
		lister:      lister,
		opts:        opts,
		ctx:         ctx,
		cancel:      cancel,
		tasks:       map[string]*task{},
		lastRequeue: map[string]time.Time{},
		modelLocks:  map[string]*modelLock{},
		global:      make(chan struct{}, opts.GlobalConcurrency),
	}
}

func (s *Service) CacheModel(ctx context.Context, req *Request) (*CacheStatus, error) {
	if err := validateRevision(req.Revision); err != nil {
		return nil, err
	}
	sel, err := compileSelection(req.Selection)
	if err != nil {
		return nil, err
	}
	prj, reg, err := s.upstream(ctx, req.Project)
	if err != nil {
		return nil, err
	}

	if req.DryRun {
		remote, err := s.listUpstream(ctx, prj, reg, req.Name, req.Revision)
		if err != nil {
			return nil, err
		}
		p := sel.plan(remote.Files)
		key := taskKey(req.Project, req.Name, remote.Commit, p.objects)
		s.gc()
		gitCached := s.localHasCommit(ctx, req.Project, req.Name, remote.Commit)
		return s.buildStatus(ctx, req.Project, req.Name, req.Revision, remote.Commit, p, s.lookupTask(key), gitCached), nil
	}

	// Pulls of one model take turns, so a pull never deletes a model that a
	// concurrent pull created.
	unlock, err := s.lockModel(ctx, req.Project+"/"+req.Name)
	if err != nil {
		return nil, err
	}
	defer unlock()

	existed, err := s.modelExists(ctx, req.Project, req.Name)
	if err != nil {
		return nil, err
	}
	// remote is only listed for a new model: the sync creates the model, so
	// everything that upstream can tell is checked first, and a request that
	// cannot succeed leaves nothing behind.
	var remote *git.RevisionFiles
	if !existed {
		if remote, err = s.listUpstream(ctx, prj, reg, req.Name, req.Revision); err != nil {
			return nil, err
		}
		if len(sel.plan(remote.Files).files) == 0 {
			return nil, fmt.Errorf("%w: the selection matched no files", ErrInvalidSelection)
		}
		if err := s.checkCapacity(); err != nil {
			return nil, err
		}
	}
	if err := s.models.CheckOrSyncFromRemote(ctx, req.Project, req.Name); err != nil {
		if !existed {
			s.rollbackNewModel(context.WithoutCancel(ctx), req.Project, req.Name)
		}
		return nil, fmt.Errorf("%w: %w", ErrUpstreamSync, err)
	}

	local, err := s.listLocal(ctx, req.Project, req.Name, req.Revision)
	if errors.Is(err, ErrRevisionNotFound) {
		// The revision may exist upstream but be newer than the last sync.
		if rerr := s.models.RefreshFromRemote(ctx, req.Project, req.Name); rerr != nil {
			return nil, fmt.Errorf("%w: %w", ErrUpstreamSync, rerr)
		}
		local, err = s.listLocal(ctx, req.Project, req.Name, req.Revision)
	}
	if errors.Is(err, ErrRevisionNotFound) {
		return nil, s.explainMissingRevision(ctx, prj, reg, req, remote, err)
	}
	if err != nil {
		return nil, err
	}

	p := sel.plan(local.Files)
	if len(p.files) == 0 {
		return nil, fmt.Errorf("%w: the selection matched no files", ErrInvalidSelection)
	}
	key := taskKey(req.Project, req.Name, local.Commit, p.objects)
	s.gc()

	var missing []string
	for _, oid := range p.objects {
		if !s.git.LFSObjectExists(ctx, oid) {
			missing = append(missing, oid)
		}
	}
	t, err := s.startOrAttach(key, req.Project, req.Name, local.Commit, missing, p.sizes)
	if err != nil {
		return nil, err
	}
	return s.buildStatus(ctx, req.Project, req.Name, req.Revision, local.Commit, p, t, true), nil
}

func (s *Service) GetModelCache(ctx context.Context, project, name, revision string) (*CacheStatus, error) {
	exists, err := s.modelExists(ctx, project, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%w: %s/%s", ErrModelNotFound, project, name)
	}
	local, err := s.listLocal(ctx, project, name, revision)
	if err != nil {
		return nil, err
	}
	s.gc()
	t := s.latestTask(project, name, local.Commit)
	return s.buildStatus(ctx, project, name, revision, local.Commit, selectAll.plan(local.Files), t, true), nil
}

func (s *Service) Close() {
	// startOrAttach checks the context and adds to s.wg under s.mu, so no
	// task starts once Close waits.
	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
}

// validateRevision rejects refs that are not branches or tags: MatrixHub
// mirrors only those, so a ref such as refs/pr/1 could be previewed upstream
// but never pulled.
func validateRevision(revision string) error {
	if strings.HasPrefix(revision, "refs/") &&
		!strings.HasPrefix(revision, "refs/heads/") &&
		!strings.HasPrefix(revision, "refs/tags/") {
		return fmt.Errorf("%w: %s", ErrUnsupportedRevision, revision)
	}
	return nil
}

// explainMissingRevision tells a revision that upstream does not have from
// one it has outside the mirrored branches and tags, such as a commit that is
// only reachable from a pull request. remote is the upstream listing of the
// revision when the caller already has it.
func (s *Service) explainMissingRevision(ctx context.Context, prj *project.Project, reg *registry.Registry, req *Request, remote *git.RevisionFiles, notFound error) error {
	if remote == nil {
		var err error
		if remote, err = s.listUpstream(ctx, prj, reg, req.Name, req.Revision); err != nil {
			return notFound
		}
	}
	return fmt.Errorf("%w: %s resolves to %s upstream", ErrRevisionNotMirrored, revisionLabel(req.Revision), remote.Commit)
}

func revisionLabel(revision string) string {
	if revision == "" {
		return "the default branch"
	}
	return revision
}

// lockModel serializes the pulls of one model until the returned function is
// called.
func (s *Service) lockModel(ctx context.Context, key string) (func(), error) {
	s.mu.Lock()
	l := s.modelLocks[key]
	if l == nil {
		l = &modelLock{ch: make(chan struct{}, 1)}
		s.modelLocks[key] = l
	}
	l.refs++
	s.mu.Unlock()

	release := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if l.refs--; l.refs == 0 {
			delete(s.modelLocks, key)
		}
	}
	select {
	case l.ch <- struct{}{}:
		return func() {
			<-l.ch
			release()
		}, nil
	case <-ctx.Done():
		release()
		return nil, ctx.Err()
	}
}

// upstream returns the proxy project and its registry.
func (s *Service) upstream(ctx context.Context, projectName string) (*project.Project, *registry.Registry, error) {
	prj, err := s.projects.GetProjectByName(ctx, projectName)
	if err != nil {
		return nil, nil, err
	}
	if !prj.HasProxy() {
		return nil, nil, fmt.Errorf("%w: %s", ErrNotProxyProject, projectName)
	}
	reg, err := s.registries.GetRegistry(ctx, *prj.RegistryID)
	if err != nil {
		return nil, nil, fmt.Errorf("get registry(id=%d): %w", *prj.RegistryID, err)
	}
	return prj, reg, nil
}

func (s *Service) listUpstream(ctx context.Context, prj *project.Project, reg *registry.Registry, name, revision string) (*git.RevisionFiles, error) {
	files, err := s.lister.ListRevisionFiles(ctx, reg, prj.Organization, name, revision)
	switch {
	case err == nil:
		return files, nil
	case errors.Is(err, registrydiscovery.ErrRemoteNotFound):
		return nil, fmt.Errorf("%w: %w", ErrRevisionNotFound, err)
	case errors.Is(err, registrydiscovery.ErrRemoteAccessDenied):
		return nil, fmt.Errorf("%w: %w", ErrUpstreamAccessDenied, err)
	case errors.Is(err, registrydiscovery.ErrRemoteUnavailable):
		return nil, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	default:
		return nil, err
	}
}

func (s *Service) listLocal(ctx context.Context, project, name, revision string) (*git.RevisionFiles, error) {
	files, err := s.git.ListRevisionFiles(ctx, repoTypeModels, project, name, revision)
	if err != nil && isNotFound(err) {
		return nil, fmt.Errorf("%w: %w", ErrRevisionNotFound, err)
	}
	return files, err
}

// localHasCommit reports whether the local repository already has commit,
// i.e. whether the files stored in git are cached.
func (s *Service) localHasCommit(ctx context.Context, project, name, commit string) bool {
	exists, err := s.modelExists(ctx, project, name)
	if err != nil || !exists {
		return false
	}
	_, err = s.git.ListRevisionFiles(ctx, repoTypeModels, project, name, commit)
	return err == nil
}

func (s *Service) modelExists(ctx context.Context, project, name string) (bool, error) {
	_, err := s.models.GetModel(ctx, project, name)
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, err
}

// rollbackNewModel removes a model whose first sync failed, unless the model
// was synced in the meantime, e.g. by a proxy download.
func (s *Service) rollbackNewModel(ctx context.Context, project, name string) {
	mod, err := s.models.GetModel(ctx, project, name)
	if err != nil || mod.SyncedAt != nil {
		return
	}
	if err := s.models.DeleteModel(ctx, project, name); err != nil {
		log.Warnf("model cache: roll back model %s/%s after failed sync: %v", project, name, err)
	}
}

func (s *Service) lookupTask(key string) *task {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tasks[key]
}

// latestTask returns the task of a commit that is running, or else the one
// started last. Tasks of narrower selections count too.
func (s *Service) latestTask(project, name, commit string) *task {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest *task
	latestRunning := false
	for _, t := range s.tasks {
		if t.project != project || t.name != name || t.commit != commit {
			continue
		}
		running := t.running()
		if latest == nil ||
			(running && !latestRunning) ||
			(running == latestRunning && t.startedAt.After(latest.startedAt)) {
			latest, latestRunning = t, running
		}
	}
	return latest
}

// checkCapacity reports ErrTooManyTasks before a new model is created;
// startOrAttach checks again when it starts the task.
func (s *Service) checkCapacity() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := s.runningTasksLocked(); n >= s.opts.MaxRunningTasks {
		return fmt.Errorf("%w: %d running", ErrTooManyTasks, n)
	}
	return nil
}

func (s *Service) runningTasksLocked() int {
	running := 0
	for _, t := range s.tasks {
		if t.running() {
			running++
		}
	}
	return running
}

// startOrAttach returns the running task of key, or starts one for the
// missing objects. It returns the last finished task, possibly nil, when
// nothing is missing.
func (s *Service) startOrAttach(key, project, name, commit string, missing []string, sizes map[string]int64) (*task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx.Err() != nil {
		return nil, ErrClosed
	}
	if t, ok := s.tasks[key]; ok && t.running() {
		return t, nil
	}
	if len(missing) == 0 {
		return s.tasks[key], nil
	}
	if running := s.runningTasksLocked(); running >= s.opts.MaxRunningTasks {
		return nil, fmt.Errorf("%w: %d running", ErrTooManyTasks, running)
	}

	// A new run replaces the finished one of the same key and gets its own
	// ID.
	t := newTask(key, project, name, commit, missing, sizes, s.opts.Now())
	s.tasks[key] = t
	s.wg.Add(1)
	go s.run(t)
	log.Infof("model cache: started task %s (%s) for %d objects", t.id, key, len(missing))
	return t, nil
}

// runningObjects returns the download progress of objects that running tasks
// of the same model, other than exclude, are downloading.
func (s *Service) runningObjects(project, name string, exclude *task) map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]int64{}
	for _, t := range s.tasks {
		if t == exclude || t.project != project || t.name != name || !t.running() {
			continue
		}
		for oid, o := range t.snapshotObjects() {
			if o.state == FileDownloading {
				out[oid] = o.downloaded
			}
		}
	}
	return out
}

// gc drops finished tasks older than FinishedTaskTTL.
func (s *Service) gc() {
	now := s.opts.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, t := range s.tasks {
		if finishedAt := t.finished(); finishedAt != nil && now.Sub(*finishedAt) > s.opts.FinishedTaskTTL {
			delete(s.tasks, key)
		}
	}
	for key, at := range s.lastRequeue {
		if now.Sub(at) > s.opts.FinishedTaskTTL {
			delete(s.lastRequeue, key)
		}
	}
}

// buildStatus combines the selected files, the local storage and the task
// state. gitCached tells whether files stored in git are present locally.
func (s *Service) buildStatus(ctx context.Context, project, name, revision, commit string, p *plan, t *task, gitCached bool) *CacheStatus {
	st := &CacheStatus{
		Project:           project,
		Name:              name,
		RequestedRevision: revision,
		Commit:            commit,
		Warnings:          p.warnings,
		Files:             make([]FileStatus, 0, len(p.files)),
	}

	var objects map[string]objectState
	if t != nil {
		var ts TaskStatus
		objects, ts = t.snapshot()
		st.Task = &ts
	}
	running := st.Task != nil && st.Task.State == TaskRunning
	others := s.runningObjects(project, name, t)

	allCached := len(p.files) > 0
	cachedObjects := 0
	for _, f := range p.files {
		fs := FileStatus{Path: f.Path, Size: f.Size, IsLFS: f.IsLFS, OID: f.OID, State: FileMissing}
		o, tracked := objects[f.OID]
		switch {
		case !f.IsLFS:
			if gitCached {
				fs.State = FileCached
			}
		case s.git.LFSObjectExists(ctx, f.OID):
			fs.State = FileCached
		case tracked && o.state != FileCached:
			fs.State, fs.DownloadedBytes, fs.Error = o.state, o.downloaded, o.err
		case hasKey(others, f.OID):
			fs.State, fs.DownloadedBytes = FileDownloading, others[f.OID]
		}
		if fs.State == FileQueued && !running {
			fs.State = FileMissing
		}
		addTotals(&st.Totals, fs)
		switch {
		case fs.State != FileCached:
			allCached = false
		case f.IsLFS:
			cachedObjects++
		}
		st.Files = append(st.Files, fs)
	}

	switch {
	case allCached:
		st.State = CacheStateCached
	case cachedObjects > 0:
		st.State = CacheStatePartial
	default:
		st.State = CacheStateMissing
	}
	return st
}

func addTotals(t *Totals, f FileStatus) {
	t.Files++
	t.Bytes += f.Size
	switch f.State {
	case FileCached:
		t.CachedFiles++
		t.CachedBytes += f.Size
	case FileDownloading:
		t.DownloadingFiles++
		t.DownloadingBytes += f.DownloadedBytes
	case FileQueued:
		t.QueuedFiles++
	case FileFailed:
		t.FailedFiles++
	default:
		t.MissingFiles++
	}
}

func hasKey(m map[string]int64, k string) bool {
	_, ok := m[k]
	return ok
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist")
}
