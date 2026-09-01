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
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	infradb "github.com/matrixhub-ai/matrixhub/internal/infra/db"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type sessionRepo struct {
	*scs.SessionManager
	user.SessionConfig
	stopCleanup func()
	closeOnce   sync.Once
}

type sessionStore interface {
	scs.Store
	StopCleanup()
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

func (s *sessionRepo) Close() {
	s.closeOnce.Do(s.stopCleanup)
}

func NewSessionRepository(db *gorm.DB, config *config.Config) (user.ISessionRepo, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database connection for session store: %w", err)
	}
	store, err := newSessionStore(db.Name(), sqlDB)
	if err != nil {
		return nil, err
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
	sessionManager.Store = store

	return &sessionRepo{
		SessionManager: sessionManager,
		SessionConfig:  sessionConfig,
		stopCleanup:    store.StopCleanup,
	}, nil
}

func newSessionStore(driver string, sqlDB *sql.DB) (sessionStore, error) {
	switch driver {
	case infradb.DriverMySQL:
		return mysqlstore.New(sqlDB), nil
	case infradb.DriverPostgres:
		return postgresstore.New(sqlDB), nil
	case infradb.DriverSQLite:
		return sqlite3store.New(sqlDB), nil
	default:
		return nil, fmt.Errorf("unsupported session database driver %q", driver)
	}
}
