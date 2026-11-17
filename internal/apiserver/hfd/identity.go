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
	"fmt"
	"net/http"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

// gitAuth holds hfd's credential validators; they name callers by their authcodec-encoded identity.
type gitAuth struct {
	basicAuthValidator authenticate.BasicAuthValidator
	publicKeyValidator authenticate.PublicKeyValidator
	tokenValidator     authenticate.TokenValidator
	tokenSignValidator authenticate.TokenSignValidator
}

func (b *Backend) initGitAuth(
	akRepo user.IAccessTokenRepo,
	userRepo user.IUserRepo,
	robotRepo robot.IRobotRepo,
	sshKeyRepo user.ISSHKeyRepo,
) {
	b.auth.basicAuthValidator = middleware.GitBasicAuthAuthn(akRepo, userRepo, robotRepo)
	b.auth.publicKeyValidator = middleware.GitPublicKeyAuthn(sshKeyRepo, userRepo)
	b.auth.tokenValidator = middleware.GitHTTPAuthn(akRepo, userRepo, robotRepo)
}

// authenticateHTTP tries HF CLI credentials before hfd's validators, which add signed LFS tokens and answer rejected credentials with 401.
func (b *Backend) authenticateHTTP(next http.Handler) http.Handler {
	handler := authenticate.NewHandler(
		authenticate.WithNext(normalizeHTTPIdentity(next)),
		authenticate.WithBasicAuthValidator(b.auth.basicAuthValidator),
		authenticate.WithTokenValidator(b.auth.tokenValidator),
		authenticate.WithTokenSignValidator(b.auth.tokenSignValidator),
	)
	return middleware.HFAuthnMiddleware(b.akRepo, b.sessionRepo, b.userRepo, b.robotRepo)(handler)
}

// normalizeIdentity makes the caller a Principal, decoding the authcodec name hfd's validators set, and attaches its domain identity.
func normalizeIdentity(ctx context.Context) (context.Context, bool) {
	identity := authenticate.IdentityFrom(ctx)
	principal, ok := identity.(middleware.Principal)
	if !ok {
		if authenticate.IsAnonymous(identity) {
			return ctx, true
		}
		decoded, err := authcodec.Unmarshal(identity.Name())
		if err != nil {
			return ctx, false
		}
		principal = middleware.Principal{Identity: decoded}
		ctx = authenticate.WithIdentity(ctx, principal)
	}
	return auth.WithIdentity(ctx, principal.Identity), true
}

func normalizeHTTPIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx, ok := normalizeIdentity(request.Context())
		if !ok {
			http.Error(writer, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// tokenSignValidator preserves matrixhub identity types and IDs in signed LFS tokens.
type tokenSignValidator struct {
	inner authenticate.TokenSignValidator
}

func newTokenSignValidator(key []byte) authenticate.TokenSignValidator {
	return &tokenSignValidator{inner: authenticate.NewTokenSignValidator(key)}
}

func (validator *tokenSignValidator) Sign(ctx context.Context, method, path, username string, expiration time.Duration) (string, error) {
	if username == "" || username == authenticate.AnonymousName {
		return "", nil // anonymous LFS download links carry no token
	}
	subject, err := signedSubject(ctx, username)
	if err != nil {
		return "", err
	}
	return validator.inner.Sign(ctx, method, path, subject, expiration)
}

// signedSubject encodes the identity hfd names: the ctx Principal over HTTP, the encoded user over SSH.
func signedSubject(ctx context.Context, username string) (string, error) {
	if principal, ok := authenticate.IdentityFrom(ctx).(middleware.Principal); ok && principal.Name() == username {
		return authcodec.Marshal(principal.Identity)
	}
	if _, err := authcodec.Unmarshal(username); err != nil {
		return "", fmt.Errorf("cannot sign token for identity %q", username)
	}
	return username, nil
}

func (validator *tokenSignValidator) Validate(ctx context.Context, method, path, token string) (string, bool, bool, error) {
	subject, next, ok, err := validator.inner.Validate(ctx, method, path, token)
	if err != nil || !ok {
		return "", next, false, err
	}
	if _, err := authcodec.Unmarshal(subject); err != nil {
		return "", false, false, nil
	}
	return subject, false, true, nil
}
