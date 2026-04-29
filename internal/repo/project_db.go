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

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type ProjectDBRepo struct {
	db *gorm.DB
}

var _ project.IProjectRepo = (*ProjectDBRepo)(nil)

func NewProjectDBRepo(db *gorm.DB) *ProjectDBRepo {
	return &ProjectDBRepo{db: db}
}

func (r *ProjectDBRepo) CreateProject(ctx context.Context, param *project.Project) (*project.Project, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(param).Error; err != nil {
			return err
		}

		creatorID := user.GetCurrentUserId(ctx)
		if creatorID != 0 {
			member := &project.ProjectMember{
				ProjectID:  &param.ID,
				MemberID:   creatorID,
				MemberType: project.MemberTypeUser,
				RoleID:     role.ProjectRoleAdmin,
			}
			if err := tx.Create(member).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return param, nil
}

func (r *ProjectDBRepo) GetProjectByID(ctx context.Context, id int) (*project.Project, error) {
	var p project.Project
	err := r.db.WithContext(ctx).
		Select(`projects.*,
			(SELECT COUNT(*) FROM models WHERE models.project_id = projects.id) as model_count,
			(SELECT COUNT(*) FROM datasets WHERE datasets.project_id = projects.id) as dataset_count,
			registries.url as registry_url`).
		Joins("LEFT JOIN registries ON registries.id = projects.registry_id").
		Where("projects.id = ?", id).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectDBRepo) GetProjectByName(ctx context.Context, name string) (*project.Project, error) {
	var p project.Project
	err := r.db.WithContext(ctx).
		Select(`projects.*,
			(SELECT COUNT(*) FROM models WHERE models.project_id = projects.id) as model_count,
			(SELECT COUNT(*) FROM datasets WHERE datasets.project_id = projects.id) as dataset_count,
			registries.url as registry_url`).
		Joins("LEFT JOIN registries ON registries.id = projects.registry_id").
		Where("projects.name = ?", name).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectDBRepo) GetProjectIDByName(ctx context.Context, name string) (int, error) {
	var p project.Project
	err := r.db.WithContext(ctx).
		Select("id").
		Where("name = ?", name).
		First(&p).Error
	if err != nil {
		return 0, err
	}
	return p.ID, nil
}

func (r *ProjectDBRepo) ListProjects(ctx context.Context, name string, projectType project.ProjectType, managedOnly bool, hasPlatformProjectGet bool, page, pageSize int) ([]*project.Project, int64, error) {
	var projects []*project.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&project.Project{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if projectType != project.ProjectTypeUnspecified {
		query = query.Where("type = ?", projectType)
	}

	if !hasPlatformProjectGet {
		userID := user.GetCurrentUserId(ctx)

		accessibleIDs, err := r.userProjectIDsWithPermission(ctx, userID, role.ProjectGet)
		if err != nil {
			return nil, 0, err
		}

		if managedOnly {
			if len(accessibleIDs) == 0 {
				return []*project.Project{}, 0, nil
			}
			query = query.Where("projects.id IN ?", accessibleIDs)
		} else {
			if len(accessibleIDs) == 0 {
				query = query.Where("type = ?", project.ProjectTypePublic)
			} else {
				query = query.Where("type = ? OR projects.id IN ?", project.ProjectTypePublic, accessibleIDs)
			}
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Select(`projects.*,
		(SELECT COUNT(*) FROM models WHERE models.project_id = projects.id) as model_count,
		(SELECT COUNT(*) FROM datasets WHERE datasets.project_id = projects.id) as dataset_count`)

	if utils.IsFullPageData(page, pageSize) {
		if err := query.Order("projects.name ASC").Find(&projects).Error; err != nil {
			return nil, 0, err
		}
	} else {
		offset := (page - 1) * pageSize
		if err := query.Order("projects.name ASC").Offset(offset).Limit(pageSize).Find(&projects).Error; err != nil {
			return nil, 0, err
		}
	}

	return projects, total, nil
}

func (r *ProjectDBRepo) userProjectIDsWithPermission(ctx context.Context, userID int, perm role.Permission) ([]int, error) {
	type binding struct {
		ProjectID   int
		Permissions role.PermissionList
	}
	var bindings []binding
	err := r.db.WithContext(ctx).
		Table("members_roles_projects AS mrp").
		Select("mrp.project_id AS project_id, roles.permissions AS permissions").
		Joins("INNER JOIN roles ON roles.id = mrp.role_id").
		Where("mrp.member_id = ? AND mrp.member_type = ? AND mrp.project_id IS NOT NULL",
			userID, project.MemberTypeUser).
		Scan(&bindings).Error
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(bindings))
	for _, b := range bindings {
		if role.MatchPermissions(b.Permissions, perm) {
			ids = append(ids, b.ProjectID)
		}
	}
	return ids, nil
}

func (r *ProjectDBRepo) UpdateProject(ctx context.Context, p *project.Project) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *ProjectDBRepo) DeleteProject(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&project.Project{}, id).Error
}

func (r *ProjectDBRepo) ListProjectMembers(ctx context.Context, projectID int, memberName string, page, pageSize int) ([]*project.ProjectMember, int64, error) {
	var members []*project.ProjectMember
	var total int64

	// Count query
	countQuery := r.db.WithContext(ctx).Model(&project.ProjectMember{}).Where("project_id = ?", projectID)
	if memberName != "" {
		countQuery = countQuery.Where("EXISTS (SELECT 1 FROM users WHERE users.id = members_roles_projects.member_id AND users.username LIKE ?)", "%"+memberName+"%")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := r.db.WithContext(ctx).
		Select(`members_roles_projects.*, COALESCE(users.username) as member_name`).
		Table("members_roles_projects").
		Joins("LEFT JOIN users ON users.id = members_roles_projects.member_id").
		Where("members_roles_projects.project_id = ?", projectID)

	if memberName != "" {
		query = query.Where("users.username LIKE ?", "%"+memberName+"%")
	}

	if utils.IsFullPageData(page, pageSize) {
		if err := query.Order("users.username ASC").Find(&members).Error; err != nil {
			return nil, 0, err
		}
	} else {
		offset := (page - 1) * pageSize
		if err := query.Order("users.username ASC").Offset(offset).Limit(pageSize).Find(&members).Error; err != nil {
			return nil, 0, err
		}
	}

	return members, total, nil
}

func (r *ProjectDBRepo) AddProjectMemberWithRole(ctx context.Context, pm *project.ProjectMember) error {
	return r.db.WithContext(ctx).Create(pm).Error
}

func (r *ProjectDBRepo) RemoveProjectMembers(ctx context.Context, projectID int, members []*project.Member) error {
	if len(members) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, m := range members {
			if err := tx.Where("project_id = ? AND member_id = ? AND member_type = ?",
				projectID, m.MemberID, m.MemberType).
				Delete(&project.ProjectMember{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ProjectDBRepo) UpdateProjectMemberRole(ctx context.Context, projectID int, member project.Member, newRole role.RoleType) error {
	result := r.db.WithContext(ctx).
		Model(&project.ProjectMember{}).
		Where("project_id = ? AND member_id = ? AND member_type = ?", projectID, member.MemberID, member.MemberType).
		Update("role_id", newRole)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("project member not found")
	}

	return nil
}

func (r *ProjectDBRepo) GetUserProjectPermissions(ctx context.Context, userID int, projectID int) ([]role.Permission, error) {
	var ro role.Role
	err := r.db.WithContext(ctx).
		Table("roles").
		Select("roles.permissions").
		Joins("INNER JOIN members_roles_projects mrp ON mrp.role_id = roles.id").
		Where("mrp.project_id = ? AND mrp.member_id = ? AND mrp.member_type = ?", projectID, userID, project.MemberTypeUser).
		First(&ro).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return ro.Permissions, nil
}

func (r *ProjectDBRepo) GetUserProjectRole(ctx context.Context, userID int, projectID int) (int, error) {
	var member project.ProjectMember
	err := r.db.WithContext(ctx).
		Select("role_id").
		Where("project_id = ? AND member_id = ? AND member_type = ?", projectID, userID, project.MemberTypeUser).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return int(member.RoleID), nil
}

func (r *ProjectDBRepo) GetUserPlatformPermissions(ctx context.Context, userID int) ([]role.Permission, error) {
	var ro role.Role
	err := r.db.WithContext(ctx).
		Table("roles").
		Select("roles.permissions").
		Joins("INNER JOIN members_roles_projects mrp ON mrp.role_id = roles.id").
		Where("mrp.project_id IS NULL AND mrp.member_id = ? AND mrp.member_type = ?", userID, project.MemberTypeUser).
		First(&ro).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return ro.Permissions, nil
}

func (r *ProjectDBRepo) IsUserSysAdmin(ctx context.Context, userID int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&project.ProjectMember{}).
		Where("member_id = ? AND member_type = ? AND project_id IS NULL AND role_id = ?", userID, project.MemberTypeUser, role.PlatformRoleAdmin).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ProjectDBRepo) SetUserSysAdmin(ctx context.Context, userID int, isAdmin bool) error {
	if isAdmin {
		isAdminAlready, err := r.IsUserSysAdmin(ctx, userID)
		if err != nil {
			return err
		}
		if isAdminAlready {
			return nil
		}

		member := &project.ProjectMember{
			MemberID:   userID,
			MemberType: project.MemberTypeUser,
			RoleID:     role.PlatformRoleAdmin,
			ProjectID:  nil,
		}
		return r.db.WithContext(ctx).Create(member).Error
	} else {
		return r.db.WithContext(ctx).
			Where("member_id = ? AND member_type = ? AND project_id IS NULL AND role_id = ?", userID, project.MemberTypeUser, role.PlatformRoleAdmin).
			Delete(&project.ProjectMember{}).Error
	}
}

func (r *ProjectDBRepo) GetUserAllProjectRoles(ctx context.Context, userID int) (map[string]int, error) {
	type projectRole struct {
		ProjectName string
		RoleID      int
	}

	var results []projectRole
	err := r.db.WithContext(ctx).
		Table("members_roles_projects").
		Select("projects.name as project_name, members_roles_projects.role_id").
		Joins("INNER JOIN projects ON projects.id = members_roles_projects.project_id").
		Where("members_roles_projects.member_id = ? AND members_roles_projects.member_type = ? AND members_roles_projects.project_id IS NOT NULL", userID, project.MemberTypeUser).
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	roles := make(map[string]int)
	for _, result := range results {
		roles[result.ProjectName] = result.RoleID
	}

	return roles, nil
}

func (r *ProjectDBRepo) ListProjectInfoByNames(ctx context.Context, names []string) (projects []*project.Project, err error) {
	if len(names) == 0 {
		return nil, nil
	}
	err = r.db.WithContext(ctx).Where("name IN (?)", names).Find(&projects).Error
	return
}
