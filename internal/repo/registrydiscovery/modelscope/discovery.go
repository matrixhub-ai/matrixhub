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

// Package modelscope implements ModelScope's paginated model discovery API.
package modelscope

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

type Discovery struct{}

func New() *Discovery { return &Discovery{} }

func (d *Discovery) ListRepositories(ctx context.Context, reg *registry.Registry, filter registrydiscovery.Filter) ([]registrydiscovery.RemoteRepository, error) {
	if reg == nil || strings.TrimSpace(reg.URL) == "" {
		return nil, fmt.Errorf("modelscope: registry URL is required")
	}
	if filter.ResourceType != "model" {
		return nil, fmt.Errorf("modelscope discovery currently supports models only")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: reg.Insecure} // #nosec G402 -- explicit registry setting
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 60 * time.Second}
	const pageSize = 100
	result := make([]registrydiscovery.RemoteRepository, 0)
	seen := make(map[string]bool)
	for page := 1; page <= 10000; page++ {
		body, err := json.Marshal(map[string]any{"Path": filter.Namespace, "PageNumber": page, "PageSize": pageSize})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.TrimRight(reg.URL, "/")+"/api/v1/models/", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if cred := registry.AsBasic(reg.GetCredential()); cred != nil && cred.Password != "" {
			req.Header.Set("Authorization", "Bearer "+cred.Password)
			req.AddCookie(&http.Cookie{Name: "m_session_id", Value: cred.Password})
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("modelscope discovery: %w", err)
		}
		var payload struct {
			Code int
			Data struct {
				Models     []struct{ Name, Path string }
				TotalCount int
			}
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("modelscope discovery returned HTTP %d", resp.StatusCode)
		}
		if err != nil {
			return nil, fmt.Errorf("decode modelscope discovery: %w", err)
		}
		if payload.Code != http.StatusOK {
			return nil, fmt.Errorf("modelscope discovery returned code %d", payload.Code)
		}
		added := 0
		for _, item := range payload.Data.Models {
			if item.Path == "" {
				item.Path = filter.Namespace
			}
			if item.Name == "" || item.Path == "" || item.Name == "." || item.Name == ".." || item.Path == "." || item.Path == ".." || strings.ContainsAny(item.Name+item.Path, "/\\\x00") {
				return nil, fmt.Errorf("modelscope discovery returned an invalid repository path")
			}
			key := item.Path + "/" + item.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			added++
			result = append(result, registrydiscovery.RemoteRepository{Namespace: item.Path, Name: item.Name, ResourceType: "model"})
		}
		if len(result) >= payload.Data.TotalCount && payload.Data.TotalCount > 0 {
			return result, nil
		}
		if len(payload.Data.Models) < pageSize {
			if payload.Data.TotalCount > len(result) {
				return nil, fmt.Errorf("modelscope discovery returned an incomplete listing")
			}
			return result, nil
		}
		if added == 0 {
			return nil, fmt.Errorf("modelscope discovery pagination made no progress")
		}
	}
	return nil, fmt.Errorf("modelscope discovery exceeded pagination limit")
}
