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
}

func (l *fakeLister) ListRevisionFiles(context.Context, *registry.Registry, string, string, string) (*git.RevisionFiles, error) {
	l.calls.Add(1)
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
	svc    *modelcache.Service
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
		st, err = f.svc.GetModelCache(context.Background(), "proj", "model", "", modelcache.Selection{})
		return err == nil && st.Task.State == want
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
	require.Equal(t, modelcache.TaskIdle, st.Task.State)
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
	require.Equal(t, modelcache.TaskSucceeded, st.Task.State)
	require.Equal(t, int64(3), st.Totals.CachedFiles)
}

func TestCacheModel_DownloadsMissingObjects(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.models.EXPECT().CheckOrSyncFromRemote(gomock.Any(), "proj", "model").Return(nil)
	f.completeOnFetch()

	st, err := f.svc.CacheModel(context.Background(), &modelcache.Request{Project: "proj", Name: "model"})
	require.NoError(t, err)
	require.NotEmpty(t, st.Task.ID)

	done := waitForTask(t, f, modelcache.TaskSucceeded)
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
	require.Equal(t, modelcache.TaskSucceeded, st.Task.State)
}

func TestGetModelCache_IsReadOnly(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.existingModel()
	f.disk.Store(oidA, true)

	// Only GetModel, ListRevisionFiles and LFSObjectExists are expected; the
	// strict mocks fail on any download, sync or refresh.
	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "", modelcache.Selection{})

	require.NoError(t, err)
	require.Equal(t, modelcache.TaskIdle, st.Task.State)
	require.Equal(t, int64(2), st.Totals.CachedFiles)
	require.Equal(t, int64(1), st.Totals.MissingFiles)
}

func TestGetModelCache_ModelNotFound(t *testing.T) {
	f := newFixture(t, modelcache.Options{})
	f.models.EXPECT().GetModel(gomock.Any(), "proj", "model").Return(nil, errModelNotFound)

	_, err := f.svc.GetModelCache(context.Background(), "proj", "model", "", modelcache.Selection{})

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
	require.NotNil(t, done.Task.StartedAt)

	mu.Lock()
	now = now.Add(2 * time.Hour)
	mu.Unlock()

	st, err := f.svc.GetModelCache(context.Background(), "proj", "model", "", modelcache.Selection{})
	require.NoError(t, err)
	require.Equal(t, modelcache.TaskSucceeded, st.Task.State)
	require.Nil(t, st.Task.StartedAt)
}
