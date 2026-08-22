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
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	datasetdomain "github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	datasetmocks "github.com/matrixhub-ai/matrixhub/internal/domain/dataset/mocks"
	gitdomain "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
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
