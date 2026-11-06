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

package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	authzmocks "github.com/matrixhub-ai/matrixhub/internal/domain/authz/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

func TestRepoEnforcerProxyProjects(t *testing.T) {
	registryID := 1
	local := &project.Project{Name: "local"}
	proxy := &project.Project{Name: "proxy", RegistryID: &registryID}
	lookupErr := errors.New("database is down")

	type lookup struct {
		name string
		prj  *project.Project
		err  error
	}
	tests := []struct {
		name       string
		op         permission.Operation
		repo       string
		opCtx      permission.Context
		permission role.Permission
		denied     bool
		lookups    []lookup
		wantPassed bool
		wantErr    error
	}{
		{
			name:       "push to a local project passes",
			op:         permission.OperationUpdateRepo,
			repo:       "local/model",
			permission: role.ModelPush,
			lookups:    []lookup{{name: "local", prj: local}},
			wantPassed: true,
		},
		{
			name:       "push to a proxy project is rejected",
			op:         permission.OperationUpdateRepo,
			repo:       "proxy/model",
			permission: role.ModelPush,
			lookups:    []lookup{{name: "proxy", prj: proxy}},
			wantErr:    project.ErrProxyProjectReadOnly,
		},
		{
			name:       "create in a proxy project is rejected",
			op:         permission.OperationCreateRepo,
			repo:       "proxy/model",
			permission: role.ModelPush,
			lookups:    []lookup{{name: "proxy", prj: proxy}},
			wantErr:    project.ErrProxyProjectReadOnly,
		},
		{
			name:       "move into a proxy project is rejected",
			op:         permission.OperationUpdateRepo,
			repo:       "local/model",
			opCtx:      permission.Context{DestRepo: "proxy/model"},
			permission: role.ModelPush,
			lookups:    []lookup{{name: "local", prj: local}, {name: "proxy", prj: proxy}},
			wantErr:    project.ErrProxyProjectReadOnly,
		},
		{
			name:       "move out of a proxy project is rejected",
			op:         permission.OperationUpdateRepo,
			repo:       "proxy/model",
			opCtx:      permission.Context{DestRepo: "local/model"},
			permission: role.ModelPush,
			lookups:    []lookup{{name: "proxy", prj: proxy}},
			wantErr:    project.ErrProxyProjectReadOnly,
		},
		{
			name:       "move into a project without a record keeps passing",
			op:         permission.OperationUpdateRepo,
			repo:       "local/model",
			opCtx:      permission.Context{DestRepo: "missing/model"},
			permission: role.ModelPush,
			lookups:    []lookup{{name: "local", prj: local}, {name: "missing", err: gorm.ErrRecordNotFound}},
			wantPassed: true,
		},
		{
			name:       "read of a proxy project passes without a lookup",
			op:         permission.OperationReadRepo,
			repo:       "proxy/model",
			permission: role.ModelPull,
			wantPassed: true,
		},
		{
			name:       "delete in a proxy project passes without a lookup",
			op:         permission.OperationDeleteRepo,
			repo:       "proxy/model",
			permission: role.ModelPush,
			wantPassed: true,
		},
		{
			name:       "dataset push passes without a lookup",
			op:         permission.OperationUpdateRepo,
			repo:       "datasets/proxy/dataset",
			permission: role.DatasetPush,
			wantPassed: true,
		},
		{
			name:       "project named like the dataset prefix holds models",
			op:         permission.OperationUpdateRepo,
			repo:       "datasets-proxy/model",
			permission: role.ModelPush,
			lookups:    []lookup{{name: "datasets-proxy", prj: proxy}},
			wantErr:    project.ErrProxyProjectReadOnly,
		},
		{
			name:       "user without permission does not learn the project type",
			op:         permission.OperationUpdateRepo,
			repo:       "proxy/model",
			permission: role.ModelPush,
			denied:     true,
		},
		{
			name:       "failed project lookup denies the write",
			op:         permission.OperationUpdateRepo,
			repo:       "proxy/model",
			permission: role.ModelPush,
			lookups:    []lookup{{name: "proxy", err: lookupErr}},
			wantErr:    lookupErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			authzSvc := authzmocks.NewMockIAuthzService(ctrl)
			projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
			authzSvc.EXPECT().
				VerifyProjectPermissionByName(gomock.Any(), gomock.Any(), tt.permission).
				Return(!tt.denied, nil)
			for _, l := range tt.lookups {
				projectRepo.EXPECT().GetProjectByName(ctx, l.name).Return(l.prj, l.err)
			}

			passed, err := NewRepoEnforcer(authzSvc, projectRepo)(ctx, tt.op, tt.repo, tt.opCtx)

			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, tt.wantPassed, passed)
		})
	}
}

func TestRepoPermissionErrorStatus(t *testing.T) {
	require.Equal(t, http.StatusForbidden, RepoPermissionErrorStatus(project.ErrProxyProjectReadOnly))
	require.Equal(t, http.StatusInternalServerError, RepoPermissionErrorStatus(errors.New("database is down")))
}
