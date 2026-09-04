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
	"fmt"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

const (
	resourceDataset = "datasets"
	resourceModel   = "models"
)

var (
	resourcePermissions = map[string]map[bool]role.Permission{
		resourceDataset: {
			true:  role.DatasetPull,
			false: role.DatasetPush,
		},
		resourceModel: {
			true:  role.ModelPull,
			false: role.ModelPush,
		},
	}
)

func NewRepoEnforcer(authzSvc authz.IAuthzService) func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error) {
	return func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (passed bool, err error) {
		// Identity is stored by the authn middlewares or the repo/hfd identity
		// normalizer before this hook runs.
		if _, ok := auth.IdentityFromContext(ctx); !ok {
			if userinfo, ok := authenticate.GetUserInfo(ctx); ok && userinfo.User != authenticate.Anonymous {
				return false, fmt.Errorf("no identity in context for authenticated user")
			}
		}

		resourceType := resourceModel
		if strings.HasPrefix(repoName, resourceDataset) {
			resourceType = resourceDataset
			repoName = strings.TrimPrefix(repoName, resourceDataset+"/")
		}
		infos := strings.SplitN(repoName, "/", 2)
		if len(infos) != 2 {
			return
		}

		project := infos[0]
		if project == "" {
			return
		}

		var ps role.Permission
		switch {
		case op.IsRead():
			ps = resourcePermissions[resourceType][true]
		case op.IsWrite(): // create, update, delete all require push permission
			ps = resourcePermissions[resourceType][false]
		}
		if ps == "" {
			return
		}
		passed, err = authzSvc.VerifyProjectPermissionByName(ctx, project, ps)
		if err != nil {
			log.Errorf("Failed to verify project permission: %s", err)
		}

		return passed, nil
	}
}
