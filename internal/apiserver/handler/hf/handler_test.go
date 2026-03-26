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
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	backendlfs "github.com/matrixhub-ai/hfd/pkg/backend/lfs"
	"github.com/matrixhub-ai/hfd/pkg/storage"
)

func setupTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	dataDir, err := os.MkdirTemp("", "hf-test-data")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })

	storage := storage.NewStorage(storage.WithRootDir(dataDir))

	// Set up handler chain (same order as main.go)
	var handler http.Handler

	handler = NewHandler(
		WithStorage(storage),
	)

	handler = backendlfs.NewHandler(
		backendlfs.WithStorage(storage),
		backendlfs.WithNext(handler),
	)

	handler = backendhttp.NewHandler(
		backendhttp.WithStorage(storage),
		backendhttp.WithNext(handler),
	)

	server := httptest.NewServer(handler)
	t.Cleanup(func() { server.Close() })

	return server, dataDir
}

func TestHuggingFaceCreateRepo(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	body := `{"type":"model","name":"test-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result createRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if result.URL == "" {
		t.Error("Expected url in response")
	}

	// Creating the same repo again should succeed (exist_ok behavior)
	resp2, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create repo again: %v", err)
	}
	_ = resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Expected 200 for existing repo, got %d: %s", resp2.StatusCode, respBody)
	}
}

func TestHuggingFacePreupload(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repo first
	createBody := `{"type":"model","name":"test-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Test preupload
	preuploadBody := `{"files":[{"path":"README.md","size":20,"sample":""},{"path":"large.bin","size":20000000,"sample":""}]}`
	resp, err = http.Post(endpoint+"/api/models/test-user/test-model/preupload/main", "application/json", strings.NewReader(preuploadBody))
	if err != nil {
		t.Fatalf("Failed to preupload: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result preuploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(result.Files))
	}
	if result.Files[0].UploadMode != "regular" {
		t.Errorf("Expected regular mode for README.md, got %s", result.Files[0].UploadMode)
	}
	if result.Files[1].UploadMode != "lfs" {
		t.Errorf("Expected lfs mode for large.bin, got %s", result.Files[1].UploadMode)
	}
}

func TestHuggingFaceCommitAndResolve(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repo first
	createBody := `{"type":"model","name":"test-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Commit a regular file
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Initial commit\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"# Test Model\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/test-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var commitResult commitResponse
	if err := json.NewDecoder(resp.Body).Decode(&commitResult); err != nil {
		t.Fatalf("Failed to decode commit response: %v", err)
	}

	if commitResult.CommitOid == "" {
		t.Error("Expected commitOid in response")
	}
	if commitResult.CommitMessage != "Initial commit" {
		t.Errorf("Expected commit message 'Initial commit', got %q", commitResult.CommitMessage)
	}

	// Verify the file is accessible via resolve endpoint
	resp, err = http.Get(endpoint + "/test-user/test-model/resolve/main/README.md")
	if err != nil {
		t.Fatalf("Failed to get file: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for resolve, got %d: %s", resp.StatusCode, respBody)
	}

	content, _ := io.ReadAll(resp.Body)
	if string(content) != "# Test Model\n" {
		t.Errorf("Unexpected content: %q", content)
	}

	// Verify the model info endpoint shows the file
	resp, err = http.Get(endpoint + "/api/models/test-user/test-model")
	if err != nil {
		t.Fatalf("Failed to get model info: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for model info, got %d: %s", resp.StatusCode, respBody)
	}

	var repoInfo repoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		t.Fatalf("Failed to decode model info: %v", err)
	}

	if len(repoInfo.Siblings) != 2 {
		t.Fatalf("Expected 2 siblings, got %d", len(repoInfo.Siblings))
	}
	foundGitAttrs := false
	foundReadme := false
	for _, s := range repoInfo.Siblings {
		switch s.RFilename {
		case ".gitattributes":
			foundGitAttrs = true
		case "README.md":
			foundReadme = true
		}
	}
	if !foundGitAttrs {
		t.Errorf("Expected a sibling with filename '.gitattributes', but none was found")
	}
	if !foundReadme {
		t.Errorf("Expected a sibling with filename 'README.md', but none was found")
	}
}

func TestHuggingFaceCommitMultipleFiles(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repo first
	createBody := `{"type":"model","name":"multi-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Commit multiple files
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add multiple files\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"# Multi Model\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"config data\\n\",\"path\":\"config.json\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/multi-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	_ = resp.Body.Close()

	// Verify both files
	for _, file := range []struct {
		path    string
		content string
	}{
		{"README.md", "# Multi Model\n"},
		{"config.json", "config data\n"},
	} {
		func() {
			resp, err = http.Get(endpoint + "/test-user/multi-model/resolve/main/" + file.path)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", file.path, err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				t.Fatalf("Expected 200 for %s, got %d: %s", file.path, resp.StatusCode, respBody)
			}

			content, _ := io.ReadAll(resp.Body)
			if string(content) != file.content {
				t.Errorf("Unexpected content for %s: %q", file.path, content)
			}
		}()
	}
}

func TestHuggingFaceCommitLFSFile(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repo first
	createBody := `{"type":"model","name":"lfs-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Commit an LFS file (pointer)
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add LFS file\"}}\n" +
		"{\"key\":\"lfsFile\",\"value\":{\"path\":\"model.bin\",\"algo\":\"sha256\",\"oid\":\"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\",\"size\":1024}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/lfs-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var commitResult commitResponse
	if err := json.NewDecoder(resp.Body).Decode(&commitResult); err != nil {
		t.Fatalf("Failed to decode commit response: %v", err)
	}

	if commitResult.CommitOid == "" {
		t.Error("Expected commitOid in response")
	}
}

func TestHuggingFaceDatasetCreateAndCommit(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create a dataset repo
	body := `{"type":"dataset","name":"test-dataset","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create dataset repo: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result createRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.Contains(result.URL, "/datasets/test-user/test-dataset") {
		t.Errorf("Expected URL to contain '/datasets/test-user/test-dataset', got %q", result.URL)
	}

	// Commit a file via datasets API
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add dataset readme\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"# Test Dataset\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/datasets/test-user/test-dataset/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit to dataset: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	// Resolve file via datasets resolve endpoint
	resp, err = http.Get(endpoint + "/datasets/test-user/test-dataset/resolve/main/README.md")
	if err != nil {
		t.Fatalf("Failed to resolve dataset file: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for dataset resolve, got %d: %s", resp.StatusCode, respBody)
	}

	content, _ := io.ReadAll(resp.Body)
	if string(content) != "# Test Dataset\n" {
		t.Errorf("Unexpected dataset content: %q", content)
	}

	// Verify dataset info endpoint works
	resp, err = http.Get(endpoint + "/api/datasets/test-user/test-dataset")
	if err != nil {
		t.Fatalf("Failed to get dataset info: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for dataset info, got %d: %s", resp.StatusCode, respBody)
	}

	var repoInfo repoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		t.Fatalf("Failed to decode dataset info: %v", err)
	}
	if len(repoInfo.Siblings) != 2 {
		t.Errorf("Expected 2 siblings, got %v", repoInfo.Siblings)
	}
	foundGitAttrs := false
	foundReadme := false
	for _, s := range repoInfo.Siblings {
		switch s.RFilename {
		case ".gitattributes":
			foundGitAttrs = true
		case "README.md":
			foundReadme = true
		}
	}
	if !foundGitAttrs {
		t.Errorf("Expected a sibling with filename '.gitattributes', but none was found")
	}
	if !foundReadme {
		t.Errorf("Expected a sibling with filename 'README.md', but none was found")
	}
}

func TestHuggingFaceSpaceCreateAndCommit(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create a space repo
	body := `{"type":"space","name":"test-space","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create space repo: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result createRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.Contains(result.URL, "/spaces/test-user/test-space") {
		t.Errorf("Expected URL to contain '/spaces/test-user/test-space', got %q", result.URL)
	}

	// Commit a file via spaces API
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add space app\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"# Test Space\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/spaces/test-user/test-space/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit to space: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	// Resolve file via spaces resolve endpoint
	resp, err = http.Get(endpoint + "/spaces/test-user/test-space/resolve/main/README.md")
	if err != nil {
		t.Fatalf("Failed to resolve space file: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for space resolve, got %d: %s", resp.StatusCode, respBody)
	}

	content, _ := io.ReadAll(resp.Body)
	if string(content) != "# Test Space\n" {
		t.Errorf("Unexpected space content: %q", content)
	}

	// Verify space info endpoint works
	resp, err = http.Get(endpoint + "/api/spaces/test-user/test-space")
	if err != nil {
		t.Fatalf("Failed to get space info: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for space info, got %d: %s", resp.StatusCode, respBody)
	}

	var repoInfo repoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		t.Fatalf("Failed to decode space info: %v", err)
	}
	if len(repoInfo.Siblings) != 2 {
		t.Errorf("Expected 2 siblings, got %v", repoInfo.Siblings)
	}
	foundGitAttrs := false
	foundReadme := false
	for _, s := range repoInfo.Siblings {
		switch s.RFilename {
		case ".gitattributes":
			foundGitAttrs = true
		case "README.md":
			foundReadme = true
		}
	}
	if !foundGitAttrs {
		t.Errorf("Expected a sibling with filename '.gitattributes', but none was found")
	}
	if !foundReadme {
		t.Errorf("Expected a sibling with filename 'README.md', but none was found")
	}
}

func TestHuggingFaceDatasetPreupload(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create dataset repo first
	createBody := `{"type":"dataset","name":"test-dataset","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Test preupload via datasets API
	preuploadBody := `{"files":[{"path":"data.csv","size":100,"sample":""}]}`
	resp, err = http.Post(endpoint+"/api/datasets/test-user/test-dataset/preupload/main", "application/json", strings.NewReader(preuploadBody))
	if err != nil {
		t.Fatalf("Failed to preupload: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result preuploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(result.Files))
	}
	if result.Files[0].UploadMode != "regular" {
		t.Errorf("Expected regular mode, got %s", result.Files[0].UploadMode)
	}
}

func TestHuggingFaceRepoTypeIsolation(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repos with the same name but different types
	for _, repoType := range []string{"model", "dataset", "space"} {
		body := `{"type":"` + repoType + `","name":"shared-name","organization":"test-user"}`
		resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create %s repo: %v", repoType, err)
		}
		_ = resp.Body.Close()
	}

	// Commit different content to each
	for _, tc := range []struct {
		repoType  string
		apiPrefix string
		content   string
	}{
		{"model", "/api/models", "model content"},
		{"dataset", "/api/datasets", "dataset content"},
		{"space", "/api/spaces", "space content"},
	} {
		ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add readme\"}}\n" +
			"{\"key\":\"file\",\"value\":{\"content\":\"" + tc.content + "\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n"

		resp, err := http.Post(endpoint+tc.apiPrefix+"/test-user/shared-name/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
		if err != nil {
			t.Fatalf("Failed to commit to %s: %v", tc.repoType, err)
		}
		_ = resp.Body.Close()
	}

	// Verify each type has its own content
	for _, tc := range []struct {
		resolvePrefix string
		expected      string
	}{
		{"", "model content\n"},
		{"/datasets", "dataset content\n"},
		{"/spaces", "space content\n"},
	} {
		resp, err := http.Get(endpoint + tc.resolvePrefix + "/test-user/shared-name/resolve/main/README.md")
		if err != nil {
			t.Fatalf("Failed to resolve: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 200 for prefix %q, got %d: %s", tc.resolvePrefix, resp.StatusCode, respBody)
		}

		content, _ := io.ReadAll(resp.Body)
		if string(content) != tc.expected {
			t.Errorf("For prefix %q: expected %q, got %q", tc.resolvePrefix, tc.expected, content)
		}
	}
}

func TestHuggingFaceCommitDeleteFile(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repo first
	createBody := `{"type":"model","name":"delete-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// First commit - add a file
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add file\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"to be deleted\\n\",\"path\":\"temp.txt\",\"encoding\":\"utf-8\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"# Keep me\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/delete-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to first commit: %v", err)
	}
	_ = resp.Body.Close()

	// Second commit - delete the file
	ndjson = "{\"key\":\"header\",\"value\":{\"summary\":\"Delete file\"}}\n" +
		"{\"key\":\"deletedFile\",\"value\":{\"path\":\"temp.txt\"}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/delete-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to second commit: %v", err)
	}
	_ = resp.Body.Close()

	// Verify temp.txt is deleted
	resp, err = http.Get(endpoint + "/test-user/delete-model/resolve/main/temp.txt")
	if err != nil {
		t.Fatalf("Failed to get temp.txt: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for deleted file, got %d", resp.StatusCode)
	}

	// Verify README.md still exists
	resp, err = http.Get(endpoint + "/test-user/delete-model/resolve/main/README.md")
	if err != nil {
		t.Fatalf("Failed to get README.md: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for README.md, got %d: %s", resp.StatusCode, respBody)
	}

	content, _ := io.ReadAll(resp.Body)
	if string(content) != "# Keep me\n" {
		t.Errorf("Unexpected content for README.md: %q", content)
	}
}

func TestHuggingFacePreuploadWithGitAttributes(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create repo first
	createBody := `{"type":"model","name":"attrs-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	// Commit a .gitattributes file that marks *.bin and *.safetensors as LFS
	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add gitattributes\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"*.bin filter=lfs diff=lfs merge=lfs -text\\n*.safetensors filter=lfs diff=lfs merge=lfs -text\\n\",\"path\":\".gitattributes\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/attrs-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	_ = resp.Body.Close()

	// Test preupload: small .bin file should be LFS (matches .gitattributes pattern),
	// small .txt file should be regular (doesn't match any LFS pattern)
	preuploadBody := `{"files":[{"path":"model.bin","size":100,"sample":""},{"path":"README.txt","size":100,"sample":""},{"path":"weights.safetensors","size":50,"sample":""}]}`
	resp, err = http.Post(endpoint+"/api/models/test-user/attrs-model/preupload/main", "application/json", strings.NewReader(preuploadBody))
	if err != nil {
		t.Fatalf("Failed to preupload: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var result preuploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(result.Files))
	}
	// model.bin matches *.bin pattern → lfs (even though size is small)
	if result.Files[0].UploadMode != "lfs" {
		t.Errorf("Expected lfs mode for model.bin (matches .gitattributes), got %s", result.Files[0].UploadMode)
	}
	// README.txt doesn't match any LFS pattern and is small → regular
	if result.Files[1].UploadMode != "regular" {
		t.Errorf("Expected regular mode for README.txt, got %s", result.Files[1].UploadMode)
	}
	// weights.safetensors matches *.safetensors pattern → lfs
	if result.Files[2].UploadMode != "lfs" {
		t.Errorf("Expected lfs mode for weights.safetensors (matches .gitattributes), got %s", result.Files[2].UploadMode)
	}
}

func TestHuggingFaceTreeSize(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	// Create and populate a repo with two files
	createBody := `{"type":"model","name":"treesize-model","organization":"test-user"}`
	resp, err := http.Post(endpoint+"/api/repos/create", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}
	_ = resp.Body.Close()

	ndjson := "{\"key\":\"header\",\"value\":{\"summary\":\"Add files\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"hello\\n\",\"path\":\"README.md\",\"encoding\":\"utf-8\"}}\n" +
		"{\"key\":\"file\",\"value\":{\"content\":\"world\\n\",\"path\":\"sub/data.txt\",\"encoding\":\"utf-8\"}}\n"

	resp, err = http.Post(endpoint+"/api/models/test-user/treesize-model/commit/main", "application/x-ndjson", strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
	_ = resp.Body.Close()

	// Get size of the root tree
	resp, err = http.Get(endpoint + "/api/models/test-user/treesize-model/treesize/main/")
	if err != nil {
		t.Fatalf("Failed to get treesize: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for treesize, got %d: %s", resp.StatusCode, respBody)
	}

	var result treeSize
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode treesize response: %v", err)
	}

	// Compute the size of .gitattributes dynamically to avoid brittle, hard-coded values.
	gitattributesResp, err := http.Get(endpoint + "/test-user/treesize-model/resolve/main/.gitattributes")
	if err != nil {
		t.Fatalf("Failed to get .gitattributes: %v", err)
	}
	defer func() {
		_ = gitattributesResp.Body.Close()
	}()

	if gitattributesResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(gitattributesResp.Body)
		t.Fatalf("Expected 200 for .gitattributes, got %d: %s", gitattributesResp.StatusCode, respBody)
	}

	gitattributesBody, err := io.ReadAll(gitattributesResp.Body)
	if err != nil {
		t.Fatalf("Failed to read .gitattributes body: %v", err)
	}

	// "hello\n" = 6 bytes, "world\n" = 6 bytes, ".gitattributes" = len(gitattributesBody) bytes
	expectedRootSize := int64(6 + 6 + len(gitattributesBody)) // README.md + sub/data.txt + .gitattributes
	if result.Size != expectedRootSize {
		t.Errorf("Expected treesize %d, got %d", expectedRootSize, result.Size)
	}

	// Get size of the sub/ subdirectory only
	resp, err = http.Get(endpoint + "/api/models/test-user/treesize-model/treesize/main/sub")
	if err != nil {
		t.Fatalf("Failed to get treesize for sub: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 for treesize sub, got %d: %s", resp.StatusCode, respBody)
	}

	var subResult treeSize
	if err := json.NewDecoder(resp.Body).Decode(&subResult); err != nil {
		t.Fatalf("Failed to decode treesize sub response: %v", err)
	}

	// "world\n" = 6 bytes
	if subResult.Size != 6 {
		t.Errorf("Expected treesize 6 for sub/, got %d", subResult.Size)
	}
}

func TestHuggingFaceTreeSizeNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	endpoint := server.URL

	resp, err := http.Get(endpoint + "/api/models/nonexistent/no-repo/treesize/main/")
	if err != nil {
		t.Fatalf("Failed to request treesize: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 for nonexistent repo, got %d", resp.StatusCode)
	}
}
