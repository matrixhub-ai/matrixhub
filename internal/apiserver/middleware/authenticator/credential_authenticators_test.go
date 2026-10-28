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
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	robotmocks "github.com/matrixhub-ai/matrixhub/internal/domain/robot/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestTokenAuthenticator(t *testing.T) {
	ctx := context.Background()
	token := utils.TokenPrefix + "test-secret"

	t.Run("authenticates a valid bearer token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		tokenRepo := usermocks.NewMockIAccessTokenRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		tokenRepo.EXPECT().GetByTokenHash(ctx, utils.Sha256Hex(token)).Return(&user.AccessToken{UserId: 42, Enabled: true}, nil)
		userRepo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Username: "alice"}, nil)
		request := httptest.NewRequest("GET", "/", nil)
		request.Header.Set("Authorization", "Bearer "+token)

		identity, next, ok, err := NewTokenAuthenticator(tokenRepo, userRepo).Authenticate(ctx, request)
		require.NoError(t, err)
		require.False(t, next)
		require.True(t, ok)
		require.Equal(t, 42, identity.GetID())
		require.Equal(t, "alice", identity.GetName())
	})

	t.Run("skips an unrelated credential", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		tokenRepo := usermocks.NewMockIAccessTokenRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)

		identity, next, ok, err := NewTokenAuthenticator(tokenRepo, userRepo).AuthenticateToken(ctx, "", "not-a-matrixhub-token")
		require.NoError(t, err)
		require.Nil(t, identity)
		require.True(t, next)
		require.False(t, ok)
	})

	t.Run("rejects an expired token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		tokenRepo := usermocks.NewMockIAccessTokenRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		expired := time.Now().Add(-time.Minute)
		tokenRepo.EXPECT().GetByTokenHash(ctx, utils.Sha256Hex(token)).Return(&user.AccessToken{UserId: 42, Enabled: true, ExpireAt: &expired}, nil)

		identity, next, ok, err := NewTokenAuthenticator(tokenRepo, userRepo).AuthenticateToken(ctx, "", token)
		require.NoError(t, err)
		require.Nil(t, identity)
		require.False(t, next)
		require.False(t, ok)
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		tokenRepo := usermocks.NewMockIAccessTokenRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		wantErr := errors.New("database unavailable")
		tokenRepo.EXPECT().GetByTokenHash(ctx, utils.Sha256Hex(token)).Return(nil, wantErr)

		identity, next, ok, err := NewTokenAuthenticator(tokenRepo, userRepo).AuthenticateToken(ctx, "", token)
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, identity)
		require.False(t, next)
		require.False(t, ok)
	})
}

func TestSSHKeyAuthenticator(t *testing.T) {
	ctx := context.Background()
	fingerprint := "SHA256:test"

	t.Run("authenticates an active key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		keyRepo := usermocks.NewMockISSHKeyRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		keyRepo.EXPECT().GetByFingerprint(ctx, fingerprint).Return(&user.SSHKey{Id: 1, UserId: 42}, nil)
		userRepo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Username: "alice"}, nil)

		identity, next, ok, err := NewSSHKeyAuthenticator(keyRepo, userRepo).Authenticate(ctx, fingerprint)
		require.NoError(t, err)
		require.False(t, next)
		require.True(t, ok)
		require.Equal(t, "alice", identity.GetName())
	})

	t.Run("rejects an expired key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		keyRepo := usermocks.NewMockISSHKeyRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		expired := time.Now().Add(-time.Minute)
		keyRepo.EXPECT().GetByFingerprint(ctx, fingerprint).Return(&user.SSHKey{Id: 1, UserId: 42, ExpireAt: &expired}, nil)

		identity, next, ok, err := NewSSHKeyAuthenticator(keyRepo, userRepo).Authenticate(ctx, fingerprint)
		require.NoError(t, err)
		require.Nil(t, identity)
		require.False(t, next)
		require.False(t, ok)
	})
}

func TestRobotAuthenticators(t *testing.T) {
	ctx := context.Background()
	token := utils.RobotTokenPrefix + "Abcdefg1"
	rb := &robot.Robot{ID: 7, Name: "robot$ci", Enabled: true, TokenHash: utils.Sha256Hex(token)}

	t.Run("basic credentials", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := robotmocks.NewMockIRobotRepo(ctrl)
		repo.EXPECT().GetRobotByName(ctx, rb.Name).Return(rb, nil)

		identity, next, ok, err := NewRobotAuthenticator(repo).AuthenticateToken(ctx, rb.Name, token)
		require.NoError(t, err)
		require.False(t, next)
		require.True(t, ok)
		require.Equal(t, 7, identity.GetID())
	})

	t.Run("bearer token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := robotmocks.NewMockIRobotRepo(ctrl)
		repo.EXPECT().GetRobotByTokenHash(ctx, utils.Sha256Hex(token)).Return(rb, nil)

		identity, next, ok, err := NewRobotTokenAuthenticator(repo).AuthenticateToken(ctx, "", token)
		require.NoError(t, err)
		require.False(t, next)
		require.True(t, ok)
		require.Equal(t, "robot$ci", identity.GetName())
	})
}
