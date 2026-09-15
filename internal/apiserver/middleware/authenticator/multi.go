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

type MultiAuthenticator struct {
	authenticators []HTTPAuthenticator
}

func NewMultiAuthenticator(auths ...HTTPAuthenticator) *MultiAuthenticator {
	return &MultiAuthenticator{
		authenticators: auths,
	}
}

func (m *MultiAuthenticator) Authenticate(ctx context.Context, r *http.Request) (succeeded HTTPAuthenticator, identity auth.Identity, next, ok bool, err error) {
	for _, auth := range m.authenticators {
		identity, next, ok, err := auth.Authenticate(ctx, r)
		if err != nil {
			return nil, nil, false, false, err
		}
		if ok {
			return auth, identity, false, true, nil
		}
		if !next {
			return nil, nil, false, false, nil
		}
	}

	return nil, nil, true, false, nil
}

func (m *MultiAuthenticator) AuthenticateToken(ctx context.Context, username, token string) (succeeded HTTPAuthenticator, identity auth.Identity, next, ok bool, err error) {
	for _, auth := range m.authenticators {
		identity, next, ok, err := auth.AuthenticateToken(ctx, username, token)
		if err != nil {
			return nil, nil, false, false, err
		}
		if ok {
			return auth, identity, false, true, nil
		}
		if !next {
			return nil, nil, false, false, nil
		}
	}

	return nil, nil, true, false, nil
}
