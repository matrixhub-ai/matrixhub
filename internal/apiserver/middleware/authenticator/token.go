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
	"net/http"
	"strings"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// TokenAuthenticator
// Use cases : HF CLI, Git HTTP, LFS HTTP
type TokenAuthenticator struct {
	tokenRepo user.IAccessTokenRepo
	userRepo  user.IUserRepo
}

func NewTokenAuthenticator(tokenRepo user.IAccessTokenRepo, userRepo user.IUserRepo) *TokenAuthenticator {
	return &TokenAuthenticator{tokenRepo: tokenRepo, userRepo: userRepo}
}

func (a *TokenAuthenticator) Authenticate(ctx context.Context, r *http.Request) (auth.Identity, bool, bool, error) {
	token := extractTokenCredential(ctx, r)
	return a.AuthenticateToken(ctx, "", token)
}

func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, _, token string) (auth.Identity, bool, bool, error) {
	if token == "" || !strings.HasPrefix(token, utils.TokenPrefix) || len(token) == len(utils.TokenPrefix) {
		return nil, true, false, nil
	}
	ak, err := a.tokenRepo.GetByTokenHash(ctx, utils.Sha256Hex(token))
	if err != nil {
		return nil, false, false, err
	}
	if ak == nil || !ak.IsValid(time.Now()) {
		return nil, false, false, nil
	}
	u, err := a.userRepo.GetUser(ctx, ak.UserId)
	if err != nil {
		return nil, false, false, err
	}

	return user.NewUserIdentity(ak.UserId, u.Username), false, true, nil
}

func parseBearerToken(r *http.Request) (token string) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth == "" {
		return
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return
	}
	token = strings.TrimSpace(parts[1])
	return
}

// extractTokenCredential extracts a token from the request.
func extractTokenCredential(ctx context.Context, r *http.Request) (token string) {
	var err error
	token, err = utils.ParseBearerTokenFromGRPCContext(ctx)
	if err == nil && token != "" {
		return token
	}
	if r == nil {
		return
	}
	token = parseBearerToken(r)
	if token != "" {
		return
	}

	_, password, ok := r.BasicAuth()
	if !ok {
		return
	}
	return password
}
