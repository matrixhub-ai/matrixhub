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
	"net/http/httptest"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

func TestIdentityNormalizerDecodesBlob(t *testing.T) {
	blob, err := authcodec.Marshal(user.NewUserIdentity(42, "alice"))
	if err != nil {
		t.Fatalf("marshal identity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(authenticate.WithContext(req.Context(), authenticate.UserInfo{
		User:  blob,
		Email: "alice@example.com",
	}))

	var called bool
	identityNormalizer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		identity, ok := auth.IdentityFromContext(r.Context())
		if !ok {
			t.Fatal("expected identity in context")
		}
		if identity.GetID() != 42 || identity.GetName() != "alice" || identity.TypeName() != "user" {
			t.Errorf("unexpected identity: %+v", identity)
		}
		userInfo, ok := authenticate.GetUserInfo(r.Context())
		if !ok {
			t.Fatal("expected user info in context")
		}
		if userInfo.User != "alice" {
			t.Errorf("expected plain username, got %q", userInfo.User)
		}
		if userInfo.Email != "alice@example.com" {
			t.Errorf("expected email preserved, got %q", userInfo.Email)
		}
	})).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler not called")
	}
}

func TestIdentityNormalizerPassesThroughNonBlob(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(authenticate.WithContext(req.Context(), authenticate.UserInfo{
		User: authenticate.Anonymous,
	}))

	var called bool
	identityNormalizer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := auth.IdentityFromContext(r.Context()); ok {
			t.Error("expected no identity in context")
		}
		userInfo, ok := authenticate.GetUserInfo(r.Context())
		if !ok {
			t.Fatal("expected user info in context")
		}
		if userInfo.User != authenticate.Anonymous {
			t.Errorf("expected user info unchanged, got %q", userInfo.User)
		}
	})).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler not called")
	}
}

func TestIdentityNormalizerPassesThroughWithoutUserInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var called bool
	identityNormalizer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := auth.IdentityFromContext(r.Context()); ok {
			t.Error("expected no identity in context")
		}
		if _, ok := authenticate.GetUserInfo(r.Context()); ok {
			t.Error("expected no user info in context")
		}
	})).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler not called")
	}
}

// Covers non-HTTP call paths (SSH sessions, signed LFS tokens) where only the
// blob-carrying UserInfo reaches the permission hook.
func TestNormalizePermissionHookDecodesBlob(t *testing.T) {
	blob, err := authcodec.Marshal(user.NewUserIdentity(42, "alice"))
	if err != nil {
		t.Fatalf("marshal identity: %v", err)
	}
	ctx := authenticate.WithContext(context.Background(), authenticate.UserInfo{User: blob})

	var called bool
	hook := normalizePermissionHook(func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error) {
		called = true
		identity, ok := auth.IdentityFromContext(ctx)
		if !ok {
			t.Fatal("expected identity in context")
		}
		if identity.GetID() != 42 || identity.GetName() != "alice" {
			t.Errorf("unexpected identity: %+v", identity)
		}
		userInfo, ok := authenticate.GetUserInfo(ctx)
		if !ok || userInfo.User != "alice" {
			t.Errorf("expected plain username in user info, got %+v", userInfo)
		}
		return true, nil
	})

	passed, err := hook(ctx, permission.OperationReadRepo, "models/proj/repo", permission.Context{})
	if err != nil || !passed {
		t.Fatalf("hook returned (%v, %v), want (true, nil)", passed, err)
	}
	if !called {
		t.Fatal("wrapped hook not called")
	}
	// The caller's context must keep the blob: hfd's ssh backend embeds it in
	// signed LFS tokens after the permission check.
	userInfo, _ := authenticate.GetUserInfo(ctx)
	if userInfo.User != blob {
		t.Errorf("expected caller context to keep blob, got %q", userInfo.User)
	}
}

func TestNormalizePermissionHookNilKeepsAllowAllContract(t *testing.T) {
	hook := normalizePermissionHook(nil)
	if hook != nil {
		t.Fatal("expected nil hook to stay nil")
	}
	if err := hook.Check(context.Background(), permission.OperationReadRepo, "models/proj/repo", permission.Context{}); err != nil {
		t.Fatalf("expected nil hook Check to allow, got %v", err)
	}
}

func TestNormalizeIdentityContextKeepsExistingIdentity(t *testing.T) {
	identity := user.NewUserIdentity(7, "bob")
	ctx := auth.WithIdentity(context.Background(), identity)
	// A decodable blob for a different identity must NOT be re-decoded once an
	// identity is already present.
	otherBlob, err := authcodec.Marshal(user.NewUserIdentity(99, "mallory"))
	if err != nil {
		t.Fatalf("marshal identity: %v", err)
	}
	ctx = authenticate.WithContext(ctx, authenticate.UserInfo{User: otherBlob})

	got := normalizeIdentityContext(ctx)

	gotIdentity, ok := auth.IdentityFromContext(got)
	if !ok || gotIdentity.GetID() != 7 || gotIdentity.GetName() != "bob" {
		t.Fatalf("expected existing identity preserved, got %+v", gotIdentity)
	}
	userInfo, _ := authenticate.GetUserInfo(got)
	if userInfo.User != otherBlob {
		t.Errorf("expected user info unchanged, got %q", userInfo.User)
	}
}
