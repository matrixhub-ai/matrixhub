package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	projectv1alpha1 "matrixhub.ai/api/protogen/v1alpha1/project"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type ProjectHandler struct {
	ProjectRepo project.IProjectRepository
}

func NewProjectHandler(repo project.IProjectRepository) *ProjectHandler {
	handler := &ProjectHandler{
		ProjectRepo: repo,
	}

	return handler
}

func (ph *ProjectHandler) RegisterToServer(opt *ServerOptions) {
	// Register GRPC Handler
	projectv1alpha1.RegisterProjectsServer(opt.GRPCServer, ph)
	if err := projectv1alpha1.RegisterProjectsHandlerServer(context.Background(), opt.GatewayMux, ph); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func (ph *ProjectHandler) GetProject(ctx context.Context, request *projectv1alpha1.GetProjectRequest) (*projectv1alpha1.GetProjectResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	result, err := ph.ProjectRepo.GetProject(ctx, &project.Project{Name: request.Name})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	output := &projectv1alpha1.GetProjectResponse{
		Name: result.Name,
	}
	return output, nil
}

func (ph *ProjectHandler) CreateProject(ctx context.Context, request *projectv1alpha1.CreateProjectRequest) (*projectv1alpha1.CreateProjectResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	param := &project.Project{
		Name: request.Name,
	}
	err := ph.ProjectRepo.CreateProject(ctx, param)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &projectv1alpha1.CreateProjectResponse{}, nil
}
