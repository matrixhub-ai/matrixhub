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

package dataset_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	datasetmocks "github.com/matrixhub-ai/matrixhub/internal/domain/dataset/mocks"
	gitdomain "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
)

func TestDatasetServiceCreateGetListAndDelete(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	service := dataset.NewDatasetService(datasetRepo, labelRepo, gitRepo)

	require.Error(t, createDataset(service, ctx, "", "dataset"))
	require.Error(t, createDataset(service, ctx, "project", ""))
	datasetRepo.EXPECT().GetByProjectAndName(ctx, "project", "dataset").Return(nil, errors.New("not found"))
	gitRepo.EXPECT().CreateRepository(ctx, "datasets", "project", "dataset").Return(nil)
	datasetRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, value *dataset.Dataset) (*dataset.Dataset, error) {
		require.Equal(t, "project", value.ProjectName)
		require.Equal(t, "dataset", value.Name)
		value.ID = 7
		return value, nil
	})
	created, err := service.CreateDataset(ctx, "project", "dataset")
	require.NoError(t, err)
	require.EqualValues(t, 7, created.ID)

	require.Error(t, getDataset(service, ctx, "", "dataset"))
	require.Error(t, getDataset(service, ctx, "project", ""))
	datasetRepo.EXPECT().GetByProjectAndName(ctx, "project", "dataset").Return(created, nil)
	got, err := service.GetDataset(ctx, "project", "dataset")
	require.NoError(t, err)
	require.Equal(t, created, got)

	datasetRepo.EXPECT().List(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, filter *model.Filter) ([]*dataset.Dataset, int64, error) {
		require.EqualValues(t, 1, filter.Page)
		require.EqualValues(t, 20, filter.PageSize)
		return []*dataset.Dataset{created}, 1, nil
	})
	items, total, err := service.ListDatasets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.EqualValues(t, 1, total)

	require.Error(t, service.DeleteDataset(ctx, "", "dataset"))
	require.Error(t, service.DeleteDataset(ctx, "project", ""))
	gitRepo.EXPECT().DeleteRepository(ctx, "datasets", "project", "dataset").Return(nil)
	datasetRepo.EXPECT().Delete(ctx, "project", "dataset").Return(nil)
	require.NoError(t, service.DeleteDataset(ctx, "project", "dataset"))
}

func createDataset(service dataset.IDatasetService, ctx context.Context, project, name string) error {
	_, err := service.CreateDataset(ctx, project, name)
	return err
}

func getDataset(service dataset.IDatasetService, ctx context.Context, project, name string) error {
	_, err := service.GetDataset(ctx, project, name)
	return err
}

func TestDatasetServiceGitQueries(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	service := dataset.NewDatasetService(datasetRepo, labelRepo, gitRepo)
	found := &dataset.Dataset{ID: 7, ProjectName: "project", Name: "dataset"}

	datasetRepo.EXPECT().GetByProjectAndName(ctx, "project", "dataset").Return(found, nil).Times(4)
	revisions := &gitdomain.Revisions{Branches: []*gitdomain.Revision{{Name: "main"}}}
	gitRepo.EXPECT().ListRevisions(ctx, "datasets", "project", "dataset").Return(revisions, nil)
	gotRevisions, err := service.ListDatasetRevisions(ctx, "project", "dataset")
	require.NoError(t, err)
	require.Equal(t, revisions, gotRevisions)

	commits := []*gitdomain.Commit{{ID: "commit"}}
	gitRepo.EXPECT().ListCommits(ctx, "datasets", "project", "dataset", "main", 1, 20).Return(commits, int64(1), nil)
	gotCommits, total, err := service.ListDatasetCommits(ctx, "project", "dataset", "main", 0, 0)
	require.NoError(t, err)
	require.Equal(t, commits, gotCommits)
	require.EqualValues(t, 1, total)

	commit := &gitdomain.Commit{ID: "commit"}
	gitRepo.EXPECT().GetCommit(ctx, "datasets", "project", "dataset", "commit").Return(commit, nil)
	gotCommit, err := service.GetDatasetCommit(ctx, "project", "dataset", "commit")
	require.NoError(t, err)
	require.Equal(t, commit, gotCommit)

	tree := []*gitdomain.TreeEntry{{Path: "README.md"}}
	gitRepo.EXPECT().GetTree(ctx, "datasets", "project", "dataset", "main", "").Return(tree, nil)
	gotTree, err := service.GetDatasetTree(ctx, "project", "dataset", "main", "")
	require.NoError(t, err)
	require.Equal(t, tree, gotTree)

	datasetRepo.EXPECT().GetByProjectAndName(ctx, "project", "dataset").Return(found, nil)
	blob := &gitdomain.TreeEntry{Path: "README.md"}
	gitRepo.EXPECT().GetBlob(ctx, "datasets", "project", "dataset", "main", "README.md").Return(blob, nil)
	gotBlob, err := service.GetDatasetBlob(ctx, "project", "dataset", "main", "README.md")
	require.NoError(t, err)
	require.Equal(t, blob, gotBlob)

	labelRepo.EXPECT().ListByCategoryAndScope(ctx, "task", "dataset").Return([]*model.Label{{Name: "task"}}, nil)
	labels, err := service.ListDatasetTaskLabels(ctx)
	require.NoError(t, err)
	require.Len(t, labels, 1)
}
