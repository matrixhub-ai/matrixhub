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

package model_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	registrymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registry/mocks"
)

func newProjectRepoMock(ctrl *gomock.Controller) *projectmocks.MockIProjectRepo {
	repo := projectmocks.NewMockIProjectRepo(ctrl)
	repo.EXPECT().
		GetProjectByName(gomock.Any(), "proj").
		Return(&project.Project{Name: "proj"}, nil)
	return repo
}

func newProxyRepoMocks(ctrl *gomock.Controller) (*projectmocks.MockIProjectRepo, *registrymocks.MockIRegistryRepo) {
	registryID := 1
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	projectRepo.EXPECT().
		GetProjectByName(gomock.Any(), "proj").
		Return(&project.Project{
			Name:         "proj",
			RegistryID:   &registryID,
			Organization: "upstream-org",
		}, nil)

	registryRepo := registrymocks.NewMockIRegistryRepo(ctrl)
	registryRepo.EXPECT().
		GetRegistry(gomock.Any(), registryID).
		Return(&registry.Registry{ID: registryID, URL: "https://huggingface.co"}, nil)
	return projectRepo, registryRepo
}

func TestModelService_CreateModelCreatesRecordWhenRepositoryAlreadyExists(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	wantErr := errors.New("model not found")
	want := &model.Model{Name: "model", ProjectName: "proj"}

	gomock.InOrder(
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(nil, wantErr),
		gitRepo.EXPECT().
			RepositoryExists(ctx, "models", "proj", "model").
			Return(true, nil),
		modelRepo.EXPECT().
			Create(ctx, &model.Model{Name: "model", ProjectName: "proj"}).
			Return(want, nil),
	)

	service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)
	got, err := service.CreateModel(ctx, "proj", "model")

	require.NoError(t, err)
	require.Same(t, want, got)
}

func TestModelService_EnsureModelReturnsExistingModel(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	want := &model.Model{Name: "model", ProjectName: "proj"}
	modelRepo.EXPECT().
		GetByProjectAndName(ctx, "proj", "model").
		Return(want, nil)

	service := model.NewModelService(modelRepo, nil, gitRepo, newProjectRepoMock(ctrl), nil)
	got, err := service.EnsureModel(ctx, "proj", "model")

	require.NoError(t, err)
	require.Same(t, want, got)
}
func TestModelService_EnsureModelCreatesRepoAndModelWhenBothMissing(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	wantErr := errors.New("model not found")
	want := &model.Model{Name: "model", ProjectName: "proj"}

	gomock.InOrder(
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(nil, wantErr),
		gitRepo.EXPECT().
			RepositoryExists(ctx, "models", "proj", "model").
			Return(false, nil),
		gitRepo.EXPECT().
			CreateRepository(ctx, "models", "proj", "model").
			Return(nil),
		modelRepo.EXPECT().
			Create(ctx, &model.Model{Name: "model", ProjectName: "proj"}).
			Return(want, nil),
	)

	service := model.NewModelService(modelRepo, nil, gitRepo, newProjectRepoMock(ctrl), nil)
	got, err := service.EnsureModel(ctx, "proj", "model")

	require.NoError(t, err)
	require.Same(t, want, got)
}

func TestModelService_EnsureModelCreatesOnlyModelWhenRepoExists(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	wantErr := errors.New("model not found")
	want := &model.Model{Name: "model", ProjectName: "proj"}

	gomock.InOrder(
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(nil, wantErr),
		gitRepo.EXPECT().
			RepositoryExists(ctx, "models", "proj", "model").
			Return(true, nil),
		modelRepo.EXPECT().
			Create(ctx, &model.Model{Name: "model", ProjectName: "proj"}).
			Return(want, nil),
	)

	service := model.NewModelService(modelRepo, nil, gitRepo, newProjectRepoMock(ctrl), nil)
	got, err := service.EnsureModel(ctx, "proj", "model")

	require.NoError(t, err)
	require.Same(t, want, got)
}

func TestModelService_EnsureModelPropagatesModelLookupError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	wantErr := errors.New("database unavailable")

	modelRepo.EXPECT().
		GetByProjectAndName(ctx, "proj", "model").
		Return(nil, wantErr)

	service := model.NewModelService(modelRepo, nil, gitRepo, newProjectRepoMock(ctrl), nil)
	got, err := service.EnsureModel(ctx, "proj", "model")

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
}

func TestModelService_CheckOrSyncFromRemoteReturnsForNonProxyProject(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	projectRepo.EXPECT().
		GetProjectByName(ctx, "proj").
		Return(&project.Project{Name: "proj"}, nil)

	service := model.NewModelService(
		modelRepo,
		nil,
		gitRepo,
		projectRepo,
		nil,
	)

	err := service.CheckOrSyncFromRemote(ctx, "proj", "model")

	require.NoError(t, err)
}

func TestModelService_CheckOrSyncFromRemoteSkipsFreshExistingModel(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	registryID := 1
	syncedAt := time.Now()
	projectRepo.EXPECT().
		GetProjectByName(ctx, "proj").
		Return(&project.Project{
			Name:         "proj",
			RegistryID:   &registryID,
			Organization: "upstream-org",
		}, nil)

	modelRepo.EXPECT().
		GetByProjectAndName(ctx, "proj", "model").
		Return(&model.Model{
			Name:        "model",
			ProjectName: "proj",
			SyncedAt:    &syncedAt,
		}, nil)

	service := model.NewModelService(
		modelRepo,
		nil,
		gitRepo,
		projectRepo,
		nil,
	)

	err := service.CheckOrSyncFromRemote(ctx, "proj", "model")

	require.NoError(t, err)
}

func TestModelService_CheckOrSyncFromRemoteSyncsNewModelImmediately(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo, registryRepo := newProxyRepoMocks(ctrl)
	pullErr := errors.New("pull attempted")
	createdModel := &model.Model{
		ID:          42,
		Name:        "model",
		ProjectName: "proj",
	}

	gomock.InOrder(
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(nil, errors.New("model not found")),
		gitRepo.EXPECT().
			RepositoryExists(ctx, "models", "proj", "model").
			Return(false, nil),
		gitRepo.EXPECT().
			CreateRepository(ctx, "models", "proj", "model").
			Return(nil),
		modelRepo.EXPECT().
			Create(ctx, &model.Model{Name: "model", ProjectName: "proj"}).
			Return(createdModel, nil),
		gitRepo.EXPECT().
			PullFromRemote(ctx, &git.GitRepository{
				RemoteRegistryURL:  "https://huggingface.co",
				RemoteProjectName:  "upstream-org",
				RemoteResourceName: "model",
				ProjectName:        "proj",
				ResourceName:       "model",
				ResourceType:       "model",
			}).
			Return(pullErr),
	)

	service := model.NewModelService(
		modelRepo,
		nil,
		gitRepo,
		projectRepo,
		registryRepo,
	)

	err := service.CheckOrSyncFromRemote(ctx, "proj", "model")

	require.ErrorIs(t, err, pullErr)
}

func TestModelService_CheckOrSyncFromRemoteSyncsStaleExistingModel(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo, registryRepo := newProxyRepoMocks(ctrl)
	pullErr := errors.New("pull attempted")
	syncedAt := time.Now().Add(-2 * time.Minute)

	modelRepo.EXPECT().
		GetByProjectAndName(ctx, "proj", "model").
		Return(&model.Model{
			ID:          42,
			Name:        "model",
			ProjectName: "proj",
			SyncedAt:    &syncedAt,
		}, nil)
	gitRepo.EXPECT().
		PullFromRemote(ctx, &git.GitRepository{
			RemoteRegistryURL:  "https://huggingface.co",
			RemoteProjectName:  "upstream-org",
			RemoteResourceName: "model",
			ProjectName:        "proj",
			ResourceName:       "model",
			ResourceType:       "model",
		}).
		Return(pullErr)

	service := model.NewModelService(
		modelRepo,
		nil,
		gitRepo,
		projectRepo,
		registryRepo,
	)

	err := service.CheckOrSyncFromRemote(ctx, "proj", "model")

	require.ErrorIs(t, err, pullErr)
}

func TestModelService_CheckOrSyncFromRemoteRecordsSuccessfulSync(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo, registryRepo := newProxyRepoMocks(ctrl)
	mod := &model.Model{ID: 42, Name: "model", ProjectName: "proj"}
	syncStartedAt := time.Now()

	gomock.InOrder(
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(mod, nil),
		gitRepo.EXPECT().
			PullFromRemote(ctx, &git.GitRepository{
				RemoteRegistryURL:  "https://huggingface.co",
				RemoteProjectName:  "upstream-org",
				RemoteResourceName: "model",
				ProjectName:        "proj",
				ResourceName:       "model",
				ResourceType:       "model",
			}).
			Return(nil),
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(mod, nil),
		gitRepo.EXPECT().
			ExtractMetadata(ctx, "models", "proj", "model").
			Return(&git.RepoMetadataFiles{}, nil),
		modelRepo.EXPECT().
			UpdateMetadata(ctx, int64(42), gomock.Any()).
			Return(nil),
		labelRepo.EXPECT().
			UpdateModelLabels(ctx, int64(42), []int(nil)).
			Return(nil),
		modelRepo.EXPECT().
			UpdateSyncedAt(ctx, int64(42), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, syncedAt *time.Time) error {
				require.NotNil(t, syncedAt)
				require.False(t, syncedAt.Before(syncStartedAt))
				return nil
			}),
	)

	service := model.NewModelService(modelRepo, labelRepo, gitRepo, projectRepo, registryRepo)
	err := service.CheckOrSyncFromRemote(ctx, "proj", "model")

	require.NoError(t, err)
}

func TestModelService_EnsureModelPropagatesProjectLookupError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	wantErr := errors.New("project not found")
	projectRepo.EXPECT().
		GetProjectByName(ctx, "proj").
		Return(nil, wantErr)

	service := model.NewModelService(
		modelRepo,
		nil,
		gitRepo,
		projectRepo,
		nil,
	)

	got, err := service.EnsureModel(ctx, "proj", "model")

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
}
