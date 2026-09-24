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
	"testing"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestGitAuthReturnsEncodedIdentity(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	for _, test := range []struct {
		name     string
		token    string
		access   *user.AccessToken
		robot    *robot.Robot
		err      error
		next     bool
		identity auth.Identity
	}{
		{"unrecognized", "forged", nil, nil, nil, true, nil},
		{"deleted", utils.TokenPrefix + "deleted", &user.AccessToken{}, nil, nil, false, nil},
		{"expired", utils.TokenPrefix + "expired", &user.AccessToken{Enabled: true, ExpireAt: &expired}, nil, nil, false, nil},
		{"missing robot", utils.RobotTokenPrefix + "missing", nil, nil, nil, false, nil},
		{"disabled robot", utils.RobotTokenPrefix + "disabled", nil, &robot.Robot{}, nil, false, nil},
		{"storage failure", utils.TokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), false, nil},
		{"robot storage failure", utils.RobotTokenPrefix + "unavailable", nil, nil, errors.New("database unavailable"), false, nil},
		{"valid user", utils.TokenPrefix + "valid", &user.AccessToken{Enabled: true, UserId: 42}, nil, nil, false, user.NewUserIdentity(42, "alice")},
		{"valid robot", utils.RobotTokenPrefix + "valid", nil, &robot.Robot{ID: 7, Name: "bot", Enabled: true}, nil, false, robot.NewRobotIdentity(7, "bot")},
	} {
		for _, scheme := range []string{"basic", "bearer"} {
			t.Run(test.name+"/"+scheme, func(t *testing.T) {
				accessRepo := &gitAuthAccessRepo{token: test.access, err: test.err}
				robotRepo := &gitAuthRobotRepo{robot: test.robot, err: test.err}
				userRepo := &gitAuthUserRepo{user: &user.User{Username: "alice"}}
				ctx := context.Background()
				var encoded string
				var next, ok bool
				var err error
				if scheme == "basic" {
					encoded, next, ok, err = GitBasicAuthAuthn(accessRepo, userRepo, robotRepo)(ctx, "alice", test.token)
				} else {
					encoded, next, ok, err = GitHTTPAuthn(accessRepo, userRepo, robotRepo)(ctx, test.token)
				}
				assertGitAuthResult(t, encoded, next, ok, err, test.next, test.identity, test.err)
			})
		}
	}
}

func assertGitAuthResult(t *testing.T, encoded string, next, ok bool, err error, wantNext bool, wantIdentity auth.Identity, wantErr error) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if next != wantNext || ok != (wantIdentity != nil) {
		t.Fatalf("(next, ok) = (%t, %t), want (%t, %t)", next, ok, wantNext, wantIdentity != nil)
	}
	if wantIdentity == nil {
		if encoded != "" {
			t.Fatalf("user = %q, want empty", encoded)
		}
		return
	}
	got, err := authcodec.Unmarshal(encoded)
	if err != nil {
		t.Fatalf("user = %q: %v", encoded, err)
	}
	if got.GetID() != wantIdentity.GetID() || got.GetName() != wantIdentity.GetName() || got.TypeName() != wantIdentity.TypeName() {
		t.Errorf("identity = %+v, want %+v", got, wantIdentity)
	}
}

func TestGitPublicKeyAuthnReturnsEncodedIdentity(t *testing.T) {
	for _, test := range []struct {
		name     string
		key      *user.SSHKey
		err      error
		identity auth.Identity
	}{
		{"unknown", &user.SSHKey{}, nil, nil},
		{"storage failure", nil, errors.New("db"), nil},
		{"valid", &user.SSHKey{Id: 1, UserId: 42}, nil, user.NewUserIdentity(42, "alice")},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &gitAuthSSHKeyRepo{key: test.key, err: test.err}
			userRepo := &gitAuthUserRepo{user: &user.User{Username: "alice"}}
			encoded, next, ok, err := GitPublicKeyAuthn(repo, userRepo)(context.Background(), "alice", "ssh-ed25519", []byte("key"))
			assertGitAuthResult(t, encoded, next, ok, err, false, test.identity, test.err)
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
