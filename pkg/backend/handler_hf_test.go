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

package backend_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/handlers"

	"github.com/matrixhub-ai/matrixhub/pkg/backend"
)

// TestHuggingFaceAPI tests the HuggingFace-compatible API endpoints.
func TestHuggingFaceAPI(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "matrixhub-hf-test")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("ModelInfoNotFound", func(t *testing.T) {
		// Request model info for non-existent repo
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/nonexistent/repo", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("ModelInfoAfterCreate", func(t *testing.T) {
		repoName := "test-model.git"

		// Create repository first
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201 for create, got %d", resp.StatusCode)
		}

		// Request model info (without .git suffix for HF API)
		req, _ = http.NewRequest(http.MethodGet, server.URL+"/api/models/test-model", nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get model info: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify response structure
		var modelInfo map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check required fields
		if modelInfo["id"] != "test-model" {
			t.Errorf("Expected id 'test-model', got %v", modelInfo["id"])
		}
		if modelInfo["modelId"] != "test-model" {
			t.Errorf("Expected modelId 'test-model', got %v", modelInfo["modelId"])
		}
		if _, ok := modelInfo["siblings"]; !ok {
			t.Errorf("Missing siblings field in response")
		}
	})

	t.Run("ResolveFileNotFound", func(t *testing.T) {
		repoName := "resolve-test.git"

		// Create repository first
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		_ = resp.Body.Close()

		// Try to resolve a file that doesn't exist
		req, _ = http.NewRequest(http.MethodGet, server.URL+"/resolve-test/resolve/main/nonexistent.txt", nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("NestedModelInfo", func(t *testing.T) {
		repoName := "org/model-name.git"

		// Create nested repository
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201 for create, got %d", resp.StatusCode)
		}

		// Request model info with nested path
		req, _ = http.NewRequest(http.MethodGet, server.URL+"/api/models/org/model-name", nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get model info: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify response structure
		var modelInfo map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check that nested repo ID is correct
		if modelInfo["id"] != "org/model-name" {
			t.Errorf("Expected id 'org/model-name', got %v", modelInfo["id"])
		}
	})
}

// TestHuggingFaceTreeAPI tests the HuggingFace tree API with recursive and expand options.
func TestHuggingFaceTreeAPI(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "matrixhub-hf-tree-test")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "matrixhub-hf-tree-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	repoName := "tree-test.git"

	// Create repository
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 for create, got %d", resp.StatusCode)
	}

	// Clone and add nested structure
	gitWorkDir := filepath.Join(clientDir, "git-work")
	cmd := exec.Command("git", "clone", server.URL+"/"+repoName, gitWorkDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
	}

	// Configure git user
	runGitCmd(t, gitWorkDir, "config", "user.email", "test@test.com")
	runGitCmd(t, gitWorkDir, "config", "user.name", "Test User")

	// Create nested structure: dir1/dir2/file.txt
	dir1 := filepath.Join(gitWorkDir, "dir1")
	dir2 := filepath.Join(dir1, "dir2")
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitWorkDir, "root.txt"), []byte("root file"), 0644); err != nil {
		t.Fatalf("Failed to create root.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "file1.txt"), []byte("file in dir1"), 0644); err != nil {
		t.Fatalf("Failed to create file1.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "file2.txt"), []byte("file in dir2"), 0644); err != nil {
		t.Fatalf("Failed to create file2.txt: %v", err)
	}

	// Add, commit, and push
	runGitCmd(t, gitWorkDir, "add", ".")
	runGitCmd(t, gitWorkDir, "commit", "-m", "Add nested structure")
	runGitCmd(t, gitWorkDir, "branch", "-M", "main")
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = gitWorkDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to push: %v\nOutput: %s", err, output)
	}

	t.Run("TreeNonRecursive", func(t *testing.T) {
		// Request tree without recursive option (default)
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/tree-test/tree/main", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var entries []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should only have root.txt and dir1 (not nested entries)
		if len(entries) != 2 {
			t.Errorf("Expected 2 entries, got %d", len(entries))
		}

		// Check that entries have size (expand=true by default)
		for _, entry := range entries {
			if entry["type"] == "file" {
				if _, ok := entry["size"]; !ok {
					t.Errorf("Expected file entry to have size field")
				}
			}
		}
	})

	t.Run("TreeRecursive", func(t *testing.T) {
		// Request tree with recursive option
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/tree-test/tree/main?recursive=true", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var entries []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have all entries: root.txt, dir1, dir1/file1.txt, dir1/dir2, dir1/dir2/file2.txt
		if len(entries) != 5 {
			t.Errorf("Expected 5 entries, got %d", len(entries))
		}

		// Check specific paths
		paths := make(map[string]bool)
		for _, entry := range entries {
			paths[entry["path"].(string)] = true
		}
		expectedPaths := []string{"root.txt", "dir1", "dir1/file1.txt", "dir1/dir2", "dir1/dir2/file2.txt"}
		for _, p := range expectedPaths {
			if !paths[p] {
				t.Errorf("Expected path %q not found in response", p)
			}
		}
	})

	t.Run("TreeExpandFalse", func(t *testing.T) {
		// Request tree with expand=false
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/tree-test/tree/main?expand=false", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var entries []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// When expand=false, entries should NOT have lastCommit field
		for _, entry := range entries {
			if _, hasLastCommit := entry["lastCommit"]; hasLastCommit {
				t.Errorf("Expected no lastCommit when expand=false, but found one for %v", entry["path"])
			}
		}
	})

	t.Run("TreeRecursiveAndExpandFalse", func(t *testing.T) {
		// Request tree with recursive=true and expand=false
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/tree-test/tree/main?recursive=true&expand=false", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var entries []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have all 5 entries
		if len(entries) != 5 {
			t.Errorf("Expected 5 entries, got %d", len(entries))
		}

		// All entries should NOT have lastCommit field
		for _, entry := range entries {
			if _, hasLastCommit := entry["lastCommit"]; hasLastCommit {
				t.Errorf("Expected no lastCommit when expand=false, but found one for %v", entry["path"])
			}
		}
	})

	t.Run("TreeExpandTrue", func(t *testing.T) {
		// Request tree with expand=true (default)
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/tree-test/tree/main?expand=true", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var entries []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// When expand=true, entries should have lastCommit field
		hasLastCommitCount := 0
		for _, entry := range entries {
			if _, hasLastCommit := entry["lastCommit"]; hasLastCommit {
				hasLastCommitCount++
				// Verify lastCommit has expected fields
				lastCommit := entry["lastCommit"].(map[string]any)
				if _, ok := lastCommit["id"]; !ok {
					t.Errorf("Expected lastCommit to have 'id' field")
				}
				if _, ok := lastCommit["title"]; !ok {
					t.Errorf("Expected lastCommit to have 'title' field")
				}
				if _, ok := lastCommit["date"]; !ok {
					t.Errorf("Expected lastCommit to have 'date' field")
				}
			}
		}
		if hasLastCommitCount == 0 {
			t.Errorf("Expected at least one entry with lastCommit when expand=true")
		}
	})
}

// TestHuggingFaceCLI tests the HuggingFace CLI (hf command) integration.
// This is an e2e test that requires the huggingface_hub package to be installed.
func TestHuggingFaceCLI(t *testing.T) {
	// Check if hf CLI is available
	if _, err := exec.LookPath("hf"); err != nil {
		t.Skip("hf CLI not found, skipping test. Install with: pip install huggingface_hub")
	}

	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "matrixhub-hf-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "matrixhub-hf-cli-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	// Create handler and test server
	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	repoName := "test-org/test-model.git"
	repoID := "test-org/test-model"

	// Create repository on server
	t.Run("CreateRepository", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}
	})

	// Clone and add content to the repository
	gitWorkDir := filepath.Join(clientDir, "git-work")
	t.Run("CloneAndAddContent", func(t *testing.T) {
		// Clone the repository using git
		cmd := exec.Command("git", "clone", server.URL+"/"+repoName, gitWorkDir)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
		}

		// Configure git user
		runGitCmd(t, gitWorkDir, "config", "user.email", "test@test.com")
		runGitCmd(t, gitWorkDir, "config", "user.name", "Test User")

		// Create test files
		configContent := `{"model_type": "test", "hidden_size": 768}`
		if err := os.WriteFile(filepath.Join(gitWorkDir, "config.json"), []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create config.json: %v", err)
		}

		readmeContent := "# Test Model\n\nThis is a test model for e2e testing."
		if err := os.WriteFile(filepath.Join(gitWorkDir, "README.md"), []byte(readmeContent), 0644); err != nil {
			t.Fatalf("Failed to create README.md: %v", err)
		}

		// Add and commit
		runGitCmd(t, gitWorkDir, "add", ".")
		runGitCmd(t, gitWorkDir, "commit", "-m", "Add model files")

		// Rename branch to main to match server default
		runGitCmd(t, gitWorkDir, "branch", "-M", "main")

		// Push
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = gitWorkDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push: %v\nOutput: %s", err, output)
		}
	})

	// Add LFS content to the repository
	t.Run("AddLFSContent", func(t *testing.T) {
		// Check if git-lfs is available
		if _, err := exec.LookPath("git-lfs"); err != nil {
			t.Skip("git-lfs not found, skipping LFS test")
		}

		// Initialize LFS
		runGitLFSCmd(t, gitWorkDir, "install", "--local")

		// Track binary files with LFS
		runGitLFSCmd(t, gitWorkDir, "track", "*.bin")
		runGitLFSCmd(t, gitWorkDir, "track", "*.safetensors")

		// Commit .gitattributes
		runGitCmd(t, gitWorkDir, "add", ".gitattributes")
		runGitCmd(t, gitWorkDir, "commit", "-m", "Configure LFS tracking")

		// Create a binary file that will be tracked by LFS (simulating model weights)
		binFile := filepath.Join(gitWorkDir, "model.safetensors")
		// Create 10KB of binary data to simulate a model file
		data := make([]byte, 10*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if err := os.WriteFile(binFile, data, 0644); err != nil {
			t.Fatalf("Failed to create model.safetensors: %v", err)
		}

		// Add and commit
		runGitCmd(t, gitWorkDir, "add", "model.safetensors")
		runGitCmd(t, gitWorkDir, "commit", "-m", "Add model weights (LFS)")

		// Push
		cmd := exec.Command("git", "push")
		cmd.Dir = gitWorkDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push LFS content: %v\nOutput: %s", err, output)
		}

		// Verify LFS is tracking the file
		cmd = exec.Command("git", "lfs", "ls-files")
		cmd.Dir = gitWorkDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list LFS files: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "model.safetensors") {
			t.Errorf("model.safetensors should be tracked by LFS, got: %s", output)
		}
	})

	// Test hf models info command
	t.Run("HFModelsInfo", func(t *testing.T) {
		cmd := exec.Command("hf", "models", "info", repoID)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf models info failed: %v\nOutput: %s", err, output)
		}

		// Verify the output contains expected fields
		outputStr := string(output)
		if !strings.Contains(outputStr, repoID) {
			t.Errorf("Expected output to contain repo ID '%s', got: %s", repoID, outputStr)
		}
	})

	// Test hf download command for a single file
	hfDownloadDir := filepath.Join(clientDir, "hf-download")
	t.Run("HFDownloadSingleFile", func(t *testing.T) {
		if err := os.MkdirAll(hfDownloadDir, 0755); err != nil {
			t.Fatalf("Failed to create download dir: %v", err)
		}

		cmd := exec.Command("hf", "download", repoID, "config.json", "--local-dir", hfDownloadDir)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf download failed: %v\nOutput: %s", err, output)
		}

		// Verify the file was downloaded
		downloadedFile := filepath.Join(hfDownloadDir, "config.json")
		content, err := os.ReadFile(downloadedFile)
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		if !strings.Contains(string(content), "model_type") {
			t.Errorf("Downloaded file content doesn't match expected. Got: %s", content)
		}
	})

	// Test hf download command for entire repo
	hfDownloadFullDir := filepath.Join(clientDir, "hf-download-full")
	t.Run("HFDownloadFullRepo", func(t *testing.T) {
		if err := os.MkdirAll(hfDownloadFullDir, 0755); err != nil {
			t.Fatalf("Failed to create download dir: %v", err)
		}

		cmd := exec.Command("hf", "download", repoID, "--local-dir", hfDownloadFullDir)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf download (full) failed: %v\nOutput: %s", err, output)
		}

		// Verify both files were downloaded
		configFile := filepath.Join(hfDownloadFullDir, "config.json")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("config.json was not downloaded")
		}

		readmeFile := filepath.Join(hfDownloadFullDir, "README.md")
		if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
			t.Error("README.md was not downloaded")
		}

		// Verify content
		content, err := os.ReadFile(readmeFile)
		if err != nil {
			t.Fatalf("Failed to read README.md: %v", err)
		}
		if !strings.Contains(string(content), "Test Model") {
			t.Errorf("README.md content doesn't match expected. Got: %s", content)
		}
	})

	// Test hf download command for LFS content
	hfDownloadLFSDir := filepath.Join(clientDir, "hf-download-lfs")
	t.Run("HFDownloadLFSContent", func(t *testing.T) {
		// Skip if git-lfs is not available (LFS content wasn't added)
		if _, err := exec.LookPath("git-lfs"); err != nil {
			t.Skip("git-lfs not found, skipping LFS download test")
		}

		if err := os.MkdirAll(hfDownloadLFSDir, 0755); err != nil {
			t.Fatalf("Failed to create download dir: %v", err)
		}

		// Download the LFS file specifically
		cmd := exec.Command("hf", "download", repoID, "model.safetensors", "--local-dir", hfDownloadLFSDir)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf download (LFS) failed: %v\nOutput: %s", err, output)
		}

		// Verify the LFS file was downloaded with correct content
		lfsFile := filepath.Join(hfDownloadLFSDir, "model.safetensors")
		content, err := os.ReadFile(lfsFile)
		if err != nil {
			t.Fatalf("Failed to read downloaded LFS file: %v", err)
		}

		// Verify the file size matches what we created (10KB)
		expectedSize := 10 * 1024
		if len(content) != expectedSize {
			t.Errorf("LFS file size mismatch: expected %d bytes, got %d bytes", expectedSize, len(content))
		}

		// Verify the content is correct (we wrote i % 256 for each byte)
		l := min(100, len(content))
		for i := range l {
			if content[i] != byte(i%256) {
				t.Errorf("LFS content mismatch at byte %d: expected %d, got %d", i, i%256, content[i])
				break
			}
		}
	})

	// Cleanup
	t.Run("DeleteRepository", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, server.URL+"/api/repositories/"+repoName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", resp.StatusCode)
		}
	})
}

// TestHuggingFaceHubTreeAPI tests the HuggingFace Hub CLI for downloading files.
// This is an e2e test that requires the huggingface_hub package to be installed.
func TestHuggingFaceHubTreeAPI(t *testing.T) {
	// Check if hf CLI is available
	if _, err := exec.LookPath("hf"); err != nil {
		t.Skip("hf CLI not found, skipping test. Install with: pip install huggingface_hub")
	}

	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "matrixhub-hf-tree-api-test")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "matrixhub-hf-tree-api-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	// Create handler and test server
	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	repoName := "tree-api-test.git"
	repoID := "tree-api-test"

	// Create repository on server
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 for create, got %d", resp.StatusCode)
	}

	// Clone and add nested content to the repository
	gitWorkDir := filepath.Join(clientDir, "git-work")
	cmd := exec.Command("git", "clone", server.URL+"/"+repoName, gitWorkDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
	}

	// Configure git user
	runGitCmd(t, gitWorkDir, "config", "user.email", "test@test.com")
	runGitCmd(t, gitWorkDir, "config", "user.name", "Test User")

	// Create nested structure: subdir/nested_file.txt
	subDir := filepath.Join(gitWorkDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitWorkDir, "root_file.txt"), []byte("root content"), 0644); err != nil {
		t.Fatalf("Failed to create root_file.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested_file.txt"), []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested_file.txt: %v", err)
	}

	// Add, commit, and push
	runGitCmd(t, gitWorkDir, "add", ".")
	runGitCmd(t, gitWorkDir, "commit", "-m", "Add nested files")
	runGitCmd(t, gitWorkDir, "branch", "-M", "main")
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = gitWorkDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to push: %v\nOutput: %s", err, output)
	}

	// Test hf download for specific file (this uses the resolve endpoint)
	t.Run("HFDownloadSingleFile", func(t *testing.T) {
		downloadDir := filepath.Join(clientDir, "hf-download-single")
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			t.Fatalf("Failed to create download dir: %v", err)
		}

		// Download single file
		cmd := exec.Command("hf", "download", repoID, "root_file.txt", "--local-dir", downloadDir)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf download failed: %v\nOutput: %s", err, output)
		}

		// Verify file was downloaded
		content, err := os.ReadFile(filepath.Join(downloadDir, "root_file.txt"))
		if err != nil {
			t.Fatalf("Failed to read root_file.txt: %v", err)
		}
		if string(content) != "root content" {
			t.Errorf("Content mismatch: expected 'root content', got '%s'", content)
		}
	})

	// Test hf download for specific nested file (this uses the resolve endpoint)
	t.Run("HFDownloadNestedFile", func(t *testing.T) {
		downloadDir := filepath.Join(clientDir, "hf-download-nested")
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			t.Fatalf("Failed to create download dir: %v", err)
		}

		// Download specific nested file
		cmd := exec.Command("hf", "download", repoID, "subdir/nested_file.txt", "--local-dir", downloadDir)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf download nested file failed: %v\nOutput: %s", err, output)
		}

		// Verify nested file was downloaded
		content, err := os.ReadFile(filepath.Join(downloadDir, "subdir", "nested_file.txt"))
		if err != nil {
			t.Fatalf("Failed to read nested_file.txt: %v", err)
		}
		if string(content) != "nested content" {
			t.Errorf("Nested file content mismatch: expected 'nested content', got '%s'", content)
		}
	})

	// Test hf models info command
	t.Run("HFModelsInfo", func(t *testing.T) {
		cmd := exec.Command("hf", "models", "info", repoID)
		cmd.Env = append(os.Environ(),
			"HF_ENDPOINT="+server.URL,
			"HF_HUB_DISABLE_TELEMETRY=1",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hf models info failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		// Verify the output contains the repo ID
		if !strings.Contains(outputStr, repoID) {
			t.Errorf("Expected output to contain repo ID '%s', got: %s", repoID, outputStr)
		}
	})
}

// runGitCmd runs a git command in the specified directory.
func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git command failed: git %s\nError: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
}

// runGitLFSCmd runs a git-lfs command in the specified directory.
func runGitLFSCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	fullArgs := append([]string{"lfs"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git LFS command failed: git lfs %s\nError: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
}
