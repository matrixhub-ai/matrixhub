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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ProjectResponse represents the simple API response for project operations
type ProjectResponse struct {
	HTTPStatusCode int
	Success        bool
	Data           *Project
	Error          *APIError
}

// ListProjectsResponse represents the API response for listing projects
type ListProjectsResponse struct {
	HTTPStatusCode int
	Success        bool
	Projects       []*Project
	Pagination     *Pagination
	Error          *APIError
}

// GetProjectResponse represents the API response for getting a project
type GetProjectResponse struct {
	HTTPStatusCode int
	Success        bool
	Data           *ProjectDetail
	Error          *APIError
}

// ListProjectMembersResponse represents the API response for listing project members
type ListProjectMembersResponse struct {
	HTTPStatusCode int
	Success        bool
	Members        []*ProjectMember
	Pagination     *Pagination
	Error          *APIError
}

// ProjectMemberResponse represents the API response for project member operations
type ProjectMemberResponse struct {
	HTTPStatusCode int
	Success        bool
	Error          *APIError
}

// Project represents a MatrixHub project (from list)
type Project struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	EnabledRegistry bool   `json:"enabled_registry"`
	ModelCount      uint32 `json:"model_count"`
	DatasetCount    uint32 `json:"dataset_count"`
}

// ProjectDetail represents detailed project info (from get)
type ProjectDetail struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	RegistryURL  string `json:"registry_url"`
	Organization string `json:"organization"`
	ModelCount   uint32 `json:"model_count"`
	DatasetCount uint32 `json:"dataset_count"`
}

// ProjectMember represents a project member
type ProjectMember struct {
	MemberID   string `json:"memberId"`
	MemberName string `json:"memberName"`
	MemberType string `json:"memberType"`
	Role       string `json:"role"`
}

// Pagination represents pagination info
type Pagination struct {
	Total    int32 `json:"total"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"pageSize"`
}

// --- Project API methods ---

// CreateProjectOption defines options for creating a project
type CreateProjectOption func(*createProjectOptions)

type createProjectOptions struct {
	projectType  string
	registryID   *int32
	organization string
}

// CreateProject creates a new project
func (c *MatrixHubClient) CreateProject(ctx context.Context, name string, opts ...CreateProjectOption) (*ProjectResponse, error) {
	options := &createProjectOptions{}
	for _, opt := range opts {
		opt(options)
	}

	reqBody := map[string]any{
		"name": name,
	}
	if options.projectType != "" {
		reqBody["type"] = options.projectType
	}
	if options.registryID != nil {
		reqBody["registry_id"] = *options.registryID
	}
	if options.organization != "" {
		reqBody["organization"] = options.organization
	}

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
func (c *MatrixHubClient) GetProject(ctx context.Context, name string) (*GetProjectResponse, error) {
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

	result := &GetProjectResponse{
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

	// Parse project data (use ProjectDetail which matches GetProject API response)
	if result.Success && len(body) > 0 {
		var projectDetail ProjectDetail
		if err := json.Unmarshal(body, &projectDetail); err == nil {
			result.Data = &projectDetail
		}
	}

	return result, nil
}

// DeleteProject deletes a project by name
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

// ListProjects lists all projects
func (c *MatrixHubClient) ListProjects(ctx context.Context, name string, projectType string, managedOnly bool, page, pageSize int32) (*ListProjectsResponse, error) {
	url := fmt.Sprintf("%s/api/v1alpha1/projects", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	if name != "" {
		q.Add("name", name)
	}
	if projectType != "" {
		q.Add("type", projectType)
	}
	if managedOnly {
		q.Add("managed_only", "true")
	}
	if page > 0 {
		q.Add("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		q.Add("page_size", fmt.Sprintf("%d", pageSize))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &ListProjectsResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	// Check if response contains gRPC-style error
	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Parse response
	if result.Success {
		var respData struct {
			Projects   []*Project  `json:"projects"`
			Pagination *Pagination `json:"pagination"`
		}
		if err := json.Unmarshal(body, &respData); err == nil {
			result.Projects = respData.Projects
			result.Pagination = respData.Pagination
		}
	}

	return result, nil
}

// UpdateProject updates a project
func (c *MatrixHubClient) UpdateProject(ctx context.Context, name string, projectType string) (*ProjectResponse, error) {
	reqBody := map[string]string{"name": name, "type": projectType}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/api/v1alpha1/projects/%s", c.BaseURL, name), io.NopCloser(bytes.NewReader(bodyBytes)))
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

	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	return result, nil
}

// ListProjectMembers lists members of a project
func (c *MatrixHubClient) ListProjectMembers(ctx context.Context, projectName string, memberName string, page, pageSize int32) (*ListProjectMembersResponse, error) {
	url := fmt.Sprintf("%s/api/v1alpha1/projects/%s/members", c.BaseURL, projectName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	if memberName != "" {
		q.Add("member_name", memberName)
	}
	if page > 0 {
		q.Add("page", fmt.Sprintf("%d", page))
	}
	if pageSize > 0 {
		q.Add("page_size", fmt.Sprintf("%d", pageSize))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	result := &ListProjectMembersResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	if result.Success {
		var respData struct {
			Members    []*ProjectMember `json:"members"`
			Pagination *Pagination      `json:"pagination"`
		}
		if err := json.Unmarshal(body, &respData); err == nil {
			result.Members = respData.Members
			result.Pagination = respData.Pagination
		}
	}

	return result, nil
}

// AddProjectMemberWithRole adds a member to a project with a role
func (c *MatrixHubClient) AddProjectMemberWithRole(ctx context.Context, projectName, memberID, memberType, role string) (*ProjectMemberResponse, error) {
	reqBody := map[string]string{
		"name":        projectName,
		"member_id":   memberID,
		"member_type": memberType,
		"role":        role,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v1alpha1/projects/%s/members", c.BaseURL, projectName), io.NopCloser(bytes.NewReader(bodyBytes)))
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

	result := &ProjectMemberResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	return result, nil
}

// RemoveProjectMembers removes members from a project
func (c *MatrixHubClient) RemoveProjectMembers(ctx context.Context, projectName string, members []map[string]string) (*ProjectMemberResponse, error) {
	reqBody := map[string]any{
		"name":    projectName,
		"members": members,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/api/v1alpha1/projects/%s/members", c.BaseURL, projectName), io.NopCloser(bytes.NewReader(bodyBytes)))
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

	result := &ProjectMemberResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	return result, nil
}

// UpdateProjectMemberRole updates a member's role in a project
func (c *MatrixHubClient) UpdateProjectMemberRole(ctx context.Context, projectName, memberID, memberType, role string) (*ProjectMemberResponse, error) {
	reqBody := map[string]string{
		"name":        projectName,
		"member_id":   memberID,
		"member_type": memberType,
		"role":        role,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		fmt.Sprintf("%s/api/v1alpha1/projects/%s/members/%s/role", c.BaseURL, projectName, memberID), io.NopCloser(bytes.NewReader(bodyBytes)))
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

	result := &ProjectMemberResponse{
		HTTPStatusCode: resp.StatusCode,
	}

	var grpcError APIError
	if err := json.Unmarshal(body, &grpcError); err == nil && grpcError.Code != 0 {
		result.Success = false
		result.Error = &grpcError
		return result, nil
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	return result, nil
}
