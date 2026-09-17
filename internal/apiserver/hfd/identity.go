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
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

// tokenSignValidator preserves matrixhub identity types and IDs in signed LFS tokens.
type tokenSignValidator struct {
	inner authenticate.TokenSignValidator
}

func newTokenSignValidator(key []byte) authenticate.TokenSignValidator {
	return &tokenSignValidator{inner: authenticate.NewTokenSignValidator(key)}
}

func (validator *tokenSignValidator) Sign(ctx context.Context, method, path string, id authenticate.Identity, expiration time.Duration) (string, error) {
	if authenticate.IsAnonymous(id) {
		return "", nil // anonymous LFS download links carry no token
	}
	principal, ok := id.(middleware.Principal)
	if !ok {
		return "", fmt.Errorf("cannot sign token for %T identity", id)
	}
	blob, err := authcodec.Marshal(principal.Identity)
	if err != nil {
		return "", err
	}
	return validator.inner.Sign(ctx, method, path, authenticate.NewIdentity(blob, ""), expiration)
}

func (validator *tokenSignValidator) Validate(ctx context.Context, method, path, token string) (authenticate.Identity, error) {
	id, err := validator.inner.Validate(ctx, method, path, token)
	if err != nil || id == nil {
		return nil, err
	}
	identity, err := authcodec.Unmarshal(id.Name())
	if err != nil {
		return nil, authenticate.ErrUnauthenticated
	}
	return middleware.Principal{Identity: identity}, nil
}
