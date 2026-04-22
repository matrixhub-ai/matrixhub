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

	"github.com/alexedwards/scs/v2"
)

const (
	CookieName       = "token"
	UsernameCtxKey   = "username"
	UserIdCtxKey     = "user_id"
	LoginAtCtxKey    = "login_at"
	LastActiveCtxKey = "last_active"
	RememberMeCtxKey = "__rememberMe"

	MaxPersistentSessionLifetime        = time.Hour * 24 * 30
	DefaultPersistentSessionIdleTimeout = time.Hour * 24 * 7
	DefaultSessionIdleTimeout           = time.Hour * 8
)

type SessionConfig struct {
	PersistentSessionLifetime    time.Duration
	PersistentSessionIdleTimeout time.Duration
	NonPersistentIdleTimeout     time.Duration
}

type ISessionRepo interface {
	GetSessionCookie() scs.SessionCookie
	GetSessionConfig() SessionConfig
	LoadSession(ctx context.Context) (context.Context, error)
	WriteSessionCookie(ctx context.Context, token string, expiry time.Time) error
	CommitAndWriteSessionCookie(ctx context.Context) error
	Manager() *scs.SessionManager
}
