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
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

func TestTokenSignValidatorRoundTrip(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	for _, principal := range []middleware.Principal{
		{Identity: user.NewUserIdentity(42, "alice")},
		{Identity: robot.NewRobotIdentity(7, "bot")},
	} {
		t.Run(principal.TypeName(), func(t *testing.T) {
			ctx := context.Background()
			token, err := validator.Sign(ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", principal, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			identity, err := validator.Validate(ctx, http.MethodPost, "/p/n.git/info/lfs/objects/batch", token)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := identity.(middleware.Principal)
			if !ok {
				t.Fatalf("identity type = %T, want middleware.Principal", identity)
			}
			if got.GetID() != principal.GetID() || got.Name() != principal.Name() || got.TypeName() != principal.TypeName() {
				t.Errorf("identity = %+v, want %+v", got, principal)
			}
		})
	}
}

func TestTokenSignValidatorRejectsPlainIdentity(t *testing.T) {
	ctx := context.Background()
	validator := newTokenSignValidator([]byte("secret"))
	token, err := authenticate.NewTokenSignValidator([]byte("secret")).Sign(
		ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", authenticate.NewIdentity("alice", ""), time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := validator.Validate(ctx, http.MethodPost, "/p/n.git/info/lfs/objects/batch", token)
	if identity != nil || !errors.Is(err, authenticate.ErrUnauthenticated) {
		t.Fatalf("Validate = (%v, %v), want (nil, ErrUnauthenticated)", identity, err)
	}
}

func TestTokenSignValidatorIgnoresUnsignedToken(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	identity, err := validator.Validate(context.Background(), http.MethodPost, "/p/n.git/info/lfs/objects/batch", "mh_whatever")
	if identity != nil || err != nil {
		t.Fatalf("Validate = (%v, %v), want (nil, nil)", identity, err)
	}
}

func TestTokenSignValidatorRejectsDifferentKey(t *testing.T) {
	ctx := context.Background()
	principal := middleware.Principal{Identity: user.NewUserIdentity(42, "alice")}
	token, err := newTokenSignValidator([]byte("other-secret")).Sign(
		ctx, http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", principal, time.Hour,
	)
	if err != nil {
		t.Fatal(err)
	}
	validator := newTokenSignValidator([]byte("secret"))
	identity, err := validator.Validate(ctx, http.MethodPost, "/p/n.git/info/lfs/objects/batch", token)
	if identity != nil || !errors.Is(err, authenticate.ErrUnauthenticated) {
		t.Fatalf("Validate = (%v, %v), want (nil, ErrUnauthenticated)", identity, err)
	}
}

func TestTokenSignValidatorCannotSignForeignIdentity(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	token, err := validator.Sign(
		context.Background(), http.MethodPost, "http://host/p/n.git/info/lfs/objects/batch", authenticate.NewIdentity("alice", ""), time.Hour,
	)
	if token != "" || err == nil {
		t.Fatalf("Sign = (%q, %v), want empty token and error", token, err)
	}
}

func TestTokenSignValidatorSkipsAnonymous(t *testing.T) {
	validator := newTokenSignValidator([]byte("secret"))
	token, err := validator.Sign(
		context.Background(), http.MethodGet, "http://host/objects/abc", authenticate.Anonymous, time.Hour,
	)
	if token != "" || err != nil {
		t.Fatalf("Sign = (%q, %v), want empty token without error", token, err)
	}
}
