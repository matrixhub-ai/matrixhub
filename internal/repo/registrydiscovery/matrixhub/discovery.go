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

package matrixhub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

const defaultPageSize = 100

// Discovery implements registrydiscovery.Discovery for a remote MatrixHub instance.
type Discovery struct {
	client *http.Client
}

// New creates a new MatrixHub Discovery.
func New() *Discovery {
	return &Discovery{
		client: &http.Client{},
	}
}

// ListRepositories discovers remote repositories on a MatrixHub registry.
func (d *Discovery) ListRepositories(ctx context.Context, reg *registry.Registry, filter registrydiscovery.Filter) ([]registrydiscovery.RemoteRepository, error) {
	base := strings.TrimSuffix(reg.URL, "/")

	if filter.ResourceType == "dataset" {
		return d.listDatasets(ctx, base, reg, filter)
	}
	return d.listModels(ctx, base, reg, filter)
}

func (d *Discovery) listModels(ctx context.Context, base string, reg *registry.Registry, filter registrydiscovery.Filter) ([]registrydiscovery.RemoteRepository, error) {
	var out []registrydiscovery.RemoteRepository
	var page int32 = 1
	for {
		u := fmt.Sprintf("%s/api/v1alpha1/models?project=%s&page=%d&page_size=%d", base, filter.Namespace, page, defaultPageSize)
		var resp v1alpha1.ListModelsResponse
		if err := d.doRequest(ctx, u, reg, &resp); err != nil {
			return nil, fmt.Errorf("matrixhub list models: %w", err)
		}

		for _, m := range resp.Items {
			out = append(out, registrydiscovery.RemoteRepository{
				Namespace:    m.Project,
				Name:         m.Name,
				ResourceType: "model",
			})
		}

		if resp.Pagination == nil || int32(len(out)) >= resp.Pagination.Total {
			break
		}
		page++
	}
	return out, nil
}

func (d *Discovery) listDatasets(ctx context.Context, base string, reg *registry.Registry, filter registrydiscovery.Filter) ([]registrydiscovery.RemoteRepository, error) {
	var out []registrydiscovery.RemoteRepository
	var page int32 = 1
	for {
		u := fmt.Sprintf("%s/api/v1alpha1/datasets?project=%s&page=%d&page_size=%d", base, filter.Namespace, page, defaultPageSize)
		var resp v1alpha1.ListDatasetsResponse
		if err := d.doRequest(ctx, u, reg, &resp); err != nil {
			return nil, fmt.Errorf("matrixhub list datasets: %w", err)
		}

		for _, ds := range resp.Items {
			out = append(out, registrydiscovery.RemoteRepository{
				Namespace:    ds.Project,
				Name:         ds.Name,
				ResourceType: "dataset",
			})
		}

		if resp.Pagination == nil || int32(len(out)) >= resp.Pagination.Total {
			break
		}
		page++
	}
	return out, nil
}

func (d *Discovery) doRequest(ctx context.Context, url string, reg *registry.Registry, out proto.Message) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if basic := registry.AsBasic(reg.GetCredential()); basic != nil && basic.Password != "" {
		req.Header.Set("Authorization", "Bearer "+basic.Password)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if err := protojson.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
