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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	registryv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type RegistryHandler struct {
	registryRepo registry.IRegistryRepo
}

func NewRegistryHandler(repo registry.IRegistryRepo) IHandler {
	handler := &RegistryHandler{
		registryRepo: repo,
	}
	return handler
}

func (rh *RegistryHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	registryv1alpha1.RegisterRegistriesServer(options.GRPCServer, rh)
	if err := registryv1alpha1.RegisterRegistriesHandlerServer(context.Background(), options.GatewayMux, rh); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func (rh *RegistryHandler) ListRegistries(ctx context.Context, request *registryv1alpha1.ListRegistriesRequest) (*registryv1alpha1.ListRegistriesResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	registries, total, err := rh.registryRepo.ListRegistries(ctx, int(request.Page), int(request.PageSize), request.Search)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var list []*registryv1alpha1.Registry
	for _, registry := range registries {
		list = append(list, convertToAPI(registry))
	}

	return &registryv1alpha1.ListRegistriesResponse{
		Registries: list,
		Pagination: &registryv1alpha1.Pagination{
			Total:    int32(total),
			Page:     int32(request.Page),
			PageSize: int32(request.PageSize),
		},
	}, nil
}

func (rh *RegistryHandler) GetRegistry(ctx context.Context, request *registryv1alpha1.GetRegistryRequest) (*registryv1alpha1.GetRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	registry, err := rh.registryRepo.GetRegistry(ctx, int(request.Id))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &registryv1alpha1.GetRegistryResponse{
		Registry: convertToAPI(registry),
	}, nil
}

func (rh *RegistryHandler) CreateRegistry(ctx context.Context, request *registryv1alpha1.CreateRegistryRequest) (*registryv1alpha1.CreateRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	domainRegistry := registry.Registry{
		Name:        request.Name,
		Description: request.Description,
		Type:        request.Type.String(),
		URL:         request.Url,
		Insecure:    request.Insecure,
	}

	created, err := rh.registryRepo.CreateRegistry(ctx, domainRegistry)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &registryv1alpha1.CreateRegistryResponse{
		Registry: convertToAPI(created),
	}, nil
}

func (rh *RegistryHandler) UpdateRegistry(ctx context.Context, request *registryv1alpha1.UpdateRegistryRequest) (*registryv1alpha1.UpdateRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	domainRegistry := registry.Registry{
		ID:          int(request.Id),
		Name:        request.Name,
		Description: request.Description,
		URL:         request.Url,
		Insecure:    request.Insecure,
	}

	if err := rh.registryRepo.UpdateRegistry(ctx, domainRegistry); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// GORM updates the passed struct in-place, so domainRegistry should now
	// contain any DB-managed fields (e.g. UpdatedAt). Use convertToAPI for response.
	return &registryv1alpha1.UpdateRegistryResponse{
		Registry: convertToAPI(&domainRegistry),
	}, nil
}

func (rh *RegistryHandler) DeleteRegistry(ctx context.Context, request *registryv1alpha1.DeleteRegistryRequest) (*registryv1alpha1.DeleteRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := rh.registryRepo.DeleteRegistry(ctx, int(request.Id)); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &registryv1alpha1.DeleteRegistryResponse{}, nil
}

func (rh *RegistryHandler) PingRegistry(ctx context.Context, request *registryv1alpha1.PingRegistryRequest) (*registryv1alpha1.PingRegistryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func convertToDomain(r registryv1alpha1.Registry) *registry.Registry {
	return nil
}

func convertToAPI(d *registry.Registry) *registryv1alpha1.Registry {
	if d == nil {
		return nil
	}

	r := &registryv1alpha1.Registry{
		Id:          int32(d.ID),
		Name:        d.Name,
		Description: d.Description,
		Url:         d.URL,
		Insecure:    d.Insecure,
		CreatedAt:   timestamppb.New(d.CreatedAt),
		UpdatedAt:   timestamppb.New(d.UpdatedAt),
	}

	switch d.Type {
	case "huggingface":
		r.Type = registryv1alpha1.RegistryType_REGISTRY_TYPE_HUGGINGFACE
	default:
		r.Type = registryv1alpha1.RegistryType_REGISTRY_TYPE_UNSPECIFIED
	}

	switch d.Status {
	case 1:
		r.Status = registryv1alpha1.RegistryStatus_REGISTRY_STATUS_HEALTHY
	case 2:
		r.Status = registryv1alpha1.RegistryStatus_REGISTRY_STATUS_UNHEALTHY
	default:
		r.Status = registryv1alpha1.RegistryStatus_REGISTRY_STATUS_UNSPECIFIED
	}

	// Map domain credential back to proto oneof.
	return r
}
