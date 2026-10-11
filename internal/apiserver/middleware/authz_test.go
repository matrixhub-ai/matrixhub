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
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

func TestNewRepoEnforcer(t *testing.T) {
	principal := Principal{user.NewUserIdentity(42, "alice")}
	foreign := authenticate.NewIdentity("mallory", "")
	author := permission.Context{Author: "proj"}
	for _, test := range []struct {
		name       string
		identity   authenticate.Identity
		op         permission.Operation
		repo       string
		opCtx      permission.Context
		permission role.Permission
		calls      int
		identityID int
		passed     bool
		err        error
	}{
		{"principal read", principal, permission.OperationReadRepo, "proj/model", permission.Context{}, role.ModelPull, 1, 42, true, nil},
		{"principal write", principal, permission.OperationUpdateRepo, "datasets/proj/ds", permission.Context{}, role.DatasetPush, 1, 42, true, nil},
		{"anonymous", nil, permission.OperationReadRepo, "proj/model", permission.Context{}, role.ModelPull, 1, 0, true, nil},
		{"foreign identity", foreign, permission.OperationReadRepo, "proj/model", permission.Context{}, "", 0, 0, false, nil},
		{"ListRepos", principal, permission.OperationListRepos, "models", permission.Context{}, "", 0, 0, false, nil},
		{"ListRepos valid path", principal, permission.OperationListRepos, "proj/model", author, "", 0, 0, false, nil},
		{"service error", principal, permission.OperationReadRepo, "proj/model", permission.Context{}, role.ModelPull, 1, 42, false, errors.New("database unavailable")},
		{"list models", principal, permission.OperationListRepos, "models", author, role.ModelPull, 1, 42, true, nil},
		{"list datasets", principal, permission.OperationListRepos, "datasets", author, role.DatasetPull, 1, 42, true, nil},
		{"list anonymous", nil, permission.OperationListRepos, "models", author, role.ModelPull, 1, 0, true, nil},
		{"list private denied", principal, permission.OperationListRepos, "models", author, role.ModelPull, 1, 42, false, nil},
		{"list spaces", principal, permission.OperationListRepos, "spaces", author, "", 0, 0, false, nil},
		{"list foreign identity", foreign, permission.OperationListRepos, "models", author, "", 0, 0, false, nil},
		{"list service error", principal, permission.OperationListRepos, "models", author, role.ModelPull, 1, 42, false, errors.New("database unavailable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.identity != nil {
				ctx = authenticate.WithIdentity(ctx, test.identity)
			}
			service := &repoEnforcerAuthzService{passed: test.passed, err: test.err}
			passed, err := NewRepoEnforcer(service)(ctx, test.op, test.repo, test.opCtx)
			if passed != test.passed || err != nil {
				t.Errorf("got (%v, %v), want (%v, nil)", passed, err, test.passed)
			}
			if service.calls != test.calls {
				t.Fatalf("service calls = %d, want %d", service.calls, test.calls)
			}
			if test.calls == 0 {
				return
			}
			if service.project != "proj" || service.permission != test.permission {
				t.Errorf("service got (%q, %q), want (proj, %q)", service.project, service.permission, test.permission)
			}
			identity, ok := auth.IdentityFromContext(service.ctx)
			if test.identityID == 0 {
				if ok {
					t.Errorf("domain identity = %v, want none", identity)
				}
			} else if !ok || identity.GetID() != test.identityID {
				t.Errorf("domain identity = %v, present = %v, want ID %d", identity, ok, test.identityID)
			}
		})
	}
}

type repoEnforcerAuthzService struct {
	ctx        context.Context
	project    string
	permission role.Permission
	calls      int
	passed     bool
	err        error
}

func (service *repoEnforcerAuthzService) VerifyProjectPermissionByName(ctx context.Context, project string, perm role.Permission) (bool, error) {
	service.ctx, service.project, service.permission = ctx, project, perm
	service.calls++
	return service.passed, service.err
}

func (*repoEnforcerAuthzService) GetUserPermissions(context.Context, int, int) ([]role.Permission, error) {
	return nil, nil
}

func (*repoEnforcerAuthzService) VerifyPlatformPermission(context.Context, role.Permission) (bool, error) {
	return false, nil
}

func (*repoEnforcerAuthzService) VerifyProjectPermission(context.Context, int, role.Permission) (bool, error) {
	return false, nil
}

func (*repoEnforcerAuthzService) GetUserAccessibleProjectIDs(context.Context, int) ([]int, error) {
	return nil, nil
}
