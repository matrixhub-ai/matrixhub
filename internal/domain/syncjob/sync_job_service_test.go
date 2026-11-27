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

package syncjob_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	datasetdomain "github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	datasetmocks "github.com/matrixhub-ai/matrixhub/internal/domain/dataset/mocks"
	gitdomain "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	jobdomain "github.com/matrixhub-ai/matrixhub/internal/domain/job"
	modeldomain "github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	projectdomain "github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	registrydomain "github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	registrymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registry/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob/mocks"
)

type pullFixture struct {
	syncJobRepo  *mocks.MockISyncJobRepo
	registryRepo *registrymocks.MockIRegistryRepo
	projectRepo  *projectmocks.MockIProjectRepo
	modelRepo    *modelmocks.MockIModelRepo
	datasetRepo  *datasetmocks.MockIDatasetRepo
	gitRepo      *gitmocks.MockIGitRepo
	modelMeta    *mocks.MockMetadataSyncer
	datasetMeta  *mocks.MockMetadataSyncer
}

func TestSyncJobService_ClaimPendingSyncJob(t *testing.T) {
	ctx := context.Background()
	f, svc := newPullFixture(t)
	f.syncJobRepo.EXPECT().GetSyncJob(ctx, 42).Return(&syncjob.SyncJob{
		ID:     42,
		Status: syncjob.SyncJobStatusPending,
	}, nil)
	f.syncJobRepo.EXPECT().UpdateJobStatusCAS(ctx, 42, syncjob.SyncJobStatusPending, syncjob.SyncJobStatusRunning).Return(true, nil)

	due, claimed, err := svc.(interface {
		ClaimPendingSyncJob(context.Context, int) (jobdomain.DueJob, bool, error)
	}).ClaimPendingSyncJob(ctx, 42)

	require.NoError(t, err)
	require.True(t, claimed)
	require.Equal(t, 42, due.ID)
}

func TestSyncJobService_CreateSyncJobsTriggersAfterBatchIsPersisted(t *testing.T) {
	ctx := context.Background()
	f, svc := newPullFixture(t)
	var persistedJobIDs []int
	f.syncJobRepo.EXPECT().CreateSyncJobs(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, created []*syncjob.SyncJob) error {
			for index, job := range created {
				job.ID = 41 + index
				persistedJobIDs = append(persistedJobIDs, job.ID)
			}
			return nil
		})
	var triggeredJobIDs []int
	svc.(interface{ SetOnJobCreated(func(int) bool) }).SetOnJobCreated(func(id int) bool {
		require.Equal(t, []int{41, 42}, persistedJobIDs, "the complete batch must be persisted before triggering")
		triggeredJobIDs = append(triggeredJobIDs, id)
		return true
	})

	require.NoError(t, svc.CreateSyncJobs(ctx, []*syncjob.SyncJob{{}, {}}))
	require.Equal(t, []int{41, 42}, triggeredJobIDs)
}

func newPullFixture(t *testing.T) (*pullFixture, syncjob.ISyncJobService) {
	t.Helper()
	ctrl := gomock.NewController(t)

	f := &pullFixture{
		syncJobRepo:  mocks.NewMockISyncJobRepo(ctrl),
		registryRepo: registrymocks.NewMockIRegistryRepo(ctrl),
		projectRepo:  projectmocks.NewMockIProjectRepo(ctrl),
		modelRepo:    modelmocks.NewMockIModelRepo(ctrl),
		datasetRepo:  datasetmocks.NewMockIDatasetRepo(ctrl),
		gitRepo:      gitmocks.NewMockIGitRepo(ctrl),
		modelMeta:    mocks.NewMockMetadataSyncer(ctrl),
		datasetMeta:  mocks.NewMockMetadataSyncer(ctrl),
	}

	svc := syncjob.NewSyncJobService(
		f.syncJobRepo,
		f.registryRepo,
		f.projectRepo,
		f.modelRepo,
		f.datasetRepo,
		f.gitRepo,
		f.modelMeta,
		f.datasetMeta,
		nil,
	)
	return f, svc
}

func pullJob(resourceType string) *syncjob.SyncJob {
	return &syncjob.SyncJob{
		ID:                 7,
		RemoteRegistryID:   3,
		RemoteProjectName:  "remote-project",
		RemoteResourceName: "remote-resource",
		ProjectName:        "local-project",
		ResourceName:       "local-resource",
		ResourceType:       resourceType,
		SyncType:           "pull",
	}
}

func TestExecuteSyncJob_PullRefreshesMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("model pull syncs metadata after the git pull", func(t *testing.T) {
		f, svc := newPullFixture(t)

		f.registryRepo.EXPECT().GetRegistry(gomock.Any(), 3).
			Return(&registrydomain.Registry{ID: 3, Name: "remote", URL: "https://remote.example.com"}, nil)
		f.projectRepo.EXPECT().GetProjectByName(gomock.Any(), "local-project").
			Return(&projectdomain.Project{ID: 11, Name: "local-project"}, nil)
		// Model record does not exist yet, so the job creates it with empty metadata.
		f.modelRepo.EXPECT().GetByProjectAndName(gomock.Any(), "local-project", "local-resource").
			Return(nil, errors.New("not found"))
		f.modelRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, m *modeldomain.Model) (*modeldomain.Model, error) {
				m.ID = 42
				return m, nil
			})

		gomock.InOrder(
			f.gitRepo.EXPECT().PullFromRemote(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, gr *gitdomain.GitRepository) error {
					require.Equal(t, "model", gr.ResourceType)
					return nil
				}),
			// The regression this guards: README/size/labels only land in the
			// database when the job explicitly refreshes metadata after the pull.
			f.modelMeta.EXPECT().SyncMetadata(gomock.Any(), "local-project", "local-resource").Return(nil),
		)

		f.syncJobRepo.EXPECT().UpdateSyncJob(gomock.Any(), gomock.Any()).Return(nil)

		require.NoError(t, svc.ExecuteSyncJob(ctx, pullJob("model")))
	})

	t.Run("dataset pull creates a dataset record and syncs dataset metadata", func(t *testing.T) {
		f, svc := newPullFixture(t)

		f.registryRepo.EXPECT().GetRegistry(gomock.Any(), 3).
			Return(&registrydomain.Registry{ID: 3, Name: "remote", URL: "https://remote.example.com"}, nil)
		f.projectRepo.EXPECT().GetProjectByName(gomock.Any(), "local-project").
			Return(&projectdomain.Project{ID: 11, Name: "local-project"}, nil)
		f.datasetRepo.EXPECT().GetByProjectAndName(gomock.Any(), "local-project", "local-resource").
			Return(nil, errors.New("not found"))
		f.datasetRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, d *datasetdomain.Dataset) (*datasetdomain.Dataset, error) {
				require.Equal(t, "local-resource", d.Name)
				d.ID = 99
				return d, nil
			})

		gomock.InOrder(
			f.gitRepo.EXPECT().PullFromRemote(gomock.Any(), gomock.Any()).Return(nil),
			f.datasetMeta.EXPECT().SyncMetadata(gomock.Any(), "local-project", "local-resource").Return(nil),
		)

		f.syncJobRepo.EXPECT().UpdateSyncJob(gomock.Any(), gomock.Any()).Return(nil)

		require.NoError(t, svc.ExecuteSyncJob(ctx, pullJob("dataset")))
	})

	t.Run("metadata sync failure fails the job instead of reporting success", func(t *testing.T) {
		f, svc := newPullFixture(t)

		f.registryRepo.EXPECT().GetRegistry(gomock.Any(), 3).
			Return(&registrydomain.Registry{ID: 3, Name: "remote", URL: "https://remote.example.com"}, nil)
		f.projectRepo.EXPECT().GetProjectByName(gomock.Any(), "local-project").
			Return(&projectdomain.Project{ID: 11, Name: "local-project"}, nil)
		f.modelRepo.EXPECT().GetByProjectAndName(gomock.Any(), "local-project", "local-resource").
			Return(&modeldomain.Model{ID: 42, Name: "local-resource"}, nil)
		f.gitRepo.EXPECT().PullFromRemote(gomock.Any(), gomock.Any()).Return(nil)
		f.modelMeta.EXPECT().SyncMetadata(gomock.Any(), "local-project", "local-resource").
			Return(errors.New("database is down"))

		f.syncJobRepo.EXPECT().UpdateSyncJob(gomock.Any(), gomock.Any()).Return(nil)

		err := svc.ExecuteSyncJob(ctx, pullJob("model"))
		require.ErrorContains(t, err, "sync metadata for model local-project/local-resource")
	})
}

func proxyProject() *projectdomain.Project {
	return &projectdomain.Project{ID: 11, Name: "proxy", RegistryID: lo.ToPtr(3), Organization: "upstream-org"}
}

func TestSyncJobService_CreateModelCacheJob(t *testing.T) {
	ctx := context.Background()

	t.Run("creates a pending job that pulls from the project's upstream", func(t *testing.T) {
		f, svc := newPullFixture(t)
		f.projectRepo.EXPECT().GetProjectByName(ctx, "proxy").Return(proxyProject(), nil)
		var created *syncjob.SyncJob
		f.syncJobRepo.EXPECT().CreateSyncJobs(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, jobs []*syncjob.SyncJob) error {
				require.Len(t, jobs, 1)
				created = jobs[0]
				created.ID = 21
				return nil
			})
		var triggered []int
		svc.(interface{ SetOnJobCreated(func(int) bool) }).SetOnJobCreated(func(id int) bool {
			triggered = append(triggered, id)
			return true
		})

		job, err := svc.CreateModelCacheJob(ctx, "proxy", "gpt2", "v1.0")
		require.NoError(t, err)
		require.Same(t, created, job)
		require.Equal(t, []int{21}, triggered)
		require.Equal(t, syncjob.JobTypeModelCache, job.JobType)
		require.Equal(t, syncjob.SyncJobStatusPending, job.Status)
		require.Nil(t, job.SyncTaskID)
		require.Equal(t, 3, job.RemoteRegistryID)
		require.Equal(t, "upstream-org", job.RemoteProjectName)
		require.Equal(t, "gpt2", job.RemoteResourceName)
		require.Equal(t, "proxy", job.ProjectName)
		require.Equal(t, "gpt2", job.ResourceName)
		require.Equal(t, "model", job.ResourceType)
		require.Equal(t, "pull", job.SyncType)
		require.JSONEq(t, `{"revision":"v1.0"}`, string(job.Params))
	})

	t.Run("rejects a project without upstream", func(t *testing.T) {
		f, svc := newPullFixture(t)
		f.projectRepo.EXPECT().GetProjectByName(ctx, "local").Return(&projectdomain.Project{ID: 12, Name: "local"}, nil)

		_, err := svc.CreateModelCacheJob(ctx, "local", "gpt2", "")
		require.ErrorIs(t, err, syncjob.ErrNotProxyProject)
	})

	t.Run("returns a failed project lookup", func(t *testing.T) {
		f, svc := newPullFixture(t)
		f.projectRepo.EXPECT().GetProjectByName(ctx, "missing").Return(nil, gorm.ErrRecordNotFound)

		_, err := svc.CreateModelCacheJob(ctx, "missing", "gpt2", "")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("accepts branches, tags and commits", func(t *testing.T) {
		for _, revision := range []string{
			"", "main", "v1.0", "release/2024-01", "refs/heads/main", "refs/tags/v1",
			"0123456789abcdef0123456789abcdef01234567",
		} {
			f, svc := newPullFixture(t)
			f.projectRepo.EXPECT().GetProjectByName(ctx, "proxy").Return(proxyProject(), nil)
			f.syncJobRepo.EXPECT().CreateSyncJobs(ctx, gomock.Any()).Return(nil)

			job, err := svc.CreateModelCacheJob(ctx, "proxy", "gpt2", revision)
			require.NoError(t, err, revision)
			params, err := job.ModelCacheParams()
			require.NoError(t, err)
			require.Equal(t, revision, params.Revision)
		}
	})

	t.Run("rejects other refs and revision expressions without a lookup", func(t *testing.T) {
		for _, revision := range []string{
			"refs/pr/1", "main~1", "main^", "HEAD@{1}", "user@host", "@", "-main", "/main", "main/",
			"a..b", "a//b", ".hidden", "main.lock", "with space", "main:path", "ma*n", "ma?n", "ma[n",
		} {
			_, svc := newPullFixture(t)

			_, err := svc.CreateModelCacheJob(ctx, "proxy", "gpt2", revision)
			require.ErrorIs(t, err, syncjob.ErrInvalidRevision, revision)
		}
	})
}

// bufferLogStore keeps the log of every job in memory.
type bufferLogStore struct {
	logs map[int]*bytes.Buffer
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func (s *bufferLogStore) Writer(jobID int) (io.WriteCloser, error) {
	if s.logs[jobID] == nil {
		s.logs[jobID] = &bytes.Buffer{}
	}
	return nopWriteCloser{s.logs[jobID]}, nil
}

func (s *bufferLogStore) Reader(int) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (s *bufferLogStore) log(jobID int) string {
	return s.logs[jobID].String()
}

const (
	oidA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	oidB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

// objectBehavior is how the download of an LFS object goes.
type objectBehavior int

const (
	// downloads makes progress and stores the object.
	downloads objectBehavior = iota
	// stalls makes no progress.
	stalls
	// finishesUnstored downloads every byte but never stores the object,
	// as when a reader keeps the download open.
	finishesUnstored
)

// lfsFake is the LFS storage and the mirror downloads of a test.
type lfsFake struct {
	mu        sync.Mutex
	stored    map[string]bool
	queued    map[string]*fakeDownload
	behaviors map[string]objectBehavior
	// unqueued are objects that a pull does not queue, for the given number
	// of pulls.
	unqueued map[string]int
	sizes    map[string]int64
	pulls    int
}

type fakeDownload struct {
	lfs  *lfsFake
	oid  string
	done int64
}

func (d *fakeDownload) Downloaded() int64 {
	d.lfs.mu.Lock()
	defer d.lfs.mu.Unlock()
	total := d.lfs.sizes[d.oid]
	switch d.lfs.behaviors[d.oid] {
	case downloads:
		d.done = min(d.done+total/2+1, total)
		if d.done == total {
			d.lfs.stored[d.oid] = true
			delete(d.lfs.queued, d.oid)
		}
	case finishesUnstored:
		d.done = total
	case stalls:
	}
	return d.done
}

func (d *fakeDownload) Total() int64 {
	d.lfs.mu.Lock()
	defer d.lfs.mu.Unlock()
	return d.lfs.sizes[d.oid]
}

func (l *lfsFake) exists(_ context.Context, oid string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stored[oid]
}

func (l *lfsFake) fetch(_ context.Context, oid string) gitdomain.LFSDownload {
	l.mu.Lock()
	defer l.mu.Unlock()
	if d, ok := l.queued[oid]; ok {
		return d
	}
	return nil
}

// pull queues every object that is not stored, as a pull of the mirror does.
func (l *lfsFake) pull(context.Context, *gitdomain.GitRepository) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pulls++
	for oid := range l.sizes {
		if l.stored[oid] || l.queued[oid] != nil {
			continue
		}
		if l.unqueued[oid] > 0 {
			l.unqueued[oid]--
			continue
		}
		l.queued[oid] = &fakeDownload{lfs: l, oid: oid}
	}
	return nil
}

func (l *lfsFake) pullCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.pulls
}

const cacheJobCommit = "0123456789abcdef0123456789abcdef01234567"

type cacheJobFixture struct {
	syncJobRepo  *mocks.MockISyncJobRepo
	registryRepo *registrymocks.MockIRegistryRepo
	projectRepo  *projectmocks.MockIProjectRepo
	modelRepo    *modelmocks.MockIModelRepo
	gitRepo      *gitmocks.MockIGitRepo
	modelMeta    *mocks.MockMetadataSyncer
	svc          syncjob.ISyncJobService
	lfs          *lfsFake
	logs         *bufferLogStore
	job          *syncjob.SyncJob
	revision     string

	mu sync.Mutex
	// modelExists tells whether the model record exists.
	modelExists bool
	// afterMetadata runs when the job has synced the model's metadata.
	afterMetadata func()
	// createdMeanwhile makes someone else create the model record just
	// before the job does.
	createdMeanwhile bool
	// saved is the job as the service last saved it.
	saved *syncjob.SyncJob
}

// newCacheJobFixture runs job 7, which caches a revision of proxy/gpt2. The
// revision has two LFS objects, which download at once unless a test says
// otherwise.
func newCacheJobFixture(t *testing.T, revision string) *cacheJobFixture {
	t.Helper()
	params, err := json.Marshal(syncjob.ModelCacheParams{Revision: revision})
	require.NoError(t, err)
	ctrl := gomock.NewController(t)
	f := &cacheJobFixture{
		syncJobRepo:  mocks.NewMockISyncJobRepo(ctrl),
		registryRepo: registrymocks.NewMockIRegistryRepo(ctrl),
		projectRepo:  projectmocks.NewMockIProjectRepo(ctrl),
		modelRepo:    modelmocks.NewMockIModelRepo(ctrl),
		gitRepo:      gitmocks.NewMockIGitRepo(ctrl),
		modelMeta:    mocks.NewMockMetadataSyncer(ctrl),
		lfs: &lfsFake{
			stored:    map[string]bool{},
			queued:    map[string]*fakeDownload{},
			behaviors: map[string]objectBehavior{},
			unqueued:  map[string]int{},
			sizes:     map[string]int64{oidA: 100, oidB: 50},
		},
		logs: &bufferLogStore{logs: map[int]*bytes.Buffer{}},
		job: &syncjob.SyncJob{
			ID:                 7,
			JobType:            syncjob.JobTypeModelCache,
			RemoteRegistryID:   3,
			RemoteProjectName:  "upstream-org",
			RemoteResourceName: "gpt2",
			ProjectName:        "proxy",
			ResourceName:       "gpt2",
			ResourceType:       "model",
			SyncType:           "pull",
			Status:             syncjob.SyncJobStatusRunning,
			Params:             params,
		},
		revision: revision,
	}
	f.svc = syncjob.NewSyncJobService(f.syncJobRepo, f.registryRepo, f.projectRepo, f.modelRepo,
		nil, f.gitRepo, f.modelMeta, nil, f.logs)
	f.setOptions(syncjob.ModelCacheOptions{
		PollInterval:    time.Millisecond,
		RetryBackoff:    time.Millisecond,
		StallTimeout:    50 * time.Millisecond,
		StoreTimeout:    50 * time.Millisecond,
		RequeueInterval: time.Nanosecond,
	})
	f.svc.SetOnJobDone(func(context.Context, int) error {
		t.Error("a model cache job belongs to no sync task")
		return nil
	})

	f.syncJobRepo.EXPECT().GetSyncJob(gomock.Any(), 7).Return(f.job, nil)
	f.syncJobRepo.EXPECT().UpdateSyncJob(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, job *syncjob.SyncJob) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			saved := *job
			f.saved = &saved
			return nil
		})
	f.gitRepo.EXPECT().LFSObjectExists(gomock.Any(), gomock.Any()).DoAndReturn(f.lfs.exists).AnyTimes()
	f.gitRepo.EXPECT().FetchLFSObject(gomock.Any(), gomock.Any()).DoAndReturn(f.lfs.fetch).AnyTimes()
	f.modelRepo.EXPECT().GetByProjectAndName(gomock.Any(), "proxy", "gpt2").DoAndReturn(
		func(context.Context, string, string) (*modeldomain.Model, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if !f.modelExists {
				return nil, fmt.Errorf("failed to get model: %w", gorm.ErrRecordNotFound)
			}
			return &modeldomain.Model{ID: 42, Name: "gpt2"}, nil
		}).AnyTimes()
	return f
}

// setOptions tunes the job. Unset options take the defaults of the service.
func (f *cacheJobFixture) setOptions(opts syncjob.ModelCacheOptions) {
	syncjob.SetModelCacheOptions(f.svc, opts)
}

// expectLookups expects the project and registry lookups of a run.
func (f *cacheJobFixture) expectLookups() {
	f.projectRepo.EXPECT().GetProjectByName(gomock.Any(), "proxy").Return(proxyProject(), nil)
	f.registryRepo.EXPECT().GetRegistry(gomock.Any(), 3).
		Return(&registrydomain.Registry{ID: 3, Name: "hf", URL: "https://hf.example.com"}, nil)
}

// expectPull expects a run up to the metadata of the model: the lookups, then
// the pull, the model record and the metadata in this order. It returns the
// metadata sync.
func (f *cacheJobFixture) expectPull() *gomock.Call {
	f.expectLookups()
	metadata := f.modelMeta.EXPECT().SyncMetadata(gomock.Any(), "proxy", "gpt2").DoAndReturn(
		func(context.Context, string, string) error {
			if f.afterMetadata != nil {
				f.afterMetadata()
			}
			return nil
		})
	gomock.InOrder(
		f.gitRepo.EXPECT().PullFromRemote(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, gr *gitdomain.GitRepository) error {
				if gr.RemoteRegistryURL != "https://hf.example.com" || gr.RemoteProjectName != "upstream-org" ||
					gr.RemoteResourceName != "gpt2" || gr.ProjectName != "proxy" || gr.ResourceName != "gpt2" {
					return fmt.Errorf("unexpected pull: %+v", gr)
				}
				return f.lfs.pull(ctx, gr)
			}),
		// The record is created only once the pull succeeded.
		f.modelRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, m *modeldomain.Model) (*modeldomain.Model, error) {
				f.mu.Lock()
				defer f.mu.Unlock()
				f.modelExists = true
				if f.createdMeanwhile {
					return nil, gorm.ErrDuplicatedKey
				}
				m.ID = 42
				return m, nil
			}),
		metadata,
	)
	return metadata
}

// expectRun expects a run up to the files of the revision: the pull, then
// the listing of the revision, in which the two LFS objects back three files.
func (f *cacheJobFixture) expectRun() {
	metadata := f.expectPull()
	revision := f.revision
	if revision == "" {
		revision = "main"
	}
	f.gitRepo.EXPECT().ListRevisionFiles(gomock.Any(), "model", "proxy", "gpt2", f.revision).Return(&gitdomain.RevisionFiles{
		Revision: revision,
		Commit:   cacheJobCommit,
		Files: []gitdomain.RevisionFile{
			{Path: "config.json", Size: 10},
			{Path: "model.safetensors", Size: 100, IsLFS: true, OID: oidA},
			{Path: "copy/model.safetensors", Size: 100, IsLFS: true, OID: oidA},
			{Path: "tokenizer.json", Size: 50, IsLFS: true, OID: oidB},
		},
	}, nil).After(metadata)
}

// pullsAgain expects the pulls that queue missing objects again.
func (f *cacheJobFixture) pullsAgain(times int) {
	f.gitRepo.EXPECT().PullFromRemote(gomock.Any(), gomock.Any()).DoAndReturn(f.lfs.pull).Times(times)
}

func (f *cacheJobFixture) run() (*syncjob.SyncJob, error) {
	return f.runWithContext(context.Background())
}

func (f *cacheJobFixture) runWithContext(ctx context.Context) (*syncjob.SyncJob, error) {
	err := f.svc.ExecuteSyncJobWithLog(ctx, 7)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saved, err
}

func TestModelCacheJob_StoresEveryLFSObject(t *testing.T) {
	f := newCacheJobFixture(t, "v1.0")
	f.expectRun()

	saved, err := f.run()
	require.NoError(t, err)
	require.Equal(t, syncjob.SyncJobStatusSucceeded, saved.Status)
	require.NotZero(t, saved.CompletedTimestamp)
	require.True(t, f.lfs.exists(context.Background(), oidA))
	require.True(t, f.lfs.exists(context.Background(), oidB))
	require.Equal(t, 1, f.lfs.pullCount())

	log := f.logs.log(7)
	require.Contains(t, log, "source=upstream-org/gpt2 -> target=proxy/gpt2, revision=v1.0")
	require.Contains(t, log, "[INFO] pull completed")
	require.Contains(t, log, "[INFO] metadata refreshed")
	require.Contains(t, log, "revision v1.0 is commit "+cacheJobCommit+": 2 LFS objects, 150 bytes")
	require.Contains(t, log, "2 of 2 LFS objects done")
	require.Contains(t, log, "[INFO] job completed successfully; every LFS object of commit "+cacheJobCommit+" is stored")
}

func TestModelCacheJob_CachesTheDefaultBranch(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.expectRun()

	saved, err := f.run()
	require.NoError(t, err)
	require.Equal(t, syncjob.SyncJobStatusSucceeded, saved.Status)
	log := f.logs.log(7)
	require.Contains(t, log, "revision=the default branch")
	require.Contains(t, log, "revision main is commit "+cacheJobCommit)
}

func TestModelCacheJob_FailsForARevisionThatUpstreamDoesNotHave(t *testing.T) {
	f := newCacheJobFixture(t, "v9")
	metadata := f.expectPull()
	f.gitRepo.EXPECT().ListRevisionFiles(gomock.Any(), "model", "proxy", "gpt2", "v9").
		Return(nil, fmt.Errorf("%w: %q: %w", gitdomain.ErrRevisionNotFound, "v9", errors.New("reference not found"))).
		After(metadata)

	saved, err := f.run()
	require.ErrorIs(t, err, gitdomain.ErrRevisionNotFound)
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	// Upstream has the model, so its record stays.
	require.True(t, f.modelExists)
	require.Contains(t, f.logs.log(7), `[ERROR] revision not found: "v9": reference not found`)
}

func TestModelCacheJob_UsesAModelCreatedMeanwhile(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.createdMeanwhile = true
	f.expectRun()

	saved, err := f.run()
	require.NoError(t, err)
	require.Equal(t, syncjob.SyncJobStatusSucceeded, saved.Status)
}

func TestModelCacheJob_SkipsStoredObjects(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.lfs.stored[oidA] = true
	f.lfs.stored[oidB] = true
	f.expectRun()

	saved, err := f.run()
	require.NoError(t, err)
	require.Equal(t, syncjob.SyncJobStatusSucceeded, saved.Status)
}

func TestModelCacheJob_PullsAgainForAnObjectThatIsNotQueued(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.lfs.unqueued[oidB] = 1
	f.expectRun()
	f.pullsAgain(1)

	saved, err := f.run()
	require.NoError(t, err)
	require.Equal(t, syncjob.SyncJobStatusSucceeded, saved.Status)
	require.Equal(t, 2, f.lfs.pullCount())
	require.Contains(t, f.logs.log(7), "pulling the model again to queue its missing LFS objects")
}

func TestModelCacheJob_FailsForAnObjectThatIsNeverQueued(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.setOptions(syncjob.ModelCacheOptions{
		MaxAttempts:     2,
		PollInterval:    time.Millisecond,
		RetryBackoff:    time.Millisecond,
		RequeueInterval: time.Nanosecond,
	})
	f.lfs.unqueued[oidB] = 100
	f.expectRun()
	// One pull for each attempt.
	f.pullsAgain(2)

	saved, err := f.run()
	require.ErrorContains(t, err, "1 of 2 LFS objects were not stored")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	require.True(t, f.lfs.exists(context.Background(), oidA))
	log := f.logs.log(7)
	require.Contains(t, log, "tokenizer.json: attempt 1 of 2 failed: not queued for download")
	require.Contains(t, log, "[ERROR] tokenizer.json: not queued for download")
}

func TestModelCacheJob_PullsAgainAtMostOncePerInterval(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.setOptions(syncjob.ModelCacheOptions{
		MaxAttempts:     3,
		PollInterval:    time.Millisecond,
		RetryBackoff:    time.Millisecond,
		RequeueInterval: time.Hour,
	})
	f.lfs.unqueued[oidA] = 100
	f.lfs.unqueued[oidB] = 100
	f.expectRun()
	// The pull that started the job is the last one for an hour, so no
	// object pulls again.

	saved, err := f.run()
	require.ErrorContains(t, err, "2 of 2 LFS objects were not stored")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	require.Equal(t, 1, f.lfs.pullCount())
}

func TestModelCacheJob_StalledDownloadIsNotRetried(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.lfs.behaviors[oidB] = stalls
	f.expectRun()

	saved, err := f.run()
	require.ErrorContains(t, err, "1 of 2 LFS objects were not stored")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	log := f.logs.log(7)
	require.Contains(t, log, "[ERROR] tokenizer.json: the download made no progress for 50ms")
	require.NotContains(t, log, "attempt 1 of 3 failed")
}

func TestModelCacheJob_UnstoredDownloadFails(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.lfs.behaviors[oidA] = finishesUnstored
	f.expectRun()

	saved, err := f.run()
	require.ErrorContains(t, err, "1 of 2 LFS objects were not stored")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	require.Contains(t, f.logs.log(7), "[ERROR] model.safetensors: the download finished but was not stored")
}

func TestModelCacheJob_DoesNotPullADeletedModelAgain(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.lfs.unqueued[oidB] = 1
	f.expectRun()
	// The model is deleted while the job runs. Pulling it again would
	// re-create its repository.
	f.afterMetadata = func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.modelExists = false
	}

	saved, err := f.run()
	require.ErrorContains(t, err, "1 of 2 LFS objects were not stored")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	require.Equal(t, 1, f.lfs.pullCount())
	log := f.logs.log(7)
	require.Contains(t, log, "[ERROR] tokenizer.json: the model was deleted")
	require.NotContains(t, log, "attempt 1 of 3 failed")
}

func TestModelCacheJob_FailedPullCreatesNoModel(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.expectLookups()
	f.gitRepo.EXPECT().PullFromRemote(gomock.Any(), gomock.Any()).Return(errors.New("repository not found"))

	saved, err := f.run()
	require.ErrorContains(t, err, "repository not found")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
	require.False(t, f.modelExists)
	require.Contains(t, f.logs.log(7), "[ERROR] repository not found")
}

func TestModelCacheJob_RequiresAProxyProject(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.projectRepo.EXPECT().GetProjectByName(gomock.Any(), "proxy").
		Return(&projectdomain.Project{ID: 11, Name: "proxy"}, nil)

	saved, err := f.run()
	require.ErrorIs(t, err, syncjob.ErrNotProxyProject)
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
}

func TestModelCacheJob_StopsWhenCanceled(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.setOptions(syncjob.ModelCacheOptions{
		PollInterval: time.Millisecond,
		StallTimeout: time.Hour,
	})
	f.lfs.behaviors[oidA] = stalls
	f.expectRun()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var (
		saved *syncjob.SyncJob
		err   error
		done  = make(chan struct{})
	)
	go func() {
		defer close(done)
		saved, err = f.runWithContext(ctx)
	}()
	require.Eventually(t, func() bool { return f.lfs.exists(ctx, oidB) }, 5*time.Second, time.Millisecond)
	cancel()
	<-done

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, syncjob.SyncJobStatusStopped, saved.Status)
}

func TestModelCacheJob_ReportsTheTimeLimit(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.setOptions(syncjob.ModelCacheOptions{
		PollInterval: time.Millisecond,
		StallTimeout: time.Hour,
	})
	f.lfs.behaviors[oidA] = stalls
	f.expectRun()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	saved, err := f.runWithContext(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "the job reached its time limit")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
}

func TestExecuteSyncJobWithLog_RejectsAnUnknownJobType(t *testing.T) {
	f := newCacheJobFixture(t, "")
	f.job.JobType = "garbage_collection"

	saved, err := f.run()
	require.ErrorContains(t, err, "unknown job type: garbage_collection")
	require.Equal(t, syncjob.SyncJobStatusFailed, saved.Status)
}

func TestExecuteSyncJobWithLog_ReportsOnlyToASyncTask(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		syncTaskID *int
		reported   []int
	}{
		{name: "a job of a sync task", syncTaskID: lo.ToPtr(5), reported: []int{5}},
		{name: "a job without a sync task", syncTaskID: nil, reported: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, svc := newPullFixture(t)
			// A job that was stopped before it started ends without running.
			f.syncJobRepo.EXPECT().GetSyncJob(ctx, 8).Return(&syncjob.SyncJob{
				ID:         8,
				Status:     syncjob.SyncJobStatusStopped,
				SyncTaskID: tc.syncTaskID,
			}, nil)
			f.syncJobRepo.EXPECT().UpdateSyncJob(ctx, gomock.Any()).Return(nil)
			var reported []int
			svc.SetOnJobDone(func(_ context.Context, taskID int) error {
				reported = append(reported, taskID)
				return nil
			})

			require.NoError(t, svc.ExecuteSyncJobWithLog(ctx, 8))
			require.Equal(t, tc.reported, reported)
		})
	}
}
