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

package user

import (
	"context"
	"time"
)

type AccessToken struct {
	Id        int `gorm:"primary_key"`
	Name      string
	UserId    int
	TokenHash string
	Enabled   bool
	ExpireAt  *time.Time
	CreatedAt time.Time
}

func (at AccessToken) IsExpired(t time.Time) bool {
	return at.ExpireAt != nil && at.ExpireAt.Before(t)
}

func (at AccessToken) IsValid(t time.Time) bool {
	return at.Enabled && !at.IsExpired(t)
}

func (AccessToken) TableName() string {
	return "access_tokens"
}

type IAccessTokenRepo interface {
	GetByTokenHash(ctx context.Context, hash string) (*AccessToken, error)
	GetAccessToken(ctx context.Context, userId, id int) (*AccessToken, error)
	ListUserAccessTokens(ctx context.Context, userId int) ([]*AccessToken, error)
	CreateAccessToken(ctx context.Context, token AccessToken) error
	DeleteAccessToken(ctx context.Context, userId, id int) error
}
