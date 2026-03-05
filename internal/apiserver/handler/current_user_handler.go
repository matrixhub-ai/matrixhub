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

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type CurrentUserHandler struct {
	userRepo user.IUserRepo
}

func (cu *CurrentUserHandler) GetCurrentUser(ctx context.Context, request *v1alpha1.GetCurrentUserRequest) (*v1alpha1.GetCurrentUserResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (cu *CurrentUserHandler) ResetPassword(ctx context.Context, request *v1alpha1.ResetPasswordRequest) (*v1alpha1.ResetPasswordResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (cu *CurrentUserHandler) ListAccessTokens(ctx context.Context, request *v1alpha1.ListAccessTokensRequest) (*v1alpha1.ListAccessTokensResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (cu *CurrentUserHandler) CreateAccessToken(ctx context.Context, request *v1alpha1.CreateAccessTokenRequest) (*v1alpha1.CreateAccessTokenResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (cu *CurrentUserHandler) DeleteAccessToken(ctx context.Context, request *v1alpha1.DeleteAccessTokenRequest) (*v1alpha1.DeleteAccessTokenResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (cu *CurrentUserHandler) GetProjectRoles(ctx context.Context, request *v1alpha1.GetProjectRolesRequest) (*v1alpha1.GetProjectRolesResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (cu *CurrentUserHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterCurrentUserServer(options.GRPCServer, cu)
	if err := v1alpha1.RegisterCurrentUserHandlerServer(context.Background(), options.GatewayMux, cu); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewCurrentUserHandler(repo user.IUserRepo) IHandler {
	handler := &CurrentUserHandler{
		userRepo: repo,
	}

	return handler
}
