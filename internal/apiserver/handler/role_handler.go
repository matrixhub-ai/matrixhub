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

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type RoleHandler struct{}

func (r *RoleHandler) ListAllPermissions(ctx context.Context, _ *v1alpha1.ListAllPermissionsRequest) (*v1alpha1.ListAllPermissionsResponse, error) {
	transferFunc := func(ps role.PermissionCategoriesList) []*v1alpha1.RoleCategory {
		return lo.Map(ps, func(item role.PermissionCategories, index int) *v1alpha1.RoleCategory {
			return &v1alpha1.RoleCategory{
				Name: item.Category.String(),
				Permissions: lo.Map(item.Permissions, func(item role.Permission, index int) *v1alpha1.Permission {
					return &v1alpha1.Permission{
						Name:       item.String(),
						Permission: item.String(),
					}
				}),
			}
		})
	}
	return &v1alpha1.ListAllPermissionsResponse{
		SystemCategories:  transferFunc(role.PlatformPermissions),
		ProjectCategories: transferFunc(role.ProjectPermissions),
	}, nil
}

func (r *RoleHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterRolesServer(options.GRPCServer, r)
	if err := v1alpha1.RegisterRolesHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewRoleHandler() IHandler {
	return &RoleHandler{}
}
