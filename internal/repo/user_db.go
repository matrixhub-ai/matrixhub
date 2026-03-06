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

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
	"github.com/matrixhub-ai/matrixhub/internal/utils"
)

type UserRepo struct {
	db *gorm.DB
}

func (u *UserRepo) CreateUser(ctx context.Context, user user.User) error {
	password, err := crypto.HashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = password
	return u.db.WithContext(ctx).Create(&user).Error
}

func (u *UserRepo) GetUser(ctx context.Context, id string) (*user.User, error) {
	var user user.User
	err := u.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *UserRepo) ListUsers(ctx context.Context, page, pageSize int, search string) (us []*user.User, total int64, err error) {
	query := u.db.WithContext(ctx).Model(&user.User{})
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if err = query.Count(&total).Error; err != nil {
		return
	}

	if utils.IsFullPageData(page, pageSize) {
		err = query.Find(&us).Error
	} else {
		offset := (page - 1) * pageSize
		err = query.Offset(offset).Limit(pageSize).Find(&us).Error
	}
	return
}

func (u *UserRepo) DeleteUser(ctx context.Context, id string) error {
	return u.db.WithContext(ctx).Where("id = ?", id).Delete(&user.User{}).Error
}

func NewUserRepo(db *gorm.DB) user.IUserRepo {
	return &UserRepo{
		db: db,
	}
}
