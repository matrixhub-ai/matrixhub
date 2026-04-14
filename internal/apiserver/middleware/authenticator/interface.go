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
)

type Identity struct {
	UserId   int
	Username string
	Via      AuthMethod // records which auth method was used, useful for logging and auditing
}

type AuthMethod string

const (
	MethodCookie   AuthMethod = "cookie"
	MethodToken    AuthMethod = "token"
	MethodSSHKey   AuthMethod = "ssh_key"
	MethodPassword AuthMethod = "password"
)

type HTTPAuthenticator interface {
	// Authenticate attempts to extract credentials from the request and verify them.
	//
	// Return semantics:
	//   (nil, nil)      — this method does not apply to this request; skip and try the next one
	//   (nil, err)      — credentials were present but invalid; reject immediately, do not try further
	//   (identity, nil) — authentication succeeded
	Authenticate(ctx context.Context, r *http.Request) (*Identity, error)
}
