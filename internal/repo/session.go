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
	"net/http"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type sessionRepo struct {
	*scs.SessionManager
	user.SessionConfig
}

func (s *sessionRepo) LoadSession(ctx context.Context) (context.Context, error) {
	token := utils.GetCookieFromContext(ctx)
	return s.Manager().Load(ctx, token)
}

func (s *sessionRepo) Manager() *scs.SessionManager {
	return s.SessionManager
}

func (s *sessionRepo) GetSessionCookie() scs.SessionCookie {
	return s.Cookie
}

func (s *sessionRepo) GetSessionConfig() user.SessionConfig {
	return s.SessionConfig
}

func (s *sessionRepo) WriteSessionCookie(ctx context.Context, token string, expiry time.Time) error {
	sessionCookie := s.GetSessionCookie()
	cookie := &http.Cookie{
		Value:       token,
		Name:        sessionCookie.Name,
		Domain:      sessionCookie.Domain,
		HttpOnly:    sessionCookie.HttpOnly,
		Path:        sessionCookie.Path,
		SameSite:    sessionCookie.SameSite,
		Secure:      sessionCookie.Secure,
		Partitioned: sessionCookie.Partitioned,
	}

	if expiry.IsZero() {
		cookie.Expires = time.Unix(1, 0)
		cookie.MaxAge = -1
	} else if sessionCookie.Persist || s.Manager().GetBool(ctx, user.RememberMeCtxKey) {
		cookie.Expires = time.Unix(expiry.Unix()+1, 0)
		cookie.MaxAge = int(time.Until(expiry).Seconds() + 1)
	}

	return grpc.SetHeader(ctx, metadata.Pairs("set-cookie", cookie.String()))
}

func (s *sessionRepo) CommitAndWriteSessionCookie(ctx context.Context) error {
	switch s.Manager().Status(ctx) {
	case scs.Modified:
		token, expiry, err := s.Manager().Commit(ctx)
		if err != nil {
			return err
		}

		return s.WriteSessionCookie(ctx, token, expiry)
	default:
		return s.WriteSessionCookie(ctx, "", time.Time{})
	}

}

func NewSessionRepository(db *gorm.DB, config *config.Config) user.ISessionRepo {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("fail to initialize database connection: %s", err)
	}
	sessionConfig := user.SessionConfig{
		PersistentSessionLifetime:    config.Session.PersistentSessionLifetime,
		PersistentSessionIdleTimeout: config.Session.PersistentSessionIdleTimeout,
		NonPersistentIdleTimeout:     config.Session.NonPersistentIdleTimeout,
	}
	sessionManager := scs.New()
	sessionManager.Lifetime = sessionConfig.PersistentSessionLifetime
	sessionManager.IdleTimeout = 0
	sessionManager.Cookie.Name = user.CookieName
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Persist = false
	sessionManager.Store = mysqlstore.New(sqlDB)

	return &sessionRepo{
		SessionManager: sessionManager,
		SessionConfig:  sessionConfig,
	}
}
