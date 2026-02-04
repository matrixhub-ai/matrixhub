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

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	userv1alpha1 "matrixhub.ai/api/protogen/v1alpha1/user"
)

type UserHandler struct {
	userRepo user.IUserRepository
}

func (u *UserHandler) CreateUser(ctx context.Context, request *userv1alpha1.CreateUserRequest) (*userv1alpha1.CreateUserResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := u.userRepo.CreateUser(ctx, user.User{
		Username: request.Username,
		Password: request.Password,
		Email:    request.Email,
	}); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &userv1alpha1.CreateUserResponse{}, nil
}

func (u *UserHandler) GetUser(ctx context.Context, request *userv1alpha1.GetUserRequest) (*userv1alpha1.GetUserResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	user, err := u.userRepo.GetUser(ctx, request.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &userv1alpha1.GetUserResponse{
		Id:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (u *UserHandler) DeleteUser(ctx context.Context, request *userv1alpha1.DeleteUserRequest) (*userv1alpha1.DeleteUserResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := u.userRepo.DeleteUser(ctx, request.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &userv1alpha1.DeleteUserResponse{}, nil
}

func (u *UserHandler) ListUsers(ctx context.Context, request *userv1alpha1.ListUsersRequest) (*userv1alpha1.ListUsersResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	users, total, err := u.userRepo.ListUsers(ctx, int(request.Page), int(request.PageSize), request.Search)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	result := lo.Map(users, func(item *user.User, index int) *userv1alpha1.User {
		return &userv1alpha1.User{
			Id:       item.ID,
			Username: item.Username,
			Email:    item.Email,
		}
	})

	return &userv1alpha1.ListUsersResponse{
		Users: result,
		Pagination: &userv1alpha1.Pagination{
			Total:    uint32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
		},
	}, nil

}

func (u *UserHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	userv1alpha1.RegisterUsersServer(options.GRPCServer, u)
	if err := userv1alpha1.RegisterUsersHandlerServer(context.Background(), options.GatewayMux, u); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewUserHandler(repo user.IUserRepository) IHandler {
	handler := &UserHandler{
		userRepo: repo,
	}

	return handler
}
