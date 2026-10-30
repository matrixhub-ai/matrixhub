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

package modelcache_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/modelcache"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	registrymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registry/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	registrydiscoverymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery/mocks"
)

const (
	oidA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	oidB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	oidC = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

var errModelNotFound = errors.New("failed to get model: record not found")

// revisionNotFound is how the local repository reports a revision it does not
// have.
func revisionNotFound(revision string) error {
	return fmt.Errorf("%w: %q: reference not found", git.ErrRevisionNotFound, revision)
}

// syncedModel is a model that synced from upstream.
func syncedModel(name string) *model.Model {
	syncedAt := time.Unix(1_000_000, 0)
	return &model.Model{Name: name, ProjectName: "proj", SyncedAt: &syncedAt}
}

func revisionFiles() *git.RevisionFiles {
	return &git.RevisionFiles{Revision: "main", Commit: "c1", Files: []git.RevisionFile{
		{Path: "config.json", Size: 10},
		{Path: "model.safetensors", Size: 100, IsLFS: true, OID: oidA},
		{Path: "tokenizer.json", Size: 50, IsLFS: true, OID: oidB},
	}}
}

// upstream is what the mocked RemoteFileLister answers.
type upstream struct {
	files *git.RevisionFiles
	err   error
	calls atomic.Int32
	// onList, when set, runs inside every listing.
	onList func()
}

type download struct{ done, total int64 }

func (d download) Downloaded() int64 { return d.done }
func (d download) Total() int64      { return d.total }

type fixture struct {
	models *modelmocks.MockIModelService
	git    *gitmocks.MockIGitRepo
	lister *upstream
	disk   sync.Map
	svc    modelcache.IModelCacheService
	// hasCommits counts the reads of local repositories for a commit.
	hasCommits atomic.Int32
}

func newFixture(t *testing.T, opts modelcache.Options) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := &fixture{
		models: modelmocks.NewMockIModelService(ctrl),
		git:    gitmocks.NewMockIGitRepo(ctrl),
		lister: &upstream{files: revisionFiles()},
	}
	lister := registrydiscoverymocks.NewMockRemoteFileLister(ctrl)
	lister.EXPECT().ListRevisionFiles(gomock.Any(), gomock.Any(), "Org", gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, *registry.Registry, string, string, string) (*git.RevisionFiles, error) {
			f.lister.calls.Add(1)
			if f.lister.onList != nil {
				f.lister.onList()
			}
			return f.lister.files, f.lister.err
		}).AnyTimes()

	registryID := 1
	projects := projectmocks.NewMockIProjectRepo(ctrl)
	projects.EXPECT().GetProjectByName(gomock.Any(), "proj").
		Return(&project.Project{Name: "proj", RegistryID: &registryID, Organization: "Org"}, nil).AnyTimes()
	projects.EXPECT().GetProjectByName(gomock.Any(), "local").
		Return(&project.Project{Name: "local"}, nil).AnyTimes()
	registries := registrymocks.NewMockIRegistryRepo(ctrl)
	registries.EXPECT().GetRegistry(gomock.Any(), registryID).
		Return(&registry.Registry{ID: registryID, URL: "https://hf.example"}, nil).AnyTimes()
	f.git.EXPECT().LFSObjectExists(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) bool {
			_, ok := f.disk.Load(oid)
			return ok
		}).AnyTimes()
	// Local repositories have the commit that revisionFiles lists.
	f.git.EXPECT().HasCommit(gomock.Any(), "models", "proj", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _, commit string) bool {
			f.hasCommits.Add(1)
			return commit == "c1"
		}).AnyTimes()

	if opts.PollInterval == 0 {
		opts.PollInterval = time.Millisecond
	}
	if opts.RetryBackoff == 0 {
		opts.RetryBackoff = time.Millisecond
	}
	if opts.RequeueMinInterval == 0 {
		opts.RequeueMinInterval = time.Nanosecond
	}
	f.svc = modelcache.NewModelCacheService(f.models, projects, registries, f.git, lister, opts)
	// Registered after the controller, so it runs first: workers stop before
	// the controller verifies its expectations.
	t.Cleanup(f.svc.Close)
	return f
}

// existingModel makes the model exist locally and list revisionFiles.
func (f *fixture) existingModel() {
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").
		Return(syncedModel("model"), nil).AnyTimes()
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", gomock.Any()).
		Return(revisionFiles(), nil).AnyTimes()
}

// completeOnFetch stores an object as soon as its download is observed.
func (f *fixture) completeOnFetch() {
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			f.disk.Store(oid, true)
			return download{done: 1, total: 1}
		}).AnyTimes()
}

func waitForTask(t *testing.T, f *fixture, want modelcache.TaskState) *modelcache.CacheStatus {
	t.Helper()
	var st *modelcache.CacheStatus
	require.Eventually(t, func() bool {
		var err error
		st, err = f.svc.GetModelCache(context.Background(), "proj", "model", "")
		return err == nil && st.Task != nil && st.Task.State == want
	}, 5*time.Second, 2*time.Millisecond)
	return st
}

func TestCacheModel_RejectsNonProxyProject(t *testing.T) {
	f := newFixture(t, modelcache.Options{})

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "local", Name: "model"})

	require.ErrorIs(t, err, modelcache.ErrNotProxyProject)
}

func TestCacheModel_DryRunIsReadOnly(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound).AnyTimes()
	f.disk.Store(oidA, true)

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})

	require.NoError(t, err)
	// No revision was requested: the status names the default branch.
	require.Empty(t, st.RequestedRevision)
	require.Equal(t, "main", st.Revision)
	require.Equal(t, "c1", st.Commit)
	require.Equal(t, int64(3), st.Totals.Files)
	require.Equal(t, int64(160), st.Totals.Bytes)
	// Only the stored LFS object counts: git files are not local yet.
	require.Equal(t, int64(1), st.Totals.CachedFiles)
	require.Equal(t, int64(100), st.Totals.CachedBytes)
	require.Equal(t, modelcache.CacheStatePartial, st.State)
	require.Nil(t, st.Task)
	require.Equal(t, int32(1), f.lister.calls.Load())
}

func TestCacheModel_DryRunMapsUpstreamErrors(t *testing.T) {
	tests := []struct {
		upstream error
		want     error
	}{
		{registrydiscovery.ErrRemoteNotFound, modelcache.ErrRevisionNotFound},
		{registrydiscovery.ErrRemoteAccessDenied, modelcache.ErrUpstreamAccessDenied},
		{registrydiscovery.ErrRemoteUnavailable, modelcache.ErrUpstreamUnavailable},
		{registrydiscovery.ErrRemoteTooLarge, modelcache.ErrTooManyFiles},
	}
	for _, tt := range tests {
		t.Run(tt.want.Error(), func(t *testing.T) {
			f := newFixture(t, modelcache.Options{})
			f.lister.err = fmt.Errorf("%w: detail", tt.upstream)

			_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})

			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestCacheModel_NewModelIsCheckedUpstreamBeforeSync(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "typo").Return(nil, errModelNotFound)
	f.lister.err = registrydiscovery.ErrRemoteNotFound

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "typo"})

	// No CheckOrSyncFromRemote expectation: the strict mock fails the test
	// if the sync, which creates the model, ran.
	require.ErrorIs(t, err, modelcache.ErrRevisionNotFound)
}

// Upstream was checked before the sync, so a sync that fails keeps the new
// model, as a failed proxy download does: the sync may have queued downloads
// already, and the next pull or download syncs the model again.
func TestCacheModel_KeepsNewModelWhenSyncFails(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	syncErr := errors.New("update model metadata: context canceled")
	gomock.InOrder(
		f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound),
		f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(syncErr),
	)

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})

	// No DeleteModel expectation: the strict mock fails the test if the
	// model is deleted.
	require.ErrorIs(t, err, modelcache.ErrUpstreamSync)
	require.ErrorIs(t, err, syncErr)
}

func TestCacheModel_KeepsExistingModelWhenSyncFails(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil)
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(errors.New("upstream down"))

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})

	require.ErrorIs(t, err, modelcache.ErrUpstreamSync)
}

func TestCacheModel_AllCachedSucceedsWithoutTask(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidA, true)
	f.disk.Store(oidB, true)

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
	require.Nil(t, st.Task)
	require.Equal(t, int64(3), st.Totals.CachedFiles)
}

func TestCacheModel_DownloadsMissingObjects(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.completeOnFetch()

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)
	require.NotNil(t, st.Task)
	require.NotEmpty(t, st.Task.ID)
	require.NotContains(t, st.Task.ID, "proj/model")

	done := waitForTask(t, f, modelcache.TaskSucceeded)
	require.Equal(t, modelcache.CacheStateCached, done.State)
	require.Equal(t, int64(3), done.Totals.CachedFiles)
	require.Equal(t, st.Task.ID, done.Task.ID)
	require.NotNil(t, done.Task.FinishedAt)
}

func TestCacheModel_RequeuesObjectsTheMirrorForgot(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	var refreshed atomic.Bool
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error {
			refreshed.Store(true)
			return nil
		}).MinTimes(1)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			if !refreshed.Load() {
				return nil
			}
			f.disk.Store(oid, true)
			return download{done: 1, total: 1}
		}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)

	waitForTask(t, f, modelcache.TaskSucceeded)
}

func TestCacheModel_FailsAfterRetries(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 2})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidB, true)
	// Each attempt sees the download once, then it disappears without being
	// stored, which is how a failed mirror download looks.
	var calls atomic.Int32
	f.git.EXPECT().FetchLFSObject(gomock.Any(), oidA).
		DoAndReturn(func(context.Context, string) git.LFSDownload {
			if calls.Add(1)%2 == 1 {
				return download{done: 10, total: 100}
			}
			return nil
		}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	require.Equal(t, int64(1), st.Totals.FailedFiles)
	for _, file := range st.Files {
		if file.OID == oidA {
			require.Equal(t, modelcache.FileFailed, file.State)
			require.NotEmpty(t, file.Error)
		}
	}
}

func TestCacheModel_SecondTriggerAttachesToRunningTask(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	release := make(chan struct{})
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			select {
			case <-release:
				f.disk.Store(oid, true)
			default:
			}
			return download{done: 1, total: 100}
		}).AnyTimes()

	first, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)
	second, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)

	require.Equal(t, modelcache.TaskRunning, second.Task.State)
	require.Equal(t, first.Task.ID, second.Task.ID)
	require.Equal(t, first.Task.StartedAt, second.Task.StartedAt)

	close(release)
	waitForTask(t, f, modelcache.TaskSucceeded)
}

func TestCacheModel_LimitsRunningTasks(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxRunningTasks: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(download{done: 1, total: 100}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{
		Project: "proj", Name: "model",
		Selection: modelcache.Selection{AllowPatterns: []string{"model.safetensors"}},
	})
	require.NoError(t, err)
	_, err = f.svc.CacheModel(context.Background(), &modelcache.Request{
		Project: "proj", Name: "model",
		Selection: modelcache.Selection{AllowPatterns: []string{"tokenizer.json"}},
	})

	require.ErrorIs(t, err, modelcache.ErrTooManyTasks)
}

func TestCacheModel_RefreshesWhenUpstreamHasANewerRevision(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	gomock.InOrder(
		f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c2").
			Return(nil, revisionNotFound("c2")),
		f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil),
		f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c2").
			Return(revisionFiles(), nil),
	)
	f.disk.Store(oidA, true)
	f.disk.Store(oidB, true)

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "c2"})

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
	// Upstream was asked first whether it has the revision.
	require.Equal(t, int32(1), f.lister.calls.Load())
}

// A revision that upstream does not have either forces no sync, however often
// it is asked for.
func TestCacheModel_UnknownRevisionDoesNotForceASync(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.lister.err = registrydiscovery.ErrRemoteNotFound
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(3)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _, revision string) (*git.RevisionFiles, error) {
			return nil, revisionNotFound(revision)
		}).Times(3)

	// No RefreshFromRemote expectation: the strict mock fails the test if a
	// sync is forced.
	for i := range 3 {
		_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: fmt.Sprintf("nope-%d", i)})
		require.ErrorIs(t, err, modelcache.ErrRevisionNotFound)
	}
}

// A revision that upstream has outside the mirrored refs forces a sync, but
// asking again within RequeueMinInterval reuses its result.
func TestCacheModel_UnmirroredRevisionIsRefreshedOncePerInterval(t *testing.T) {
	f := newFixture(t, modelcache.Options{RequeueMinInterval: time.Hour})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(3)
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(1)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c9").Return(nil, revisionNotFound("c9")).AnyTimes()

	for range 3 {
		_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "c9"})
		require.ErrorIs(t, err, modelcache.ErrRevisionNotMirrored)
	}
}

func TestCacheModel_RejectsRevisionExpressions(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()

	for _, revision := range []string{
		"main~1", "HEAD^", "main@{1}", "main^{/fix}", "../other/repo", "a..b", "-main", "main.lock",
		"refs/heads/.hidden", "with space", "trailing/", "trailing.", "a//b", "@", "a:b", "a?", "a*", "a[b", `a\b`,
	} {
		for _, dryRun := range []bool{true, false} {
			_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: revision, DryRun: dryRun})
			require.ErrorIs(t, err, modelcache.ErrUnsupportedRevision, revision)
		}
	}
	require.Zero(t, f.lister.calls.Load())

	for _, revision := range []string{"main", "v1.0", "release/2.x", "refs/heads/main", "refs/tags/v1", "c1899de289a04d12100db370d81485cdf75e47ca"} {
		_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: revision, DryRun: true})
		require.NoError(t, err, revision)
	}
}

func TestGetModelCache_IsReadOnly(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.disk.Store(oidA, true)

	// Only GetModel, ListRevisionFiles and LFSObjectExists are expected; the
	// strict mocks fail on any download, sync or refresh.
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")

	require.NoError(t, err)
	require.Equal(t, "main", st.Revision)
	require.Nil(t, st.Task)
	require.Equal(t, modelcache.CacheStatePartial, st.State)
	require.Equal(t, int64(2), st.Totals.CachedFiles)
	require.Equal(t, int64(1), st.Totals.MissingFiles)
}

func TestGetModelCache_StateCountsOnlyLFSObjects(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()

	// config.json is in git, so it is cached as soon as the model is synced.
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), st.Totals.CachedFiles)
	require.Equal(t, modelcache.CacheStateMissing, st.State)

	f.disk.Store(oidA, true)
	f.disk.Store(oidB, true)
	st, err = f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
}

// A proxy model whose first sync has not succeeded holds only the commit that
// creating it wrote, which must not look cached.
func TestGetModelCache_NeverSyncedProxyModelIsMissing(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model", ProjectName: "proj"}, nil)

	// No ListRevisionFiles expectation: the placeholder commit is not listed.
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateMissing, st.State)
	require.Empty(t, st.Files)
	require.Empty(t, st.Commit)
	require.NotEmpty(t, st.Warnings)
}

// Models of local projects never sync, and their files are what was pushed.
func TestGetModelCache_LocalModelIsListed(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "local", "model").Return(&model.Model{Name: "model", ProjectName: "local"}, nil)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "local", "model", "").Return(revisionFiles(), nil)
	f.disk.Store(oidA, true)
	f.disk.Store(oidB, true)

	st, err := f.svc.GetModelCache(context.Background(), "local", "model", "")

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
	require.Equal(t, int64(3), st.Totals.CachedFiles)
}

func TestGetModelCache_ModelNotFound(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound)

	_, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")

	require.ErrorIs(t, err, modelcache.ErrModelNotFound)
}

func TestGetModelCache_DropsFinishedTasksAfterTTL(t *testing.T) {
	var mu sync.Mutex
	now := time.Unix(1_000_000, 0)
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	f := newFixture(t, modelcache.Options{Now: clock, FinishedTaskTTL: time.Hour})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.completeOnFetch()

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)
	done := waitForTask(t, f, modelcache.TaskSucceeded)
	require.False(t, done.Task.StartedAt.IsZero())

	mu.Lock()
	now = now.Add(2 * time.Hour)
	mu.Unlock()

	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
	require.Nil(t, st.Task)
}

var pullModel = &modelcache.Request{Project: "proj", Name: "model"}

func TestCacheModel_RejectsRefsOtherThanBranchesAndTags(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound).AnyTimes()

	for _, dryRun := range []bool{true, false} {
		_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "refs/pr/1", DryRun: dryRun})
		require.ErrorIs(t, err, modelcache.ErrUnsupportedRevision)
	}
	require.Zero(t, f.lister.calls.Load())

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "refs/tags/v1", DryRun: true})
	require.NoError(t, err)
}

func TestCacheModel_NewModelWithEmptySelectionFailsBeforeSync(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound)

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{
		Project: "proj", Name: "model",
		Selection: modelcache.Selection{AllowPatterns: []string{"*.gguf"}},
	})

	// No CheckOrSyncFromRemote expectation: the model is never created.
	require.ErrorIs(t, err, modelcache.ErrInvalidSelection)
}

func TestCacheModel_NewModelOverCapacityFailsBeforeSync(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxRunningTasks: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(download{done: 1, total: 100}).AnyTimes()
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	f.models.EXPECT().GetModel(gomock.Any(), "proj", "other").Return(nil, errModelNotFound)
	_, err = f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "other"})

	require.ErrorIs(t, err, modelcache.ErrTooManyTasks)
}

func TestCacheModel_RevisionMissingLocally(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		// refreshes tells whether the mirror is synced again.
		refreshes bool
		want      error
	}{
		// For example a commit that only a pull request ref reaches.
		{"upstream has it outside mirrored refs", nil, true, modelcache.ErrRevisionNotMirrored},
		{"upstream does not have it", registrydiscovery.ErrRemoteNotFound, false, modelcache.ErrRevisionNotFound},
		// The revision may exist, so the failure is not reported as NotFound.
		{"upstream is unavailable", registrydiscovery.ErrRemoteUnavailable, false, modelcache.ErrUpstreamUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, modelcache.Options{})
			f.lister.err = tt.upstream
			f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
			f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
			listings := 1
			if tt.refreshes {
				f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil)
				listings = 2
			}
			f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c9").
				Return(nil, revisionNotFound("c9")).Times(listings)

			_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "c9"})

			require.ErrorIs(t, err, tt.want)
			require.Equal(t, int32(1), f.lister.calls.Load())
		})
	}
}

func TestCacheModel_NewModelWithUnmirroredRevisionIsKept(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	gomock.InOrder(
		f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound),
		f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil),
	)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c9").Return(nil, revisionNotFound("c9"))

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "c9"})

	// The model synced and exists upstream, so it stays: no DeleteModel
	// expectation. It synced just now, so no RefreshFromRemote either. The
	// upstream listing taken before the sync explains the failure.
	require.ErrorIs(t, err, modelcache.ErrRevisionNotMirrored)
	require.Equal(t, int32(1), f.lister.calls.Load())
}

func TestCacheModel_PullsOfANewModelTakeTurns(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.disk.Store(oidA, true)
	f.disk.Store(oidB, true)
	var exists atomic.Bool
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) (*model.Model, error) {
			if exists.Load() {
				return syncedModel("model"), nil
			}
			return nil, errModelNotFound
		}).AnyTimes()
	// The first sync creates the model; the second one fails.
	var syncs atomic.Int32
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error {
			if syncs.Add(1) == 1 {
				exists.Store(true)
				return nil
			}
			return errors.New("upstream down")
		}).Times(2)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", gomock.Any()).Return(revisionFiles(), nil).AnyTimes()
	listing := make(chan struct{})
	var once sync.Once
	f.lister.onList = func() {
		once.Do(func() {
			close(listing)
			// Give the second pull time to look the model up.
			time.Sleep(100 * time.Millisecond)
		})
	}

	var wg sync.WaitGroup
	var first, second error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, first = f.svc.CacheModel(context.Background(), pullModel)
	}()
	<-listing
	go func() {
		defer wg.Done()
		_, second = f.svc.CacheModel(context.Background(), pullModel)
	}()
	wg.Wait()

	// The second pull starts after the first created the model, so its
	// failure does not roll the model back: there is no DeleteModel
	// expectation. Only the first pull, of a new model, listed upstream:
	// without the lock the second one would have found no model either.
	require.NoError(t, first)
	require.ErrorIs(t, second, modelcache.ErrUpstreamSync)
	require.Equal(t, int32(1), f.lister.calls.Load())
}

func TestCacheModel_RetryStartsANewRun(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	f.disk.Store(oidB, true)
	var healthy atomic.Bool
	var calls atomic.Int32
	f.git.EXPECT().FetchLFSObject(gomock.Any(), oidA).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			if healthy.Load() {
				f.disk.Store(oid, true)
				return download{done: 1, total: 1}
			}
			// Seen once, then gone without being stored: a failed download.
			if calls.Add(1)%2 == 1 {
				return download{done: 10, total: 100}
			}
			return nil
		}).AnyTimes()

	first, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	failed := waitForTask(t, f, modelcache.TaskFailed)
	require.Equal(t, first.Task.ID, failed.Task.ID)

	healthy.Store(true)
	second, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	require.NotEqual(t, first.Task.ID, second.Task.ID)
	done := waitForTask(t, f, modelcache.TaskSucceeded)
	require.Equal(t, second.Task.ID, done.Task.ID)
}

func TestCacheModel_PreviewShowsRunningDownloads(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(download{done: 30, total: 100}).AnyTimes()
	narrow := modelcache.Selection{AllowPatterns: []string{"model.safetensors"}}

	started, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Selection: narrow})
	require.NoError(t, err)
	require.Equal(t, int64(1), started.Totals.Files)

	// A preview of the same selection attaches to the run.
	require.Eventually(t, func() bool {
		st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Selection: narrow, DryRun: true})
		return err == nil && st.Task != nil && st.Task.ID == started.Task.ID &&
			st.Files[0].State == modelcache.FileDownloading && st.Files[0].DownloadedBytes == 30
	}, 5*time.Second, 2*time.Millisecond)

	// A wider preview has no run of its own but shows the download. The
	// local repository has the commit, so files stored in git are cached.
	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})
	require.NoError(t, err)
	require.Nil(t, st.Task)
	for _, file := range st.Files {
		switch {
		case file.OID == oidA:
			require.Equal(t, modelcache.FileDownloading, file.State)
		case !file.IsLFS:
			require.Equal(t, modelcache.FileCached, file.State)
		}
	}

	// The status of the commit reports the narrower run.
	st, err = f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), st.Totals.Files)
	require.Equal(t, started.Task.ID, st.Task.ID)
	require.Equal(t, modelcache.TaskRunning, st.Task.State)
}

func TestClose_InterruptsRunningTasks(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	// Once: the pull after Close fails before it syncs.
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(download{done: 1, total: 100}).AnyTimes()
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	f.svc.Close()

	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.TaskFailed, st.Task.State)
	require.Contains(t, st.Task.Error, "interrupted")
	_, err = f.svc.CacheModel(context.Background(), pullModel)
	require.ErrorIs(t, err, modelcache.ErrClosed)
}

func TestCacheModel_StalledDownloadFailsWithoutRetry(t *testing.T) {
	// A retry would wait out the backoff, so the task only fails in time if
	// the stall ends the object.
	f := newFixture(t, modelcache.Options{StallTimeout: 20 * time.Millisecond, RetryBackoff: time.Hour})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidB, true)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), oidA).Return(download{done: 10, total: 100}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	for _, file := range st.Files {
		if file.OID == oidA {
			require.Equal(t, modelcache.FileFailed, file.State)
			require.Contains(t, file.Error, "no progress")
		}
	}
}

func TestCacheModel_RequeueIsRateLimited(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1, RequeueMinInterval: time.Hour})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	// The mirror tracks neither object, even after a refresh.
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(1)

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	require.Equal(t, int64(2), st.Totals.FailedFiles)
}

func TestCacheModel_RequeueDoesNotRecreateADeletedModel(t *testing.T) {
	// A retry would wait out the backoff, so the task only fails in time if
	// a deleted model ends the object.
	f := newFixture(t, modelcache.Options{MaxAttempts: 3, RetryBackoff: time.Hour})
	var deleted atomic.Bool
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) (*model.Model, error) {
			if deleted.Load() {
				return nil, errModelNotFound
			}
			return syncedModel("model"), nil
		}).AnyTimes()
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", gomock.Any()).Return(revisionFiles(), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	// The model is deleted while the task runs, and the mirror forgot its
	// objects.
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string) git.LFSDownload {
			deleted.Store(true)
			return nil
		}).AnyTimes()

	// RefreshFromRemote never creates a model, so it fails for a deleted
	// one; see TestModelService_RefreshFromRemoteDoesNotCreateModel.
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
		Return(fmt.Errorf("%w: proj/model", model.ErrModelNotFound)).MinTimes(1)

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	// The model is gone, so the run is observed through a preview.
	var st *modelcache.CacheStatus
	require.Eventually(t, func() bool {
		st, err = f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})
		return err == nil && st.Task != nil && st.Task.State == modelcache.TaskFailed
	}, 5*time.Second, 2*time.Millisecond)
	for _, file := range st.Files {
		if file.IsLFS {
			require.Contains(t, file.Error, "deleted")
		}
	}
}

func TestCacheModel_RequeueWithinIntervalReportsLastError(t *testing.T) {
	// One object at a time, so the second object's requeue falls within the
	// interval of the first one's.
	f := newFixture(t, modelcache.Options{MaxAttempts: 1, PerTaskConcurrency: 1, RequeueMinInterval: time.Hour})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	// The mirror tracks neither object, and the one refresh fails.
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
		Return(errors.New("open /data/repositories/proj/model.git: permission denied")).Times(1)

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	// Both objects report the failed refresh, although only the first ran
	// it, and neither shows the server path.
	st := waitForTask(t, f, modelcache.TaskFailed)
	for _, file := range st.Files {
		if file.IsLFS {
			require.Contains(t, file.Error, "refreshing the model from upstream failed")
			require.NotContains(t, file.Error, "/data")
		}
	}
}

// hangingRefresh makes RefreshFromRemote hang, ignoring its context as git
// does while upstream stays silent, until the test ends. The returned channel
// closes once a refresh hangs.
func (f *fixture) hangingRefresh(t *testing.T) <-chan struct{} {
	hang, hanging := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(hang) })
	var once sync.Once
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error {
			once.Do(func() { close(hanging) })
			<-hang
			return nil
		}).MinTimes(1)
	return hanging
}

func TestCacheModel_RefreshIsBounded(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1, RefreshTimeout: 20 * time.Millisecond})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	f.hangingRefresh(t)

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	for _, file := range st.Files {
		if file.IsLFS {
			require.Contains(t, file.Error, "refreshing the model from upstream failed")
		}
	}
}

func TestClose_DoesNotWaitForAHungRefresh(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	hanging := f.hangingRefresh(t)
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	<-hanging

	closed := make(chan struct{})
	go func() {
		f.svc.Close()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close() waits for a refresh that hangs")
	}
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.TaskFailed, st.Task.State)
	require.Contains(t, st.Task.Error, "interrupted")
}

// A pull stops waiting for a sync when its request ends, although the sync
// does not stop at once.
func TestCacheModel_StopsWaitingForTheSyncWhenTheRequestEnds(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	hang := make(chan struct{})
	t.Cleanup(func() { close(hang) })
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error {
			<-hang
			return nil
		})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := f.svc.CacheModel(ctx, pullModel)

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// A failed lookup during the sync says "not found" without the model being
// gone: the object is retried.
func TestCacheModel_FailedRefreshIsRetried(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 2})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidB, true)
	var refreshed atomic.Bool
	gomock.InOrder(
		f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
			Return(errors.New("model not found: failed to get model: database is locked")),
		f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
			DoAndReturn(func(context.Context, string, string) error {
				refreshed.Store(true)
				return nil
			}),
	)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), oidA).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			if !refreshed.Load() {
				return nil
			}
			f.disk.Store(oid, true)
			return download{done: 1, total: 1}
		}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	waitForTask(t, f, modelcache.TaskSucceeded)
}

// The mirror stores a finished download before it forgets it, so a download
// that disappears between two checks is found in storage.
func TestCacheModel_DownloadThatDisappearsIsFoundInStorage(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidB, true)
	var calls atomic.Int32
	f.git.EXPECT().FetchLFSObject(gomock.Any(), oidA).
		DoAndReturn(func(context.Context, string) git.LFSDownload {
			if calls.Add(1) == 1 {
				return download{done: 50, total: 100}
			}
			// The download finished: it is stored, then forgotten.
			f.disk.Store(oidA, true)
			return nil
		}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	waitForTask(t, f, modelcache.TaskSucceeded)
}

func TestCacheModel_FinishedDownloadThatIsNeverStoredFails(t *testing.T) {
	// A retry would wait out the backoff, so the task only fails in time if
	// the store timeout ends the object.
	f := newFixture(t, modelcache.Options{StoreTimeout: 20 * time.Millisecond, RetryBackoff: time.Hour})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidB, true)
	// The download is complete, but a client keeps reading it, so the
	// mirror never stores it. Waiting for readers is not a stall.
	f.git.EXPECT().FetchLFSObject(gomock.Any(), oidA).Return(download{done: 100, total: 100}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	for _, file := range st.Files {
		if file.OID == oidA {
			require.Equal(t, modelcache.FileFailed, file.State)
			require.Contains(t, file.Error, "not stored")
		}
	}
}

// fileState returns the state of the file with oid in st.
func fileState(st *modelcache.CacheStatus, oid string) modelcache.FileStatus {
	for _, file := range st.Files {
		if file.OID == oid {
			return file
		}
	}
	return modelcache.FileStatus{}
}

// Pulling again while a run goes on retries the run's failed objects at once.
func TestCacheModel_AttachRetriesFailedObjects(t *testing.T) {
	// A retry within the run would wait out the backoff, so the stall ends
	// the object.
	f := newFixture(t, modelcache.Options{StallTimeout: 20 * time.Millisecond, RetryBackoff: time.Hour})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	var healthy atomic.Bool
	var progress atomic.Int64
	release := make(chan struct{})
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			switch {
			case oid == oidA && healthy.Load():
				f.disk.Store(oid, true)
				return download{done: 1, total: 1}
			case oid == oidA:
				// Stalls.
				return download{done: 10, total: 100}
			}
			// oidB keeps the run going until it is released.
			select {
			case <-release:
				f.disk.Store(oid, true)
				return download{done: 1, total: 1}
			default:
				return download{done: progress.Add(1), total: 1 << 40}
			}
		}).AnyTimes()

	first, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
		return err == nil && st.Task.State == modelcache.TaskRunning && fileState(st, oidA).State == modelcache.FileFailed
	}, 5*time.Second, 2*time.Millisecond)

	healthy.Store(true)
	second, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	require.Equal(t, first.Task.ID, second.Task.ID)
	require.Eventually(t, func() bool {
		st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
		return err == nil && fileState(st, oidA).State == modelcache.FileCached
	}, 5*time.Second, 2*time.Millisecond)

	close(release)
	done := waitForTask(t, f, modelcache.TaskSucceeded)
	require.Equal(t, first.Task.ID, done.Task.ID)
}

// A run that failed an object does not hide another run that downloads it.
func TestCacheModel_StatusPrefersADownloadOverAnEarlierFailure(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil).AnyTimes()
	var queued atomic.Bool
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string) git.LFSDownload {
			if !queued.Load() {
				return nil
			}
			return download{done: 30, total: 100}
		}).AnyTimes()
	narrow := modelcache.Selection{AllowPatterns: []string{"model.safetensors"}}

	// The mirror does not track the object, so the narrow run fails.
	failed, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Selection: narrow})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Selection: narrow, DryRun: true})
		return err == nil && st.Task != nil && st.Task.State == modelcache.TaskFailed
	}, 5*time.Second, 2*time.Millisecond)

	// A wider run downloads it now.
	queued.Store(true)
	_, err = f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Selection: narrow, DryRun: true})
		return err == nil && st.Task.ID == failed.Task.ID &&
			fileState(st, oidA).State == modelcache.FileDownloading && fileState(st, oidA).DownloadedBytes == 30
	}, 5*time.Second, 2*time.Millisecond)
}

// Runs are keyed by their objects: when the branch moves to a commit with the
// same objects during a pull, the status still shows the run, and pulling
// again attaches to it.
func TestCacheModel_RunFollowsTheBranch(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	var commit atomic.Value
	commit.Store("c1")
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", gomock.Any()).
		DoAndReturn(func(context.Context, string, string, string, string) (*git.RevisionFiles, error) {
			files := revisionFiles()
			files.Commit = commit.Load().(string)
			return files, nil
		}).AnyTimes()
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(download{done: 1, total: 100}).AnyTimes()

	first, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	// An upstream README edit moved main.
	commit.Store("c2")
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, "c2", st.Commit)
	require.Equal(t, first.Task.ID, st.Task.ID)
	require.Equal(t, modelcache.TaskRunning, st.Task.State)

	second, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	require.Equal(t, first.Task.ID, second.Task.ID)
}

// Objects are shared between models, so a new model may need no download,
// and then no task and no capacity.
func TestCacheModel_NewModelWithStoredObjectsNeedsNoCapacity(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxRunningTasks: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(download{done: 1, total: 100}).AnyTimes()
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	other := &git.RevisionFiles{Revision: "main", Commit: "c3", Files: []git.RevisionFile{
		{Path: "model.safetensors", Size: 100, IsLFS: true, OID: oidC},
	}}
	f.lister.files = other
	f.disk.Store(oidC, true)
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "other").Return(nil, errModelNotFound)
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "other").Return(nil)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "other", "").Return(other, nil)

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "other"})

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
	require.Nil(t, st.Task)
}

func TestCacheModel_ExistingModelWithEmptySelectionFails(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)

	// No FetchLFSObject expectation: nothing starts.
	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{
		Project: "proj", Name: "model",
		Selection: modelcache.Selection{AllowPatterns: []string{"*.gguf"}},
	})

	require.ErrorIs(t, err, modelcache.ErrInvalidSelection)
}

// Once everything is stored, a failed run is no longer the news.
func TestCacheModel_PullWithNothingMissingHidesAnEarlierFailure(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil).AnyTimes()
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	f.disk.Store(oidB, true)
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	waitForTask(t, f, modelcache.TaskFailed)

	// A client downloaded the object meanwhile.
	f.disk.Store(oidA, true)
	st, err := f.svc.CacheModel(context.Background(), pullModel)

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
	require.Nil(t, st.Task)
}

// A finished download waits for its readers to close before it is stored;
// meanwhile it downloads nothing, so its slot serves other downloads.
func TestCacheModel_UnstoredDownloadFreesItsSlot(t *testing.T) {
	f := newFixture(t, modelcache.Options{GlobalConcurrency: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.disk.Store(oidB, true)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			if oid == oidA {
				// Complete, but a client keeps reading it.
				return download{done: 100, total: 100}
			}
			f.disk.Store(oid, true)
			return download{done: 1, total: 1}
		}).AnyTimes()
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
		return err == nil && fileState(st, oidA).DownloadedBytes == 100
	}, 5*time.Second, 2*time.Millisecond)

	other := &git.RevisionFiles{Revision: "main", Commit: "c3", Files: []git.RevisionFile{
		{Path: "model.safetensors", Size: 1, IsLFS: true, OID: oidC},
	}}
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "other").Return(syncedModel("other"), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "other").Return(nil)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "other", "").Return(other, nil).AnyTimes()
	_, err = f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "other"})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		st, err := f.svc.GetModelCache(context.Background(), "proj", "other", "")
		return err == nil && st.Task != nil && st.Task.State == modelcache.TaskSucceeded
	}, 5*time.Second, 2*time.Millisecond)
}

// The sync runs on a goroutine of its own, where no request handler recovers
// a panic of the mirror: a panic fails the pull instead of the server.
func TestCacheModel_SyncPanicFailsThePull(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil)
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error {
			panic("runtime error: invalid memory address or nil pointer dereference")
		})

	_, err := f.svc.CacheModel(context.Background(), pullModel)

	require.ErrorIs(t, err, modelcache.ErrUpstreamSync)
}

func TestCacheModel_RefreshPanicFailsTheObject(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error { panic("boom") }).AnyTimes()
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	for _, file := range st.Files {
		if file.IsLFS {
			require.Equal(t, "refreshing the model from upstream failed; see server logs for details", file.Error)
		}
	}
}

// A worker that panics fails its object, frees its download slot and lets
// the run end.
func TestCacheModel_DownloadPanicFailsTheObject(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1, GlobalConcurrency: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			if oid == oidA {
				panic("boom")
			}
			f.disk.Store(oid, true)
			return download{done: 1, total: 1}
		}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	st := waitForTask(t, f, modelcache.TaskFailed)
	require.Equal(t, modelcache.FileFailed, fileState(st, oidA).State)
	require.Equal(t, "download failed; see server logs for details", fileState(st, oidA).Error)
	require.Equal(t, modelcache.FileCached, fileState(st, oidB).State)
}

// A preview does not read the repository of a model that never synced: it
// holds only its placeholder commit, so none of the revision's files are in it.
func TestCacheModel_DryRunDoesNotReadANeverSyncedModel(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model", ProjectName: "proj"}, nil).AnyTimes()

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})

	require.NoError(t, err)
	require.Zero(t, f.hasCommits.Load())
	require.Equal(t, modelcache.CacheStateMissing, st.State)
}

// A model whose first sync failed, for example because upstream is gated, is
// checked upstream like a new one, so the pull reports why.
func TestCacheModel_NeverSyncedModelIsCheckedUpstreamBeforeSync(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model", ProjectName: "proj"}, nil)
	f.lister.err = fmt.Errorf("%w: %w", registrydiscovery.ErrRemoteAccessDenied, registrydiscovery.ErrRemoteGated)

	// No CheckOrSyncFromRemote expectation: the strict mock fails the test
	// if the sync ran.
	_, err := f.svc.CacheModel(context.Background(), pullModel)

	require.ErrorIs(t, err, modelcache.ErrUpstreamAccessDenied)
	require.ErrorIs(t, err, registrydiscovery.ErrRemoteGated)
}

// Closing the service ends the pulls that wait for a sync, a forced re-sync
// or the lock of their model, so they do not hold up the server's shutdown.
func TestClose_EndsPullsThatWait(t *testing.T) {
	t.Run("for a sync and for the model lock", func(t *testing.T) {
		f := newFixture(t, modelcache.Options{})
		f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
		hang, syncing := make(chan struct{}), make(chan struct{})
		t.Cleanup(func() { close(hang) })
		// The mirror cannot always stop git at once: the sync ignores its
		// context.
		f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").
			DoAndReturn(func(context.Context, string, string) error {
				close(syncing)
				<-hang
				return nil
			})
		errs := make(chan error, 2)
		go func() {
			_, err := f.svc.CacheModel(context.Background(), pullModel)
			errs <- err
		}()
		<-syncing
		go func() {
			_, err := f.svc.CacheModel(context.Background(), pullModel)
			errs <- err
		}()

		closeWithin(t, f, 5*time.Second)
		for range 2 {
			require.ErrorIs(t, <-errs, modelcache.ErrClosed)
		}
	})

	t.Run("for a forced re-sync", func(t *testing.T) {
		f := newFixture(t, modelcache.Options{})
		f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
		f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
		f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "v2").Return(nil, revisionNotFound("v2")).AnyTimes()
		f.lister.files = &git.RevisionFiles{Revision: "v2", Commit: "c2", Files: revisionFiles().Files}
		hanging := f.hangingRefresh(t)
		errs := make(chan error, 1)
		go func() {
			_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "v2"})
			errs <- err
		}()
		<-hanging

		closeWithin(t, f, 5*time.Second)
		require.ErrorIs(t, <-errs, modelcache.ErrClosed)
	})
}

func closeWithin(t *testing.T, f *fixture, d time.Duration) {
	t.Helper()
	closed := make(chan struct{})
	go func() {
		f.svc.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(d):
		t.Fatal("Close() did not return")
	}
}

// branchFiles lists a branch int4 of commit c2 that shares tokenizer.json (B)
// with main and has weights C of its own.
func branchFiles() *git.RevisionFiles {
	return &git.RevisionFiles{Revision: "int4", Commit: "c2", Files: []git.RevisionFile{
		{Path: "config.json", Size: 10},
		{Path: "model-int4.safetensors", Size: 40, IsLFS: true, OID: oidC},
		{Path: "tokenizer.json", Size: 50, IsLFS: true, OID: oidB},
	}}
}

func (f *fixture) twoBranches() {
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "").Return(revisionFiles(), nil).AnyTimes()
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "int4").Return(branchFiles(), nil).AnyTimes()
	var progress atomic.Int64
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string) git.LFSDownload {
			return download{done: progress.Add(1), total: 1 << 40}
		}).AnyTimes()
}

// A run of another branch that shares an object is not a run of main: main's
// own weights are not pulled by it.
func TestGetModelCache_IgnoresTheRunOfAnotherBranch(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.twoBranches()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	int4, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "int4"})
	require.NoError(t, err)
	require.Equal(t, modelcache.TaskRunning, int4.Task.State)

	// The shared object shows its download in the status of main too.
	var st *modelcache.CacheStatus
	require.Eventually(t, func() bool {
		st, err = f.svc.GetModelCache(context.Background(), "proj", "model", "")
		return err == nil && fileState(st, oidB).State == modelcache.FileDownloading
	}, 5*time.Second, time.Millisecond)

	require.Equal(t, "c1", st.Commit)
	require.Nil(t, st.Task)
	require.Equal(t, modelcache.FileMissing, fileState(st, oidA).State)
}

func TestGetModelCache_PrefersTheRunOfItsCommit(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.twoBranches()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(3)
	own, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	// A later run of int4 that pulls only the shared object.
	shared := &modelcache.Request{Project: "proj", Name: "model", Revision: "int4", Selection: modelcache.Selection{AllowPatterns: []string{"tokenizer.json"}}}
	other, err := f.svc.CacheModel(context.Background(), shared)
	require.NoError(t, err)
	require.NotEqual(t, own.Task.ID, other.Task.ID)
	_, err = f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "int4"})
	require.NoError(t, err)

	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")

	require.NoError(t, err)
	require.Equal(t, own.Task.ID, st.Task.ID)
}

// After the branch moved to a commit with the same objects, the status of the
// branch keeps showing the run, also once it failed.
func TestGetModelCache_KeepsTheOutcomeOfARunThatFollowedTheBranch(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(syncedModel("model"), nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil).AnyTimes()
	var commit atomic.Value
	commit.Store("c1")
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", gomock.Any()).
		DoAndReturn(func(context.Context, string, string, string, string) (*git.RevisionFiles, error) {
			files := revisionFiles()
			files.Commit = commit.Load().(string)
			return files, nil
		}).AnyTimes()
	f.disk.Store(oidB, true)
	fail := make(chan struct{})
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string) git.LFSDownload {
			select {
			case <-fail:
				return nil // the mirror forgot it: the run fails
			default:
				return download{done: 1, total: 100}
			}
		}).AnyTimes()
	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	// An upstream README edit moved main, then the run failed.
	commit.Store("c2")
	close(fail)

	st := waitForTask(t, f, modelcache.TaskFailed)
	require.Equal(t, "c2", st.Commit)
	require.Equal(t, modelcache.FileFailed, fileState(st, oidA).State)
}

// A failed run whose failed objects are all stored by now is not reported:
// a preview and the status agree with a pull, which hides it too.
func TestCacheModel_FailureThatNoLongerHoldsIsHiddenEverywhere(t *testing.T) {
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil).AnyTimes()
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	narrow := &modelcache.Request{Project: "proj", Name: "model", Selection: modelcache.Selection{AllowPatterns: []string{"model.safetensors"}}}
	_, err := f.svc.CacheModel(context.Background(), narrow)
	require.NoError(t, err)
	waitForTask(t, f, modelcache.TaskFailed)

	// A client downloaded the failed object meanwhile; tokenizer.json is
	// still missing.
	f.disk.Store(oidA, true)

	pull, err := f.svc.CacheModel(context.Background(), narrow)
	require.NoError(t, err)
	require.Nil(t, pull.Task)
	preview := *narrow
	preview.DryRun = true
	dry, err := f.svc.CacheModel(context.Background(), &preview)
	require.NoError(t, err)
	require.Nil(t, dry.Task)
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStatePartial, st.State)
	require.Nil(t, st.Task)
}

// An object that gave its download slot back for a re-sync, and waits for a
// slot again, has not started: it shows as queued, not downloading.
func TestCacheModel_ObjectWaitingForASlotIsQueued(t *testing.T) {
	f := newFixture(t, modelcache.Options{GlobalConcurrency: 1})
	f.existingModel()
	f.disk.Store(oidA, true)
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", gomock.Any()).Return(nil).Times(2)
	inRefresh, release := make(chan struct{}), make(chan struct{})
	f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) error {
			close(inRefresh)
			<-release
			return nil
		})
	other := &git.RevisionFiles{Revision: "main", Commit: "c3", Files: []git.RevisionFile{
		{Path: "model.safetensors", Size: 100, IsLFS: true, OID: oidC},
	}}
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "other").Return(syncedModel("other"), nil).AnyTimes()
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "other", gomock.Any()).Return(other, nil).AnyTimes()
	var progress atomic.Int64
	f.git.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, oid string) git.LFSDownload {
			if oid == oidB {
				return nil // forgotten: B re-syncs
			}
			return download{done: progress.Add(1), total: 1 << 40} // C downloads slowly
		}).AnyTimes()

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)
	<-inRefresh
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.FileQueued, fileState(st, oidB).State, "B waits for a re-sync")

	_, err = f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "other"})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		st, err := f.svc.GetModelCache(context.Background(), "proj", "other", "")
		return err == nil && st.Files[0].State == modelcache.FileDownloading
	}, 5*time.Second, time.Millisecond)
	close(release) // B now waits for the slot that C holds
	time.Sleep(50 * time.Millisecond)

	st, err = f.svc.GetModelCache(context.Background(), "proj", "model", "")
	require.NoError(t, err)
	require.Equal(t, modelcache.FileQueued, fileState(st, oidB).State, "B waits for a slot")
	require.Zero(t, st.Totals.DownloadingFiles)
}

func TestCacheModel_RejectsRevisionsThatMatrixHubCannotResolve(t *testing.T) {
	f := newFixture(t, modelcache.Options{})

	// No GetModel expectation: the request fails before any lookup.
	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "release@2024"})

	require.ErrorIs(t, err, modelcache.ErrUnsupportedRevision)
}

func TestGetModelCache_RejectsRevisionExpressions(t *testing.T) {
	f := newFixture(t, modelcache.Options{})

	for _, revision := range []string{"main~1", "refs/pr/1", "a@b"} {
		_, err := f.svc.GetModelCache(context.Background(), "proj", "model", revision)
		require.ErrorIs(t, err, modelcache.ErrUnsupportedRevision, revision)
	}
}
