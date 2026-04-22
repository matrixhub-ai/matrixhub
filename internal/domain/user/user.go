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

	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
)

type IdentityKey struct{}
type Identity struct {
	UserId   int
	Username string
}

type User struct {
	ID        int `gorm:"primary_key"`
	Username  string
	Password  string
	Email     string
	IsAdmin   bool `gorm:"-"` // Platform admin flag (not read from DB directly, determined via members_roles_projects table)
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u User) CheckPassword(password string) bool {
	return crypto.CheckPasswordHash(password, u.Password)
}

func (User) TableName() string {
	return "users"
}

type IUserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id int) (*User, error)
	GetUserByName(ctx context.Context, username string) (*User, error)
	ListUsers(ctx context.Context, page, pageSize int, search string) ([]*User, int64, error)
	DeleteUser(ctx context.Context, id int) error
	UpdateUserPassword(ctx context.Context, id int, password string) error

	// SetUserSysAdmin sets user as system admin
	SetUserSysAdmin(ctx context.Context, userID int, isAdmin bool) error

	// IsUserSysAdmin checks if user is system admin
	IsUserSysAdmin(ctx context.Context, userID int) (bool, error)

	IsUsersSysAdmin(ctx context.Context, users []int) (map[int]bool, error)

	// GetUserAllProjectRoles gets user's roles in all projects
	GetUserAllProjectRoles(ctx context.Context, userID int) (map[string]int, error)
}
