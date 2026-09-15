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

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
)

type AuthMethod string

const (
	MethodCookie   AuthMethod = "cookie"
	MethodToken    AuthMethod = "token"
	MethodSSHKey   AuthMethod = "ssh_key"
	MethodPassword AuthMethod = "password"
)

// HTTPAuthenticator verifies request credentials or an extracted token.
// Both methods return (identity, next, ok, err):
//   - (nil, true, false, nil): absent or unrecognized credentials; try the next method.
//   - (identity, false, true, nil): authenticated with a non-nil identity.
//   - (nil, false, false, nil): recognized but rejected credentials; stop.
//   - (nil, false, false, err): infrastructure failure; stop.
type HTTPAuthenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (identity auth.Identity, next, ok bool, err error)
	AuthenticateToken(ctx context.Context, username, token string) (identity auth.Identity, next, ok bool, err error)
}

type SessionRenewer interface {
	Renew(ctx context.Context) error
}
