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
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	modelv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type ModelHandler struct {
	ms model.IModelService
}

func NewModelHandler(modelService model.IModelService) IHandler {
	handler := &ModelHandler{
		ms: modelService,
	}
	return handler
}

// formatSize converts bytes to string without unit conversion
func formatSize(bytes int64) string {
	return fmt.Sprintf("%d", bytes)
}

// formatParameterCount converts parameter count to string without unit conversion
func formatParameterCount(count int64) string {
	return fmt.Sprintf("%d", count)
}

// modelToProto converts domain Model to proto Model
func modelToProto(m *model.Model) *modelv1alpha1.Model {
	labels := make([]*modelv1alpha1.Label, len(m.Labels))
	for i, l := range m.Labels {
		labels[i] = &modelv1alpha1.Label{
			Id:        int32(l.ID),
			Name:      l.Name,
			Category:  modelv1alpha1.Category_TASK, // Default to TASK
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
			UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
		}
		// Map category string to proto enum
		if l.Category == "library" {
			labels[i].Category = modelv1alpha1.Category_LIBRARY
		}
	}

	return &modelv1alpha1.Model{
		Id:            int32(m.ID),
		Name:          m.Name,
		Nickname:      "", // Not implemented yet
		DefaultBranch: m.DefaultBranch,
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     m.UpdatedAt.Format(time.RFC3339),
		CloneUrls: &modelv1alpha1.CloneUrls{
			SshUrl:  "",
			HttpUrl: "",
		},
		Labels:         labels,
		Project:        m.ProjectName,
		ReadmeContent:  m.ReadmeContent,
		Size:           formatSize(m.Size),
		ParameterCount: formatParameterCount(m.ParameterCount),
	}
}

func (mh *ModelHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	modelv1alpha1.RegisterModelsServer(options.GRPCServer, mh)
	if err := modelv1alpha1.RegisterModelsHandlerServer(context.Background(), options.GatewayMux, mh); err != nil {
		log.Errorf("register model handler error: %s", err.Error())
	}
}

func (mh *ModelHandler) ListModelTaskLabels(ctx context.Context, request *modelv1alpha1.ListModelTaskLabelsRequest) (*modelv1alpha1.ListModelTaskLabelsResponse, error) {
	labels, err := mh.ms.ListModelTaskLabels(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list task labels")
	}

	items := make([]*modelv1alpha1.Label, len(labels))
	for i, l := range labels {
		items[i] = &modelv1alpha1.Label{
			Id:        int32(l.ID),
			Name:      l.Name,
			Category:  modelv1alpha1.Category_TASK,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
			UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &modelv1alpha1.ListModelTaskLabelsResponse{
		Items: items,
	}, nil
}

func (mh *ModelHandler) ListModelFrameLabels(ctx context.Context, request *modelv1alpha1.ListModelFrameLabelsRequest) (*modelv1alpha1.ListModelFrameLabelsResponse, error) {
	labels, err := mh.ms.ListModelFrameLabels(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list library labels")
	}

	items := make([]*modelv1alpha1.Label, len(labels))
	for i, l := range labels {
		items[i] = &modelv1alpha1.Label{
			Id:        int32(l.ID),
			Name:      l.Name,
			Category:  modelv1alpha1.Category_LIBRARY,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
			UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &modelv1alpha1.ListModelFrameLabelsResponse{
		Items: items,
	}, nil
}

func (mh *ModelHandler) ListModels(ctx context.Context, request *modelv1alpha1.ListModelsRequest) (*modelv1alpha1.ListModelsResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Build filter
	filter := &model.Filter{
		Label:    request.Labels,
		Search:   request.Search,
		Sort:     request.Sort,
		Project:  request.Project,
		Page:     request.Page,
		PageSize: request.PageSize,
		Popular:  &request.Popular, // Pass popular parameter to filter
	}

	// Call service
	models, total, err := mh.ms.ListModels(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list models")
	}

	// Convert to proto
	items := make([]*modelv1alpha1.Model, len(models))
	for i, m := range models {
		items[i] = modelToProto(m)
	}

	// Build response
	return &modelv1alpha1.ListModelsResponse{
		Items: items,
		Pagination: &modelv1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
		},
	}, nil
}

func (mh *ModelHandler) GetModel(ctx context.Context, request *modelv1alpha1.GetModelRequest) (*modelv1alpha1.Model, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	model, err := mh.ms.GetModel(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return modelToProto(model), nil
}

func (mh *ModelHandler) CreateModel(ctx context.Context, request *modelv1alpha1.CreateModelRequest) (*modelv1alpha1.CreateModelResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	_, err := mh.ms.CreateModel(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &modelv1alpha1.CreateModelResponse{}, nil
}

func (mh *ModelHandler) DeleteModel(ctx context.Context, request *modelv1alpha1.DeleteModelRequest) (*modelv1alpha1.DeleteModelResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call service
	err := mh.ms.DeleteModel(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &modelv1alpha1.DeleteModelResponse{}, nil
}

func (mh *ModelHandler) ListModelRevisions(ctx context.Context, request *modelv1alpha1.ListModelRevisionsRequest) (*modelv1alpha1.ListModelRevisionsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (mh *ModelHandler) ListModelCommits(ctx context.Context, request *modelv1alpha1.ListModelCommitsRequest) (*modelv1alpha1.ListModelCommitsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (mh *ModelHandler) GetModelCommit(ctx context.Context, request *modelv1alpha1.GetModelCommitRequest) (*modelv1alpha1.Commit, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (mh *ModelHandler) GetModelTree(ctx context.Context, request *modelv1alpha1.GetModelTreeRequest) (*modelv1alpha1.GetModelTreeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (mh *ModelHandler) GetModelBlob(ctx context.Context, request *modelv1alpha1.GetModelBlobRequest) (*modelv1alpha1.GetModelBlobResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}
