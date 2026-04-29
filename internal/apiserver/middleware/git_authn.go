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
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

func GitHTTPAuthn(akRepo user.IAccessTokenRepo, robotRepo robot.IRobotRepo) authenticate.TokenValidatorFunc {
	return func(ctx context.Context, token string) (user string, next, ok bool, err error) {
		auth := authenticator.NewGitAuthenticator(akRepo, robotRepo)
		_, identity, err := auth.AuthenticateToken(ctx, "", token)
		if err != nil {
			return "", false, false, err
		}
		id, err := authcodec.Marshal(identity)
		if err != nil {
			return "", false, false, err
		}
		return id, true, true, nil
	}
}

func GitBasicAuthAuthn(akRepo user.IAccessTokenRepo, robotRepo robot.IRobotRepo) authenticate.BasicAuthValidatorFunc {
	return func(ctx context.Context, username, password string) (user string, next, ok bool, err error) {
		auth := authenticator.NewGitAuthenticator(akRepo, robotRepo)
		_, identity, err := auth.AuthenticateToken(ctx, username, password)
		if err != nil {
			return "", false, false, err
		}
		id, err := authcodec.Marshal(identity)
		if err != nil {
			return "", false, false, err
		}
		return id, true, true, nil
	}
}

func GitPublicKeyAuthn(sshKeyRepo user.ISSHKeyRepo, userRepo user.IUserRepo) authenticate.PublicKeyValidatorFunc {
	return func(ctx context.Context, username string, keyType string, marshaledKey []byte) (user string, next, ok bool, err error) {
		auth := authenticator.NewSSHKeyAuthenticator(sshKeyRepo, userRepo)
		sha256sum := sha256.Sum256(marshaledKey)
		hash := base64.RawStdEncoding.EncodeToString(sha256sum[:])
		fg := "SHA256:" + hash
		identity, err := auth.Authenticate(ctx, fg)
		if err != nil || identity == nil || identity.GetID() == 0 {
			return "", false, false, err
		}
		id, err := authcodec.Marshal(identity)
		if err != nil {
			return "", false, false, err
		}
		return id, true, true, nil
	}
}
