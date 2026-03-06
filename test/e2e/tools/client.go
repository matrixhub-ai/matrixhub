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

package tools

import (
	"context"
	"fmt"
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
