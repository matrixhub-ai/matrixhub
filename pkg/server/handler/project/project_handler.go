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

package project

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/pkg/db"
	"github.com/matrixhub-ai/matrixhub/pkg/project"
	"github.com/matrixhub-ai/matrixhub/pkg/server/handler"
)

type ProjectHandler struct {
	project *project.Project
}

func NewProjectHandler(rawDB *db.RawDB) handler.Registry {
	return &ProjectHandler{
		project: project.NewProject(rawDB),
	}
}

func (ph *ProjectHandler) Register(opt *handler.ServerOptions) error {
	err := ph.project.MigrateTable(context.Background())
	if err != nil {
		return err
	}
	v1alpha1.RegisterProjectsServer(opt.GRPCServer, ph)
	err = v1alpha1.RegisterProjectsHandlerServer(context.Background(), opt.GatewayMux, ph)
	if err != nil {
		return err
	}
	return nil
}

func (ph *ProjectHandler) Get(ctx context.Context, request *v1alpha1.GetProjectRequest) (*v1alpha1.ProjectItem, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return ph.project.Get(ctx, request)
}

func (ph *ProjectHandler) Create(ctx context.Context, request *v1alpha1.CreateProjectRequest) (*v1alpha1.ProjectItem, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := ph.project.Create(ctx, request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return ph.project.Get(ctx, &v1alpha1.GetProjectRequest{
		Name: request.Name,
	})
}

func (ph *ProjectHandler) List(ctx context.Context, request *v1alpha1.ListProjectsRequest) (*v1alpha1.ListProjects, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	items, cursor, err := ph.project.List(ctx, request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	count, err := ph.project.Count(ctx, request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.ListProjects{
		Items:      items,
		NextCursor: cursor,
		Count:      count,
	}, nil
}
