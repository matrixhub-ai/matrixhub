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
	CookieName                = "token"
	DefaultSessionLifetime    = time.Hour * 24 * 7
	DefaultSessionIdleTimeout = time.Hour * 8
)

type ISessionRepo interface {
	GetSessionCookie() scs.SessionCookie
	Load(ctx context.Context, token string) (context.Context, error)
	Commit(ctx context.Context) (string, time.Time, error)
	RenewToken(ctx context.Context) error
	RememberMe(ctx context.Context, val bool)
	GetString(ctx context.Context, key string) string
	GetInt(ctx context.Context, key string) int
	Put(ctx context.Context, key string, val interface{})
	Destroy(ctx context.Context) error
	Status(ctx context.Context) scs.Status
	Exists(ctx context.Context, key string) bool
	GetBool(ctx context.Context, key string) bool
}
