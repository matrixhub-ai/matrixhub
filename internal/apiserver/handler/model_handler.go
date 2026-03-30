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
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type ModelHandler struct {
	ms           model.IModelService
	authzService authz.IAuthzService
}

func NewModelHandler(modelService model.IModelService, authzService authz.IAuthzService) IHandler {
	handler := &ModelHandler{
		ms:           modelService,
		authzService: authzService,
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

func labelCategoryToProto(category string) modelv1alpha1.Category {
	switch category {
	case "library":
		return modelv1alpha1.Category_LIBRARY
	case "license":
		return modelv1alpha1.Category_LICENSE
	case "language":
		return modelv1alpha1.Category_LANGUAGE
	case "other":
		return modelv1alpha1.Category_OTHER
	case "task":
		return modelv1alpha1.Category_TASK
	default:
		return modelv1alpha1.Category_OTHER
	}
}

// modelToProto converts domain Model to proto Model
func modelToProto(m *model.Model) *modelv1alpha1.Model {
	labels := make([]*modelv1alpha1.Label, len(m.Labels))
	for i, l := range m.Labels {
		labels[i] = &modelv1alpha1.Label{
			Id:        int32(l.ID),
			Name:      l.Name,
			Category:  labelCategoryToProto(l.Category),
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
			UpdatedAt: l.UpdatedAt.Format(time.RFC3339),
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
		Popular:        m.IsPopular,
	}
}

// revisionToProto converts domain git.Revision to proto Revision
func revisionToProto(r *git.Revision) *modelv1alpha1.Revision {
	return &modelv1alpha1.Revision{
		Name: r.Name,
	}
}

// revisionsToProto converts domain git.Revisions to proto Revisions
func revisionsToProto(revisions *git.Revisions) *modelv1alpha1.Revisions {
	branches := make([]*modelv1alpha1.Revision, len(revisions.Branches))
	for i, b := range revisions.Branches {
		branches[i] = revisionToProto(b)
	}

	tags := make([]*modelv1alpha1.Revision, len(revisions.Tags))
	for i, t := range revisions.Tags {
		tags[i] = revisionToProto(t)
	}

	return &modelv1alpha1.Revisions{
		Branches: branches,
		Tags:     tags,
	}
}

// commitToProto converts domain git.Commit to proto Commit
func commitToProto(c *git.Commit) *modelv1alpha1.Commit {
	return &modelv1alpha1.Commit{
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

// treeEntryToProtoFile converts domain git.TreeEntry to proto File
func treeEntryToProtoFile(entry *git.TreeEntry) *modelv1alpha1.File {
	var protoCommit *modelv1alpha1.Commit
	if entry.Commit != nil {
		protoCommit = commitToProto(entry.Commit)
	}

	return &modelv1alpha1.File{
		Name:   entry.Name,
		Type:   modelv1alpha1.FileType(entry.Type),
		Path:   entry.Path,
		Size:   entry.Size,
		Lfs:    entry.IsLFS,
		Sha256: entry.Hash,
		Commit: protoCommit,
		Url:    entry.URL,
	}
}

func (mh *ModelHandler) RegisterToServer(opt *ServerOptions) {
	modelv1alpha1.RegisterModelsServer(opt.GRPCServer, mh)
	if err := modelv1alpha1.RegisterModelsHandlerFromEndpoint(context.Background(), opt.GatewayMux, opt.GRPCAddr, opt.GRPCDialOpt); err != nil {
		log.Errorf("register model handler error: %s", err.Error())
	}
}

func (mh *ModelHandler) getAccessibleProjectIDs(ctx context.Context) ([]int, error) {
	userIDVal := ctx.Value(user.UserIdCtxKey)
	if userIDVal == nil {
		return mh.authzService.GetUserAccessibleProjectIDs(ctx, 0)
	}
	userID, ok := userIDVal.(int)
	if !ok {
		return mh.authzService.GetUserAccessibleProjectIDs(ctx, 0)
	}

	allowed, err := mh.authzService.VerifyPlatformPermission(ctx, "*.*")
	if err != nil {
		return nil, err
	}
	if allowed {
		return nil, nil
	}

	return mh.authzService.GetUserAccessibleProjectIDs(ctx, userID)
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

	page := utils.NewPage(request.Page, request.PageSize)

	// Build filter
	filter := &model.Filter{
		Label:    request.Labels,
		Search:   request.Search,
		Sort:     request.Sort,
		Project:  request.Project,
		Page:     page.Page,
		PageSize: page.PageSize,
		Popular:  &request.Popular,
	}

	// Inject accessible project IDs
	projectIDs, err := mh.getAccessibleProjectIDs(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get accessible projects")
	}
	if projectIDs != nil {
		if len(projectIDs) == 0 {
			return &modelv1alpha1.ListModelsResponse{
				Items:      []*modelv1alpha1.Model{},
				Pagination: utils.SetPageTotal(page, 0),
			}, nil
		}
		filter.ProjectIDs = projectIDs
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
		Items:      items,
		Pagination: utils.SetPageTotal(page, int32(total)),
	}, nil
}

func (mh *ModelHandler) GetModel(ctx context.Context, request *modelv1alpha1.GetModelRequest) (*modelv1alpha1.Model, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
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

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelPush); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
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

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelDelete); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
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
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	// Call service
	revisions, err := mh.ms.ListModelRevisions(ctx, request.Project, request.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return nil, status.Errorf(codes.NotFound, "model %s not found in project %s", request.Name, request.Project)
		}
		return nil, status.Errorf(codes.Internal, "failed to list revisions: %v", err)
	}

	// Convert to proto and return
	return &modelv1alpha1.ListModelRevisionsResponse{
		Items: revisionsToProto(revisions),
	}, nil
}

func (mh *ModelHandler) ListModelCommits(ctx context.Context, request *modelv1alpha1.ListModelCommitsRequest) (*modelv1alpha1.ListModelCommitsResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	// Call service (diff parameter is ignored for now)
	commits, total, err := mh.ms.ListModelCommits(ctx, request.Project, request.Name, request.Revision, int(request.Page), int(request.PageSize))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to list commits")
	}

	// Convert to proto
	items := make([]*modelv1alpha1.Commit, len(commits))
	for i, c := range commits {
		items[i] = commitToProto(c)
	}

	return &modelv1alpha1.ListModelCommitsResponse{
		Items: items,
		Pagination: &modelv1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
			Pages:    utils.CalculatePages(total, request.PageSize),
		},
	}, nil
}

func (mh *ModelHandler) GetModelCommit(ctx context.Context, request *modelv1alpha1.GetModelCommitRequest) (*modelv1alpha1.Commit, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	// Call service
	commit, err := mh.ms.GetModelCommit(ctx, request.Project, request.Name, request.Id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get commit")
	}

	return commitToProto(commit), nil
}

func (mh *ModelHandler) GetModelTree(ctx context.Context, request *modelv1alpha1.GetModelTreeRequest) (*modelv1alpha1.GetModelTreeResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	// Call service
	entries, err := mh.ms.GetModelTree(ctx, request.Project, request.Name, request.Revision, request.Path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get tree")
	}

	// Convert to proto
	items := make([]*modelv1alpha1.File, len(entries))
	for i, entry := range entries {
		items[i] = treeEntryToProtoFile(entry)
	}

	return &modelv1alpha1.GetModelTreeResponse{
		Items: items,
	}, nil
}

func (mh *ModelHandler) UpdateModelSetting(ctx context.Context, request *modelv1alpha1.UpdateModelSettingRequest) (*modelv1alpha1.UpdateModelSettingResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelSetting); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	update := &model.SettingUpdate{}
	if request.Popular != nil {
		v := request.Popular.GetValue()
		update.IsPopular = &v
	}

	err := mh.ms.UpdateModelSetting(ctx, request.Project, request.Name, update)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &modelv1alpha1.UpdateModelSettingResponse{}, nil
}

func (mh *ModelHandler) GetModelBlob(ctx context.Context, request *modelv1alpha1.GetModelBlobRequest) (*modelv1alpha1.File, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Verify permission
	if allowed, err := mh.authzService.VerifyProjectPermissionByName(ctx, request.Project, role.ModelGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	entry, err := mh.ms.GetModelBlob(ctx, request.Project, request.Name, request.Revision, request.Path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get blob")
	}

	if entry.Type == git.FileTypeDir {
		return nil, status.Error(codes.InvalidArgument, "path must reference a file blob")
	}

	return treeEntryToProtoFile(entry), nil
}
