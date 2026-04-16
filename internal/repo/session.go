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
	"net/http"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type sessionRepo struct {
	*scs.SessionManager
}

func (s *sessionRepo) GetSessionCookie() scs.SessionCookie {
	return s.Cookie
}

func NewSessionRepository(db *gorm.DB) user.ISessionRepo {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("fail to initialize database connection: %s", err)
	}

	sessionManager := scs.New()
	sessionManager.Lifetime = user.DefaultSessionLifetime
	sessionManager.IdleTimeout = user.DefaultSessionIdleTimeout
	sessionManager.Cookie.Name = user.CookieName
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Store = mysqlstore.New(sqlDB)

	return &sessionRepo{
		SessionManager: sessionManager,
	}
}
