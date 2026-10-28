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

package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
)

func TestUserHandlerCreateUser(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.CreateUserRequest{Username: "alice", Password: "secret", IsAdmin: true}

	t.Run("creates an administrator", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().GetUserByName(ctx, "alice").Return(nil, errors.New("not found"))
		repo.EXPECT().CreateUser(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, account *user.User) error {
			require.Equal(t, "alice", account.Username)
			require.Equal(t, "secret", account.Password)
			account.ID = 42
			return nil
		})
		repo.EXPECT().SetUserSysAdmin(ctx, 42, true).Return(nil)

		response, err := (&UserHandler{userRepo: repo}).CreateUser(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("rejects duplicate username", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().GetUserByName(ctx, "alice").Return(&user.User{ID: 1}, nil)

		response, err := (&UserHandler{userRepo: repo}).CreateUser(ctx, request)
		require.Nil(t, response)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestUserHandlerCRUD(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Unix(1_700_000_000, 0)

	t.Run("gets a user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Username: "alice", CreatedAt: createdAt}, nil)

		response, err := (&UserHandler{userRepo: repo}).GetUser(ctx, &v1alpha1.GetUserRequest{Id: 42})
		require.NoError(t, err)
		require.Equal(t, uint32(42), response.Id)
		require.Equal(t, "alice", response.Username)
		require.Equal(t, createdAt.Unix(), response.CreatedAt.AsTime().Unix())
	})

	t.Run("deletes a user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().DeleteUser(ctx, 42).Return(nil)

		response, err := (&UserHandler{userRepo: repo}).DeleteUser(ctx, &v1alpha1.DeleteUserRequest{Id: 42})
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("resets a password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().UpdateUserPassword(ctx, 42, "new-secret").Return(nil)

		response, err := (&UserHandler{userRepo: repo}).ResetUserPassword(ctx, &v1alpha1.ResetUserPasswordRequest{Id: 42, Password: "new-secret"})
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}

func TestUserHandlerListUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := usermocks.NewMockIUserRepo(ctrl)
	ctx := context.Background()
	createdAt := time.Unix(1_700_000_000, 0)
	accounts := []*user.User{
		{ID: 1, Username: "alice", CreatedAt: createdAt},
		{ID: 2, Username: "bob", CreatedAt: createdAt},
	}
	repo.EXPECT().ListUsers(ctx, 2, 10, "a", "demo").Return(accounts, int64(12), nil)
	repo.EXPECT().IsUsersSysAdmin(ctx, []int{1, 2}).Return(map[int]bool{1: true}, nil)

	response, err := (&UserHandler{userRepo: repo}).ListUsers(ctx, &v1alpha1.ListUsersRequest{
		Page: 2, PageSize: 10, Search: "a", ExcludeProject: "demo",
	})
	require.NoError(t, err)
	require.Len(t, response.Users, 2)
	require.True(t, response.Users[0].IsAdmin)
	require.False(t, response.Users[1].IsAdmin)
	require.Equal(t, int32(2), response.Pagination.Pages)
}

func TestUserHandlerSetUserSysAdminRequiresPermission(t *testing.T) {
	ctx := context.Background()

	t.Run("updates when authorized", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		authzService := mocks.NewMockIAuthzService(ctrl)
		authzService.EXPECT().VerifyPlatformPermission(ctx, role.UserAuthorize).Return(true, nil)
		repo.EXPECT().SetUserSysAdmin(ctx, 42, true).Return(nil)

		response, err := (&UserHandler{userRepo: repo, authzService: authzService}).SetUserSysAdmin(ctx, &v1alpha1.SetUserSysAdminRequest{Id: 42, SysadminFlag: true})
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("denies without permission", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		authzService := mocks.NewMockIAuthzService(ctrl)
		authzService.EXPECT().VerifyPlatformPermission(ctx, role.UserAuthorize).Return(false, nil)

		response, err := (&UserHandler{userRepo: repo, authzService: authzService}).SetUserSysAdmin(ctx, &v1alpha1.SetUserSysAdminRequest{Id: 42, SysadminFlag: true})
		require.Nil(t, response)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}
