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
	"testing"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	registrymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registry/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUpdateRegistryRejectsIncompletePut(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request *v1alpha1.UpdateRegistryRequest
	}{
		{name: "missing name", request: &v1alpha1.UpdateRegistryRequest{Id: 7, Url: "https://huggingface.co"}},
		{name: "missing url", request: &v1alpha1.UpdateRegistryRequest{Id: 7, Name: "upstream-hf"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := registrymocks.NewMockIRegistryRepo(gomock.NewController(t))
			h := &RegistryHandler{registryRepo: repo}

			_, err := h.UpdateRegistry(t.Context(), tc.request)

			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.Contains(t, status.Convert(err).Message(), "name and url are required")
		})
	}
}

func TestDeleteRegistryReturnsFailedPreconditionWhenInUse(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := registrymocks.NewMockIRegistryRepo(ctrl)
	repo.EXPECT().DeleteRegistry(gomock.Any(), 7).Return(registry.ErrInUse)

	h := &RegistryHandler{registryRepo: repo}
	_, err := h.DeleteRegistry(t.Context(), &v1alpha1.DeleteRegistryRequest{Id: 7})

	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "referenced")
}

func TestCreateModelScopeRegistryPreservesProviderAndCredentials(t *testing.T) {
	mock := registrymocks.NewMockIRegistryRepo(gomock.NewController(t))
	mock.EXPECT().CreateRegistry(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r registry.Registry) (*registry.Registry, error) {
		require.Equal(t, registry.TypeModelScope, r.Type)
		require.Equal(t, "https://www.modelscope.cn", r.URL)
		require.Equal(t, "secret", registry.AsBasic(r.GetCredential()).Password)
		r.ID = 7
		return &r, nil
	})
	handler := &RegistryHandler{registryRepo: mock}
	response, err := handler.CreateRegistry(t.Context(), &v1alpha1.CreateRegistryRequest{Name: "modelscope", Type: v1alpha1.RegistryType_REGISTRY_TYPE_MODELSCOPE, Url: "https://www.modelscope.cn", Credential: &v1alpha1.CreateRegistryRequest_Basic{Basic: &v1alpha1.RegistryBasicCredential{Username: "alice", Password: "secret"}}})
	require.NoError(t, err)
	require.Equal(t, v1alpha1.RegistryType_REGISTRY_TYPE_MODELSCOPE, response.Registry.Type)
	mock.EXPECT().PingRegistry(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, r registry.Registry) (int, string, error) {
		require.Equal(t, registry.TypeModelScope, r.Type)
		return 200, "OK", nil
	})
	_, err = handler.PingRegistry(t.Context(), &v1alpha1.PingRegistryRequest{Type: v1alpha1.RegistryType_REGISTRY_TYPE_MODELSCOPE, Url: "https://www.modelscope.cn"})
	require.NoError(t, err)
}
