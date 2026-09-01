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

func TestModelService_SyncMetadataPersistsZeroValues(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	modelRepo.EXPECT().GetByProjectAndName(gomock.Any(), "proj", "empty").
		Return(&model.Model{ID: 42, Name: "empty", ProjectName: "proj"}, nil)
	gitRepo.EXPECT().ExtractMetadata(gomock.Any(), "models", "proj", "empty").
		Return(&git.RepoMetadataFiles{ReadmeContent: []byte("# Empty")}, nil)
	modelRepo.EXPECT().UpdateMetadata(gomock.Any(), int64(42), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, update *model.MetadataUpdate) error {
			require.NotNil(t, update.Size)
			require.Zero(t, *update.Size)
			require.NotNil(t, update.ParameterCount)
			require.Zero(t, *update.ParameterCount)
			return nil
		})
	labelRepo.EXPECT().UpdateModelLabels(gomock.Any(), int64(42), []int(nil)).Return(nil)

	service := model.NewModelService(modelRepo, labelRepo, gitRepo, nil, nil)
	require.NoError(t, service.SyncMetadata(ctx, "proj", "empty"))
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

func TestModelService_CreateModelValidatesPath(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		project string
		model   string
	}{
		{name: "empty project", model: "model"},
		{name: "empty name", project: "proj"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := model.NewModelService(
				modelmocks.NewMockIModelRepo(ctrl),
				nil,
				gitmocks.NewMockIGitRepo(ctrl),
				nil,
				nil,
			)

			got, err := service.CreateModel(ctx, tc.project, tc.model)

			require.Error(t, err)
			require.Nil(t, got)
		})
	}
}

func TestModelService_GetModelValidatesAndDelegates(t *testing.T) {
	ctx := context.Background()

	t.Run("returns invalid input errors before repository access", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := model.NewModelService(modelmocks.NewMockIModelRepo(ctrl), nil, nil, nil, nil)

		got, err := service.GetModel(ctx, "", "model")

		require.Error(t, err)
		require.Nil(t, got)
	})

	t.Run("returns the repository model", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		want := &model.Model{ID: 1, ProjectName: "proj", Name: "model"}
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(want, nil)
		service := model.NewModelService(modelRepo, nil, nil, nil, nil)

		got, err := service.GetModel(ctx, "proj", "model")

		require.NoError(t, err)
		require.Same(t, want, got)
	})
}

func TestModelService_ListModelsUsesDefaultFilter(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	want := []*model.Model{{ID: 1, ProjectName: "proj", Name: "model"}}
	modelRepo.EXPECT().
		List(ctx, gomock.Eq(&model.Filter{Page: 1, PageSize: 20})).
		Return(want, int64(1), nil)
	service := model.NewModelService(modelRepo, nil, nil, nil, nil)

	got, total, err := service.ListModels(ctx, nil)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, want, got)
}

func TestModelService_DeleteModelDeletesRepositoryBeforeRecord(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	gomock.InOrder(
		gitRepo.EXPECT().DeleteRepository(ctx, "models", "proj", "model").Return(nil),
		modelRepo.EXPECT().Delete(ctx, "proj", "model").Return(nil),
	)
	service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)

	err := service.DeleteModel(ctx, "proj", "model")

	require.NoError(t, err)
}

func TestModelService_ListModelLabelsUsesModelScope(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	taskLabels := []*model.Label{{Name: "text-generation"}}
	frameLabels := []*model.Label{{Name: "transformers"}}
	gomock.InOrder(
		labelRepo.EXPECT().ListByCategoryAndScope(ctx, "task", "model").Return(taskLabels, nil),
		labelRepo.EXPECT().ListByCategoryAndScope(ctx, "library", "model").Return(frameLabels, nil),
	)
	service := model.NewModelService(nil, labelRepo, nil, nil, nil)

	gotTask, err := service.ListModelTaskLabels(ctx)
	require.NoError(t, err)
	require.Equal(t, taskLabels, gotTask)

	gotFrame, err := service.ListModelFrameLabels(ctx)
	require.NoError(t, err)
	require.Equal(t, frameLabels, gotFrame)
}

func TestModelService_ListModelRevisionsChecksModelBeforeGit(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	want := &git.Revisions{Branches: []*git.Revision{{Name: "main"}}}
	gomock.InOrder(
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(&model.Model{ID: 1}, nil),
		gitRepo.EXPECT().ListRevisions(ctx, "models", "proj", "model").Return(want, nil),
	)
	service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)

	got, err := service.ListModelRevisions(ctx, "proj", "model")

	require.NoError(t, err)
	require.Same(t, want, got)
}

func TestModelService_ListModelCommitsUsesDefaultPagination(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	want := []*git.Commit{{ID: "commit"}}
	gomock.InOrder(
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(&model.Model{ID: 1}, nil),
		gitRepo.EXPECT().ListCommits(ctx, "models", "proj", "model", "", 1, 20).Return(want, int64(1), nil),
	)
	service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)

	got, total, err := service.ListModelCommits(ctx, "proj", "model", "", 0, 0)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, want, got)
}

func TestModelService_GitReadsCheckModelBeforeDelegating(t *testing.T) {
	ctx := context.Background()

	t.Run("get commit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		gitRepo := gitmocks.NewMockIGitRepo(ctrl)
		want := &git.Commit{ID: "commit"}
		gomock.InOrder(
			modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(&model.Model{ID: 1}, nil),
			gitRepo.EXPECT().GetCommit(ctx, "models", "proj", "model", "commit").Return(want, nil),
		)
		service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)

		got, err := service.GetModelCommit(ctx, "proj", "model", "commit")

		require.NoError(t, err)
		require.Same(t, want, got)
	})

	t.Run("get tree", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		gitRepo := gitmocks.NewMockIGitRepo(ctrl)
		want := []*git.TreeEntry{{Path: "README.md"}}
		gomock.InOrder(
			modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(&model.Model{ID: 1}, nil),
			gitRepo.EXPECT().GetTree(ctx, "models", "proj", "model", "main", "").Return(want, nil),
		)
		service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)

		got, err := service.GetModelTree(ctx, "proj", "model", "main", "")

		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("get blob", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		gitRepo := gitmocks.NewMockIGitRepo(ctrl)
		want := &git.TreeEntry{Path: "README.md"}
		gomock.InOrder(
			modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(&model.Model{ID: 1}, nil),
			gitRepo.EXPECT().GetBlob(ctx, "models", "proj", "model", "main", "README.md").Return(want, nil),
		)
		service := model.NewModelService(modelRepo, nil, gitRepo, nil, nil)

		got, err := service.GetModelBlob(ctx, "proj", "model", "main", "README.md")

		require.NoError(t, err)
		require.Same(t, want, got)
	})
}

func TestModelService_CreateModelCommitCreatesRecordAndSyncsMetadata(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	created := &model.Model{ID: 42, ProjectID: 3, ProjectName: "proj", Name: "model"}
	commit := &git.Commit{Message: "add README"}
	ops := []git.CommitOperation{{Type: git.CommitOperationAdd, Path: "README.md", Content: []byte("# Model")}}

	gomock.InOrder(
		projectRepo.EXPECT().GetProjectByName(ctx, "proj").Return(&project.Project{ID: 3, Name: "proj"}, nil),
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, errors.New("model not found")),
		modelRepo.EXPECT().Create(ctx, &model.Model{Name: "model", ProjectID: 3, ProjectName: "proj"}).Return(created, nil),
		gitRepo.EXPECT().CreateCommit(ctx, "models", "proj", "model", "main", commit, ops).Return("commit", nil),
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(created, nil),
		gitRepo.EXPECT().ExtractMetadata(ctx, "models", "proj", "model").Return(&git.RepoMetadataFiles{}, nil),
		modelRepo.EXPECT().UpdateMetadata(ctx, int64(42), gomock.Any()).Return(nil),
		labelRepo.EXPECT().UpdateModelLabels(ctx, int64(42), []int(nil)).Return(nil),
	)
	service := model.NewModelService(modelRepo, labelRepo, gitRepo, projectRepo, nil)

	commitID, err := service.CreateModelCommit(ctx, "proj", "model", "main", commit, ops)

	require.NoError(t, err)
	require.Equal(t, "commit", commitID)
}

func TestModelService_UpdateModelSettingUsesModelID(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	popular := true
	update := &model.SettingUpdate{IsPopular: &popular}
	gomock.InOrder(
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(&model.Model{ID: 42}, nil),
		modelRepo.EXPECT().UpdateSetting(ctx, int64(42), update).Return(nil),
	)
	service := model.NewModelService(modelRepo, nil, nil, nil, nil)

	err := service.UpdateModelSetting(ctx, "proj", "model", update)

	require.NoError(t, err)
}

func TestModelService_CreateModelStopsAtEachFailedDependency(t *testing.T) {
	ctx := context.Background()
	notFoundErr := errors.New("model not found")

	t.Run("existing model", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		modelRepo.EXPECT().
			GetByProjectAndName(ctx, "proj", "model").
			Return(&model.Model{ID: 1}, nil)

		got, err := model.NewModelService(modelRepo, nil, nil, nil, nil).CreateModel(ctx, "proj", "model")

		require.EqualError(t, err, "model already exists")
		require.Nil(t, got)
	})

	t.Run("repository lookup", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		gitRepo := gitmocks.NewMockIGitRepo(ctrl)
		wantErr := errors.New("git unavailable")
		gomock.InOrder(
			modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, notFoundErr),
			gitRepo.EXPECT().RepositoryExists(ctx, "models", "proj", "model").Return(false, wantErr),
		)

		got, err := model.NewModelService(modelRepo, nil, gitRepo, nil, nil).CreateModel(ctx, "proj", "model")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})

	t.Run("repository creation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		gitRepo := gitmocks.NewMockIGitRepo(ctrl)
		wantErr := errors.New("create repository failed")
		gomock.InOrder(
			modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, notFoundErr),
			gitRepo.EXPECT().RepositoryExists(ctx, "models", "proj", "model").Return(false, nil),
			gitRepo.EXPECT().CreateRepository(ctx, "models", "proj", "model").Return(wantErr),
		)

		got, err := model.NewModelService(modelRepo, nil, gitRepo, nil, nil).CreateModel(ctx, "proj", "model")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})

	t.Run("model record creation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		gitRepo := gitmocks.NewMockIGitRepo(ctrl)
		wantErr := errors.New("insert failed")
		gomock.InOrder(
			modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, notFoundErr),
			gitRepo.EXPECT().RepositoryExists(ctx, "models", "proj", "model").Return(true, nil),
			modelRepo.EXPECT().Create(ctx, &model.Model{Name: "model", ProjectName: "proj"}).Return(nil, wantErr),
		)

		got, err := model.NewModelService(modelRepo, nil, gitRepo, nil, nil).CreateModel(ctx, "proj", "model")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})
}

func TestModelService_DeleteModelDoesNotDeleteRecordWhenGitDeletionFails(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	wantErr := errors.New("delete repository failed")
	gitRepo.EXPECT().DeleteRepository(ctx, "models", "proj", "model").Return(wantErr)

	err := model.NewModelService(modelRepo, nil, gitRepo, nil, nil).DeleteModel(ctx, "proj", "model")

	require.ErrorIs(t, err, wantErr)
}

func TestModelService_ModelReadsReturnLookupErrors(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("model not found")

	t.Run("revisions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, wantErr)

		got, err := model.NewModelService(modelRepo, nil, nil, nil, nil).ListModelRevisions(ctx, "proj", "model")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})

	t.Run("commits", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, wantErr)

		got, total, err := model.NewModelService(modelRepo, nil, nil, nil, nil).ListModelCommits(ctx, "proj", "model", "main", 1, 20)

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
		require.Zero(t, total)
	})

	t.Run("commit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, wantErr)

		got, err := model.NewModelService(modelRepo, nil, nil, nil, nil).GetModelCommit(ctx, "proj", "model", "commit")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})

	t.Run("tree", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, wantErr)

		got, err := model.NewModelService(modelRepo, nil, nil, nil, nil).GetModelTree(ctx, "proj", "model", "main", "")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})

	t.Run("blob", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		modelRepo := modelmocks.NewMockIModelRepo(ctrl)
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(nil, wantErr)

		got, err := model.NewModelService(modelRepo, nil, nil, nil, nil).GetModelBlob(ctx, "proj", "model", "main", "README.md")

		require.ErrorIs(t, err, wantErr)
		require.Nil(t, got)
	})
}

func TestModelService_SyncMetadataCreatesAndAssignsClassifiedLabels(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	mod := &model.Model{ID: 42, Name: "model", ProjectName: "proj"}
	files := &git.RepoMetadataFiles{
		ReadmeContent: []byte(`---
pipeline_tag: text-generation
library_name: transformers
language: en
license: apache-2.0
tags:
  - instruction-tuned
---
# Model
`),
		ConfigJSON:           []byte(`{"torch_dtype":"float16"}`),
		SafetensorsIndexJSON: []byte(`{"metadata":{"total_size":100}}`),
		Size:                 200,
	}

	gomock.InOrder(
		modelRepo.EXPECT().GetByProjectAndName(ctx, "proj", "model").Return(mod, nil),
		gitRepo.EXPECT().ExtractMetadata(ctx, "models", "proj", "model").Return(files, nil),
		modelRepo.EXPECT().UpdateMetadata(ctx, int64(42), gomock.Any()).DoAndReturn(
			func(_ context.Context, _ int64, update *model.MetadataUpdate) error {
				require.Equal(t, string(files.ReadmeContent), *update.ReadmeContent)
				require.EqualValues(t, 200, *update.Size)
				require.EqualValues(t, 50, *update.ParameterCount)
				return nil
			},
		),
		labelRepo.EXPECT().GetOrCreateByName(ctx, "text-generation", "task", "model").Return(&model.Label{ID: 1}, nil),
		labelRepo.EXPECT().GetOrCreateByName(ctx, "transformers", "library", "model").Return(&model.Label{ID: 2}, nil),
		labelRepo.EXPECT().GetOrCreateByName(ctx, "en", "language", "model").Return(&model.Label{ID: 3}, nil),
		labelRepo.EXPECT().GetOrCreateByName(ctx, "apache-2.0", "license", "model").Return(&model.Label{ID: 4}, nil),
		labelRepo.EXPECT().GetOrCreateByName(ctx, "instruction-tuned", "other", "model").Return(&model.Label{ID: 5}, nil),
		labelRepo.EXPECT().UpdateModelLabels(ctx, int64(42), []int{1, 2, 3, 4, 5}).Return(nil),
	)

	err := model.NewModelService(modelRepo, labelRepo, gitRepo, nil, nil).SyncMetadata(ctx, "proj", "model")

	require.NoError(t, err)
}
