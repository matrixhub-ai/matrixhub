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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Environment variable names
const (
	EnvMatrixHubBaseURL = "MATRIXHUB_BASE_URL"
)

// Default values
const (
	DefaultMatrixHubBaseURL    = "http://localhost:9527"
	DefaultServiceReadyTimeout = 3 * time.Minute
)

// ProjectResponse represents the API response for project operations
type ProjectResponse struct {
	HTTPStatusCode int
	Success        bool
	Data           *Project
	Error          *APIError
}

// Project represents a MatrixHub project
type Project struct {
	Name string `json:"name"`
}

// APIError represents an API error response (gRPC style)
type APIError struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`
}

// MatrixHubClient is a client for interacting with MatrixHub API
type MatrixHubClient struct {
	BaseURL string
	Client  *http.Client
}

// NewMatrixHubClient creates a new MatrixHub API client
func NewMatrixHubClient(baseURL string) *MatrixHubClient {
	return &MatrixHubClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetBaseURL returns the base URL for MatrixHub API from environment or default
func GetBaseURL() string {
	baseURL := os.Getenv(EnvMatrixHubBaseURL)
	if baseURL == "" {
		return DefaultMatrixHubBaseURL
	}
	return baseURL
}

// HealthCheck performs a health check on MatrixHub
func (c *MatrixHubClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/healthz", c.BaseURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateProject creates a new project
func (c *MatrixHubClient) CreateProject(ctx context.Context, name string) (*ProjectResponse, error) {
	reqBody := map[string]string{"name": name}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1alpha1/projects", c.BaseURL), io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &ProjectResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	// Check if response contains gRPC-style error
	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	// Success response (HTTP status is OK and no error in body)
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Try to parse project data from response
	if result.Success && len(body) > 0 && string(body) != "{}" {
		var project Project
		if err := json.Unmarshal(body, &project); err == nil && project.Name != "" {
			result.Data = &project
		}
	}

	return result, nil
}

// GetProject retrieves a project by name
func (c *MatrixHubClient) GetProject(ctx context.Context, name string) (*ProjectResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/api/v1alpha1/projects/%s", c.BaseURL, name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &ProjectResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	// Check if response contains gRPC-style error
	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	// Success response
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Parse project data
	if result.Success && len(body) > 0 {
		var project Project
		if err := json.Unmarshal(body, &project); err == nil {
			result.Data = &project
		}
	}

	return result, nil
}

// DeleteProject deletes a project by name
// Note: This API endpoint may not be implemented yet
func (c *MatrixHubClient) DeleteProject(ctx context.Context, name string) (*ProjectResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/api/v1alpha1/projects/%s", c.BaseURL, name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &ProjectResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	// Check if response contains gRPC-style error
	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	// Success response (2xx status)
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	return result, nil
}

// WaitForServiceReady waits for the service to be ready
func WaitForServiceReady(ctx context.Context, baseURL string, timeout time.Duration) error {
	client := NewMatrixHubClient(baseURL)
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for service to be ready")
		case <-ticker.C:
			if err := client.HealthCheck(ctx); err == nil {
				return nil
			}
		}
	}
}

// GenerateTestProjectName generates a unique project name for testing
func GenerateTestProjectName(prefix string) string {
	return fmt.Sprintf("%s-test-%d", prefix, time.Now().UnixNano())
}
