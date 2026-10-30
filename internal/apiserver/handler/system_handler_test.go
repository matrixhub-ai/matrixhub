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
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/system"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
)

type systemHandlerTestService struct {
	cfg *system.PublicConfig
	err error
}

func (s systemHandlerTestService) GetPublicConfig(context.Context) (*system.PublicConfig, error) {
	return s.cfg, s.err
}

func TestSystemHandler_GetSystemConfig(t *testing.T) {
	cases := []struct {
		name        string
		externalURL string
		want        string
	}{
		{"plain", "https://matrixhub.example.com", "https://matrixhub.example.com"},
		{"trailing slash trimmed", "https://matrixhub.example.com/", "https://matrixhub.example.com"},
		{"empty when not configured", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := repo.NewSystemProvider(&config.Config{
				APIServer: &config.APIServerConfig{ExternalURL: tc.externalURL},
			})
			h := NewSystemHandler(svc).(*SystemHandler)
			resp, err := h.GetSystemConfig(context.Background(), &v1alpha1.GetSystemConfigRequest{})
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if resp.GetEndpoints().GetHfBase() != tc.want {
				t.Fatalf("hf_base = %q, want %q", resp.GetEndpoints().GetHfBase(), tc.want)
			}
		})
	}
}

func TestSystemHandler_GetSystemConfigServiceError(t *testing.T) {
	h := NewSystemHandler(systemHandlerTestService{err: errors.New("config store unavailable")}).(*SystemHandler)

	resp, err := h.GetSystemConfig(context.Background(), &v1alpha1.GetSystemConfigRequest{})
	require.Nil(t, resp)
	requireHandlerTestStatus(t, err, codes.Internal, "")
}

func TestSystemHandler_RegisterToServer(t *testing.T) {
	h := NewSystemHandler(systemHandlerTestService{cfg: &system.PublicConfig{
		Endpoints: system.Endpoints{HfBase: "https://hub.example.com"},
	}})

	t.Run("serves the config over gRPC and the HTTP gateway", func(t *testing.T) {
		grpcServer, gatewayMux := registerHandlerForTest(t, h, handlerTestInsecure)
		require.Contains(t, grpcServer.GetServiceInfo(), "matrixhub.v1alpha1.SystemService")

		rec := serveHandlerTestRequest(t, gatewayMux, http.MethodGet, "/api/v1alpha1/system/config")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		got := &v1alpha1.SystemConfig{}
		require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), got))
		requireProtoEqual(t, &v1alpha1.SystemConfig{
			Endpoints: &v1alpha1.Endpoints{HfBase: "https://hub.example.com"},
		}, got)
	})

	t.Run("keeps the gRPC service when gateway registration fails", func(t *testing.T) {
		// Without transport credentials the gateway cannot create its client.
		grpcServer, gatewayMux := registerHandlerForTest(t, h)
		require.Contains(t, grpcServer.GetServiceInfo(), "matrixhub.v1alpha1.SystemService")

		rec := serveHandlerTestRequest(t, gatewayMux, http.MethodGet, "/api/v1alpha1/system/config")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
