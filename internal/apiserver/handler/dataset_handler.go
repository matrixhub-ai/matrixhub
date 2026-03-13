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
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	datasetv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type DatasetHandler struct {
	ds dataset.IDatasetService
}

func NewDatasetHandler(datasetService dataset.IDatasetService) IHandler {
	handler := &DatasetHandler{
		ds: datasetService,
	}
	return handler
}

// datasetToProto converts domain Dataset to proto Dataset
func datasetToProto(d *dataset.Dataset) *datasetv1alpha1.Dataset {
	labels := make([]*datasetv1alpha1.Label, len(d.Labels))
	for i, l := range d.Labels {
		labels[i] = &datasetv1alpha1.Label{
			Id:        int32(l.ID),
			Name:      l.Name,
			Category:  datasetv1alpha1.Category_TASK, // Default to TASK
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
			UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
		}
		// Map category string to proto enum
		if l.Category == "library" {
			labels[i].Category = datasetv1alpha1.Category_LIBRARY
		}
	}

	return &datasetv1alpha1.Dataset{
		Id:            int32(d.ID),
		Name:          d.Name,
		Project:       d.ProjectName,
		DefaultBranch: d.DefaultBranch,
		NumRows:       d.NumRows,
		CreatedAt:     d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     d.UpdatedAt.Format(time.RFC3339),
		CloneUrls: &datasetv1alpha1.CloneUrls{
			SshUrl:  "",
			HttpUrl: "",
		},
		Labels:        labels,
		Size:          formatSize(d.Size),
		ReadmeContent: d.ReadmeContent,
	}
}

func (h *DatasetHandler) RegisterToServer(options *ServerOptions) {
	datasetv1alpha1.RegisterDatasetsServer(options.GRPCServer, h)
	if err := datasetv1alpha1.RegisterDatasetsHandlerServer(context.Background(), options.GatewayMux, h); err != nil {
		log.Errorf("dataset handler error: %s", err.Error())
	}
}

func (h *DatasetHandler) ListDatasetTaskLabels(ctx context.Context, request *datasetv1alpha1.ListDatasetTaskLabelsRequest) (*datasetv1alpha1.ListDatasetTaskLabelsResponse, error) {
	labels, err := h.ds.ListDatasetTaskLabels(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list task labels")
	}

	items := make([]*datasetv1alpha1.Label, len(labels))
	for i, l := range labels {
		items[i] = &datasetv1alpha1.Label{
			Id:        int32(l.ID),
			Name:      l.Name,
			Category:  datasetv1alpha1.Category_TASK,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
			UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &datasetv1alpha1.ListDatasetTaskLabelsResponse{
		Items: items,
	}, nil
}

func (h *DatasetHandler) ListDatasets(ctx context.Context, request *datasetv1alpha1.ListDatasetsRequest) (*datasetv1alpha1.ListDatasetsResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Build filter
	filter := &model.Filter{
		Label:    request.Label,
		Search:   request.Search,
		Sort:     request.Sort,
		Project:  request.Project,
		Page:     request.Page,
		PageSize: request.PageSize,
		Popular:  &request.Popular, // Pass popular parameter to filter
	}

	// Call service
	datasets, total, err := h.ds.ListDatasets(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list datasets")
	}

	// Convert to proto
	items := make([]*datasetv1alpha1.Dataset, len(datasets))
	for i, d := range datasets {
		items[i] = datasetToProto(d)
	}

	// Build response
	return &datasetv1alpha1.ListDatasetsResponse{
		Items: items,
		Pagination: &datasetv1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
		},
	}, nil
}

func (h *DatasetHandler) GetDataset(ctx context.Context, request *datasetv1alpha1.GetDatasetRequest) (*datasetv1alpha1.Dataset, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) CreateDataset(ctx context.Context, request *datasetv1alpha1.CreateDatasetRequest) (*datasetv1alpha1.CreateDatasetResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) DeleteDataset(ctx context.Context, request *datasetv1alpha1.DeleteDatasetRequest) (*datasetv1alpha1.DeleteDatasetResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) ListDatasetRevisions(ctx context.Context, request *datasetv1alpha1.ListDatasetRevisionsRequest) (*datasetv1alpha1.ListDatasetRevisionsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) ListDatasetCommits(ctx context.Context, request *datasetv1alpha1.ListDatasetCommitsRequest) (*datasetv1alpha1.ListDatasetCommitsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) GetDatasetCommit(ctx context.Context, request *datasetv1alpha1.GetDatasetCommitRequest) (*datasetv1alpha1.Commit, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) GetDatasetTree(ctx context.Context, request *datasetv1alpha1.GetDatasetTreeRequest) (*datasetv1alpha1.GetDatasetTreeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (h *DatasetHandler) GetDatasetBlob(ctx context.Context, request *datasetv1alpha1.GetDatasetBlobRequest) (*datasetv1alpha1.GetDatasetBlobResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}
