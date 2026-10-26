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
	"net/http"
	"net/url"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware/authenticator"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

// SecuritySessionMiddleware bridges the existing web session to the opt-in
// security REST endpoints. Explicit API credentials keep their existing HF
// authentication path; browser session writes require a same-origin request.
func SecuritySessionMiddleware(sessionRepo user.ISessionRepo, userRepo user.IUserRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/security/v1alpha1/") || r.Header.Get("Authorization") != "" {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(user.CookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			identity, _, ok, err := authenticator.NewCookieAuthenticator(sessionRepo, userRepo).AuthenticateToken(r.Context(), "", cookie.Value)
			if err != nil || !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead && !securitySameOrigin(r) {
				http.Error(w, "same-origin request required", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, setUserInfo(r, identity))
		})
	}
}

func securitySameOrigin(r *http.Request) bool {
	origin, err := url.Parse(r.Header.Get("Origin"))
	return err == nil && (origin.Scheme == "http" || origin.Scheme == "https") &&
		origin.User == nil && origin.Path == "" && origin.RawQuery == "" && origin.Fragment == "" &&
		strings.EqualFold(origin.Host, r.Host) && r.Header.Get("Sec-Fetch-Site") != "cross-site"
}
