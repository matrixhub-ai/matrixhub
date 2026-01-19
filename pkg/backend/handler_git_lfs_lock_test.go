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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/handlers"

	"github.com/matrixhub-ai/matrixhub/internal/utils"
	"github.com/matrixhub-ai/matrixhub/pkg/backend"
)

// TestGitLFSLock tests the Git LFS lock functionality using git-lfs binary.
func TestGitLFSLock(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "matrixhub-lfs-lock-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "matrixhub-lfs-lock-test-client")
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

	repoName := "lfs-lock-test-repo.git"
	repoURL := server.URL + "/" + repoName

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

	workDir := filepath.Join(clientDir, "lfs-lock-work")

	t.Run("CloneAndConfigureLFS", func(t *testing.T) {
		// Clone the repository
		runGitCmd(t, "", "clone", repoURL, workDir)

		// Configure git user
		runGitCmd(t, workDir, "config", "user.email", "test@test.com")
		runGitCmd(t, workDir, "config", "user.name", "Test User")

		// Initialize LFS
		runGitLFSCmd(t, workDir, "install", "--local")

		// Track binary files with LFS
		runGitLFSCmd(t, workDir, "track", "*.bin")

		// Commit .gitattributes
		runGitCmd(t, workDir, "add", ".gitattributes")
		runGitCmd(t, workDir, "commit", "-m", "Configure LFS tracking")

		// Push initial commit
		runGitCmd(t, workDir, "push", "-u", "origin", "HEAD")
	})

	// Create and push a binary file for locking tests
	t.Run("CreateBinaryFileForLocking", func(t *testing.T) {
		// Create a binary file
		binFile := filepath.Join(workDir, "lockable.bin")
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if err := os.WriteFile(binFile, data, 0644); err != nil {
			t.Fatalf("Failed to create binary file: %v", err)
		}

		// Add and commit
		runGitCmd(t, workDir, "add", "lockable.bin")
		runGitCmd(t, workDir, "commit", "-m", "Add lockable binary file")

		// Push with LFS
		runGitCmd(t, workDir, "push")
	})

	// Test locking a file
	t.Run("LockFile", func(t *testing.T) {
		output := runGitLFSCmd(t, workDir, "lock", "lockable.bin")
		if !strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected lock output to contain 'lockable.bin', got: %s", output)
		}
	})

	// Test listing locks
	t.Run("ListLocks", func(t *testing.T) {
		output := runGitLFSCmd(t, workDir, "locks")
		if !strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected locks output to contain 'lockable.bin', got: %s", output)
		}
	})

	// Test locking the same file again should fail
	t.Run("LockSameFileShouldFail", func(t *testing.T) {
		cmd := utils.Command(t.Context(), "git", "lfs", "lock", "lockable.bin")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.Output()
		if err == nil {
			t.Errorf("Expected lock to fail for already locked file, but it succeeded. Output: %s", output)
		}
		// The error should indicate the file is already locked
		if !strings.Contains(string(output), "lock already created") && !strings.Contains(string(output), "already locked") {
			t.Logf("Lock failed with output: %s", output)
		}
	})

	// Test verify locks
	t.Run("VerifyLocks", func(t *testing.T) {
		output := runGitLFSCmd(t, workDir, "locks", "--verify")
		// Verify should show the lock as "ours"
		if !strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected verify output to contain 'lockable.bin', got: %s", output)
		}
	})

	// Test lock with JSON output
	t.Run("ListLocksJSON", func(t *testing.T) {
		output := runGitLFSCmd(t, workDir, "locks", "--json")
		if !strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected JSON locks output to contain 'lockable.bin', got: %s", output)
		}
		// Should be valid JSON
		if !strings.Contains(output, "{") {
			t.Errorf("Expected JSON output, got: %s", output)
		}
	})

	// Test unlock file
	t.Run("UnlockFile", func(t *testing.T) {
		output := runGitLFSCmd(t, workDir, "unlock", "lockable.bin")
		if !strings.Contains(output, "lockable.bin") && !strings.Contains(output, "unlocked") {
			t.Logf("Unlock output: %s", output)
		}
	})

	// Test locks should be empty after unlock
	t.Run("VerifyUnlock", func(t *testing.T) {
		output := runGitLFSCmd(t, workDir, "locks")
		if strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected no locks after unlock, but got: %s", output)
		}
	})

	// Test lock and force unlock
	t.Run("LockAndForceUnlock", func(t *testing.T) {
		// Lock the file again
		runGitLFSCmd(t, workDir, "lock", "lockable.bin")

		// Force unlock
		output := runGitLFSCmd(t, workDir, "unlock", "--force", "lockable.bin")
		if !strings.Contains(output, "lockable.bin") && !strings.Contains(output, "unlocked") {
			t.Logf("Force unlock output: %s", output)
		}

		// Verify no locks
		output = runGitLFSCmd(t, workDir, "locks")
		if strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected no locks after force unlock, but got: %s", output)
		}
	})

	// Test locking multiple files
	t.Run("LockMultipleFiles", func(t *testing.T) {
		// Create more binary files
		for _, name := range []string{"file1.bin", "file2.bin", "file3.bin"} {
			binFile := filepath.Join(workDir, name)
			data := make([]byte, 512)
			for i := range data {
				data[i] = byte(i % 256)
			}
			if err := os.WriteFile(binFile, data, 0644); err != nil {
				t.Fatalf("Failed to create binary file %s: %v", name, err)
			}
		}

		// Add and commit
		runGitCmd(t, workDir, "add", ".")
		runGitCmd(t, workDir, "commit", "-m", "Add multiple binary files")

		// Push
		runGitCmd(t, workDir, "push")

		// Lock all files
		runGitLFSCmd(t, workDir, "lock", "file1.bin")
		runGitLFSCmd(t, workDir, "lock", "file2.bin")
		runGitLFSCmd(t, workDir, "lock", "file3.bin")

		// List locks should show all three
		locksOutput := runGitLFSCmd(t, workDir, "locks")
		for _, name := range []string{"file1.bin", "file2.bin", "file3.bin"} {
			if !strings.Contains(locksOutput, name) {
				t.Errorf("Expected locks to contain %s, got: %s", name, locksOutput)
			}
		}

		// Unlock all
		runGitLFSCmd(t, workDir, "unlock", "file1.bin")
		runGitLFSCmd(t, workDir, "unlock", "file2.bin")
		runGitLFSCmd(t, workDir, "unlock", "file3.bin")
	})

	// Test lock with limit parameter
	t.Run("LockWithLimit", func(t *testing.T) {
		// Lock multiple files
		runGitLFSCmd(t, workDir, "lock", "file1.bin")
		runGitLFSCmd(t, workDir, "lock", "file2.bin")
		runGitLFSCmd(t, workDir, "lock", "file3.bin")

		// List locks with limit
		output := runGitLFSCmd(t, workDir, "locks", "--limit", "2")
		// Count how many locks are shown (should be limited)
		lines := strings.Split(strings.TrimSpace(output), "\n")
		lockCount := 0
		for _, line := range lines {
			if strings.Contains(line, ".bin") {
				lockCount++
			}
		}
		if lockCount > 2 {
			t.Errorf("Expected at most 2 locks with --limit 2, got %d locks: %s", lockCount, output)
		}

		// Cleanup
		runGitLFSCmd(t, workDir, "unlock", "file1.bin")
		runGitLFSCmd(t, workDir, "unlock", "file2.bin")
		runGitLFSCmd(t, workDir, "unlock", "file3.bin")
	})

	// Test lock by path filter
	t.Run("LockFilterByPath", func(t *testing.T) {
		// Lock a file
		runGitLFSCmd(t, workDir, "lock", "lockable.bin")

		// List locks with path filter
		output := runGitLFSCmd(t, workDir, "locks", "--path", "lockable.bin")
		if !strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected filtered locks to contain 'lockable.bin', got: %s", output)
		}

		// Filter for non-existent path should return no locks
		output = runGitLFSCmd(t, workDir, "locks", "--path", "nonexistent.bin")
		if strings.Contains(output, "lockable.bin") {
			t.Errorf("Expected filtered locks to not contain 'lockable.bin', got: %s", output)
		}

		// Cleanup
		runGitLFSCmd(t, workDir, "unlock", "lockable.bin")
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
