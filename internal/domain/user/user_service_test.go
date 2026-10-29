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

package user_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
)

type memorySessionStore struct {
	mu        sync.Mutex
	data      map[string][]byte
	commitErr error
}

func (s *memorySessionStore) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, token)
	return nil
}

func (s *memorySessionStore) Find(token string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.data[token]
	return append([]byte(nil), b...), ok, nil
}

func (s *memorySessionStore) Commit(token string, b []byte, _ time.Time) error {
	if s.commitErr != nil {
		return s.commitErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[token] = append([]byte(nil), b...)
	return nil
}

func newSessionManager(store scs.Store) *scs.SessionManager {
	manager := scs.New()
	manager.Store = store
	return manager
}

func passwordUser(t *testing.T) *user.User {
	t.Helper()
	hash, err := crypto.HashPassword("correct-password")
	require.NoError(t, err)
	return &user.User{ID: 42, Username: "alice", Password: hash, SessionVersion: 3}
}

func TestUserServiceGetCurrentUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessionRepo := usermocks.NewMockISessionRepo(ctrl)
	userRepo := usermocks.NewMockIUserRepo(ctrl)
	manager := newSessionManager(&memorySessionStore{data: make(map[string][]byte)})
	ctx, err := manager.Load(context.Background(), "")
	require.NoError(t, err)
	manager.Put(ctx, user.UsernameCtxKey, "alice")

	sessionRepo.EXPECT().Manager().Return(manager)
	want := &user.User{ID: 42, Username: "alice"}
	userRepo.EXPECT().GetUserByName(ctx, "alice").Return(want, nil)

	got, err := user.NewUserService(sessionRepo, userRepo).GetCurrentUser(ctx)
	require.NoError(t, err)
	require.Same(t, want, got)
}

func TestUserServiceGetCurrentUserRejectsMissingSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessionRepo := usermocks.NewMockISessionRepo(ctrl)
	userRepo := usermocks.NewMockIUserRepo(ctrl)
	manager := newSessionManager(&memorySessionStore{data: make(map[string][]byte)})
	ctx, err := manager.Load(context.Background(), "")
	require.NoError(t, err)
	sessionRepo.EXPECT().Manager().Return(manager)

	got, err := user.NewUserService(sessionRepo, userRepo).GetCurrentUser(ctx)
	require.ErrorIs(t, err, user.InvalidUsernameOrPassword)
	require.Nil(t, got)
}

func TestUserServiceLoginUser(t *testing.T) {
	t.Run("stores the authenticated session", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sessionRepo := usermocks.NewMockISessionRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		manager := newSessionManager(&memorySessionStore{data: make(map[string][]byte)})
		account := passwordUser(t)
		ctx := context.Background()

		userRepo.EXPECT().GetUserByName(ctx, account.Username).Return(account, nil)
		sessionRepo.EXPECT().Manager().Return(manager)
		sessionRepo.EXPECT().WriteSessionCookie(gomock.Any(), gomock.Not(""), gomock.Any()).DoAndReturn(
			func(sessionCtx context.Context, _ string, expiry time.Time) error {
				require.Equal(t, account.ID, manager.GetInt(sessionCtx, user.UserIdCtxKey))
				require.Equal(t, account.Username, manager.GetString(sessionCtx, user.UsernameCtxKey))
				require.Equal(t, account.SessionVersion, manager.GetInt(sessionCtx, user.SessionVersionKey))
				require.True(t, manager.GetBool(sessionCtx, user.RememberMeCtxKey))
				require.False(t, expiry.IsZero())
				return nil
			},
		)

		err := user.NewUserService(sessionRepo, userRepo).LoginUser(ctx, account.Username, "correct-password", true)
		require.NoError(t, err)
	})

	t.Run("hides user lookup failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sessionRepo := usermocks.NewMockISessionRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		ctx := context.Background()
		userRepo.EXPECT().GetUserByName(ctx, "alice").Return(nil, errors.New("database unavailable"))

		err := user.NewUserService(sessionRepo, userRepo).LoginUser(ctx, "alice", "password", false)
		require.ErrorIs(t, err, user.InvalidUsernameOrPassword)
	})

	t.Run("rejects the wrong password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sessionRepo := usermocks.NewMockISessionRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		account := passwordUser(t)
		ctx := context.Background()
		userRepo.EXPECT().GetUserByName(ctx, account.Username).Return(account, nil)

		err := user.NewUserService(sessionRepo, userRepo).LoginUser(ctx, account.Username, "wrong-password", false)
		require.ErrorIs(t, err, user.InvalidUsernameOrPassword)
	})

	t.Run("hides session commit failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sessionRepo := usermocks.NewMockISessionRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		manager := newSessionManager(&memorySessionStore{
			data:      make(map[string][]byte),
			commitErr: errors.New("store unavailable"),
		})
		account := passwordUser(t)
		ctx := context.Background()
		userRepo.EXPECT().GetUserByName(ctx, account.Username).Return(account, nil)
		sessionRepo.EXPECT().Manager().Return(manager)

		err := user.NewUserService(sessionRepo, userRepo).LoginUser(ctx, account.Username, "correct-password", false)
		require.ErrorIs(t, err, user.InvalidUsernameOrPassword)
	})
}

func TestUserServiceLogoutUser(t *testing.T) {
	t.Run("destroys a loaded session", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sessionRepo := usermocks.NewMockISessionRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		manager := newSessionManager(&memorySessionStore{data: make(map[string][]byte)})
		loadedCtx, err := manager.Load(context.Background(), "")
		require.NoError(t, err)
		ctx := context.Background()
		sessionRepo.EXPECT().LoadSession(ctx).Return(loadedCtx, nil)
		sessionRepo.EXPECT().Manager().Return(manager)

		require.NoError(t, user.NewUserService(sessionRepo, userRepo).LogoutUser(ctx))
		require.Equal(t, scs.Destroyed, manager.Status(loadedCtx))
	})

	t.Run("propagates load failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sessionRepo := usermocks.NewMockISessionRepo(ctrl)
		userRepo := usermocks.NewMockIUserRepo(ctrl)
		wantErr := errors.New("load failed")
		ctx := context.Background()
		sessionRepo.EXPECT().LoadSession(ctx).Return(nil, wantErr)

		err := user.NewUserService(sessionRepo, userRepo).LogoutUser(ctx)
		require.ErrorIs(t, err, wantErr)
	})
}

func TestCurrentUserIdentityHelpers(t *testing.T) {
	require.Empty(t, user.GetCurrentUsername(context.Background()))
	require.Zero(t, user.GetCurrentUserId(context.Background()))

	ctx := auth.WithIdentity(context.Background(), user.NewUserIdentity(42, "alice"))
	require.Equal(t, "alice", user.GetCurrentUsername(ctx))
	require.Equal(t, 42, user.GetCurrentUserId(ctx))
}
