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
	"net/http"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

// normalizeIdentityContext decodes the authcodec blob that the git/hf
// validators place into the hfd authenticate UserInfo.User, stores the decoded
// identity in the domain auth context, and rewrites UserInfo.User with the
// plain username so downstream hfd code never exposes the encoded blob.
// Contexts that already carry an identity, have no UserInfo, or carry a
// non-blob User are returned unchanged.
func normalizeIdentityContext(ctx context.Context) context.Context {
	if _, ok := auth.IdentityFromContext(ctx); ok {
		return ctx
	}
	userInfo, ok := authenticate.GetUserInfo(ctx)
	if !ok {
		return ctx
	}
	identity, err := authcodec.Unmarshal(userInfo.User)
	if err != nil {
		return ctx
	}
	ctx = auth.WithIdentity(ctx, identity)
	userInfo.User = identity.GetName()
	return authenticate.WithContext(ctx, userInfo)
}

// identityNormalizer applies normalizeIdentityContext to HTTP requests. It is
// assembled after the hfd authenticate handler and before the backends.
func identityNormalizer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(normalizeIdentityContext(r.Context())))
	})
}

// normalizePermissionHook wraps a permission hook with identity normalization
// for call paths that bypass the HTTP middleware chain, such as SSH sessions
// and signed LFS tokens.
func normalizePermissionHook(hook permission.PermissionHookFunc) permission.PermissionHookFunc {
	if hook == nil {
		return nil // preserve hfd's nil-hook (allow-all) contract
	}
	return func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error) {
		return hook(normalizeIdentityContext(ctx), op, repoName, opCtx)
	}
}
