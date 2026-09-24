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
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

// normalizeIdentity decodes the authcodec user hfd keeps as a built-in identity into a Principal.
func normalizeIdentity(ctx context.Context) (context.Context, bool) {
	identity := authenticate.IdentityFrom(ctx)
	if _, ok := identity.(middleware.Principal); ok || authenticate.IsAnonymous(identity) {
		return ctx, true
	}
	decoded, err := authcodec.Unmarshal(identity.Name())
	if err != nil {
		return ctx, false
	}
	return authenticate.WithIdentity(ctx, middleware.Principal{Identity: decoded}), true
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

func normalizePermissionHook(hook permission.PermissionHookFunc) permission.PermissionHookFunc {
	return func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error) {
		ctx, ok := normalizeIdentity(ctx)
		if !ok {
			return false, nil
		}
		return hook(ctx, op, repoName, opCtx)
	}
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
