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

package hf

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

func TestHandleWhoami(t *testing.T) {
	t.Run("returns the authenticated user", func(t *testing.T) {
		encoded, err := authcodec.Marshal(user.NewUserIdentity(42, "alice"))
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
		request = request.WithContext(authenticate.WithContext(request.Context(), authenticate.UserInfo{
			User: encoded, Email: "alice@example.com",
		}))
		recorder := httptest.NewRecorder()

		(&Handler{}).handleWhoami(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		var response whoamiResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.Equal(t, "alice", response.Name)
		require.Equal(t, "alice@example.com", response.Email)
		require.Equal(t, "write", response.Auth.AccessToken.Role)
	})

	t.Run("rejects an anonymous request", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/whoami-v2", nil)
		recorder := httptest.NewRecorder()

		(&Handler{}).handleWhoami(recorder, request)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.JSONEq(t, `{"error":"Unauthorized"}`, recorder.Body.String())
	})
}
