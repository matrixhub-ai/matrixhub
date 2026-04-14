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

package authenticator

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc/metadata"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

// CookieAuthenticator
// Use cases : web login
type CookieAuthenticator struct {
	sessionRepo user.ISessionRepo
	cookieName  string
}

func NewCookieAuthenticator(sessionRepo user.ISessionRepo) HTTPAuthenticator {
	return &CookieAuthenticator{sessionRepo: sessionRepo, cookieName: user.CookieName}
}

// Authenticate extract cookie from context or from http.Request
func (a *CookieAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*Identity, error) {
	if !a.sessionRepo.Exists(ctx, user.UserIdCtxKey.String()) {
		return nil, errors.New("authentication invalid")
	}

	return &Identity{
		UserId:   a.sessionRepo.GetInt(ctx, user.UserIdCtxKey.String()),
		Username: a.sessionRepo.GetString(ctx, user.UsernameCtxKey.String()),
		Via:      MethodCookie,
	}, nil
}

func GetCookieFromContext(ctx context.Context) (token string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
	cookieHeader := firstMD(md, "grpcgateway-cookie", "cookie")
	if cookieHeader == "" {
		return
	}

	return extractSessionToken(cookieHeader, user.CookieName)
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
