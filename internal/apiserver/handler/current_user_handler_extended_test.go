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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
)

func currentUserContext() context.Context {
	return auth.WithIdentity(context.Background(), user.NewUserIdentity(42, "alice"))
}

func TestCurrentUserHandlerGetCurrentUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := usermocks.NewMockIUserRepo(ctrl)
	ctx := currentUserContext()
	repo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Username: "alice"}, nil)
	repo.EXPECT().IsUserSysAdmin(ctx, 42).Return(true, nil)

	response, err := (&CurrentUserHandler{userRepo: repo}).GetCurrentUser(ctx, &v1alpha1.GetCurrentUserRequest{})
	require.NoError(t, err)
	require.Equal(t, uint32(42), response.Id)
	require.Equal(t, "alice", response.Username)
	require.True(t, response.IsAdmin)
}

func TestCurrentUserHandlerResetPassword(t *testing.T) {
	hash, err := crypto.HashPassword("old-password")
	require.NoError(t, err)
	ctx := currentUserContext()

	t.Run("updates a changed password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Password: hash}, nil)
		repo.EXPECT().UpdateUserPassword(ctx, 42, "new-password").Return(nil)

		response, err := (&CurrentUserHandler{userRepo: repo}).ResetPassword(ctx, &v1alpha1.ResetPasswordRequest{
			OldPassword: "old-password", NewPassword: "new-password",
		})
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("rejects an incorrect old password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Password: hash}, nil)

		response, err := (&CurrentUserHandler{userRepo: repo}).ResetPassword(ctx, &v1alpha1.ResetPasswordRequest{
			OldPassword: "wrong-password", NewPassword: "new-password",
		})
		require.Nil(t, response)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("rejects password reuse", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIUserRepo(ctrl)
		repo.EXPECT().GetUser(ctx, 42).Return(&user.User{ID: 42, Password: hash}, nil)

		response, err := (&CurrentUserHandler{userRepo: repo}).ResetPassword(ctx, &v1alpha1.ResetPasswordRequest{
			OldPassword: "old-password", NewPassword: "old-password",
		})
		require.Nil(t, response)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestCurrentUserHandlerAccessTokens(t *testing.T) {
	ctx := currentUserContext()
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	t.Run("lists token status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIAccessTokenRepo(ctrl)
		repo.EXPECT().ListUserAccessTokens(ctx, 42).Return([]*user.AccessToken{
			{Id: 1, Name: "never", CreatedAt: now},
			{Id: 2, Name: "expired", CreatedAt: now, ExpireAt: &past},
			{Id: 3, Name: "valid", CreatedAt: now, ExpireAt: &future},
		}, nil)

		response, err := (&CurrentUserHandler{accessTokenRepo: repo}).ListAccessTokens(ctx, &v1alpha1.ListAccessTokensRequest{})
		require.NoError(t, err)
		require.Len(t, response.Items, 3)
		require.Equal(t, v1alpha1.AccessTokenStatus_ACCESS_TOKEN_STATUS_VALID, response.Items[0].Status)
		require.Equal(t, v1alpha1.AccessTokenStatus_ACCESS_TOKEN_STATUS_EXPIRED, response.Items[1].Status)
		require.Equal(t, v1alpha1.AccessTokenStatus_ACCESS_TOKEN_STATUS_VALID, response.Items[2].Status)
		require.Equal(t, strconv.FormatInt(past.Unix(), 10), response.Items[1].ExpiredAt)
	})

	t.Run("retries a token hash collision", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIAccessTokenRepo(ctrl)
		expireAt := now.Add(24 * time.Hour).Truncate(time.Second)
		gomock.InOrder(
			repo.EXPECT().CreateAccessToken(ctx, gomock.Any()).Return(gorm.ErrDuplicatedKey),
			repo.EXPECT().CreateAccessToken(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, token user.AccessToken) error {
				require.Equal(t, "ci", token.Name)
				require.Equal(t, 42, token.UserId)
				require.True(t, token.Enabled)
				require.Equal(t, expireAt, *token.ExpireAt)
				require.NotEmpty(t, token.TokenHash)
				return nil
			}),
		)

		response, err := (&CurrentUserHandler{accessTokenRepo: repo}).CreateAccessToken(ctx, &v1alpha1.CreateAccessTokenRequest{
			Name: "ci", ExpireAt: strconv.FormatInt(expireAt.Unix(), 10),
		})
		require.NoError(t, err)
		require.NotEmpty(t, response.Token)
	})

	t.Run("deletes only the current user's token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockIAccessTokenRepo(ctrl)
		repo.EXPECT().DeleteAccessToken(ctx, 42, 7).Return(nil)

		response, err := (&CurrentUserHandler{accessTokenRepo: repo}).DeleteAccessToken(ctx, &v1alpha1.DeleteAccessTokenRequest{Id: 7})
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}

func TestCurrentUserHandlerSSHKeys(t *testing.T) {
	ctx := currentUserContext()
	now := time.Now()
	past := now.Add(-time.Hour)

	t.Run("lists key status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		repo.EXPECT().ListSSHKeys(ctx, 42).Return([]*user.SSHKey{
			{Id: 1, Name: "active", CreatedAt: now},
			{Id: 2, Name: "expired", CreatedAt: now, ExpireAt: &past},
		}, nil)

		response, err := (&CurrentUserHandler{sshKeyRepo: repo}).ListSSHKeys(ctx, &v1alpha1.ListSSHKeysRequest{})
		require.NoError(t, err)
		require.Len(t, response.Items, 2)
		require.Equal(t, v1alpha1.SSHKeyStatus_SSH_KEY_STATUS_VALID, response.Items[0].Status)
		require.Equal(t, v1alpha1.SSHKeyStatus_SSH_KEY_STATUS_EXPIRED, response.Items[1].Status)
	})

	t.Run("deletes only the current user's key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		repo.EXPECT().DeleteSSHKey(ctx, 42, 7).Return(nil)

		response, err := (&CurrentUserHandler{sshKeyRepo: repo}).DeleteSSHKey(ctx, &v1alpha1.DeleteSSHKeyRequest{Id: 7})
		require.NoError(t, err)
		require.NotNil(t, response)
	})
}

func TestCurrentUserHandlerGetProjectRoles(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := usermocks.NewMockIUserRepo(ctrl)
	ctx := currentUserContext()
	repo.EXPECT().GetUserAllProjectRoles(ctx, 42).Return(map[string]int{
		"admin-project":  int(role.ProjectRoleAdmin),
		"viewer-project": int(role.ProjectRoleViewer),
	}, nil)

	response, err := (&CurrentUserHandler{userRepo: repo}).GetProjectRoles(ctx, &v1alpha1.GetProjectRolesRequest{})
	require.NoError(t, err)
	require.Equal(t, v1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_ADMIN, response.ProjectRoles["admin-project"])
	require.Equal(t, v1alpha1.ProjectRoleType_ROLE_TYPE_PROJECT_VIEWER, response.ProjectRoles["viewer-project"])
}

func TestCurrentUserHandlerParseExpireAt(t *testing.T) {
	handler := &CurrentUserHandler{}

	got, err := handler.parseExpireAt("")
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = handler.parseExpireAt("1700000000")
	require.NoError(t, err)
	require.Equal(t, int64(1_700_000_000), got.Unix())

	got, err = handler.parseExpireAt("not-a-timestamp")
	require.Error(t, err)
	require.Nil(t, got)
}

func TestCurrentUserHandlerPropagatesRepositoryFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := usermocks.NewMockIUserRepo(ctrl)
	ctx := currentUserContext()
	repo.EXPECT().GetUser(ctx, 42).Return(nil, errors.New("database unavailable"))

	response, err := (&CurrentUserHandler{userRepo: repo}).GetCurrentUser(ctx, &v1alpha1.GetCurrentUserRequest{})
	require.Nil(t, response)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
