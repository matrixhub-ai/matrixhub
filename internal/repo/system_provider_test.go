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

package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/system"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

func TestSystemProvider_GetPublicConfig(t *testing.T) {
	for _, test := range []struct {
		name       string
		cfg        *config.Config
		wantHfBase string
	}{
		{
			name: "nil config",
		},
		{
			name: "config without API server",
			cfg:  &config.Config{},
		},
		{
			name: "external URL not configured",
			cfg:  &config.Config{APIServer: &config.APIServerConfig{HostURL: "http://localhost:9527"}},
		},
		{
			name:       "trailing slashes are trimmed",
			cfg:        &config.Config{APIServer: &config.APIServerConfig{ExternalURL: "https://hub.example.com//"}},
			wantHfBase: "https://hub.example.com",
		},
		{
			name:       "path prefix is kept",
			cfg:        &config.Config{APIServer: &config.APIServerConfig{ExternalURL: "https://example.com/matrixhub/"}},
			wantHfBase: "https://example.com/matrixhub",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := NewSystemProvider(test.cfg)

			got, err := provider.GetPublicConfig(context.Background())

			require.NoError(t, err)
			require.Equal(t, &system.PublicConfig{Endpoints: system.Endpoints{HfBase: test.wantHfBase}}, got)
		})
	}
}
