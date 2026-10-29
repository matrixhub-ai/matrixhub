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
