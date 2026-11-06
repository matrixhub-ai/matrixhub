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
	"fmt"
	"net/http"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
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

// NewRepoEnforcer returns the permission hook shared by HTTP Git, SSH, HF and
// LFS. It returns project.ErrProxyProjectReadOnly for a permitted write to a
// model of a proxy project; use RepoPermissionErrorStatus to translate its errors.
func NewRepoEnforcer(authzSvc authz.IAuthzService, projectRepo project.IProjectRepo) func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error) {
	return func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (passed bool, err error) {
		userinfo, ok := authenticate.GetUserInfo(ctx)
		if ok && userinfo.User != authenticate.Anonymous {
			identity, err := authcodec.Unmarshal(userinfo.User)
			if err != nil {
				return false, err
			}
			ctx = auth.WithIdentity(ctx, identity)
		}

		resourceType, projectName, ok := parseRepoName(repoName)
		if !ok {
			return
		}
		ps := resourcePermissions[resourceType][op.IsRead()]
		if ps == "" {
			return
		}
		passed, err = authzSvc.VerifyProjectPermissionByName(ctx, projectName, ps)
		if err != nil {
			log.Errorf("Failed to verify project permission: %s", err)
		}
		if !passed || !writesProxyManagedContent(resourceType, op) {
			return passed, nil
		}

		if err := rejectProxyProject(ctx, projectRepo, projectName); err != nil {
			return false, err
		}
		if _, destProject, ok := parseRepoName(opCtx.DestRepo); ok {
			if err := rejectProxyProject(ctx, projectRepo, destProject); err != nil {
				return false, err
			}
		}

		return true, nil
	}
}

// RepoPermissionErrorStatus returns the HTTP status for an error returned by the
// permission hook.
func RepoPermissionErrorStatus(err error) int {
	if errors.Is(err, project.ErrProxyProjectReadOnly) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

// parseRepoName returns the resource type and project of a repository name.
func parseRepoName(repoName string) (resourceType, projectName string, ok bool) {
	resourceType = resourceModel
	if rest, isDataset := strings.CutPrefix(repoName, resourceDataset+"/"); isDataset {
		resourceType = resourceDataset
		repoName = rest
	}
	infos := strings.SplitN(repoName, "/", 2)
	if len(infos) != 2 || infos[0] == "" {
		return "", "", false
	}
	return resourceType, infos[0], true
}

// writesProxyManagedContent reports whether op changes model content that a
// proxy project takes from upstream. Deleting a model only drops the local copy.
func writesProxyManagedContent(resourceType string, op permission.Operation) bool {
	return resourceType == resourceModel && !op.IsRead() && op != permission.OperationDeleteRepo
}

func rejectProxyProject(ctx context.Context, projectRepo project.IProjectRepo, projectName string) error {
	prj, err := projectRepo.GetProjectByName(ctx, projectName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get project %s: %w", projectName, err)
	}
	return prj.CheckWritable()
}
