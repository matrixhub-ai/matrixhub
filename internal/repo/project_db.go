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
	"github.com/matrixhub-ai/matrixhub/internal/utils"
)

type ProjectDBRepo struct {
	db *gorm.DB
}

var _ project.IProjectRepo = (*ProjectDBRepo)(nil)

func NewProjectDBRepo(db *gorm.DB) *ProjectDBRepo {
	return &ProjectDBRepo{db: db}
}

func (r *ProjectDBRepo) CreateProject(ctx context.Context, p *project.Project) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *ProjectDBRepo) GetProjectByName(ctx context.Context, name string) (*project.Project, error) {
	var p project.Project
	err := r.db.WithContext(ctx).
		Select(`projects.*,
			(SELECT COUNT(*) FROM models WHERE models.project_id = projects.id) as model_count,
			(SELECT COUNT(*) FROM datasets WHERE datasets.project_id = projects.id) as dataset_count`).
		Where("name = ?", name).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectDBRepo) GetProjectIDByName(ctx context.Context, name string) (int, error) {
	var id int
	err := r.db.WithContext(ctx).
		Model(&project.Project{}).
		Select("id").
		Where("name = ?", name).
		Scan(&id).Error
	if err != nil {
		return 0, err
	}
	return id, nil
}

// getCurrentUserID is a mock method to get current user ID from context.
// TODO: Replace with actual authentication logic when implemented.
func (r *ProjectDBRepo) getCurrentUserID(ctx context.Context) string {
	// Mock: return a placeholder user ID
	// In production, this should extract user ID from JWT token or session
	return "mock-user-id"
}

func (r *ProjectDBRepo) ListProjects(ctx context.Context, name string, projectType project.ProjectType, managedOnly bool, page, pageSize int) ([]*project.Project, int64, error) {
	var projects []*project.Project
	var total int64

	query := r.db.WithContext(ctx).Model(&project.Project{})

	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if projectType != project.ProjectTypeUnspecified {
		query = query.Where("type = ?", projectType)
	}

	// If managedOnly is true, only return projects where the current user has access
	if managedOnly {
		userID := r.getCurrentUserID(ctx)
		query = query.Where("EXISTS (SELECT 1 FROM members_roles_projects WHERE project_id = projects.id AND member_id = ? AND member_type = ?)", userID, project.MemberTypeUser)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Use subquery to get model_count and dataset_count in one query
	query = query.Select(`projects.*,
		(SELECT COUNT(*) FROM models WHERE models.project_id = projects.id) as model_count,
		(SELECT COUNT(*) FROM datasets WHERE datasets.project_id = projects.id) as dataset_count`)

	if utils.IsFullPageData(page, pageSize) {
		if err := query.Find(&projects).Error; err != nil {
			return nil, 0, err
		}
	} else {
		offset := (page - 1) * pageSize
		if err := query.Offset(offset).Limit(pageSize).Find(&projects).Error; err != nil {
			return nil, 0, err
		}
	}

	return projects, total, nil
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
		if err := query.Find(&members).Error; err != nil {
			return nil, 0, err
		}
	} else {
		offset := (page - 1) * pageSize
		if err := query.Offset(offset).Limit(pageSize).Find(&members).Error; err != nil {
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
