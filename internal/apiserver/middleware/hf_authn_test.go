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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	robotmocks "github.com/matrixhub-ai/matrixhub/internal/domain/robot/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func TestHFAuthnMiddlewareSetsAuthenticatedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	tokenRepo := usermocks.NewMockIAccessTokenRepo(ctrl)
	sessionRepo := usermocks.NewMockISessionRepo(ctrl)
	userRepo := usermocks.NewMockIUserRepo(ctrl)
	robotRepo := robotmocks.NewMockIRobotRepo(ctrl)
	token := utils.TokenPrefix + "test-secret"
	tokenRepo.EXPECT().GetByTokenHash(gomock.Any(), utils.Sha256Hex(token)).Return(&user.AccessToken{UserId: 42, Enabled: true}, nil)
	userRepo.EXPECT().GetUser(gomock.Any(), 42).Return(&user.User{ID: 42, Username: "alice"}, nil)

	next := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		identity, ok := auth.IdentityFromContext(request.Context())
		require.True(t, ok)
		require.Equal(t, "alice", identity.GetName())
		info, ok := authenticate.GetUserInfo(request.Context())
		require.True(t, ok)
		decoded, err := authcodec.Unmarshal(info.User)
		require.NoError(t, err)
		require.Equal(t, "alice", decoded.GetName())
		w.WriteHeader(http.StatusNoContent)
	})
	handler := HFAuthnMiddleware(tokenRepo, sessionRepo, userRepo, robotRepo)(next)
	request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestSetUserInfoPreservesRequestWhenIdentityCannotBeEncoded(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	got := setUserInfo(request, invalidIdentity{Bad: make(chan int)})

	_, ok := authenticate.GetUserInfo(got.Context())
	require.False(t, ok)
}

type invalidIdentity struct {
	Bad chan int
}

func (invalidIdentity) GetID() int       { return 0 }
func (invalidIdentity) GetName() string  { return "invalid" }
func (invalidIdentity) TypeName() string { return "unsupported" }

var _ auth.Identity = invalidIdentity{}
