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
	modeldomain "github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
)

const datasetReadme = `---
license: apache-2.0
task_categories:
- text-classification
---

# Demo dataset
`

func TestDatasetService_SyncMetadata(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	datasetRepo.EXPECT().GetByProjectAndName(gomock.Any(), "proj", "ds").
		Return(&dataset.Dataset{ID: 5, Name: "ds", ProjectName: "proj"}, nil)
	gitRepo.EXPECT().ExtractMetadata(gomock.Any(), "datasets", "proj", "ds").
		Return(&gitdomain.RepoMetadataFiles{
			ReadmeContent: []byte(datasetReadme),
			Size:          1234,
		}, nil)

	datasetRepo.EXPECT().UpdateMetadata(gomock.Any(), int64(5), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, update *dataset.MetadataUpdate) error {
			require.NotNil(t, update.ReadmeContent)
			require.Contains(t, *update.ReadmeContent, "# Demo dataset")
			require.NotNil(t, update.Size)
			require.EqualValues(t, 1234, *update.Size)
			return nil
		})

	// Labels parsed from the dataset card are stored under the dataset scope.
	labelRepo.EXPECT().GetOrCreateByName(gomock.Any(), gomock.Any(), gomock.Any(), "dataset").
		DoAndReturn(func(_ context.Context, name, category, _ string) (*modeldomain.Label, error) {
			return &modeldomain.Label{ID: 1, Name: name, Category: category}, nil
		}).AnyTimes()
	labelRepo.EXPECT().UpdateDatasetLabels(gomock.Any(), int64(5), gomock.Any()).Return(nil)

	svc := dataset.NewDatasetService(datasetRepo, labelRepo, gitRepo)
	require.NoError(t, svc.SyncMetadata(ctx, "proj", "ds"))
}

func TestDatasetService_SyncMetadataPersistsZeroSize(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	labelRepo := modelmocks.NewMockILabelRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	datasetRepo.EXPECT().GetByProjectAndName(gomock.Any(), "proj", "empty").
		Return(&dataset.Dataset{ID: 5, Name: "empty", ProjectName: "proj"}, nil)
	gitRepo.EXPECT().ExtractMetadata(gomock.Any(), "datasets", "proj", "empty").
		Return(&gitdomain.RepoMetadataFiles{ReadmeContent: []byte("# Empty")}, nil)

	datasetRepo.EXPECT().UpdateMetadata(gomock.Any(), int64(5), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, update *dataset.MetadataUpdate) error {
			require.NotNil(t, update.Size)
			require.Zero(t, *update.Size)
			return nil
		})
	labelRepo.EXPECT().UpdateDatasetLabels(gomock.Any(), int64(5), []int(nil)).Return(nil)

	svc := dataset.NewDatasetService(datasetRepo, labelRepo, gitRepo)
	require.NoError(t, svc.SyncMetadata(ctx, "proj", "empty"))
}

func TestDatasetService_SyncMetadataStopsAtFailedDependency(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("dependency unavailable")
	found := &dataset.Dataset{ID: 5, Name: "ds", ProjectName: "proj"}
	// The card yields a license label followed by an "other" label.
	files := &gitdomain.RepoMetadataFiles{ReadmeContent: []byte("---\nlicense: mit\ntags:\n- tabular\n---\n")}

	for _, tc := range []struct {
		name    string
		setup   func(datasetRepo *datasetmocks.MockIDatasetRepo, labelRepo *modelmocks.MockILabelRepo, gitRepo *gitmocks.MockIGitRepo)
		wantErr string
	}{
		{
			name: "dataset lookup",
			setup: func(datasetRepo *datasetmocks.MockIDatasetRepo, _ *modelmocks.MockILabelRepo, _ *gitmocks.MockIGitRepo) {
				datasetRepo.EXPECT().GetByProjectAndName(ctx, "proj", "ds").Return(nil, failure)
			},
			wantErr: "dataset not found: dependency unavailable",
		},
		{
			name: "metadata extraction",
			setup: func(datasetRepo *datasetmocks.MockIDatasetRepo, _ *modelmocks.MockILabelRepo, gitRepo *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					datasetRepo.EXPECT().GetByProjectAndName(ctx, "proj", "ds").Return(found, nil),
					gitRepo.EXPECT().ExtractMetadata(ctx, "datasets", "proj", "ds").Return(nil, failure),
				)
			},
			wantErr: "failed to read metadata files: dependency unavailable",
		},
		{
			name: "metadata update",
			setup: func(datasetRepo *datasetmocks.MockIDatasetRepo, _ *modelmocks.MockILabelRepo, gitRepo *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					datasetRepo.EXPECT().GetByProjectAndName(ctx, "proj", "ds").Return(found, nil),
					gitRepo.EXPECT().ExtractMetadata(ctx, "datasets", "proj", "ds").Return(files, nil),
					datasetRepo.EXPECT().UpdateMetadata(ctx, int64(5), gomock.Any()).Return(failure),
				)
			},
			wantErr: "failed to update dataset metadata: dependency unavailable",
		},
		{
			name: "label creation",
			setup: func(datasetRepo *datasetmocks.MockIDatasetRepo, labelRepo *modelmocks.MockILabelRepo, gitRepo *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					datasetRepo.EXPECT().GetByProjectAndName(ctx, "proj", "ds").Return(found, nil),
					gitRepo.EXPECT().ExtractMetadata(ctx, "datasets", "proj", "ds").Return(files, nil),
					datasetRepo.EXPECT().UpdateMetadata(ctx, int64(5), gomock.Any()).Return(nil),
					labelRepo.EXPECT().GetOrCreateByName(ctx, "mit", "license", "dataset").Return(&modeldomain.Label{ID: 1}, nil),
					labelRepo.EXPECT().GetOrCreateByName(ctx, "tabular", "other", "dataset").Return(nil, failure),
				)
			},
			wantErr: "failed to get/create label tabular (category=other): dependency unavailable",
		},
		{
			name: "label assignment",
			setup: func(datasetRepo *datasetmocks.MockIDatasetRepo, labelRepo *modelmocks.MockILabelRepo, gitRepo *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					datasetRepo.EXPECT().GetByProjectAndName(ctx, "proj", "ds").Return(found, nil),
					gitRepo.EXPECT().ExtractMetadata(ctx, "datasets", "proj", "ds").Return(files, nil),
					datasetRepo.EXPECT().UpdateMetadata(ctx, int64(5), gomock.Any()).Return(nil),
					labelRepo.EXPECT().GetOrCreateByName(ctx, "mit", "license", "dataset").Return(&modeldomain.Label{ID: 1}, nil),
					labelRepo.EXPECT().GetOrCreateByName(ctx, "tabular", "other", "dataset").Return(&modeldomain.Label{ID: 2}, nil),
					labelRepo.EXPECT().UpdateDatasetLabels(ctx, int64(5), []int{1, 2}).Return(failure),
				)
			},
			wantErr: "dependency unavailable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
			labelRepo := modelmocks.NewMockILabelRepo(ctrl)
			gitRepo := gitmocks.NewMockIGitRepo(ctrl)
			tc.setup(datasetRepo, labelRepo, gitRepo)

			err := dataset.NewDatasetService(datasetRepo, labelRepo, gitRepo).SyncMetadata(ctx, "proj", "ds")

			require.ErrorIs(t, err, failure)
			require.EqualError(t, err, tc.wantErr)
		})
	}
}
