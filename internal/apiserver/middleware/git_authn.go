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
	"crypto/sha256"
	"encoding/base64"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware/authenticator"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

// Principal adapts a matrixhub identity to hfd's authenticate.Identity.
type Principal struct{ auth.Identity }

func (p Principal) Name() string  { return p.GetName() }
func (p Principal) Email() string { return "" }

func authenticationResult(identity auth.Identity, next, ok bool, err error) (authenticate.Identity, error) {
	if err != nil {
		return nil, err
	}
	if ok {
		return Principal{identity}, nil
	}
	if next {
		return nil, nil
	}
	return nil, authenticate.ErrUnauthenticated
}

func GitHTTPAuthn(akRepo user.IAccessTokenRepo, userRepo user.IUserRepo, robotRepo robot.IRobotRepo) authenticate.TokenValidatorFunc {
	return func(ctx context.Context, token string) (authenticate.Identity, error) {
		auth := authenticator.NewGitAuthenticator(akRepo, userRepo, robotRepo)
		_, identity, next, ok, err := auth.AuthenticateToken(ctx, "", token)
		return authenticationResult(identity, next, ok, err)
	}
}

func GitBasicAuthAuthn(akRepo user.IAccessTokenRepo, userRepo user.IUserRepo, robotRepo robot.IRobotRepo) authenticate.BasicAuthValidatorFunc {
	return func(ctx context.Context, username, password string) (authenticate.Identity, error) {
		auth := authenticator.NewGitAuthenticator(akRepo, userRepo, robotRepo)
		_, identity, next, ok, err := auth.AuthenticateToken(ctx, username, password)
		return authenticationResult(identity, next, ok, err)
	}
}

func GitPublicKeyAuthn(sshKeyRepo user.ISSHKeyRepo, userRepo user.IUserRepo) authenticate.PublicKeyValidatorFunc {
	return func(ctx context.Context, username string, keyType string, marshaledKey []byte) (authenticate.Identity, error) {
		auth := authenticator.NewSSHKeyAuthenticator(sshKeyRepo, userRepo)
		sha256sum := sha256.Sum256(marshaledKey)
		hash := base64.RawStdEncoding.EncodeToString(sha256sum[:])
		fg := "SHA256:" + hash
		identity, next, ok, err := auth.Authenticate(ctx, fg)
		return authenticationResult(identity, next, ok, err)
	}
}
