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

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type UserRepo struct {
	db *gorm.DB
}

func (u *UserRepo) CreateUser(ctx context.Context, user *user.User) error {
	password, err := crypto.HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = password
	return u.db.WithContext(ctx).Create(&user).Error
}

func (u *UserRepo) GetUser(ctx context.Context, id int) (*user.User, error) {
	var user user.User
	err := u.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *UserRepo) GetUserByName(ctx context.Context, username string) (*user.User, error) {
	var user user.User
	err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *UserRepo) ListUsers(ctx context.Context, page, pageSize int, search string) (us []*user.User, total int64, err error) {
	query := u.db.WithContext(ctx).Model(&user.User{})
	if search != "" {
		query = query.Where("username LIKE ?", "%"+search+"%")
	}
	if err = query.Count(&total).Error; err != nil {
		return
	}

	if utils.IsFullPageData(page, pageSize) {
		err = query.Order("username ASC").Find(&us).Error
	} else {
		offset := (page - 1) * pageSize
		err = query.Order("username ASC").Offset(offset).Limit(pageSize).Find(&us).Error
	}
	return
}

func (u *UserRepo) DeleteUser(ctx context.Context, id int) error {
	return u.db.WithContext(ctx).Where("id = ?", id).Delete(&user.User{}).Error
}

func (u *UserRepo) UpdateUserPassword(ctx context.Context, id int, password string) error {
	user, err := u.GetUser(ctx, id)
	if err != nil {
		return err
	}
	password, err = crypto.HashPassword(password)
	if err != nil {
		return err
	}
	user.Password = password
	return u.db.WithContext(ctx).Model(user).Where("id = ?", user.ID).Updates(user).Error
}

func (u *UserRepo) SetUserSysAdmin(ctx context.Context, userID int, isAdmin bool) error {
	if isAdmin {
		isAdminAlready, err := u.IsUserSysAdmin(ctx, userID)
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
		return u.db.WithContext(ctx).Create(member).Error
	} else {
		return u.db.WithContext(ctx).
			Where("member_id = ? AND member_type = ? AND project_id IS NULL AND role_id = ?", userID, project.MemberTypeUser, role.PlatformRoleAdmin).
			Delete(&project.ProjectMember{}).Error
	}
}

func (u *UserRepo) IsUserSysAdmin(ctx context.Context, userID int) (bool, error) {
	var count int64
	err := u.db.WithContext(ctx).
		Model(&project.ProjectMember{}).
		Where("member_id = ? AND member_type = ? AND project_id IS NULL AND role_id = ?", userID, project.MemberTypeUser, role.PlatformRoleAdmin).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (u *UserRepo) GetUserAllProjectRoles(ctx context.Context, userID int) (map[string]int, error) {
	type projectRole struct {
		ProjectName string
		RoleID      int
	}

	var results []projectRole
	err := u.db.WithContext(ctx).
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

func (u *UserRepo) IsUsersSysAdmin(ctx context.Context, users []int) (map[int]bool, error) {
	var pms []*project.ProjectMember
	err := u.db.WithContext(ctx).
		Where("member_id in (?) AND member_type = ? AND project_id IS NULL AND role_id = ?", users, project.MemberTypeUser, role.PlatformRoleAdmin).
		Find(&pms).Error
	if err != nil {
		return nil, err
	}
	results := make(map[int]bool)
	for _, pm := range pms {
		results[pm.MemberID] = true
	}
	return results, nil
}

func NewUserRepo(db *gorm.DB) user.IUserRepo {
	return &UserRepo{
		db: db,
	}
}
