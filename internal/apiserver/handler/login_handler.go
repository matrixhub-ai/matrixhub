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
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type LoginHandler struct {
}

func (l *LoginHandler) Login(ctx context.Context, request *v1alpha1.LoginRequest) (*v1alpha1.LoginResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (l *LoginHandler) Logout(ctx context.Context, request *v1alpha1.LogoutRequest) (*v1alpha1.LogoutResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (l *LoginHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterLoginServer(options.GRPCServer, l)
	if err := v1alpha1.RegisterLoginHandlerServer(context.Background(), options.GatewayMux, l); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewLoginHandler() IHandler {
	handler := &LoginHandler{}

	return handler
}
