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
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestHFAuthnMiddlewarePublishesPrincipal(t *testing.T) {
	accessRepo := &gitAuthAccessRepo{token: &user.AccessToken{Enabled: true, UserId: 42}}
	userRepo := &gitAuthUserRepo{user: &user.User{Username: "alice"}}
	sessionRepo := &hfAuthnSessionRepo{manager: scs.New()}
	var hfdIdentity authenticate.Identity
	var domainIdentity auth.Identity
	handler := HFAuthnMiddleware(accessRepo, sessionRepo, userRepo, &gitAuthRobotRepo{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hfdIdentity = authenticate.IdentityFrom(r.Context())
		domainIdentity, _ = auth.IdentityFromContext(r.Context())
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
	request.Header.Set("Authorization", "Bearer "+utils.TokenPrefix+"valid")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	principal, ok := hfdIdentity.(Principal)
	if !ok || principal.GetID() != 42 || principal.Name() != "alice" {
		t.Fatalf("hfd identity = %#v, want Principal alice/42", hfdIdentity)
	}
	if domainIdentity == nil || domainIdentity.GetID() != 42 {
		t.Fatalf("domain identity = %#v, want ID 42", domainIdentity)
	}

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil))
	if !authenticate.IsAnonymous(hfdIdentity) || domainIdentity != nil {
		t.Fatalf("identities without credentials = (%#v, %#v), want anonymous and none", hfdIdentity, domainIdentity)
	}
}

type hfAuthnSessionRepo struct {
	user.ISessionRepo
	manager *scs.SessionManager
}

func (repo *hfAuthnSessionRepo) Manager() *scs.SessionManager {
	return repo.manager
}
