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
)

type accessTokenRepo struct {
	db *gorm.DB
}

func (a *accessTokenRepo) GetByTokenHash(ctx context.Context, hash string) (*user.AccessToken, error) {
	var ak user.AccessToken
	err := a.db.WithContext(ctx).Where("token_hash = ?", hash).Find(&ak).Error
	return &ak, err
}

func (a *accessTokenRepo) GetAccessToken(ctx context.Context, userId, id int) (*user.AccessToken, error) {
	var ak user.AccessToken
	err := a.db.WithContext(ctx).Where("id = ? and user_id = ?", id, userId).First(&ak).Error
	if err != nil {
		return nil, err
	}

	return &ak, nil
}

func (a *accessTokenRepo) ListUserAccessTokens(ctx context.Context, userId int) (aks []*user.AccessToken, err error) {
	err = a.db.WithContext(ctx).Where("user_id = ?", userId).Find(&aks).Error
	return
}

func (a *accessTokenRepo) CreateAccessToken(ctx context.Context, token user.AccessToken) error {
	return a.db.WithContext(ctx).Create(&token).Error
}

func (a *accessTokenRepo) DeleteAccessToken(ctx context.Context, userId, id int) error {
	return a.db.WithContext(ctx).Where("id = ? and user_id = ?", id, userId).Delete(&user.AccessToken{}).Error
}

func NewAccessTokenRepo(db *gorm.DB) user.IAccessTokenRepo {
	return &accessTokenRepo{
		db: db,
	}
}
