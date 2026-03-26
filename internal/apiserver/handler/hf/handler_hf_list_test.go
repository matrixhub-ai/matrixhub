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

package hf

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHandleListModelsEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	resp, err := http.Get(endpoint + "/api/models")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("Expected 0 models, got %d", len(items))
	}
}

func TestHandleListModels(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create two model repos
	for _, body := range []string{
		`{"type":"model","name":"alpha-model","organization":"user-a"}`,
		`{"type":"model","name":"beta-model","organization":"user-b"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Also create a dataset to ensure it's NOT listed
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json",
		strings.NewReader(`{"type":"dataset","name":"my-dataset","organization":"user-a"}`))
	if err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}
	_ = resp.Body.Close()

	// List all models
	resp, err = http.Get(endpoint + "/api/models")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(items))
	}

	// Verify modelId is set for models
	for _, item := range items {
		if item.ModelID == "" {
			t.Errorf("Expected modelId to be set for model %s", item.RepoID)
		}
	}
}

func TestHandleListModelsFilterByAuthor(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repos under different authors
	for _, body := range []string{
		`{"type":"model","name":"model-1","organization":"alice"}`,
		`{"type":"model","name":"model-2","organization":"bob"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	resp, err := http.Get(endpoint + "/api/models?author=alice")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(items))
	}
	if items[0].RepoID != "alice/model-1" {
		t.Errorf("Expected alice/model-1, got %s", items[0].RepoID)
	}
}

func TestHandleListModelsSearch(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	for _, body := range []string{
		`{"type":"model","name":"llama-7b","organization":"meta"}`,
		`{"type":"model","name":"bert-base","organization":"google"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	resp, err := http.Get(endpoint + "/api/models?search=llama")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(items))
	}
	if items[0].RepoID != "meta/llama-7b" {
		t.Errorf("Expected meta/llama-7b, got %s", items[0].RepoID)
	}
}

func TestHandleListModelsLimit(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	for _, body := range []string{
		`{"type":"model","name":"model-a","organization":"org"}`,
		`{"type":"model","name":"model-b","organization":"org"}`,
		`{"type":"model","name":"model-c","organization":"org"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	resp, err := http.Get(endpoint + "/api/models?limit=2")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(items))
	}
}

func TestHandleListModelsResponseFormat(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create a model repo and commit a file with metadata
	createBody := `{"type":"model","name":"test-model","organization":"test-org"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Commit a README with pipeline_tag and library_name
	ndjson := `{"key":"header","value":{"summary":"Initial commit"}}` + "\n" +
		`{"key":"file","value":{"content":"---\npipeline_tag: text-generation\nlibrary_name: transformers\n---\n# Test\n","path":"README.md","encoding":"utf-8"}}` + "\n"
	resp, err = http.Post(endpoint+"/api/models/test-org/test-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	_ = resp.Body.Close()

	// List models and verify response format
	resp, err = http.Get(endpoint + "/api/models")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(items))
	}

	item := items[0]
	if item.RepoID != "test-org/test-model" {
		t.Errorf("Expected id=test-org/test-model, got %s", item.RepoID)
	}
	if item.ModelID != "test-org/test-model" {
		t.Errorf("Expected modelId=test-org/test-model, got %s", item.ModelID)
	}
	if item.PipelineTag != "text-generation" {
		t.Errorf("Expected pipeline_tag=text-generation, got %s", item.PipelineTag)
	}
	if item.LibraryName != "transformers" {
		t.Errorf("Expected library_name=transformers, got %s", item.LibraryName)
	}
}

func TestHandleListDatasetsEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	resp, err := http.Get(endpoint + "/api/datasets")
	if err != nil {
		t.Fatalf("Failed to list datasets: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("Expected 0 datasets, got %d", len(items))
	}
}

func TestHandleListDatasets(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create datasets
	for _, body := range []string{
		`{"type":"dataset","name":"dataset-a","organization":"org-a"}`,
		`{"type":"dataset","name":"dataset-b","organization":"org-b"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Also create a model to ensure it's NOT listed in datasets
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json",
		strings.NewReader(`{"type":"model","name":"my-model","organization":"org-a"}`))
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	_ = resp.Body.Close()

	// List all datasets
	resp, err = http.Get(endpoint + "/api/datasets")
	if err != nil {
		t.Fatalf("Failed to list datasets: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 datasets, got %d", len(items))
	}

	// Verify modelId is NOT set for datasets
	for _, item := range items {
		if item.ModelID != "" {
			t.Errorf("Expected modelId to be empty for dataset %s, got %s", item.RepoID, item.ModelID)
		}
	}
}

func TestHandleListDatasetsFilterByAuthor(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	for _, body := range []string{
		`{"type":"dataset","name":"ds-1","organization":"alice"}`,
		`{"type":"dataset","name":"ds-2","organization":"bob"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	resp, err := http.Get(endpoint + "/api/datasets?author=alice")
	if err != nil {
		t.Fatalf("Failed to list datasets: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 dataset, got %d", len(items))
	}
	if items[0].RepoID != "alice/ds-1" {
		t.Errorf("Expected alice/ds-1, got %s", items[0].RepoID)
	}
}

func TestHandleListSpacesEmpty(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	resp, err := http.Get(endpoint + "/api/spaces")
	if err != nil {
		t.Fatalf("Failed to list spaces: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("Expected 0 spaces, got %d", len(items))
	}
}

func TestHandleListSpaces(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create spaces
	for _, body := range []string{
		`{"type":"space","name":"space-a","organization":"org-a"}`,
		`{"type":"space","name":"space-b","organization":"org-b"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Also create a model to ensure it's NOT listed in spaces
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json",
		strings.NewReader(`{"type":"model","name":"my-model","organization":"org-a"}`))
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	_ = resp.Body.Close()

	// List all spaces
	resp, err = http.Get(endpoint + "/api/spaces")
	if err != nil {
		t.Fatalf("Failed to list spaces: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 spaces, got %d", len(items))
	}

	// Verify modelId is NOT set for spaces
	for _, item := range items {
		if item.ModelID != "" {
			t.Errorf("Expected modelId to be empty for space %s, got %s", item.RepoID, item.ModelID)
		}
	}
}

func TestHandleListSpacesSearch(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	for _, body := range []string{
		`{"type":"space","name":"chatbot-demo","organization":"dev"}`,
		`{"type":"space","name":"image-gen","organization":"dev"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	resp, err := http.Get(endpoint + "/api/spaces?search=chatbot")
	if err != nil {
		t.Fatalf("Failed to list spaces: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 space, got %d", len(items))
	}
	if items[0].RepoID != "dev/chatbot-demo" {
		t.Errorf("Expected dev/chatbot-demo, got %s", items[0].RepoID)
	}
}

func TestHandleListModelsPagination(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create 5 models
	for _, name := range []string{"model-a", "model-b", "model-c", "model-d", "model-e"} {
		body := `{"type":"model","name":"` + name + `","organization":"org"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Page 1: limit=2, no cursor
	resp, err := http.Get(endpoint + "/api/models?limit=2")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	var page1 []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&page1); err != nil {
		t.Fatalf("Failed to decode page 1: %v", err)
	}
	_ = resp.Body.Close()

	if len(page1) != 2 {
		t.Fatalf("Page 1: expected 2 models, got %d", len(page1))
	}

	// Check Link header is present
	linkHeader := resp.Header.Get("Link")
	if linkHeader == "" {
		t.Fatal("Expected Link header on page 1, got none")
	}
	if !strings.Contains(linkHeader, `rel="next"`) {
		t.Errorf("Expected Link header to contain rel=\"next\", got: %s", linkHeader)
	}

	// Extract cursor URL from Link header
	nextURL := extractNextURL(linkHeader)
	if nextURL == "" {
		t.Fatalf("Could not extract next URL from Link header: %s", linkHeader)
	}

	// Page 2: follow the cursor
	resp, err = http.Get(nextURL)
	if err != nil {
		t.Fatalf("Failed to fetch page 2: %v", err)
	}

	var page2 []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&page2); err != nil {
		t.Fatalf("Failed to decode page 2: %v", err)
	}
	linkHeader2 := resp.Header.Get("Link")
	_ = resp.Body.Close()

	if len(page2) != 2 {
		t.Fatalf("Page 2: expected 2 models, got %d", len(page2))
	}

	// Page 2 should have a Link header for page 3
	if linkHeader2 == "" {
		t.Fatal("Expected Link header on page 2, got none")
	}
	nextURL2 := extractNextURL(linkHeader2)

	// Page 3: last page (1 remaining item)
	resp, err = http.Get(nextURL2)
	if err != nil {
		t.Fatalf("Failed to fetch page 3: %v", err)
	}

	var page3 []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&page3); err != nil {
		t.Fatalf("Failed to decode page 3: %v", err)
	}
	linkHeader3 := resp.Header.Get("Link")
	_ = resp.Body.Close()

	if len(page3) != 1 {
		t.Fatalf("Page 3: expected 1 model, got %d", len(page3))
	}

	// Last page should NOT have a Link header
	if linkHeader3 != "" {
		t.Errorf("Expected no Link header on last page, got: %s", linkHeader3)
	}

	// Verify no duplicates across pages
	allIDs := make(map[string]bool)
	for _, item := range page1 {
		allIDs[item.RepoID] = true
	}
	for _, item := range page2 {
		if allIDs[item.RepoID] {
			t.Errorf("Duplicate item across pages: %s", item.RepoID)
		}
		allIDs[item.RepoID] = true
	}
	for _, item := range page3 {
		if allIDs[item.RepoID] {
			t.Errorf("Duplicate item across pages: %s", item.RepoID)
		}
		allIDs[item.RepoID] = true
	}
	if len(allIDs) != 5 {
		t.Errorf("Expected 5 unique models across all pages, got %d", len(allIDs))
	}
}

func TestHandleListModelsNoLinkHeaderWhenAllResultsFit(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create 2 models
	for _, name := range []string{"model-x", "model-y"} {
		body := `{"type":"model","name":"` + name + `","organization":"org"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Request with limit=5 (more than available)
	resp, err := http.Get(endpoint + "/api/models?limit=5")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(items))
	}

	// No Link header when all results fit
	linkHeader := resp.Header.Get("Link")
	if linkHeader != "" {
		t.Errorf("Expected no Link header when all results fit, got: %s", linkHeader)
	}
}

func TestHandleListModelsNoLinkHeaderWithoutLimit(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create 2 models
	for _, name := range []string{"model-x", "model-y"} {
		body := `{"type":"model","name":"` + name + `","organization":"org"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Request without limit
	resp, err := http.Get(endpoint + "/api/models")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// No Link header when no limit is set
	linkHeader := resp.Header.Get("Link")
	if linkHeader != "" {
		t.Errorf("Expected no Link header without limit, got: %s", linkHeader)
	}
}

// extractNextURL parses the next URL from a Link header value like:
// <http://example.com/api/models?cursor=abc>; rel="next"
func extractNextURL(linkHeader string) string {
	for part := range strings.SplitSeq(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

func TestHandleListModelsFilterByTag(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create two models
	for _, body := range []string{
		`{"type":"model","name":"model-a","organization":"org"}`,
		`{"type":"model","name":"model-b","organization":"org"}`,
	} {
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Commit a README with a tag to model-a
	ndjson := `{"key":"header","value":{"summary":"add readme"}}` + "\n" +
		`{"key":"file","value":{"content":"---\ntags:\n- text-classification\npipeline_tag: text-classification\n---\n# Model A\n","path":"README.md","encoding":"utf-8"}}` + "\n"
	resp, err := http.Post(endpoint+"/api/models/org/model-a/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	_ = resp.Body.Close()

	// Filter by tag "text-classification" — should return only model-a
	resp, err = http.Get(endpoint + "/api/models?filter=text-classification")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 model with tag text-classification, got %d", len(items))
	}
	if items[0].RepoID != "org/model-a" {
		t.Errorf("Expected org/model-a, got %s", items[0].RepoID)
	}
}

func TestHandleListModelsSortByCreatedAt(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create models in order
	for _, name := range []string{"model-first", "model-second"} {
		body := `{"type":"model","name":"` + name + `","organization":"org"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()

		// Commit to give each model a commit date
		ndjson := `{"key":"header","value":{"summary":"init"}}` + "\n" +
			`{"key":"file","value":{"content":"# ` + name + `\n","path":"README.md","encoding":"utf-8"}}` + "\n"
		resp, err = http.Post(endpoint+"/api/models/org/"+name+"/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Sort by created_at — most recently created first
	resp, err := http.Get(endpoint + "/api/models?sort=created_at")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(items))
	}
}

func TestHandleListModelsExpand(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create a model with metadata
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json",
		strings.NewReader(`{"type":"model","name":"expand-test","organization":"org"}`))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Commit a README with tags, pipeline_tag, library_name
	ndjson := `{"key":"header","value":{"summary":"init"}}` + "\n" +
		`{"key":"file","value":{"content":"---\ntags:\n- safetensors\npipeline_tag: text-generation\nlibrary_name: transformers\n---\n# Test\n","path":"README.md","encoding":"utf-8"}}` + "\n"
	resp, err = http.Post(endpoint+"/api/models/org/expand-test/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	_ = resp.Body.Close()

	// Request with expand parameter
	resp, err = http.Get(endpoint + "/api/models?expand=downloads,likes,tags,pipeline_tag,library_name,createdAt,trendingScore")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(items))
	}

	item := items[0]
	if item.PipelineTag != "text-generation" {
		t.Errorf("Expected pipeline_tag=text-generation, got %s", item.PipelineTag)
	}
	if item.LibraryName != "transformers" {
		t.Errorf("Expected library_name=transformers, got %s", item.LibraryName)
	}
	if len(item.Tags) == 0 {
		t.Error("Expected tags to be populated")
	}
}

func TestHandleListModelsSortByLikes(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create models
	for _, name := range []string{"model-a", "model-b"} {
		body := `{"type":"model","name":"` + name + `","organization":"org"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Sort by likes (all zero, so should still return results ordered stably)
	resp, err := http.Get(endpoint + "/api/models?sort=likes")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(items))
	}
}

func TestHandleListModelsSortByTrendingScore(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create models
	for _, name := range []string{"model-a", "model-b"} {
		body := `{"type":"model","name":"` + name + `","organization":"org"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create repo: %v", err)
		}
		_ = resp.Body.Close()
	}

	// Sort by trending_score
	resp, err := http.Get(endpoint + "/api/models?sort=trending_score")
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var items []repoListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(items))
	}
}
