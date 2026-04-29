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

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	projectv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type ProjectHandler struct {
	projectRepo  project.IProjectRepo
	authzService authz.IAuthzService
}

func NewProjectHandler(repo project.IProjectRepo, authzService authz.IAuthzService) IHandler {
	return &ProjectHandler{
		projectRepo:  repo,
		authzService: authzService,
	}
}

func (h *ProjectHandler) RegisterToServer(opt *ServerOptions) {
	projectv1alpha1.RegisterProjectsServer(opt.GRPCServer, h)
	if err := projectv1alpha1.RegisterProjectsHandlerFromEndpoint(context.Background(), opt.GatewayMux, opt.GRPCAddr, opt.GRPCDialOpt); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func (h *ProjectHandler) CreateProject(ctx context.Context, req *projectv1alpha1.CreateProjectRequest) (*projectv1alpha1.CreateProjectResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if !hasAtLeastTwoDistinctChars(req.GetName()) {
		return nil, status.Error(codes.InvalidArgument, "project name must contain at least 2 distinct characters")
	}

	existingProject, err := h.projectRepo.GetProjectByName(ctx, req.GetName())
	if err == nil && existingProject != nil {
		return nil, status.Error(codes.AlreadyExists, "project with this name already exists")
	}

	p := &project.Project{
		Name:         req.GetName(),
		Type:         convertProtoTypeToDomain(req.GetType()),
		Organization: req.GetOrganization(),
	}

	if req.RegistryId != nil {
		registryID := int(req.RegistryId.GetValue())
		p.RegistryID = &registryID
	}

	if _, err := h.projectRepo.CreateProject(ctx, p); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.CreateProjectResponse{}, nil
}

func (h *ProjectHandler) GetProject(ctx context.Context, req *projectv1alpha1.GetProjectRequest) (*projectv1alpha1.GetProjectResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	p, err := h.projectRepo.GetProjectByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, p.ID, role.ProjectGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	resp := &projectv1alpha1.GetProjectResponse{
		Name:         p.Name,
		Type:         convertDomainTypeToProto(p.Type),
		RegistryUrl:  p.RegistryURL,
		Organization: p.Organization,
		ModelCount:   uint32(p.ModelCount),
		DatasetCount: uint32(p.DatasetCount),
		UpdatedAt:    timestamppb.New(p.UpdatedAt),
	}

	return resp, nil
}

func (h *ProjectHandler) ListProjects(ctx context.Context, req *projectv1alpha1.ListProjectsRequest) (*projectv1alpha1.ListProjectsResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
	}

	switch identity.(type) {
	case *user.Identity:
		return h.listProjectsForUser(ctx, req)
	case *robot.Identity:

		return nil, status.Error(codes.Unimplemented, "list projects for robot is not implemented yet")
	default:
		return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
	}
}

func (h *ProjectHandler) listProjectsForUser(ctx context.Context, req *projectv1alpha1.ListProjectsRequest) (*projectv1alpha1.ListProjectsResponse, error) {
	page := utils.NewPage(req.Page, req.PageSize)

	hasPlatformProjectGet, err := h.authzService.VerifyPlatformPermission(ctx, role.ProjectGet)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	projects, total, err := h.projectRepo.ListProjects(
		ctx,
		req.GetName(),
		convertProtoTypeToDomain(req.GetType()),
		req.GetManagedOnly(),
		hasPlatformProjectGet,
		int(page.Page),
		int(page.PageSize),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.ListProjectsResponse{
		Projects: lo.Map(projects, func(p *project.Project, _ int) *projectv1alpha1.Project {
			return &projectv1alpha1.Project{
				Name:            p.Name,
				Type:            convertDomainTypeToProto(p.Type),
				EnabledRegistry: p.HasProxy(),
				ModelCount:      uint32(p.ModelCount),
				DatasetCount:    uint32(p.DatasetCount),
				UpdatedAt:       timestamppb.New(p.UpdatedAt),
			}
		}),
		Pagination: &projectv1alpha1.Pagination{
			Total:    int32(total),
			Page:     req.GetPage(),
			PageSize: req.GetPageSize(),
			Pages:    utils.CalculatePages(total, req.GetPageSize()),
		},
	}, nil
}

func (h *ProjectHandler) UpdateProject(ctx context.Context, req *projectv1alpha1.UpdateProjectRequest) (*projectv1alpha1.UpdateProjectResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	p, err := h.projectRepo.GetProjectByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, p.ID, role.ProjectUpdate); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	p.Type = convertProtoTypeToDomain(req.GetType())

	if err := h.projectRepo.UpdateProject(ctx, p); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.UpdateProjectResponse{}, nil
}

func (h *ProjectHandler) DeleteProject(ctx context.Context, req *projectv1alpha1.DeleteProjectRequest) (*projectv1alpha1.DeleteProjectResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, projectID, role.ProjectDelete); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	if err := h.projectRepo.DeleteProject(ctx, projectID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.DeleteProjectResponse{}, nil
}

func (h *ProjectHandler) ListProjectMembers(ctx context.Context, req *projectv1alpha1.ListProjectMembersRequest) (*projectv1alpha1.ListProjectMembersResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page := utils.NewPage(req.Page, req.PageSize)

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, projectID, role.MemberGet); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	members, total, err := h.projectRepo.ListProjectMembers(
		ctx,
		projectID,
		req.GetMemberName(),
		int(page.Page),
		int(page.PageSize),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.ListProjectMembersResponse{
		Members: lo.Map(members, func(m *project.ProjectMember, _ int) *projectv1alpha1.ProjectMember {
			return &projectv1alpha1.ProjectMember{
				MemberId:   int32(m.MemberID),
				MemberName: m.MemberName,
				MemberType: convertDomainMemberTypeToProto(m.MemberType),
				Role:       convertDomainRoleToProto(m.RoleID),
			}
		}),
		Pagination: &projectv1alpha1.Pagination{
			Total:    int32(total),
			Page:     req.GetPage(),
			PageSize: req.GetPageSize(),
			Pages:    utils.CalculatePages(total, req.GetPageSize()),
		},
	}, nil
}

func (h *ProjectHandler) AddProjectMemberWithRole(ctx context.Context, req *projectv1alpha1.AddProjectMemberWithRoleRequest) (*projectv1alpha1.AddProjectMemberWithRoleResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, projectID, role.MemberAdd); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	pm := &project.ProjectMember{
		ProjectID:  &projectID,
		MemberID:   int(req.GetMemberId()),
		MemberType: convertProtoMemberTypeToDomain(req.GetMemberType()),
		RoleID:     convertProtoRoleToDomain(req.GetRole()),
	}

	if err := h.projectRepo.AddProjectMemberWithRole(ctx, pm); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, "member already exists in this project")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.AddProjectMemberWithRoleResponse{}, nil
}

func (h *ProjectHandler) RemoveProjectMembers(ctx context.Context, req *projectv1alpha1.RemoveProjectMembersRequest) (*projectv1alpha1.RemoveProjectMembersResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	currentUserID := user.GetCurrentUserId(ctx)
	for _, m := range req.GetMembers() {
		if m.GetMemberType() == projectv1alpha1.MemberType_MEMBER_TYPE_USER && int(m.GetMemberId()) == currentUserID {
			return nil, status.Error(codes.InvalidArgument, "cannot remove yourself from the project")
		}
	}

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, projectID, role.MemberRemove); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	members := lo.Map(req.GetMembers(), func(m *projectv1alpha1.MemberToRemove, _ int) *project.Member {
		return &project.Member{
			MemberID:   int(m.GetMemberId()),
			MemberType: convertProtoMemberTypeToDomain(m.GetMemberType()),
		}
	})

	if err := h.projectRepo.RemoveProjectMembers(ctx, projectID, members); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.RemoveProjectMembersResponse{}, nil
}

func (h *ProjectHandler) UpdateProjectMemberRole(ctx context.Context, req *projectv1alpha1.UpdateProjectMemberRoleRequest) (*projectv1alpha1.UpdateProjectMemberRoleResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if allowed, err := h.authzService.VerifyProjectPermission(ctx, projectID, role.MemberRoleUpdate); err != nil || !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	if err := h.projectRepo.UpdateProjectMemberRole(ctx, projectID, project.Member{
		MemberID:   int(req.GetMemberId()),
		MemberType: convertProtoMemberTypeToDomain(req.GetMemberType()),
	}, convertProtoRoleToDomain(req.GetRole())); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.UpdateProjectMemberRoleResponse{}, nil
}

func convertProtoTypeToDomain(t projectv1alpha1.ProjectType) project.ProjectType {
	switch t {
	case projectv1alpha1.ProjectType_PROJECT_TYPE_PRIVATE:
		return project.ProjectTypePrivate
	case projectv1alpha1.ProjectType_PROJECT_TYPE_PUBLIC:
		return project.ProjectTypePublic
	default:
		return project.ProjectTypeUnspecified
	}
}

func convertDomainTypeToProto(t project.ProjectType) projectv1alpha1.ProjectType {
	switch t {
	case project.ProjectTypePrivate:
		return projectv1alpha1.ProjectType_PROJECT_TYPE_PRIVATE
	case project.ProjectTypePublic:
		return projectv1alpha1.ProjectType_PROJECT_TYPE_PUBLIC
	default:
		return projectv1alpha1.ProjectType_PROJECT_TYPE_UNSPECIFIED
	}
}

func convertProtoMemberTypeToDomain(t projectv1alpha1.MemberType) project.MemberType {
	switch t {
	case projectv1alpha1.MemberType_MEMBER_TYPE_GROUP:
		return project.MemberTypeGroup
	default:
		return project.MemberTypeUser
	}
}

func convertDomainMemberTypeToProto(t project.MemberType) projectv1alpha1.MemberType {
	switch t {
	case project.MemberTypeGroup:
		return projectv1alpha1.MemberType_MEMBER_TYPE_GROUP
	default:
		return projectv1alpha1.MemberType_MEMBER_TYPE_USER
	}
}

func convertProtoRoleToDomain(r projectv1alpha1.ProjectRoleType) role.RoleType {
	switch r {
	case projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_ADMIN:
		return role.ProjectRoleAdmin
	case projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_EDITOR:
		return role.ProjectRoleEditor
	case projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_VIEWER:
		return role.ProjectRoleViewer
	default:
		return role.ProjectRoleViewer
	}
}

func convertDomainRoleToProto(r role.RoleType) projectv1alpha1.ProjectRoleType {
	switch r {
	case role.ProjectRoleAdmin:
		return projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_ADMIN
	case role.ProjectRoleEditor:
		return projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_EDITOR
	case role.ProjectRoleViewer:
		return projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_VIEWER
	default:
		return projectv1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_VIEWER
	}
}

func hasAtLeastTwoDistinctChars(s string) bool {
	if len(s) < 2 {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return true
		}
	}
	return false
}
