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

	// mu guards tasks, lastRefresh and modelLocks. Lock order: s.mu, then
	// task.mu.
	mu           sync.Mutex
	tasks        map[string]*task
	lastRefresh  map[string]refreshResult
	modelLocks   map[string]*modelLock
	global       chan struct{}
	refreshGroup singleflight.Group
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
		lastRefresh: map[string]refreshResult{},
		modelLocks:  map[string]*modelLock{},
		global:      make(chan struct{}, opts.GlobalConcurrency),
	}
}

func (s *Service) CacheModel(ctx context.Context, req *Request) (_ *CacheStatus, err error) {
	if s.ctx.Err() != nil {
		return nil, ErrClosed
	}
	ctx, done := s.untilClosed(ctx)
	defer func() {
		if err != nil && errors.Is(context.Cause(ctx), ErrClosed) {
			err = ErrClosed
		}
		done()
	}()
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
		p, err := sel.plan(ctx, remote.Files)
		if err != nil {
			return nil, err
		}
		s.gc()
		gitCached := s.localHasCommit(ctx, req.Project, req.Name, remote.Commit)
		t := s.reportable(ctx, s.lookupTask(taskKey(req.Project, req.Name, p.objects)))
		return s.buildStatus(ctx, req.Project, req.Name, req.Revision, remote, p, t, gitCached), nil
	}

	// Pulls of one model take turns, so two pulls of a new model do not both
	// try to create it.
	unlock, err := s.lockModel(ctx, req.Project+"/"+req.Name)
	if err != nil {
		return nil, err
	}
	defer unlock()

	m, err := s.findModel(ctx, req.Project, req.Name)
	if err != nil {
		return nil, err
	}
	// remote is only listed for a model that has not synced yet: for a new
	// model the sync creates it, so everything that upstream can tell is
	// checked first, and a request that cannot succeed creates nothing; a
	// model whose first sync failed gets the same answers, such as a gated
	// repository, instead of failing the sync again. A sync that fails after
	// that keeps the model, as a failed proxy download does, and the next
	// pull or download syncs it again.
	var remote *git.RevisionFiles
	if m == nil || m.SyncedAt == nil {
		if remote, err = s.listUpstream(ctx, prj, reg, req.Name, req.Revision); err != nil {
			return nil, err
		}
		p, err := sel.plan(ctx, remote.Files)
		if err != nil {
			return nil, err
		}
		if len(p.files) == 0 {
			return nil, fmt.Errorf("%w: the selection matched no files", ErrInvalidSelection)
		}
		// Other models may have stored the objects already, and a pull that
		// downloads nothing starts no task.
		if len(s.missingObjects(ctx, p.objects)) > 0 {
			if err := s.checkCapacity(); err != nil {
				return nil, err
			}
		}
	}
	if err := untilDone(ctx, func(ctx context.Context) error {
		return s.models.CheckOrSyncFromRemote(ctx, req.Project, req.Name)
	}); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstreamSync, err)
	}

	local, err := s.listLocal(ctx, req.Project, req.Name, req.Revision)
	if errors.Is(err, ErrRevisionNotFound) {
		local, err = s.fetchRevision(ctx, prj, reg, req, remote, err)
	}
	if err != nil {
		return nil, err
	}

	p, err := sel.plan(ctx, local.Files)
	if err != nil {
		return nil, err
	}
	if len(p.files) == 0 {
		return nil, fmt.Errorf("%w: the selection matched no files", ErrInvalidSelection)
	}
	s.gc()
	key := taskKey(req.Project, req.Name, p.objects)
	t, err := s.startOrAttach(key, req.Project, req.Name, local.Commit, s.missingObjects(ctx, p.objects), p.sizes)
	if err != nil {
		return nil, err
	}
	return s.buildStatus(ctx, req.Project, req.Name, req.Revision, local, p, t, true), nil
}

func (s *Service) GetModelCache(ctx context.Context, project, name, revision string) (*CacheStatus, error) {
	if err := validateRevision(revision); err != nil {
		return nil, err
	}
	m, err := s.models.GetModel(ctx, project, name)
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("%w: %s/%s", ErrModelNotFound, project, name)
		}
		return nil, err
	}
	if m.SyncedAt == nil {
		prj, err := s.projects.GetProjectByName(ctx, project)
		if err != nil {
			return nil, err
		}
		if prj.HasProxy() {
			// Until its first sync succeeds, a proxy model holds only the
			// placeholder commit that creating it wrote: none of its files
			// are known.
			return &CacheStatus{
				Project:           project,
				Name:              name,
				RequestedRevision: revision,
				State:             CacheStateMissing,
				Files:             []FileStatus{},
				Warnings:          []string{"the model has not been synced from upstream yet"},
			}, nil
		}
	}
	local, err := s.listLocal(ctx, project, name, revision)
	if err != nil {
		return nil, err
	}
	p, err := selectAll.plan(ctx, local.Files)
	if err != nil {
		return nil, err
	}
	s.gc()
	t := s.reportable(ctx, s.latestTask(project, name, local.Commit, p.objects))
	return s.buildStatus(ctx, project, name, revision, local, p, t, true), nil
}

func (s *Service) Close() {
	// startOrAttach checks the context and adds to s.wg under s.mu, so no
	// task starts once Close waits.
	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
}

// untilClosed returns ctx, also ended with the cause ErrClosed once the
// service closes, so that a request waiting for a model lock or a sync does
// not hold up shutdown. Call the returned function when the request ends.
func (s *Service) untilClosed(ctx context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(ctx)
	stop := context.AfterFunc(s.ctx, func() { cancel(ErrClosed) })
	return ctx, func() {
		stop()
		cancel(nil)
	}
}

// validateRevision accepts the default branch (empty), branch and tag names,
// and commit ids. MatrixHub mirrors only branches and tags, so a ref such as
// refs/pr/1 could be previewed upstream but never pulled; and a revision
// expression such as main~1 would resolve locally but not upstream.
func validateRevision(revision string) error {
	if revision == "" {
		return nil
	}
	if strings.HasPrefix(revision, "refs/") &&
		!strings.HasPrefix(revision, "refs/heads/") &&
		!strings.HasPrefix(revision, "refs/tags/") {
		return fmt.Errorf("%w: %s", ErrUnsupportedRevision, revision)
	}
	if !isRefName(revision) {
		return fmt.Errorf("%w: %q is not a branch, tag or commit name", ErrUnsupportedRevision, revision)
	}
	// git allows '@' in ref names, but the revision syntax of go-git, which
	// MatrixHub resolves revisions with, rejects it.
	if strings.Contains(revision, "@") {
		return fmt.Errorf("%w: MatrixHub cannot resolve %q, which contains '@'", ErrUnsupportedRevision, revision)
	}
	return nil
}

// isRefName applies the rules of git check-ref-format to a branch or tag
// name; commit ids follow them too.
func isRefName(name string) bool {
	if name == "@" || strings.HasPrefix(name, "-") || strings.HasPrefix(name, "/") ||
		strings.HasSuffix(name, "/") || strings.HasSuffix(name, ".") ||
		strings.Contains(name, "..") || strings.Contains(name, "//") || strings.Contains(name, "@{") {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}

// fetchRevision handles a revision that the mirror does not have, as upstream
// tells: a revision that upstream does not have either is not found; one that
// it has is newer than the last sync, which a forced sync fetches, or outside
// the mirrored branches and tags, such as a commit that only a pull request
// reaches. Only a revision that upstream has forces a sync, and refresh rate
// limits those. remote is the upstream listing that a new model was checked
// with: that model synced just now, so no sync is forced.
func (s *Service) fetchRevision(ctx context.Context, prj *project.Project, reg *registry.Registry, req *Request, remote *git.RevisionFiles, notFound error) (*git.RevisionFiles, error) {
	if remote == nil {
		var err error
		if remote, err = s.listUpstream(ctx, prj, reg, req.Name, req.Revision); err != nil {
			if errors.Is(err, ErrRevisionNotFound) {
				return nil, notFound
			}
			return nil, err
		}
		if err := s.refresh(ctx, req.Project, req.Name); err != nil {
			if errors.Is(err, model.ErrModelNotFound) {
				return nil, fmt.Errorf("%w: %s/%s", ErrModelNotFound, req.Project, req.Name)
			}
			return nil, fmt.Errorf("%w: %w", ErrUpstreamSync, err)
		}
		local, err := s.listLocal(ctx, req.Project, req.Name, req.Revision)
		if !errors.Is(err, ErrRevisionNotFound) {
			return local, err
		}
	}
	// Forced syncs of a model are rate limited, so a branch or tag created
	// upstream right after the last one is not mirrored yet.
	return nil, fmt.Errorf("%w: %s resolves to %s upstream; if it was created there just now, pull again in a few seconds",
		ErrRevisionNotMirrored, revisionLabel(req.Revision), remote.Commit)
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
		// select picks at random when the request also ended meanwhile. The
		// service context is checked too: untilClosed ends ctx only soon
		// after the service closes, and the pull that held the lock may
		// have returned already.
		if err := s.lockedCtxErr(ctx); err != nil {
			<-l.ch
			release()
			return nil, err
		}
		return func() {
			<-l.ch
			release()
		}, nil
	case <-ctx.Done():
		release()
		return nil, context.Cause(ctx)
	}
}

// lockedCtxErr reports why a request that got its model lock must not go on.
func (s *Service) lockedCtxErr(ctx context.Context) error {
	switch {
	case ctx.Err() != nil:
		return context.Cause(ctx)
	case s.ctx.Err() != nil:
		return ErrClosed
	default:
		return nil
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
	case errors.Is(err, registrydiscovery.ErrRemoteTooLarge):
		return nil, fmt.Errorf("%w: %w", ErrTooManyFiles, err)
	default:
		return nil, err
	}
}

func (s *Service) listLocal(ctx context.Context, project, name, revision string) (*git.RevisionFiles, error) {
	files, err := s.git.ListRevisionFiles(ctx, repoTypeModels, project, name, revision)
	if errors.Is(err, git.ErrRevisionNotFound) {
		return nil, fmt.Errorf("%w: %w", ErrRevisionNotFound, err)
	}
	return files, err
}

// localHasCommit reports whether the local repository already has commit,
// i.e. whether the files stored in git are cached. A model that never synced
// holds only its placeholder commit, so it has nothing cached and its
// repository is not read.
func (s *Service) localHasCommit(ctx context.Context, project, name, commit string) bool {
	m, err := s.findModel(ctx, project, name)
	if err != nil || m == nil || m.SyncedAt == nil {
		return false
	}
	return s.git.HasCommit(ctx, repoTypeModels, project, name, commit)
}

// missingObjects returns the objects of oids that are not stored.
func (s *Service) missingObjects(ctx context.Context, oids []string) []string {
	var missing []string
	for _, oid := range oids {
		if !s.git.LFSObjectExists(ctx, oid) {
			missing = append(missing, oid)
		}
	}
	return missing
}

// findModel returns the model, or nil when it does not exist.
func (s *Service) findModel(ctx context.Context, project, name string) (*model.Model, error) {
	m, err := s.models.GetModel(ctx, project, name)
	if err == nil {
		return m, nil
	}
	if isNotFound(err) {
		return nil, nil
	}
	return nil, err
}

func (s *Service) lookupTask(key string) *task {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tasks[key]
}

// latestTask returns the task to report for a revision of commit whose LFS
// objects are objects. A run counts when every object it pulls belongs to the
// revision: runs of narrower selections do, and so do runs of another commit
// with the same objects, as after the branch moved during a pull; a run that
// also pulls other objects, such as one of another branch that shares some,
// does not. Of those, a running run comes first, then a run of commit, then
// the run started last.
func (s *Service) latestTask(project, name, commit string, objects []string) *task {
	inRevision := make(map[string]struct{}, len(objects))
	for _, oid := range objects {
		inRevision[oid] = struct{}{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest *task
	var latestRank taskRank
	for _, t := range s.tasks {
		if t.project != project || t.name != name || !t.within(inRevision) {
			continue
		}
		rank := taskRank{running: t.running(), ofCommit: t.commit == commit, startedAt: t.startedAt}
		if latest == nil || rank.before(latestRank) {
			latest, latestRank = t, rank
		}
	}
	return latest
}

// taskRank orders the runs that latestTask may report.
type taskRank struct {
	running, ofCommit bool
	startedAt         time.Time
}

func (r taskRank) before(o taskRank) bool {
	if r.running != o.running {
		return r.running
	}
	if r.ofCommit != o.ofCommit {
		return r.ofCommit
	}
	return r.startedAt.After(o.startedAt)
}

// reportable returns t, unless t failed and every object that it failed is
// stored by now: its failures no longer hold, and a pull of its selection
// would not report it either; see startOrAttach.
func (s *Service) reportable(ctx context.Context, t *task) *task {
	if t == nil || !t.failed() {
		return t
	}
	for _, oid := range t.failedObjects() {
		if !s.git.LFSObjectExists(ctx, oid) {
			return t
		}
	}
	return nil
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

// startOrAttach returns the running task of key, which retries its failed
// objects that are still missing, or starts one for the missing objects. When
// nothing is missing, it returns the last run of key unless that run failed,
// because the run's failures no longer hold; possibly nil.
func (s *Service) startOrAttach(key, project, name, commit string, missing []string, sizes map[string]int64) (*task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx.Err() != nil {
		return nil, ErrClosed
	}
	last := s.tasks[key]
	if last != nil && last.retry(missing) {
		return last, nil
	}
	if len(missing) == 0 {
		if last != nil && last.failed() {
			return nil, nil
		}
		return last, nil
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
	for key, r := range s.lastRefresh {
		if now.Sub(r.at) > s.opts.FinishedTaskTTL {
			delete(s.lastRefresh, key)
		}
	}
}

// buildStatus combines the selected files of a listing, the local storage and
// the task state. gitCached tells whether files stored in git are present
// locally.
func (s *Service) buildStatus(ctx context.Context, project, name, requested string, listed *git.RevisionFiles, p *plan, t *task, gitCached bool) *CacheStatus {
	st := &CacheStatus{
		Project:           project,
		Name:              name,
		RequestedRevision: requested,
		Revision:          listed.Revision,
		Commit:            listed.Commit,
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
		// Storage decides, then a download in progress, of the task or
		// of another one, and only then the task's failure.
		switch {
		case !f.IsLFS:
			if gitCached {
				fs.State = FileCached
			}
		case s.git.LFSObjectExists(ctx, f.OID):
			fs.State = FileCached
		case tracked && (o.state == FileDownloading || o.state == FileQueued):
			fs.State, fs.DownloadedBytes, fs.Error = o.state, o.downloaded, o.err
		case hasKey(others, f.OID):
			fs.State, fs.DownloadedBytes = FileDownloading, others[f.OID]
		case tracked && o.state == FileFailed:
			fs.State, fs.Error = FileFailed, o.err
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
