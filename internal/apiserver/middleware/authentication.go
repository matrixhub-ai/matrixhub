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
	"net/http"

	"github.com/alexedwards/scs/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

var publicMethods = map[string]bool{
	"/matrixhub.v1alpha1.Login/Login": true,
}

func AuthInterceptor(sessionRepo user.ISessionRepo) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		token := getTokenFromContext(ctx, sessionRepo)
		ctx, err := sessionRepo.Load(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
		}
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		cookieStatus := sessionRepo.Status(ctx)
		if cookieStatus != scs.Unmodified && cookieStatus != scs.Modified {
			return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
		}

		// set username in context
		ctx = context.WithValue(ctx, user.UsernameCtxKey, sessionRepo.GetString(ctx, user.UsernameCtxKey.String()))

		return handler(ctx, req)
	}
}

func getTokenFromContext(ctx context.Context, session user.ISessionRepo) (token string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	sessionCookie := session.GetSessionCookie()
	cookieHeader := firstMD(md, "grpcgateway-cookie", "cookie")
	if cookieHeader == "" {
		return
	}

	return extractSessionToken(cookieHeader, sessionCookie.Name)
}

func extractSessionToken(cookieHeader, name string) string {
	h := http.Header{}
	h.Add("Cookie", cookieHeader)
	c, err := (&http.Request{Header: h}).Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func firstMD(md metadata.MD, keys ...string) string {
	for _, k := range keys {
		if vs := md.Get(k); len(vs) > 0 && vs[0] != "" {
			return vs[0]
		}
	}
	return ""
}
