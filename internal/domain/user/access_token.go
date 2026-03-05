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
	Id        string `gorm:"primary_key"`
	Name      string
	UserId    string
	Content   string
	Enabled   bool
	ExpiredAt *time.Time
}

func (at AccessToken) IsExpired(t time.Time) bool {
	return at.ExpiredAt != nil && at.ExpiredAt.Before(t)
}

func (AccessToken) TableName() string {
	return "access_tokens"
}

type IAccessTokenRepository interface {
	GetAccessToken(ctx context.Context, id string) (*AccessToken, error)
	ListUserAccessTokens(ctx context.Context, userId string) ([]*AccessToken, error)
	CreateAccessToken(ctx context.Context, token AccessToken) error
	DeleteAccessToken(ctx context.Context, id string) error
}
