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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type LoginHandler struct {
	userService user.IUserService
}

func (l *LoginHandler) Login(ctx context.Context, request *v1alpha1.LoginRequest) (*v1alpha1.LoginResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := l.userService.LoginUser(ctx, request.Username, request.Password, request.RememberMe)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return &v1alpha1.LoginResponse{}, nil
}

func (l *LoginHandler) Logout(ctx context.Context, _ *v1alpha1.LogoutRequest) (*v1alpha1.LogoutResponse, error) {
	err := l.userService.LogoutUser(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1alpha1.LogoutResponse{}, nil
}

func (l *LoginHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterLoginServer(options.GRPCServer, l)
	if err := v1alpha1.RegisterLoginHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewLoginHandler(userService user.IUserService) IHandler {
	handler := &LoginHandler{
		userService: userService,
	}

	return handler
}
