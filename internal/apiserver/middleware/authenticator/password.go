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

package authenticator

import (
	"context"
	"errors"
	"net/http"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

// PasswordAuthenticator
// Use cases : Git HTTP
// Extraction: Authorization: Basic {base64(user:password)}
// Important : must be registered after TokenAuthenticator and SSHKeyAuthenticator, otherwise tokens and fingerprints
// will be mistakenly treated as passwords.
type PasswordAuthenticator struct {
	userRepo user.IUserRepo
}

func NewPasswordAuthenticator(userRepo user.IUserRepo) *PasswordAuthenticator {
	return &PasswordAuthenticator{userRepo: userRepo}
}

func (a *PasswordAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*Identity, error) {
	username, password, ok := r.BasicAuth()
	if !ok || username == "" || password == "" {
		return nil, nil
	}

	u, err := a.userRepo.GetUserByName(ctx, username)
	if err != nil || u == nil || !u.CheckPassword(password) {
		return nil, errors.New(" invalid credentials")
	}

	return &Identity{
		UserId:   u.ID,
		Username: u.Username,
		Via:      MethodPassword,
	}, nil
}
