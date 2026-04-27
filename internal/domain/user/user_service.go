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

	"github.com/pkg/errors"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
)

var (
	InvalidUsernameOrPassword = errors.New("invalid username or password")
)

type IUserService interface {
	LoginUser(ctx context.Context, username, password string, rememberMe bool) error
	LogoutUser(ctx context.Context) error
	GetCurrentUser(ctx context.Context) (*User, error)
}

type UserService struct {
	sessionRepo ISessionRepo
	userRepo    IUserRepo
}

func (us UserService) GetCurrentUser(ctx context.Context) (*User, error) {
	username := us.sessionRepo.Manager().GetString(ctx, UsernameCtxKey)
	if username == "" {
		return nil, InvalidUsernameOrPassword
	}

	return us.userRepo.GetUserByName(ctx, username)
}

func (us UserService) LoginUser(ctx context.Context, username, password string, rememberMe bool) error {
	u, err := us.userRepo.GetUserByName(ctx, username)
	if err != nil {
		return InvalidUsernameOrPassword
	}
	if !u.CheckPassword(password) {
		return InvalidUsernameOrPassword
	}
	manager := us.sessionRepo.Manager()
	ctx, err = manager.Load(ctx, "")
	if err != nil {
		return err
	}
	if err = manager.RenewToken(ctx); err != nil {
		return err
	}
	now := time.Now().Unix()
	manager.RememberMe(ctx, rememberMe)
	manager.Put(ctx, UserIdCtxKey, u.ID)
	manager.Put(ctx, UsernameCtxKey, u.Username)
	manager.Put(ctx, LoginAtCtxKey, now)
	manager.Put(ctx, LastActiveCtxKey, now)

	token, expiry, err := manager.Commit(ctx)
	if err != nil {
		return InvalidUsernameOrPassword
	}
	return us.sessionRepo.WriteSessionCookie(ctx, token, expiry)
}

func (us UserService) LogoutUser(ctx context.Context) error {
	ctx, err := us.sessionRepo.LoadSession(ctx)
	if err != nil {
		return err
	}
	return us.sessionRepo.Manager().Destroy(ctx)
}

func NewUserService(session ISessionRepo, user IUserRepo) IUserService {
	return &UserService{
		sessionRepo: session,
		userRepo:    user,
	}
}

// GetCurrentUsername get current username from context
func GetCurrentUsername(ctx context.Context) string {
	val, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return ""
	}
	return val.GetName()
}

func GetCurrentUserId(ctx context.Context) int {
	val, ok := auth.IdentityFromContext(ctx)
	if !ok {
		return 0
	}
	return val.GetID()
}
