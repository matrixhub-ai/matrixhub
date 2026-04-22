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

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware/authenticator"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

var publicMethods = map[string]bool{
	"/matrixhub.v1alpha1.Login/Login": true,
}

func AuthInterceptor(sessionRepo user.ISessionRepo) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		auth := authenticator.NewWebAuthenticator(sessionRepo)
		succeeded, identity, err := auth.Authenticate(ctx, nil)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
		}

		if renewer, ok := succeeded.(authenticator.SessionRenewer); ok {
			defer func() {
				if err = renewer.Renew(ctx); err != nil {
					log.Errorf("failed to renew session: %s", err)
				}
			}()
		}

		ctx = context.WithValue(ctx, user.IdentityKey{}, identity)

		return handler(ctx, req)
	}
}
