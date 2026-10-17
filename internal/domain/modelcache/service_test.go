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
)

const (
	oidA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	oidB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

var errModelNotFound = errors.New("failed to get model: record not found")

func revisionFiles() *git.RevisionFiles {
	return &git.RevisionFiles{Revision: "main", Commit: "c1", Files: []git.RevisionFile{
		{Path: "config.json", Size: 10},
		{Path: "model.safetensors", Size: 100, IsLFS: true, OID: oidA},
		{Path: "tokenizer.json", Size: 50, IsLFS: true, OID: oidB},
	}}
}

type fakeLister struct {
	files *git.RevisionFiles
	err   error
	calls atomic.Int32
	// onList, when set, runs inside every listing.
	onList func()
}

func (l *fakeLister) ListRevisionFiles(context.Context, *registry.Registry, string, string, string) (*git.RevisionFiles, error) {
	l.calls.Add(1)
	if l.onList != nil {
		l.onList()
	}
	return l.files, l.err
}

type download struct{ done, total int64 }

func (d download) Downloaded() int64 { return d.done }
func (d download) Total() int64      { return d.total }

type fixture struct {
	models *modelmocks.MockIModelService
	git    *gitmocks.MockIGitRepo
	lister *fakeLister
	disk   sync.Map
	svc    modelcache.IModelCacheService
}

func newFixture(t *testing.T, opts modelcache.Options) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	f := &fixture{
		models: modelmocks.NewMockIModelService(ctrl),
		git:    gitmocks.NewMockIGitRepo(ctrl),
		lister: &fakeLister{files: revisionFiles()},
	}

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

	if opts.PollInterval == 0 {
		opts.PollInterval = time.Millisecond
	}
	if opts.RetryBackoff == 0 {
		opts.RetryBackoff = time.Millisecond
	}
	if opts.RequeueMinInterval == 0 {
		opts.RequeueMinInterval = time.Nanosecond
	}
	f.svc = modelcache.NewModelCacheService(f.models, projects, registries, f.git, f.lister, opts)
	// Registered after the controller, so it runs first: workers stop before
	// the controller verifies its expectations.
	t.Cleanup(f.svc.Close)
	return f
}

// existingModel makes the model exist locally and list revisionFiles.
func (f *fixture) existingModel() {
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").
		Return(&model.Model{Name: "model", ProjectName: "proj"}, nil).AnyTimes()
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

func TestCacheModel_RollsBackNewModelWhenSyncFails(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	syncErr := errors.New("failed to list remote refs")
	gomock.InOrder(
		f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound),
		f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(syncErr),
		f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model"}, nil),
		f.models.EXPECT().DeleteModel(gomock.Any(), "proj", "model").Return(nil),
	)

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})

	require.ErrorIs(t, err, modelcache.ErrUpstreamSync)
	require.ErrorIs(t, err, syncErr)
}

func TestCacheModel_KeepsExistingModelWhenSyncFails(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model"}, nil)
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

func TestCacheModel_RefreshesOnceWhenRevisionIsUnknownLocally(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model"}, nil).AnyTimes()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	gomock.InOrder(
		f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c2").
			Return(nil, errors.New(`revision "c2" not found: reference not found`)),
		f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil),
		f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c2").
			Return(revisionFiles(), nil),
	)
	f.disk.Store(oidA, true)
	f.disk.Store(oidB, true)

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "c2"})

	require.NoError(t, err)
	require.Equal(t, modelcache.CacheStateCached, st.State)
}

func TestGetModelCache_IsReadOnly(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.disk.Store(oidA, true)

	// Only GetModel, ListRevisionFiles and LFSObjectExists are expected; the
	// strict mocks fail on any download, sync or refresh.
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "")

	require.NoError(t, err)
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

func TestCacheModel_RevisionMissingAfterRefresh(t *testing.T) {
	tests := []struct {
		name     string
		upstream error
		want     error
	}{
		// For example a commit that only a pull request ref reaches.
		{"upstream has it outside mirrored refs", nil, modelcache.ErrRevisionNotMirrored},
		{"upstream does not have it", registrydiscovery.ErrRemoteNotFound, modelcache.ErrRevisionNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, modelcache.Options{})
			f.lister.err = tt.upstream
			f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(&model.Model{Name: "model"}, nil).AnyTimes()
			f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
			f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil)
			f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c9").
				Return(nil, errors.New(`revision "c9" not found: reference not found`)).Times(2)

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
		f.models.EXPECT().RefreshFromRemote(gomock.Any(), "proj", "model").Return(nil),
	)
	f.git.EXPECT().ListRevisionFiles(gomock.Any(), "models", "proj", "model", "c9").
		Return(nil, errors.New(`revision "c9" not found: reference not found`)).Times(2)

	_, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", Revision: "c9"})

	// The model synced and exists upstream, so it stays: no DeleteModel
	// expectation. The upstream listing taken before the sync explains the
	// failure.
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
				return &model.Model{Name: "model"}, nil
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
	// expectation.
	require.NoError(t, first)
	require.ErrorIs(t, second, modelcache.ErrUpstreamSync)
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

	// A wider preview has no run of its own but shows the download.
	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})
	require.NoError(t, err)
	require.Nil(t, st.Task)
	for _, file := range st.Files {
		if file.OID == oidA {
			require.Equal(t, modelcache.FileDownloading, file.State)
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
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil).Times(2)
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
	f := newFixture(t, modelcache.Options{MaxAttempts: 1})
	var deleted atomic.Bool
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").
		DoAndReturn(func(context.Context, string, string) (*model.Model, error) {
			if deleted.Load() {
				return nil, errModelNotFound
			}
			return &model.Model{Name: "model", ProjectName: "proj"}, nil
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

	_, err := f.svc.CacheModel(context.Background(), pullModel)
	require.NoError(t, err)

	// No RefreshFromRemote expectation: the sync would re-create the model.
	// The model is gone, so the run is observed through a preview.
	var st *modelcache.CacheStatus
	require.Eventually(t, func() bool {
		st, err = f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model", DryRun: true})
		return err == nil && st.Task != nil && st.Task.State == modelcache.TaskFailed
	}, 5*time.Second, 2*time.Millisecond)
	for _, file := range st.Files {
		if file.IsLFS {
			require.Contains(t, file.Error, "no longer available")
		}
	}
}
