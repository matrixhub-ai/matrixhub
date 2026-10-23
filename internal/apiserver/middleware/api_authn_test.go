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
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	modelv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type authTestSessionRepo struct {
	manager *scs.SessionManager
}

func (r *authTestSessionRepo) GetSessionCookie() scs.SessionCookie  { return scs.SessionCookie{} }
func (r *authTestSessionRepo) GetSessionConfig() user.SessionConfig { return user.SessionConfig{} }
func (r *authTestSessionRepo) LoadSession(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
func (r *authTestSessionRepo) WriteSessionCookie(context.Context, string, time.Time) error {
	return nil
}
func (r *authTestSessionRepo) CommitAndWriteSessionCookie(context.Context) error { return nil }
func (r *authTestSessionRepo) Manager() *scs.SessionManager                      { return r.manager }

type authTestAccessTokenRepo struct {
	user.IAccessTokenRepo
	tokenHash string
	token     *user.AccessToken
}

func (r *authTestAccessTokenRepo) GetByTokenHash(_ context.Context, tokenHash string) (*user.AccessToken, error) {
	if tokenHash != r.tokenHash {
		return nil, nil
	}
	return r.token, nil
}

type authTestUserRepo struct {
	user.IUserRepo
	user *user.User
}

func (r *authTestUserRepo) GetUser(context.Context, int) (*user.User, error) {
	return r.user, nil
}

func TestAuthInterceptorAllowsAnonymousModelList(t *testing.T) {
	interceptor := AuthInterceptor(&authTestSessionRepo{manager: scs.New()}, nil, nil, nil)
	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: modelv1alpha1.Models_ListModels_FullMethodName,
	}, func(context.Context, interface{}) (interface{}, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("expected anonymous model listing to reach the handler, got %v", err)
	}
	if !called {
		t.Fatal("expected anonymous model listing to reach the handler")
	}
}

func TestAuthInterceptorAuthenticatesModelListWhenCredentialsAreProvided(t *testing.T) {
	token := utils.TokenPrefix + "test-token"
	interceptor := AuthInterceptor(
		&authTestSessionRepo{manager: scs.New()},
		&authTestUserRepo{user: &user.User{ID: 42, Username: "test-user"}},
		&authTestAccessTokenRepo{
			tokenHash: utils.Sha256Hex(token),
			token:     &user.AccessToken{UserId: 42, Enabled: true},
		},
		nil,
	)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	called := false
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: modelv1alpha1.Models_ListModels_FullMethodName,
	}, func(ctx context.Context, _ interface{}) (interface{}, error) {
		called = true
		if got := user.GetCurrentUserId(ctx); got != 42 {
			t.Errorf("expected authenticated user ID 42 in handler context, got %d", got)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("expected authenticated model listing to reach the handler, got %v", err)
	}
	if !called {
		t.Fatal("expected authenticated model listing to reach the handler")
	}
}

func TestAuthInterceptorRejectsInvalidCredentialsForModelList(t *testing.T) {
	token := utils.TokenPrefix + "invalid-token"
	interceptor := AuthInterceptor(
		&authTestSessionRepo{manager: scs.New()},
		&authTestUserRepo{user: &user.User{ID: 42, Username: "test-user"}},
		&authTestAccessTokenRepo{tokenHash: utils.Sha256Hex(token)},
		nil,
	)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	called := false
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: modelv1alpha1.Models_ListModels_FullMethodName,
	}, func(context.Context, interface{}) (interface{}, error) {
		called = true
		return nil, nil
	})
	if called {
		t.Fatal("expected invalid credentials to be rejected")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestAuthInterceptorStillRequiresAuthenticationForModelDetails(t *testing.T) {
	interceptor := AuthInterceptor(&authTestSessionRepo{manager: scs.New()}, nil, nil, nil)
	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: modelv1alpha1.Models_GetModel_FullMethodName,
	}, func(context.Context, interface{}) (interface{}, error) {
		called = true
		return nil, nil
	})
	if called {
		t.Fatal("expected unauthenticated model detail request to be rejected")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}
