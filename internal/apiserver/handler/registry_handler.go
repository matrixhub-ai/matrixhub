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
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	registryv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	pageutils "github.com/matrixhub-ai/matrixhub/internal/utils"
)

type RegistryHandler struct {
	registryRepo registry.IRegistryRepo
}

func NewRegistryHandler(repo registry.IRegistryRepo) IHandler {
	return &RegistryHandler{
		registryRepo: repo,
	}
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

	page := pageutils.NewPage(request.Page, request.PageSize)
	domainRegistries, total, err := rh.registryRepo.ListRegistries(ctx, int(page.Page), int(page.PageSize), request.Search)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	pageutils.SetPageTotal(page, int32(total))

	var list []*registryv1alpha1.Registry
	for _, registry := range domainRegistries {
		list = append(list, convertDomainRegistryToAPIRegistry(registry))
	}

	return &registryv1alpha1.ListRegistriesResponse{
		Registries: list,
		Pagination: page,
	}, nil
}

func (rh *RegistryHandler) GetRegistry(ctx context.Context, request *registryv1alpha1.GetRegistryRequest) (*registryv1alpha1.GetRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	domainRegistry, err := rh.registryRepo.GetRegistry(ctx, request.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &registryv1alpha1.GetRegistryResponse{
		Registry: convertDomainRegistryToAPIRegistry(domainRegistry),
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
	if b := request.GetBasic(); b != nil {
		domainRegistry.SetCredential(registry.NewBasicCredential(b.Username, b.Password))
	}

	created, err := rh.registryRepo.CreateRegistry(ctx, domainRegistry)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &registryv1alpha1.CreateRegistryResponse{
		Registry: convertDomainRegistryToAPIRegistry(created),
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
	if b := request.GetBasic(); b != nil {
		domainRegistry.SetCredential(registry.NewBasicCredential(b.Username, b.Password))
	}
	if err := rh.registryRepo.UpdateRegistry(ctx, domainRegistry); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	updated, err := rh.registryRepo.GetRegistry(ctx, request.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &registryv1alpha1.UpdateRegistryResponse{
		Registry: convertDomainRegistryToAPIRegistry(updated),
	}, nil
}

func (rh *RegistryHandler) DeleteRegistry(ctx context.Context, request *registryv1alpha1.DeleteRegistryRequest) (*registryv1alpha1.DeleteRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := rh.registryRepo.DeleteRegistry(ctx, request.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &registryv1alpha1.DeleteRegistryResponse{}, nil
}

func (rh *RegistryHandler) PingRegistry(ctx context.Context, request *registryv1alpha1.PingRegistryRequest) (*registryv1alpha1.PingRegistryResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	url := strings.TrimSpace(request.Url)
	if url == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid url: "+err.Error())
	}
	if b := request.GetBasic(); b != nil && (b.Username != "" || b.Password != "") {
		req.SetBasicAuth(b.Username, b.Password)
	}

	resp, err := pingRegistryRequest(req, request.Insecure)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "connectivity check failed: "+err.Error())
	}
	defer resp.Body.Close()

	fmt.Println("resp.StatusCode", resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, status.Error(codes.Unavailable, "registry returned status "+resp.Status)
	}
	return &registryv1alpha1.PingRegistryResponse{}, nil
}

func convertDomainRegistryToAPIRegistry(d *registry.Registry) *registryv1alpha1.Registry {
	if d == nil {
		return nil
	}

	r := &registryv1alpha1.Registry{
		Id:          int32(d.ID),
		Name:        d.Name,
		Description: d.Description,
		Url:         d.URL,
		Insecure:    d.Insecure,
		Status:      registryv1alpha1.RegistryStatus(d.Status),
		CreatedAt:   timestamppb.New(d.CreatedAt),
		UpdatedAt:   timestamppb.New(d.UpdatedAt),
	}

	if basic := registry.AsBasic(d.GetCredential()); basic != nil {
		r.Credential = &registryv1alpha1.Registry_Basic{
			Basic: &registryv1alpha1.RegistryBasicCredential{
				Username: basic.Username,
				Password: basic.Password,
			},
		}
	}

	if v, ok := registryv1alpha1.RegistryType_value[d.Type]; ok {
		r.Type = registryv1alpha1.RegistryType(v)
	} else {
		r.Type = registryv1alpha1.RegistryType_REGISTRY_TYPE_UNSPECIFIED
	}

	return r
}

// pingRegistryRequest performs a basic HTTP GET to the given registry endpoint.
// TLS verification can be skipped when insecure is true.
func pingRegistryRequest(req *http.Request, insecure bool) (*http.Response, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}
	return client.Do(req)
}
