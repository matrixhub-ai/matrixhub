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
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type CtxKey string

func (c CtxKey) String() string {
	return string(c)
}

var (
	InvalidUsernameOrPassword = errors.New("invalid username or password")

	UsernameCtxKey CtxKey = "username"
	UserIdCtxKey   CtxKey = "user_id"
	LoginAtCtxKey  CtxKey = "login_at"
)

type IUserService interface {
	LoginUser(ctx context.Context, username, password string, rememberMe bool) (*http.Cookie, error)
	LogoutUser(ctx context.Context) error
	GetCurrentUser(ctx context.Context) (*User, error)
}

type UserService struct {
	sessionRepo ISessionRepo
	userRepo    IUserRepo
}

func (us UserService) GetCurrentUser(ctx context.Context) (*User, error) {
	username := us.sessionRepo.GetString(ctx, UsernameCtxKey.String())
	if username == "" {
		return nil, InvalidUsernameOrPassword
	}

	return us.userRepo.GetUserByName(ctx, username)
}

func (us UserService) LoginUser(ctx context.Context, username, password string, rememberMe bool) (*http.Cookie, error) {
	u, err := us.userRepo.GetUserByName(ctx, username)
	if err != nil {
		return nil, InvalidUsernameOrPassword
	}
	if !u.CheckPassword(password) {
		return nil, InvalidUsernameOrPassword
	}

	if err = us.sessionRepo.RenewToken(ctx); err != nil {
		return nil, err
	}

	us.sessionRepo.RememberMe(ctx, rememberMe)
	us.sessionRepo.Put(ctx, UserIdCtxKey.String(), u.ID)
	us.sessionRepo.Put(ctx, UsernameCtxKey.String(), u.Username)
	us.sessionRepo.Put(ctx, LoginAtCtxKey.String(), time.Now().Format(time.RFC3339))

	token, expiry, err := us.sessionRepo.Commit(ctx)
	if err != nil {
		return nil, InvalidUsernameOrPassword
	}
	sessionCookie := us.sessionRepo.GetSessionCookie()

	return &http.Cookie{
		Name:     sessionCookie.Name, // 默认 "session"
		Value:    token,
		Path:     sessionCookie.Path,
		Domain:   sessionCookie.Domain,
		Expires:  expiry,
		HttpOnly: sessionCookie.HttpOnly,
		Secure:   sessionCookie.Secure,
		SameSite: sessionCookie.SameSite,
	}, nil
}

func (us UserService) LogoutUser(ctx context.Context) error {
	return us.sessionRepo.Destroy(ctx)
}

func NewUserService(session ISessionRepo, user IUserRepo) IUserService {
	return &UserService{
		sessionRepo: session,
		userRepo:    user,
	}
}

// GetCurrentUsername get current username from context
func GetCurrentUsername(ctx context.Context) string {
	val := ctx.Value(UsernameCtxKey)
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}
