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

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	projectv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type ProjectHandler struct {
	projectRepo project.IProjectRepo
}

func NewProjectHandler(repo project.IProjectRepo) IHandler {
	return &ProjectHandler{
		projectRepo: repo,
	}
}

func (h *ProjectHandler) RegisterToServer(opt *ServerOptions) {
	projectv1alpha1.RegisterProjectsServer(opt.GRPCServer, h)
	if err := projectv1alpha1.RegisterProjectsHandlerServer(context.Background(), opt.GatewayMux, h); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func (h *ProjectHandler) CreateProject(ctx context.Context, req *projectv1alpha1.CreateProjectRequest) (*projectv1alpha1.CreateProjectResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	if err := h.projectRepo.CreateProject(ctx, p); err != nil {
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

	projects, total, err := h.projectRepo.ListProjects(
		ctx,
		req.GetName(),
		convertProtoTypeToDomain(req.GetType()),
		req.GetManagedOnly(),
		int(req.GetPage()),
		int(req.GetPageSize()),
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

	if err := h.projectRepo.DeleteProject(ctx, projectID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.DeleteProjectResponse{}, nil
}

func (h *ProjectHandler) ListProjectMembers(ctx context.Context, req *projectv1alpha1.ListProjectMembersRequest) (*projectv1alpha1.ListProjectMembersResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	members, total, err := h.projectRepo.ListProjectMembers(
		ctx,
		projectID,
		req.GetMemberName(),
		int(req.GetPage()),
		int(req.GetPageSize()),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.ListProjectMembersResponse{
		Members: lo.Map(members, func(m *project.ProjectMember, _ int) *projectv1alpha1.ProjectMember {
			return &projectv1alpha1.ProjectMember{
				MemberId:   m.MemberID,
				MemberName: m.MemberName,
				MemberType: convertDomainMemberTypeToProto(m.MemberType),
				Role:       convertDomainRoleToProto(m.RoleID),
			}
		}),
		Pagination: &projectv1alpha1.Pagination{
			Total:    int32(total),
			Page:     req.GetPage(),
			PageSize: req.GetPageSize(),
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

	pm := &project.ProjectMember{
		ProjectID:  projectID,
		MemberID:   req.GetMemberId(),
		MemberType: convertProtoMemberTypeToDomain(req.GetMemberType()),
		RoleID:     convertProtoRoleToDomain(req.GetRole()),
	}

	if err := h.projectRepo.AddProjectMemberWithRole(ctx, pm); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &projectv1alpha1.AddProjectMemberWithRoleResponse{}, nil
}

func (h *ProjectHandler) RemoveProjectMembers(ctx context.Context, req *projectv1alpha1.RemoveProjectMembersRequest) (*projectv1alpha1.RemoveProjectMembersResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	projectID, err := h.projectRepo.GetProjectIDByName(ctx, req.GetName())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	members := lo.Map(req.GetMembers(), func(m *projectv1alpha1.MemberToRemove, _ int) *project.Member {
		return &project.Member{
			MemberID:   m.GetMemberId(),
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

	if err := h.projectRepo.UpdateProjectMemberRole(ctx, projectID, project.Member{
		MemberID:   req.GetMemberId(),
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
