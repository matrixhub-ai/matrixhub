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

package hfd

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func encodedIdentity(t *testing.T, identity auth.Identity) string {
	t.Helper()
	encoded, err := authcodec.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestGitAuthValidatorsPreservePrincipal(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	for _, test := range []struct {
		name     string
		token    string
		access   *user.AccessToken
		robot    *robot.Robot
		err      error
		status   int
		identity auth.Identity
	}{
		{"unrecognized", "forged", nil, nil, nil, http.StatusOK, nil},
		{"deleted", utils.TokenPrefix + "deleted", &user.AccessToken{}, nil, nil, http.StatusUnauthorized, nil},
		{"expired", utils.TokenPrefix + "expired", &user.AccessToken{Enabled: true, ExpireAt: &expired}, nil, nil, http.StatusUnauthorized, nil},
		{"missing robot", utils.RobotTokenPrefix + "missing", nil, nil, nil, http.StatusUnauthorized, nil},
		{"disabled robot", utils.RobotTokenPrefix + "disabled", nil, &robot.Robot{}, nil, http.StatusUnauthorized, nil},
		{"storage failure", utils.TokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), http.StatusInternalServerError, nil},
		{"robot storage failure", utils.RobotTokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), http.StatusInternalServerError, nil},
		{"valid user", utils.TokenPrefix + "valid", &user.AccessToken{Enabled: true, UserId: 42}, nil, nil, http.StatusOK, user.NewUserIdentity(42, "alice")},
		{"valid robot", utils.RobotTokenPrefix + "valid", nil, &robot.Robot{ID: 7, Name: "bot", Enabled: true}, nil, http.StatusOK, robot.NewRobotIdentity(7, "bot")},
	} {
		for _, scheme := range []string{"basic", "bearer"} {
			t.Run(test.name+"/"+scheme, func(t *testing.T) {
				repos := &gitAuthRepos{token: test.access, robot: test.robot, err: test.err}
				backend := &Backend{}
				backend.initGitAuth(repos, repos, repos, repos)
				reached := false
				next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					reached = true
					identity := authenticate.IdentityFrom(request.Context())
					if test.identity == nil {
						if !authenticate.IsAnonymous(identity) {
							t.Errorf("identity = %v, want anonymous", identity)
						}
						return
					}
					assertPrincipal(t, identity, test.identity)
				})
				handler := authenticate.BasicAuthHandler(backend.auth.basicAuthValidator,
					authenticate.TokenValidatorHandler(backend.auth.tokenValidator,
						authenticate.AnonymousAuthenticateHandler(normalizeHTTPIdentity(next))))
				request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
				if scheme == "basic" {
					request.SetBasicAuth("alice", test.token)
				} else {
					request.Header.Set("Authorization", "Bearer "+test.token)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != test.status {
					t.Errorf("status = %d, want %d", response.Code, test.status)
				}
				if reached != (test.status == http.StatusOK) {
					t.Errorf("reached next handler = %t, want %t", reached, test.status == http.StatusOK)
				}
			})
		}
	}
}

func TestGitAuthPublicKeyValidator(t *testing.T) {
	for _, test := range []struct {
		name string
		key  *user.SSHKey
		err  error
		want auth.Identity
	}{
		{"unknown", &user.SSHKey{}, nil, nil},
		{"storage failure", nil, errors.New("db"), nil},
		{"valid", &user.SSHKey{Id: 1, UserId: 42}, nil, user.NewUserIdentity(42, "alice")},
	} {
		t.Run(test.name, func(t *testing.T) {
			repos := &gitAuthRepos{key: test.key, err: test.err}
			backend := &Backend{}
			backend.initGitAuth(repos, repos, repos, repos)
			encoded, next, ok, err := backend.auth.publicKeyValidator.Validate(context.Background(), "alice", "ssh-ed25519", []byte("key"))
			if !errors.Is(err, test.err) || next || ok != (test.want != nil) {
				t.Fatalf("Validate = (%q, %t, %t, %v), want (next false, ok %t, err %v)", encoded, next, ok, err, test.want != nil, test.err)
			}
			if test.want == nil {
				if encoded != "" {
					t.Fatalf("user = %q, want empty", encoded)
				}
				return
			}
			// The SSH server keeps the returned user as a built-in identity.
			ctx, ok := normalizeIdentity(authenticate.WithIdentity(context.Background(), authenticate.NewIdentity(encoded, "")))
			if !ok {
				t.Fatalf("normalizeIdentity rejected validator user %q", encoded)
			}
			assertPrincipal(t, authenticate.IdentityFrom(ctx), test.want)
		})
	}
}

func TestNormalizeIdentity(t *testing.T) {
	principal := middleware.Principal{Identity: user.NewUserIdentity(42, "alice")}
	for _, test := range []struct {
		name     string
		identity authenticate.Identity
		ok       bool
		want     auth.Identity
	}{
		{"principal", principal, true, principal.Identity},
		{"missing", nil, true, nil},
		{"anonymous", authenticate.Anonymous, true, nil},
		{"encoded user", authenticate.NewIdentity(encodedIdentity(t, user.NewUserIdentity(42, "alice")), ""), true, user.NewUserIdentity(42, "alice")},
		{"encoded robot", authenticate.NewIdentity(encodedIdentity(t, robot.NewRobotIdentity(7, "bot")), ""), true, robot.NewRobotIdentity(7, "bot")},
		{"plain name", authenticate.NewIdentity("alice", ""), false, nil},
		{"invalid blob", authenticate.NewIdentity("{", ""), false, nil},
		{"unknown type", authenticate.NewIdentity(`{"type":"admin","payload":{}}`, ""), false, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.identity != nil {
				ctx = authenticate.WithIdentity(ctx, test.identity)
			}
			normalized, ok := normalizeIdentity(ctx)
			if ok != test.ok {
				t.Fatalf("ok = %t, want %t", ok, test.ok)
			}
			if !ok {
				return
			}
			identity := authenticate.IdentityFrom(normalized)
			if test.want == nil {
				if !authenticate.IsAnonymous(identity) {
					t.Fatalf("identity = %v, want anonymous", identity)
				}
				return
			}
			assertPrincipal(t, identity, test.want)
		})
	}
}

func TestBindNormalizesPermissionHookIdentity(t *testing.T) {
	backend := &Backend{}
	backend.Bind(nil, nil, handlerAuthzService{}, nil, nil, nil, nil, nil, nil, nil)
	for _, test := range []struct {
		name     string
		identity authenticate.Identity
		repo     string
		passed   bool
	}{
		{"encoded member", authenticate.NewIdentity(encodedIdentity(t, user.NewUserIdentity(1, "alice")), ""), "priv/model", true},
		{"encoded other user", authenticate.NewIdentity(encodedIdentity(t, user.NewUserIdentity(2, "bob")), ""), "priv/model", false},
		{"principal member", middleware.Principal{Identity: user.NewUserIdentity(1, "alice")}, "priv/model", true},
		{"anonymous public", nil, "pub/model", true},
		{"anonymous private", nil, "priv/model", false},
		{"plain name public", authenticate.NewIdentity("alice", ""), "pub/model", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			if test.identity != nil {
				ctx = authenticate.WithIdentity(ctx, test.identity)
			}
			passed, err := backend.permissionHookFunc(ctx, permission.OperationReadRepo, test.repo, permission.Context{})
			if err != nil || passed != test.passed {
				t.Fatalf("hook = (%t, %v), want (%t, nil)", passed, err, test.passed)
			}
		})
	}
}

func assertPrincipal(t *testing.T, identity authenticate.Identity, want auth.Identity) {
	t.Helper()
	if want == nil {
		if identity != nil {
			t.Fatalf("identity = %#v, want nil", identity)
		}
		return
	}
	principal, ok := identity.(middleware.Principal)
	if !ok {
		t.Fatalf("identity = %T, want middleware.Principal", identity)
	}
	if principal.GetID() != want.GetID() || principal.Name() != want.GetName() || principal.TypeName() != want.TypeName() || principal.Email() != "" {
		t.Errorf("principal = (%d, %q, %q, %q), want (%d, %q, %q, empty)",
			principal.GetID(), principal.Name(), principal.TypeName(), principal.Email(),
			want.GetID(), want.GetName(), want.TypeName())
	}
}

type gitAuthRepos struct {
	user.IAccessTokenRepo
	user.IUserRepo
	user.ISSHKeyRepo
	robot.IRobotRepo
	token *user.AccessToken
	robot *robot.Robot
	key   *user.SSHKey
	err   error
}

func (repos *gitAuthRepos) GetByTokenHash(context.Context, string) (*user.AccessToken, error) {
	return repos.token, repos.err
}

func (repos *gitAuthRepos) GetRobotByTokenHash(context.Context, string) (*robot.Robot, error) {
	return repos.robot, repos.err
}

func (repos *gitAuthRepos) GetByFingerprint(context.Context, string) (*user.SSHKey, error) {
	return repos.key, repos.err
}

func (repos *gitAuthRepos) GetUser(context.Context, int) (*user.User, error) {
	return &user.User{Username: "alice"}, nil
}

func TestTokenSignValidatorRoundTrip(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	for _, identity := range []auth.Identity{user.NewUserIdentity(42, "alice"), robot.NewRobotIdentity(7, "bot")} {
		encoded := encodedIdentity(t, identity)
		// hfd passes IdentityFrom(ctx).Name(): the display name over HTTP, the encoded user over SSH.
		for _, source := range []struct {
			name     string
			identity authenticate.Identity
			username string
		}{
			{"http", middleware.Principal{Identity: identity}, identity.GetName()},
			{"ssh", authenticate.NewIdentity(encoded, ""), encoded},
		} {
			t.Run(identity.TypeName()+"/"+source.name, func(t *testing.T) {
				ctx := authenticate.WithIdentity(context.Background(), source.identity)
				token, err := validator.Sign(ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", source.username, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				subject, next, ok, err := validator.Validate(context.Background(), http.MethodPost, "/p/n.git/info/lfs/objects/batch", token)
				if err != nil || next || !ok {
					t.Fatalf("Validate = (%q, %t, %t, %v), want (subject, false, true, nil)", subject, next, ok, err)
				}
				normalized, ok := normalizeIdentity(authenticate.WithIdentity(context.Background(), authenticate.NewIdentity(subject, "")))
				if !ok {
					t.Fatalf("normalizeIdentity rejected validated subject %q", subject)
				}
				assertPrincipal(t, authenticate.IdentityFrom(normalized), identity)
			})
		}
	}
}

func TestTokenSignValidatorRejectsPlainSubject(t *testing.T) {
	ctx := context.Background()
	validator := newTokenSignValidator([]byte("secret"))
	token, err := authenticate.NewTokenSignValidator([]byte("secret")).Sign(
		ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", "alice", time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	subject, next, ok, err := validator.Validate(ctx, http.MethodPost, "/p/n.git/info/lfs/objects/batch", token)
	if subject != "" || next || ok || err != nil {
		t.Fatalf("Validate = (%q, %t, %t, %v), want (\"\", false, false, nil)", subject, next, ok, err)
	}
}

func TestTokenSignValidatorIgnoresUnsignedToken(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	subject, next, ok, err := validator.Validate(context.Background(), http.MethodPost, "/p/n.git/info/lfs/objects/batch", "mh_whatever")
	if subject != "" || !next || ok || err != nil {
		t.Fatalf("Validate = (%q, %t, %t, %v), want (\"\", true, false, nil)", subject, next, ok, err)
	}
}

func TestTokenSignValidatorRejectsDifferentKey(t *testing.T) {
	ctx := authenticate.WithIdentity(context.Background(), middleware.Principal{Identity: user.NewUserIdentity(42, "alice")})
	token, err := newTokenSignValidator([]byte("other-secret")).Sign(
		ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", "alice", time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	validator := newTokenSignValidator([]byte("secret"))
	subject, next, ok, err := validator.Validate(ctx, http.MethodPost, "/p/n.git/info/lfs/objects/batch", token)
	if subject != "" || next || ok || err != nil {
		t.Fatalf("Validate = (%q, %t, %t, %v), want (\"\", false, false, nil)", subject, next, ok, err)
	}
}

func TestTokenSignValidatorCannotSignForeignIdentity(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	for _, test := range []struct {
		name     string
		identity authenticate.Identity
		username string
	}{
		{"plain name", authenticate.NewIdentity("alice", ""), "alice"},
		{"principal mismatch", middleware.Principal{Identity: user.NewUserIdentity(42, "alice")}, "mallory"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := authenticate.WithIdentity(context.Background(), test.identity)
			token, err := validator.Sign(ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", test.username, time.Hour)
			if token != "" || err == nil {
				t.Fatalf("Sign = (%q, %v), want empty token and error", token, err)
			}
		})
	}
}

func TestTokenSignValidatorSkipsAnonymous(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	ctx := authenticate.WithIdentity(context.Background(), authenticate.Anonymous)
	token, err := validator.Sign(ctx, http.MethodGet, "http://host/objects/abc", authenticate.AnonymousName, time.Hour)
	if token != "" || err != nil {
		t.Fatalf("Sign = (%q, %v), want empty token without error", token, err)
	}
}
