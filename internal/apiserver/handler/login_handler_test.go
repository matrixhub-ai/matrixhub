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

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
)

func TestLoginHandlerLogin(t *testing.T) {
	ctx := context.Background()
	request := &v1alpha1.LoginRequest{Username: "alice", Password: "secret", RememberMe: true}

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := usermocks.NewMockIUserService(ctrl)
		service.EXPECT().LoginUser(ctx, "alice", "secret", true).Return(nil)

		response, err := (&LoginHandler{userService: service}).Login(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("authentication failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := usermocks.NewMockIUserService(ctrl)
		service.EXPECT().LoginUser(ctx, "alice", "secret", true).Return(errors.New("invalid credentials"))

		response, err := (&LoginHandler{userService: service}).Login(ctx, request)
		require.Nil(t, response)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

func TestLoginHandlerLogout(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := usermocks.NewMockIUserService(ctrl)
		service.EXPECT().LogoutUser(ctx).Return(nil)

		response, err := (&LoginHandler{userService: service}).Logout(ctx, &v1alpha1.LogoutRequest{})
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("session failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		service := usermocks.NewMockIUserService(ctrl)
		service.EXPECT().LogoutUser(ctx).Return(errors.New("session unavailable"))

		response, err := (&LoginHandler{userService: service}).Logout(ctx, &v1alpha1.LogoutRequest{})
		require.Nil(t, response)
		require.Equal(t, codes.Internal, status.Code(err))
	})
}
