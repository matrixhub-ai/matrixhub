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
	"strings"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// TokenAuthenticator
// Use cases : HF CLI, Git HTTP, LFS HTTP
type TokenAuthenticator struct {
	tokenRepo user.IAccessTokenRepo
}

func NewTokenAuthenticator(tokenRepo user.IAccessTokenRepo) *TokenAuthenticator {
	return &TokenAuthenticator{tokenRepo: tokenRepo}
}

func (a *TokenAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*Identity, error) {
	token := extractTokenCredential(r)
	if token == "" {
		return nil, nil
	}
	ak, err := a.tokenRepo.GetByTokenHash(r.Context(), utils.Sha256Hex(token))
	if err != nil {
		return nil, err
	}
	if ak != nil && ak.IsValid(time.Now()) {
		return &Identity{
			UserId: ak.UserId,
			Via:    MethodToken,
		}, nil
	}

	return nil, errors.New("invalid token")
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
func extractTokenCredential(r *http.Request) (token string) {
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
