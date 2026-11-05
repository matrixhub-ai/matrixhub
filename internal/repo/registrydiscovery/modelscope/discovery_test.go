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

package modelscope_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	"github.com/matrixhub-ai/matrixhub/internal/repo/registrydiscovery/modelscope"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryPaginationAndCredentials(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, "/api/v1/models/", r.URL.Path)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		cookie, err := r.Cookie("m_session_id")
		require.NoError(t, err)
		require.Equal(t, "secret-token", cookie.Value)
		var request struct {
			Path                 string
			PageNumber, PageSize int
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "Qwen", request.Path)
		require.Equal(t, calls, request.PageNumber)
		models := []map[string]string{}
		limit := 100
		if request.PageNumber == 2 {
			limit = 1
		}
		for i := 0; i < limit; i++ {
			models = append(models, map[string]string{"Name": fmt.Sprintf("model-%d-%d", calls, i), "Path": "Qwen"})
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"Code": 200, "Data": map[string]any{"Models": models, "TotalCount": 101}}))
	}))
	defer server.Close()
	reg := &registry.Registry{URL: server.URL, Insecure: true}
	reg.SetCredential(registry.NewBasicCredential("alice", "secret-token"))
	items, err := modelscope.New().ListRepositories(context.Background(), reg, registrydiscovery.Filter{Namespace: "Qwen", ResourceType: "model"})
	require.NoError(t, err)
	require.Len(t, items, 101)
	require.Equal(t, 2, calls)
	require.Equal(t, "model", items[0].ResourceType)
}

func TestDiscoveryRejectsErrorsAndIncompleteListings(t *testing.T) {
	for name, body := range map[string]string{
		"api error":   `{"Code":401,"Message":"denied"}`,
		"bad JSON":    `<html>login</html>`,
		"incomplete":  `{"Code":200,"Data":{"Models":[],"TotalCount":20}}`,
		"unsafe path": `{"Code":200,"Data":{"Models":[{"Name":"../escape","Path":"org"}],"TotalCount":1}}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			_, err := modelscope.New().ListRepositories(context.Background(), &registry.Registry{URL: server.URL}, registrydiscovery.Filter{Namespace: "org", ResourceType: "model"})
			require.Error(t, err)
		})
	}
	_, err := modelscope.New().ListRepositories(context.Background(), &registry.Registry{URL: "http://unused"}, registrydiscovery.Filter{ResourceType: "dataset"})
	require.ErrorContains(t, err, "models only")
}
