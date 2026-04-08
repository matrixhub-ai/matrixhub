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

package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func HFAuthenticationMiddleware(akRepo user.IAccessTokenRepo, sessionRepo user.ISessionRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawToken, ok := parseBearerToken(r)
			if ok {
				ak, err := akRepo.GetByTokenHash(r.Context(), utils.Sha256Hex(rawToken))
				if err != nil {
					log.Errorf("fail to get ak by secret: %s", err)
				}
				if ak != nil && ak.IsValid(time.Now()) {
					r = setUserInfo(r, ak.UserId)
				}
				next.ServeHTTP(w, r)
				return
			}

			token, ok := parseCookie(r)
			if ok {
				ctx, err := sessionRepo.Load(r.Context(), token)
				if err == nil {
					if sessionRepo.Exists(ctx, user.UsernameCtxKey.String()) {
						r = setUserInfo(r, sessionRepo.GetInt(ctx, user.UserIdCtxKey.String()))
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func setUserInfo(r *http.Request, userId int) *http.Request {
	r = r.WithContext(authenticate.WithContext(r.Context(), authenticate.UserInfo{
		User: strconv.Itoa(userId),
	}))
	r = r.WithContext(context.WithValue(r.Context(), user.UserIdCtxKey, userId))
	return r
}

func parseBearerToken(r *http.Request) (string, bool) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return "", false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	return token, token != ""
}

func parseCookie(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(user.CookieName)
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}
