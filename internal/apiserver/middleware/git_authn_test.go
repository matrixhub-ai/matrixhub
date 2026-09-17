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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestGitAuthRejectsInvalidCredentials(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	for _, test := range []struct {
		name   string
		token  string
		access *user.AccessToken
		robot  *robot.Robot
		err    error
		status int
	}{
		{"unrecognized", "forged", nil, nil, nil, http.StatusOK},
		{"deleted", utils.TokenPrefix + "deleted", &user.AccessToken{}, nil, nil, http.StatusUnauthorized},
		{"expired", utils.TokenPrefix + "expired", &user.AccessToken{Enabled: true, ExpireAt: &expired}, nil, nil, http.StatusUnauthorized},
		{"missing robot", utils.RobotTokenPrefix + "missing", nil, nil, nil, http.StatusUnauthorized},
		{"disabled robot", utils.RobotTokenPrefix + "disabled", nil, &robot.Robot{}, nil, http.StatusUnauthorized},
		{"storage failure", utils.TokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), http.StatusInternalServerError},
		{"robot storage failure", utils.RobotTokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), http.StatusInternalServerError},
		{"valid", utils.TokenPrefix + "valid", &user.AccessToken{Enabled: true, UserId: 42}, nil, nil, http.StatusOK},
	} {
		for _, scheme := range []string{"basic", "bearer"} {
			t.Run(test.name+"/"+scheme, func(t *testing.T) {
				accessRepo := &gitAuthAccessRepo{token: test.access, err: test.err}
				robotRepo := &gitAuthRobotRepo{robot: test.robot, err: test.err}
				userRepo := &gitAuthUserRepo{}
				if test.name == "valid" {
					userRepo.user = &user.User{Username: "alice"}
				}
				reached := false
				next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					reached = true
					if test.status != http.StatusOK {
						t.Error("invalid credential reached the next handler")
					}
					identity := authenticate.IdentityFrom(r.Context())
					if test.name == "valid" {
						principal, ok := identity.(Principal)
						if !ok {
							t.Fatalf("identity = %T, want Principal", identity)
						}
						if principal.GetID() != 42 || principal.Name() != "alice" || principal.TypeName() != "user" || principal.Email() != "" {
							t.Errorf("principal = (%d, %q, %q, %q), want (42, alice, user, empty)",
								principal.GetID(), principal.Name(), principal.TypeName(), principal.Email())
						}
					} else if !authenticate.IsAnonymous(identity) {
						t.Errorf("identity = %v, want anonymous", identity)
					}
					w.WriteHeader(http.StatusOK)
				})
				handler := authenticate.BasicAuthHandler(GitBasicAuthAuthn(accessRepo, userRepo, robotRepo),
					authenticate.TokenValidatorHandler(GitHTTPAuthn(accessRepo, userRepo, robotRepo),
						authenticate.AnonymousAuthenticateHandler(next)))
				request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
				if scheme == "basic" {
					request.SetBasicAuth("alice", test.token)
				} else {
					request.Header.Set("Authorization", "Bearer "+test.token)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if test.status == http.StatusOK && !reached {
					t.Error("credential did not reach the next handler")
				}
				if response.Code != test.status {
					t.Errorf("status = %d, want %d", response.Code, test.status)
				}
			})
		}
	}
}

func TestGitPublicKeyAuthnRejectsUnknownKey(t *testing.T) {
	for _, test := range []struct {
		name string
		key  *user.SSHKey
		err  error
	}{
		{"unknown", &user.SSHKey{}, nil},
		{"storage failure", nil, errors.New("db")},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &gitAuthSSHKeyRepo{key: test.key, err: test.err}
			identity, err := GitPublicKeyAuthn(repo, nil)(context.Background(), "alice", "ssh-ed25519", []byte("key"))
			wantErr := test.err
			if wantErr == nil {
				wantErr = authenticate.ErrUnauthenticated
			}
			if identity != nil || !errors.Is(err, wantErr) {
				t.Errorf("got (%v, %v), want (nil, %v)", identity, err, wantErr)
			}
		})
	}
}

type gitAuthSSHKeyRepo struct {
	user.ISSHKeyRepo
	key *user.SSHKey
	err error
}

type gitAuthUserRepo struct {
	user.IUserRepo
	user *user.User
}

func (repo *gitAuthUserRepo) GetUser(context.Context, int) (*user.User, error) {
	if repo == nil {
		return nil, nil
	}
	return repo.user, nil
}

func (repo *gitAuthSSHKeyRepo) GetByFingerprint(context.Context, string) (*user.SSHKey, error) {
	return repo.key, repo.err
}

type gitAuthAccessRepo struct {
	user.IAccessTokenRepo
	token *user.AccessToken
	err   error
}

func (repo *gitAuthAccessRepo) GetByTokenHash(context.Context, string) (*user.AccessToken, error) {
	return repo.token, repo.err
}

type gitAuthRobotRepo struct {
	robot.IRobotRepo
	robot *robot.Robot
	err   error
}

func (repo *gitAuthRobotRepo) GetRobotByTokenHash(context.Context, string) (*robot.Robot, error) {
	return repo.robot, repo.err
}
