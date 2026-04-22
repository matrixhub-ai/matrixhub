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

package authz

import (
	"context"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

// IAuthzService permission verification service interface
type IAuthzService interface {
	// GetUserPermissions gets user's permission list in a project
	GetUserPermissions(ctx context.Context, userID int, projectID int) ([]role.Permission, error)

	// VerifyPlatformPermission verifies platform-level permission (gets user info from ctx)
	VerifyPlatformPermission(ctx context.Context, perm role.Permission) (bool, error)

	VerifyProjectPermission(ctx context.Context, projectID int, perm role.Permission) (bool, error)

	// VerifyProjectPermissionByName resolves project name to ID, then verifies permission
	VerifyProjectPermissionByName(ctx context.Context, projectName string, perm role.Permission) (bool, error)

	// GetUserAccessibleProjectIDs gets all project IDs accessible to a user
	GetUserAccessibleProjectIDs(ctx context.Context, userID int) ([]int, error)
}

// AuthzService permission verification service
type AuthzService struct {
	authzRepo   IAuthzProjectRepo
	projectRepo project.IProjectRepo
}

// NewAuthzService creates permission verification service
func NewAuthzService(authzRepo IAuthzProjectRepo, projectRepo project.IProjectRepo) IAuthzService {
	return &AuthzService{
		authzRepo:   authzRepo,
		projectRepo: projectRepo,
	}
}

// GetUserPermissions gets user's permission list in a project
func (s *AuthzService) GetUserPermissions(ctx context.Context, userID int, projectID int) ([]role.Permission, error) {
	platformPerms, err := s.authzRepo.GetUserPlatformPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	projectPerms, err := s.authzRepo.GetUserProjectPermissions(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	// Merge platform and project permissions
	return append(platformPerms, projectPerms...), nil
}

// VerifyPlatformPermission verifies platform-level permission (gets user info from ctx)
func (s *AuthzService) VerifyPlatformPermission(ctx context.Context, perm role.Permission) (bool, error) {
	userID := user.GetCurrentUserId(ctx)
	if userID == 0 {
		return false, nil
	}

	// Get user's platform-level permissions
	permissions, err := s.authzRepo.GetUserPlatformPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	return role.MatchPermissions(permissions, perm), nil
}

// VerifyProjectPermissionByName resolves project name to ID, then verifies permission
func (s *AuthzService) VerifyProjectPermissionByName(ctx context.Context, projectName string, perm role.Permission) (bool, error) {
	project, err := s.projectRepo.GetProjectByName(ctx, projectName)
	if err != nil {
		return false, err
	}
	return s.verifyProjectPermission(ctx, project, perm)
}

// VerifyProjectPermission verifies project-level permission
func (s *AuthzService) VerifyProjectPermission(ctx context.Context, projectID int, perm role.Permission) (bool, error) {
	project, err := s.projectRepo.GetProjectByID(ctx, projectID)
	if err != nil {
		return false, err
	}
	return s.verifyProjectPermission(ctx, project, perm)
}

func (s *AuthzService) verifyProjectPermission(ctx context.Context, project *project.Project, perm role.Permission) (bool, error) {
	if project.IsPublic() && (perm == role.ProjectGet || perm == role.ModelGet || perm == role.ModelPull || perm == role.DatasetGet || perm == role.DatasetPull) {
		return true, nil
	}

	userID := user.GetCurrentUserId(ctx)
	if userID == 0 {
		return false, nil
	}

	// Get user's permission list
	permissions, err := s.GetUserPermissions(ctx, userID, project.ID)
	if err != nil {
		return false, err
	}

	// Check if there's matching permission using regex
	return role.MatchPermissions(permissions, perm), nil
}

// GetUserAccessibleProjectIDs gets all project IDs where the user has membership
func (s *AuthzService) GetUserAccessibleProjectIDs(ctx context.Context, userID int) ([]int, error) {
	return s.authzRepo.GetUserAccessibleProjectIDs(ctx, userID)
}
