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

package authenticator

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// CookieAuthenticator
// Use cases : web login
type CookieAuthenticator struct {
	sessionRepo user.ISessionRepo
	cookieName  string
}

func NewCookieAuthenticator(sessionRepo user.ISessionRepo) *CookieAuthenticator {
	return &CookieAuthenticator{sessionRepo: sessionRepo, cookieName: user.CookieName}
}

// Authenticate extract cookie from context or from http.Request
func (a *CookieAuthenticator) Authenticate(ctx context.Context, r *http.Request) (auth.Identity, error) {
	token := utils.GetCookieFromContext(ctx)
	return a.AuthenticateToken(ctx, "", token)
}

func (a *CookieAuthenticator) Renew(ctx context.Context) error {
	ctx, err := a.sessionRepo.LoadSession(ctx)
	if err != nil {
		return err
	}
	manager := a.sessionRepo.Manager()
	if manager.Exists(ctx, user.UserIdCtxKey) {
		manager.Put(ctx, user.LastActiveCtxKey, time.Now().Unix())
	}

	return a.sessionRepo.CommitAndWriteSessionCookie(ctx)
}

func (a *CookieAuthenticator) AuthenticateToken(ctx context.Context, _, token string) (auth.Identity, error) {
	manager := a.sessionRepo.Manager()
	ctx, err := manager.Load(ctx, token)
	if err != nil || token == "" {
		return nil, nil
	}

	if !manager.Exists(ctx, user.UserIdCtxKey) {
		return nil, errors.New("authentication invalid")
	}
	config := a.sessionRepo.GetSessionConfig()
	lastActive := manager.GetInt64(ctx, user.LastActiveCtxKey)
	if lastActive == 0 {
		return nil, errors.New("invalid session: missing last_active")
	}
	rememberMe := manager.GetBool(ctx, user.RememberMeCtxKey)
	idleTimeout := config.NonPersistentIdleTimeout
	if rememberMe {
		idleTimeout = config.PersistentSessionIdleTimeout
	}
	if time.Since(time.Unix(lastActive, 0)) > idleTimeout {
		_ = manager.Destroy(ctx)
		return nil, errors.New("session expired: idle timeout")
	}
	return user.NewUserIdentity(
		manager.GetInt(ctx, user.UserIdCtxKey),
		manager.GetString(ctx, user.UsernameCtxKey),
	), nil
}
