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

package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	datasetmocks "github.com/matrixhub-ai/matrixhub/internal/domain/dataset/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

const (
	datasetHandlerTestProject = "demo"
	datasetHandlerTestName    = "imdb"
)

func datasetHandlerTestSetup(t *testing.T) (*DatasetHandler, *datasetmocks.MockIDatasetService) {
	t.Helper()
	svc := datasetmocks.NewMockIDatasetService(gomock.NewController(t))
	h, ok := NewDatasetHandler(svc).(*DatasetHandler)
	require.True(t, ok)
	return h, svc
}

func datasetHandlerTestDataset() *dataset.Dataset {
	return &dataset.Dataset{
		ID:            11,
		Name:          datasetHandlerTestName,
		ProjectID:     3,
		ProjectName:   datasetHandlerTestProject,
		DefaultBranch: "main",
		NumRows:       "50000",
		Size:          84_125_825,
		ReadmeContent: "# IMDB",
		IsPopular:     true,
		Labels: []model.Label{
			{ID: 1, Name: "text-classification", Category: "task", Scope: "dataset", CreatedAt: handlerTestCreatedAt, UpdatedAt: handlerTestUpdatedAt},
			{ID: 2, Name: "pandas", Category: "library", Scope: "dataset", CreatedAt: handlerTestCreatedAt, UpdatedAt: handlerTestUpdatedAt},
		},
		CreatedAt: handlerTestCreatedAt,
		UpdatedAt: handlerTestUpdatedAt,
	}
}

func datasetHandlerTestDatasetProto() *v1alpha1.Dataset {
	return &v1alpha1.Dataset{
		Id:            11,
		Project:       datasetHandlerTestProject,
		Name:          datasetHandlerTestName,
		DefaultBranch: "main",
		NumRows:       "50000",
		Labels: []*v1alpha1.Label{
			{Id: 1, Name: "text-classification", Category: v1alpha1.Category_TASK, CreatedAt: handlerTestCreatedAtText, UpdatedAt: handlerTestUpdatedAtText},
			{Id: 2, Name: "pandas", Category: v1alpha1.Category_LIBRARY, CreatedAt: handlerTestCreatedAtText, UpdatedAt: handlerTestUpdatedAtText},
		},
		Size:          "84125825",
		CloneUrls:     &v1alpha1.CloneUrls{},
		ReadmeContent: "# IMDB",
		CreatedAt:     handlerTestCreatedAtText,
		UpdatedAt:     handlerTestUpdatedAtText,
	}
}

func TestNewDatasetHandler(t *testing.T) {
	svc := datasetmocks.NewMockIDatasetService(gomock.NewController(t))

	h, ok := NewDatasetHandler(svc).(*DatasetHandler)
	require.True(t, ok)
	require.Same(t, svc, h.ds)
}

func TestDatasetToProto(t *testing.T) {
	t.Run("maps every exposed field", func(t *testing.T) {
		requireProtoEqual(t, datasetHandlerTestDatasetProto(), datasetToProto(datasetHandlerTestDataset()))
	})

	t.Run("keeps optional fields empty", func(t *testing.T) {
		got := datasetToProto(&dataset.Dataset{
			ID:          12,
			Name:        "bare",
			ProjectName: datasetHandlerTestProject,
			CreatedAt:   handlerTestCreatedAt,
			UpdatedAt:   handlerTestUpdatedAt,
		})

		requireProtoEqual(t, &v1alpha1.Dataset{
			Id:        12,
			Project:   datasetHandlerTestProject,
			Name:      "bare",
			Size:      "0",
			CloneUrls: &v1alpha1.CloneUrls{},
			CreatedAt: handlerTestCreatedAtText,
			UpdatedAt: handlerTestUpdatedAtText,
		}, got)
		require.Empty(t, got.Labels)
	})
}

func TestDatasetTreeEntryToProtoFile(t *testing.T) {
	entries := handlerTestTreeEntries()
	files := handlerTestTreeFiles()

	requireProtoEqual(t, files[1], datasetTreeEntryToProtoFile(entries[1]))

	dir := datasetTreeEntryToProtoFile(entries[0])
	requireProtoEqual(t, files[0], dir)
	require.Nil(t, dir.Commit)
}

func TestDatasetHandlerListDatasetTaskLabels(t *testing.T) {
	ctx := context.Background()

	t.Run("returns task labels", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().ListDatasetTaskLabels(ctx).Return([]*model.Label{
			{ID: 1, Name: "text-classification", Category: "task", Scope: "dataset", CreatedAt: handlerTestCreatedAt, UpdatedAt: handlerTestUpdatedAt},
			{ID: 5, Name: "translation", Category: "task", Scope: "dataset", CreatedAt: handlerTestCreatedAt, UpdatedAt: handlerTestUpdatedAt},
		}, nil)

		got, err := h.ListDatasetTaskLabels(ctx, &v1alpha1.ListDatasetTaskLabelsRequest{})
		require.NoError(t, err)
		requireProtoEqual(t, &v1alpha1.ListDatasetTaskLabelsResponse{Items: []*v1alpha1.Label{
			{Id: 1, Name: "text-classification", Category: v1alpha1.Category_TASK, CreatedAt: handlerTestCreatedAtText, UpdatedAt: handlerTestUpdatedAtText},
			{Id: 5, Name: "translation", Category: v1alpha1.Category_TASK, CreatedAt: handlerTestCreatedAtText, UpdatedAt: handlerTestUpdatedAtText},
		}}, got)
	})

	t.Run("hides service errors", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().ListDatasetTaskLabels(ctx).Return(nil, errors.New("database is locked"))

		got, err := h.ListDatasetTaskLabels(ctx, &v1alpha1.ListDatasetTaskLabelsRequest{})
		require.Nil(t, got)
		requireHandlerTestStatus(t, err, codes.Internal, "failed to list task labels")
	})
}

func TestDatasetHandlerListDatasets(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.ListDatasetsRequest{
		Label:    []string{"text-classification"},
		Search:   "im",
		Sort:     "updated_at",
		Project:  datasetHandlerTestProject,
		Page:     2,
		PageSize: 5,
		Popular:  true,
	}

	t.Run("forwards filters and paginates the result", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().ListDatasets(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, filter *model.Filter) ([]*dataset.Dataset, int64, error) {
				require.Equal(t, []string{"text-classification"}, filter.Label)
				require.Equal(t, "im", filter.Search)
				require.Equal(t, "updated_at", filter.Sort)
				require.Equal(t, datasetHandlerTestProject, filter.Project)
				require.Equal(t, int32(2), filter.Page)
				require.Equal(t, int32(5), filter.PageSize)
				require.NotNil(t, filter.Popular)
				require.True(t, *filter.Popular)
				return []*dataset.Dataset{datasetHandlerTestDataset()}, 12, nil
			})

		got, err := h.ListDatasets(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, &v1alpha1.ListDatasetsResponse{
			Items:      []*v1alpha1.Dataset{datasetHandlerTestDatasetProto()},
			Pagination: &v1alpha1.Pagination{Total: 12, Page: 2, PageSize: 5, Pages: 3},
		}, got)
	})

	t.Run("passes a false popular flag through", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().ListDatasets(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, filter *model.Filter) ([]*dataset.Dataset, int64, error) {
				require.NotNil(t, filter.Popular)
				require.False(t, *filter.Popular)
				return nil, 0, nil
			})

		got, err := h.ListDatasets(ctx, &v1alpha1.ListDatasetsRequest{Page: 1, PageSize: 20})
		require.NoError(t, err)
		require.Empty(t, got.Items)
		requireProtoEqual(t, &v1alpha1.Pagination{Total: 0, Page: 1, PageSize: 20, Pages: 0}, got.Pagination)
	})

	t.Run("hides service errors", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().ListDatasets(ctx, gomock.Any()).Return(nil, int64(0), errors.New("database is locked"))

		got, err := h.ListDatasets(ctx, request)
		require.Nil(t, got)
		requireHandlerTestStatus(t, err, codes.Internal, "failed to list datasets")
	})
}

func TestDatasetHandlerRejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name  string
		field string
		call  func(context.Context, *DatasetHandler) error
	}{
		{
			name:  "GetDataset without project",
			field: "GetDatasetRequest.Project",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.GetDataset(ctx, &v1alpha1.GetDatasetRequest{Name: datasetHandlerTestName})
				return err
			},
		},
		{
			name:  "GetDataset with reserved name",
			field: "GetDatasetRequest.Name",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.GetDataset(ctx, &v1alpha1.GetDatasetRequest{Project: datasetHandlerTestProject, Name: "datasets"})
				return err
			},
		},
		{
			name:  "CreateDataset without name",
			field: "CreateDatasetRequest.Name",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.CreateDataset(ctx, &v1alpha1.CreateDatasetRequest{Project: datasetHandlerTestProject})
				return err
			},
		},
		{
			name:  "CreateDataset with reserved name",
			field: "CreateDatasetRequest.Name",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.CreateDataset(ctx, &v1alpha1.CreateDatasetRequest{Project: datasetHandlerTestProject, Name: "models"})
				return err
			},
		},
		{
			name:  "DeleteDataset with reserved name",
			field: "DeleteDatasetRequest.Name",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.DeleteDataset(ctx, &v1alpha1.DeleteDatasetRequest{Project: datasetHandlerTestProject, Name: "resolve"})
				return err
			},
		},
		{
			name:  "ListDatasetRevisions without project",
			field: "ListDatasetRevisionsRequest.Project",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.ListDatasetRevisions(ctx, &v1alpha1.ListDatasetRevisionsRequest{Name: datasetHandlerTestName})
				return err
			},
		},
		{
			name:  "ListDatasetCommits without name",
			field: "ListDatasetCommitsRequest.Name",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.ListDatasetCommits(ctx, &v1alpha1.ListDatasetCommitsRequest{Project: datasetHandlerTestProject})
				return err
			},
		},
		{
			name:  "GetDatasetCommit without commit id",
			field: "GetDatasetCommitRequest.Id",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.GetDatasetCommit(ctx, &v1alpha1.GetDatasetCommitRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName})
				return err
			},
		},
		{
			name:  "GetDatasetTree with reserved name",
			field: "GetDatasetTreeRequest.Name",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.GetDatasetTree(ctx, &v1alpha1.GetDatasetTreeRequest{Project: datasetHandlerTestProject, Name: "spaces"})
				return err
			},
		},
		{
			name:  "GetDatasetBlob without project",
			field: "GetDatasetBlobRequest.Project",
			call: func(ctx context.Context, h *DatasetHandler) error {
				_, err := h.GetDatasetBlob(ctx, &v1alpha1.GetDatasetBlobRequest{Name: datasetHandlerTestName})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// No mock expectations: validation must fail before the service is called.
			h, _ := datasetHandlerTestSetup(t)

			err := tc.call(context.Background(), h)
			require.Equal(t, codes.InvalidArgument, status.Code(err), "unexpected error: %v", err)
			require.Contains(t, status.Convert(err).Message(), tc.field)
		})
	}
}

func TestDatasetHandlerCreateDataset(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.CreateDatasetRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName}

	t.Run("creates the dataset", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().CreateDataset(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(&dataset.Dataset{ID: 11}, nil)

		got, err := h.CreateDataset(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "duplicate dataset", err: errors.New("dataset already exists"), wantCode: codes.AlreadyExists},
		{name: "missing project", err: errors.New("project not found"), wantCode: codes.NotFound},
		{name: "repository failure", err: errors.New("disk full"), wantCode: codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().CreateDataset(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(nil, tc.err)

			got, err := h.CreateDataset(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerDeleteDataset(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.DeleteDatasetRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName}

	t.Run("deletes the dataset", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().DeleteDataset(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(nil)

		got, err := h.DeleteDataset(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing dataset", err: errors.New("record not found"), wantCode: codes.NotFound},
		{name: "repository failure", err: errors.New("permission denied by filesystem"), wantCode: codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().DeleteDataset(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(tc.err)

			got, err := h.DeleteDataset(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerGetDataset(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.GetDatasetRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName}

	t.Run("returns the dataset", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().GetDataset(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(datasetHandlerTestDataset(), nil)

		got, err := h.GetDataset(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, datasetHandlerTestDatasetProto(), got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing dataset", err: errors.New("record not found"), wantCode: codes.NotFound},
		{name: "storage failure", err: errors.New("connection refused"), wantCode: codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().GetDataset(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(nil, tc.err)

			got, err := h.GetDataset(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerListDatasetRevisions(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.ListDatasetRevisionsRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName}

	t.Run("returns branches and tags", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().ListDatasetRevisions(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(&git.Revisions{
			Branches: []*git.Revision{{Name: "main"}},
			Tags:     []*git.Revision{{Name: "v1"}, {Name: "v2"}},
		}, nil)

		got, err := h.ListDatasetRevisions(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, &v1alpha1.ListDatasetRevisionsResponse{Items: &v1alpha1.Revisions{
			Branches: []*v1alpha1.Revision{{Name: "main"}},
			Tags:     []*v1alpha1.Revision{{Name: "v1"}, {Name: "v2"}},
		}}, got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing dataset record", err: errors.New("record not found"), wantCode: codes.NotFound, wantMsg: "dataset imdb not found in project demo"},
		{name: "missing git repository", err: errors.New("repository does not exist at /data/datasets/demo/imdb"), wantCode: codes.NotFound, wantMsg: "dataset imdb not found in project demo"},
		{name: "git failure", err: errors.New("bad object HEAD"), wantCode: codes.Internal, wantMsg: "failed to list revisions: bad object HEAD"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().ListDatasetRevisions(ctx, datasetHandlerTestProject, datasetHandlerTestName).Return(nil, tc.err)

			got, err := h.ListDatasetRevisions(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerListDatasetCommits(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.ListDatasetCommitsRequest{
		Project:  datasetHandlerTestProject,
		Name:     datasetHandlerTestName,
		Revision: "main",
		Page:     3,
		PageSize: 4,
	}

	t.Run("returns a page of commits", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().
			ListDatasetCommits(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", 3, 4).
			Return([]*git.Commit{handlerTestCommit()}, int64(9), nil)

		got, err := h.ListDatasetCommits(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, &v1alpha1.ListDatasetCommitsResponse{
			Items:      []*v1alpha1.Commit{handlerTestCommitProto()},
			Pagination: &v1alpha1.Pagination{Total: 9, Page: 3, PageSize: 4, Pages: 3},
		}, got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing dataset", err: errors.New("record not found"), wantCode: codes.NotFound},
		{name: "git failure", err: errors.New("bad object HEAD"), wantCode: codes.Internal, wantMsg: "failed to list commits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().
				ListDatasetCommits(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", 3, 4).
				Return(nil, int64(0), tc.err)

			got, err := h.ListDatasetCommits(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerGetDatasetCommit(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.GetDatasetCommitRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName, Id: "8f3c2a1"}

	t.Run("returns the commit", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().GetDatasetCommit(ctx, datasetHandlerTestProject, datasetHandlerTestName, "8f3c2a1").Return(handlerTestCommit(), nil)

		got, err := h.GetDatasetCommit(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, handlerTestCommitProto(), got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing commit", err: errors.New("object not found"), wantCode: codes.NotFound},
		{name: "git failure", err: errors.New("packfile corrupted"), wantCode: codes.Internal, wantMsg: "failed to get commit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().GetDatasetCommit(ctx, datasetHandlerTestProject, datasetHandlerTestName, "8f3c2a1").Return(nil, tc.err)

			got, err := h.GetDatasetCommit(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerGetDatasetTree(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.GetDatasetTreeRequest{Project: datasetHandlerTestProject, Name: datasetHandlerTestName, Revision: "main", Path: "data"}

	t.Run("returns tree entries", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().
			GetDatasetTree(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", "data").
			Return(handlerTestTreeEntries(), nil)

		got, err := h.GetDatasetTree(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, &v1alpha1.GetDatasetTreeResponse{Items: handlerTestTreeFiles()}, got)
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing path", err: errors.New("directory not found"), wantCode: codes.NotFound},
		{name: "git failure", err: errors.New("packfile corrupted"), wantCode: codes.Internal, wantMsg: "failed to get tree"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().GetDatasetTree(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", "data").Return(nil, tc.err)

			got, err := h.GetDatasetTree(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}

func TestDatasetHandlerGetDatasetBlob(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.GetDatasetBlobRequest{
		Project:  datasetHandlerTestProject,
		Name:     datasetHandlerTestName,
		Revision: "main",
		Path:     "data/shard-00000.bin",
	}

	t.Run("returns file metadata", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().
			GetDatasetBlob(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", "data/shard-00000.bin").
			Return(handlerTestTreeEntries()[1], nil)

		got, err := h.GetDatasetBlob(ctx, request)
		require.NoError(t, err)
		requireProtoEqual(t, handlerTestTreeFiles()[1], got)
	})

	t.Run("rejects directories", func(t *testing.T) {
		h, svc := datasetHandlerTestSetup(t)
		svc.EXPECT().
			GetDatasetBlob(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", "data/shard-00000.bin").
			Return(handlerTestTreeEntries()[0], nil)

		got, err := h.GetDatasetBlob(ctx, request)
		require.Nil(t, got)
		requireHandlerTestStatus(t, err, codes.InvalidArgument, "path must reference a file blob")
	})

	for _, tc := range []handlerTestErrorCase{
		{name: "missing file", err: errors.New("file or directory not found at data/shard-00000.bin"), wantCode: codes.NotFound},
		{name: "missing git repository", err: errors.New("repository does not exist"), wantCode: codes.NotFound},
		{name: "git failure", err: errors.New("packfile corrupted"), wantCode: codes.Internal, wantMsg: "failed to get blob"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, svc := datasetHandlerTestSetup(t)
			svc.EXPECT().
				GetDatasetBlob(ctx, datasetHandlerTestProject, datasetHandlerTestName, "main", "data/shard-00000.bin").
				Return(nil, tc.err)

			got, err := h.GetDatasetBlob(ctx, request)
			require.Nil(t, got)
			requireHandlerTestStatus(t, err, tc.wantCode, tc.wantMsg)
		})
	}
}
