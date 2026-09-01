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
