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

package repo

import (
	"context"
	"errors"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

type AuthzDBRepo struct {
	db *gorm.DB
}

var _ authz.IAuthzProjectRepo = (*AuthzDBRepo)(nil)

func NewAuthzDBRepo(db *gorm.DB) authz.IAuthzProjectRepo {
	return &AuthzDBRepo{db: db}
}

// GetUserProjectPermissions gets user's permissions in a project
func (r *AuthzDBRepo) GetUserProjectPermissions(ctx context.Context, userID int, projectID int) ([]role.Permission, error) {
	var result struct {
		Permissions datatypes.JSONSlice[role.Permission]
	}
	err := r.db.WithContext(ctx).
		Table("roles").
		Select("roles.permissions").
		Joins("INNER JOIN members_roles_projects mrp ON mrp.role_id = roles.id").
		Where("mrp.project_id = ? AND mrp.member_id = ? AND mrp.member_type = ?", projectID, userID, project.MemberTypeUser).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return result.Permissions, nil
}

// GetUserPlatformPermissions gets user's platform-level permissions
func (r *AuthzDBRepo) GetUserPlatformPermissions(ctx context.Context, userID int) ([]role.Permission, error) {
	var result struct {
		Permissions datatypes.JSONSlice[role.Permission]
	}
	err := r.db.WithContext(ctx).
		Table("roles").
		Select("roles.permissions").
		Joins("INNER JOIN members_roles_projects mrp ON mrp.role_id = roles.id").
		Where("mrp.project_id IS NULL AND mrp.member_id = ? AND mrp.member_type = ?", userID, project.MemberTypeUser).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return result.Permissions, nil
}

// GetRobotProjectPermissions  gets robot's permissions in a project
func (r *AuthzDBRepo) GetRobotProjectPermissions(ctx context.Context, robotID int, projectID int) ([]role.Permission, error) {
	var result struct {
		Permissions datatypes.JSONSlice[role.Permission]
	}
	err := r.db.WithContext(ctx).
		Table("robots").
		Select("robots.project_permissions").
		Joins("INNER JOIN robots_projects rp ON robots.id = robot_id").
		Where("rp.project_id = ? AND rp.robot_id = ?", projectID, robotID).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return result.Permissions, err
}

// GetUserAccessibleProjectIDs gets all project IDs where the user has membership
// or that are public (visible to everyone)
func (r *AuthzDBRepo) GetUserAccessibleProjectIDs(ctx context.Context, userID int) ([]int, error) {
	var projectIDs []int
	err := r.db.WithContext(ctx).
		Table("projects").
		Select("DISTINCT id").
		Where("type = ?", project.ProjectTypePublic).
		Or("id IN (?)",
			r.db.Table("members_roles_projects").
				Select("project_id").
				Where("member_id = ? AND member_type = ? AND project_id IS NOT NULL", userID, project.MemberTypeUser),
		).
		Find(&projectIDs).Error
	if err != nil {
		return nil, err
	}
	return projectIDs, nil
}
