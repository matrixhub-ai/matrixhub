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
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	datasetv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
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

// datasetRevisionToProto converts domain git.Revision to proto Revision for datasets
func datasetRevisionToProto(r *git.Revision) *datasetv1alpha1.Revision {
	return &datasetv1alpha1.Revision{
		Name: r.Name,
	}
}

// datasetRevisionsToProto converts domain git.Revisions to proto Revisions for datasets
func datasetRevisionsToProto(revisions *git.Revisions) *datasetv1alpha1.Revisions {
	branches := make([]*datasetv1alpha1.Revision, len(revisions.Branches))
	for i, b := range revisions.Branches {
		branches[i] = datasetRevisionToProto(b)
	}

	tags := make([]*datasetv1alpha1.Revision, len(revisions.Tags))
	for i, t := range revisions.Tags {
		tags[i] = datasetRevisionToProto(t)
	}

	return &datasetv1alpha1.Revisions{
		Branches: branches,
		Tags:     tags,
	}
}

// datasetCommitToProto converts domain git.Commit to proto Commit for datasets
func datasetCommitToProto(c *git.Commit) *datasetv1alpha1.Commit {
	return &datasetv1alpha1.Commit{
		Id:             c.ID,
		Message:        c.Message,
		AuthorName:     c.AuthorName,
		AuthorEmail:    c.AuthorEmail,
		AuthorDate:     c.AuthorDate.Format(time.RFC3339),
		CommitterName:  c.CommitterName,
		CommitterEmail: c.CommitterEmail,
		CommitterDate:  c.CommitterDate.Format(time.RFC3339),
		Diff:           c.Diff,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
	}
}

// datasetTreeEntryToProtoFile converts domain git.TreeEntry to proto File for datasets
func datasetTreeEntryToProtoFile(entry *git.TreeEntry) *datasetv1alpha1.File {
	var protoCommit *datasetv1alpha1.Commit
	if entry.Commit != nil {
		protoCommit = datasetCommitToProto(entry.Commit)
	}

	return &datasetv1alpha1.File{
		Name:   entry.Name,
		Type:   datasetv1alpha1.FileType(entry.Type),
		Path:   entry.Path,
		Size:   entry.Size,
		Lfs:    entry.IsLFS,
		Sha256: entry.Hash,
		Commit: protoCommit,
		Url:    entry.URL,
	}
}

func (h *DatasetHandler) RegisterToServer(options *ServerOptions) {
	datasetv1alpha1.RegisterDatasetsServer(options.GRPCServer, h)
	if err := datasetv1alpha1.RegisterDatasetsHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
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
			Pages:    utils.CalculatePages(total, request.PageSize),
		},
	}, nil
}

func (h *DatasetHandler) CreateDataset(ctx context.Context, request *datasetv1alpha1.CreateDatasetRequest) (*datasetv1alpha1.CreateDatasetResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	_, err := h.ds.CreateDataset(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &datasetv1alpha1.CreateDatasetResponse{}, nil
}

func (h *DatasetHandler) DeleteDataset(ctx context.Context, request *datasetv1alpha1.DeleteDatasetRequest) (*datasetv1alpha1.DeleteDatasetResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	err := h.ds.DeleteDataset(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &datasetv1alpha1.DeleteDatasetResponse{}, nil
}

func (h *DatasetHandler) GetDataset(ctx context.Context, request *datasetv1alpha1.GetDatasetRequest) (*datasetv1alpha1.Dataset, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	dataset, err := h.ds.GetDataset(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return datasetToProto(dataset), nil
}

func (h *DatasetHandler) ListDatasetRevisions(ctx context.Context, request *datasetv1alpha1.ListDatasetRevisionsRequest) (*datasetv1alpha1.ListDatasetRevisionsResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	revisions, err := h.ds.ListDatasetRevisions(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return nil, status.Errorf(codes.NotFound, "dataset %s not found in project %s", request.Name, request.Project)
		}
		return nil, status.Errorf(codes.Internal, "failed to list revisions: %v", err)
	}

	// Convert to proto and return
	return &datasetv1alpha1.ListDatasetRevisionsResponse{
		Items: datasetRevisionsToProto(revisions),
	}, nil
}

func (h *DatasetHandler) ListDatasetCommits(ctx context.Context, request *datasetv1alpha1.ListDatasetCommitsRequest) (*datasetv1alpha1.ListDatasetCommitsResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	commits, total, err := h.ds.ListDatasetCommits(ctx, request.Project, request.Name, request.Revision, int(request.Page), int(request.PageSize))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to list commits")
	}

	// Convert to proto
	items := make([]*datasetv1alpha1.Commit, len(commits))
	for i, c := range commits {
		items[i] = datasetCommitToProto(c)
	}

	return &datasetv1alpha1.ListDatasetCommitsResponse{
		Items: items,
		Pagination: &datasetv1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
			Pages:    utils.CalculatePages(total, request.PageSize),
		},
	}, nil
}

func (h *DatasetHandler) GetDatasetCommit(ctx context.Context, request *datasetv1alpha1.GetDatasetCommitRequest) (*datasetv1alpha1.Commit, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	commit, err := h.ds.GetDatasetCommit(ctx, request.Project, request.Name, request.Id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get commit")
	}

	return datasetCommitToProto(commit), nil
}

func (h *DatasetHandler) GetDatasetTree(ctx context.Context, request *datasetv1alpha1.GetDatasetTreeRequest) (*datasetv1alpha1.GetDatasetTreeResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	entries, err := h.ds.GetDatasetTree(ctx, request.Project, request.Name, request.Revision, request.Path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get tree")
	}

	// Convert to proto
	items := make([]*datasetv1alpha1.File, len(entries))
	for i, entry := range entries {
		items[i] = datasetTreeEntryToProtoFile(entry)
	}

	return &datasetv1alpha1.GetDatasetTreeResponse{
		Items: items,
	}, nil
}

func (h *DatasetHandler) GetDatasetBlob(ctx context.Context, request *datasetv1alpha1.GetDatasetBlobRequest) (*datasetv1alpha1.File, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	entry, err := h.ds.GetDatasetBlob(ctx, request.Project, request.Name, request.Revision, request.Path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get blob")
	}

	if entry.Type == git.FileTypeDir {
		return nil, status.Error(codes.InvalidArgument, "path must reference a file blob")
	}

	return datasetTreeEntryToProtoFile(entry), nil
}
