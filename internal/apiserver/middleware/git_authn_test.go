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
	} {
		for _, scheme := range []string{"basic", "bearer"} {
			t.Run(test.name+"/"+scheme, func(t *testing.T) {
				accessRepo := &gitAuthAccessRepo{token: test.access, err: test.err}
				robotRepo := &gitAuthRobotRepo{robot: test.robot, err: test.err}
				reached := false
				next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					reached = true
					if test.status != http.StatusOK {
						t.Error("invalid credential reached the next handler")
					}
					info, ok := authenticate.GetUserInfo(r.Context())
					if !ok || info.User != authenticate.Anonymous {
						t.Errorf("user info = %v, present = %v, want anonymous", info, ok)
					}
					w.WriteHeader(http.StatusOK)
				})
				handler := authenticate.BasicAuthHandler(GitBasicAuthAuthn(accessRepo, nil, robotRepo),
					authenticate.TokenValidatorHandler(GitHTTPAuthn(accessRepo, nil, robotRepo),
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
					t.Error("unrecognized credential did not reach the next handler")
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
			identity, next, ok, err := GitPublicKeyAuthn(repo, nil)(context.Background(), "alice", "ssh-ed25519", []byte("key"))
			if identity != "" || next || ok || (err != nil) != (test.err != nil) {
				t.Errorf("got (%q, %v, %v, %v), want empty identity, no next, rejected, error=%v", identity, next, ok, err, test.err)
			}
		})
	}
}

type gitAuthSSHKeyRepo struct {
	user.ISSHKeyRepo
	key *user.SSHKey
	err error
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
