package backend_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/handlers"
	"github.com/wzshiming/gitd/pkg/backend"
)

// TestGitLFSServer tests the Git LFS server using git-lfs binary.
func TestGitLFSServer(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "gitd-lfs-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "gitd-lfs-test-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer os.RemoveAll(clientDir)

	// Create handler and test server
	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	repoName := "lfs-test-repo.git"
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
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}
	})

	workDir := filepath.Join(clientDir, "lfs-work")

	t.Run("CloneAndConfigureLFS", func(t *testing.T) {
		// Clone the repository
		cmd := exec.Command("git", "clone", repoURL, workDir)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
		}

		// Configure git user
		runGitCommand(t, workDir, "config", "user.email", "test@test.com")
		runGitCommand(t, workDir, "config", "user.name", "Test User")

		// Initialize LFS
		runGitLFSCommand(t, workDir, "install", "--local")

		// Track binary files with LFS
		runGitLFSCommand(t, workDir, "track", "*.bin")
		runGitLFSCommand(t, workDir, "track", "*.dat")

		// Commit .gitattributes
		runGitCommand(t, workDir, "add", ".gitattributes")
		runGitCommand(t, workDir, "commit", "-m", "Configure LFS tracking")
	})

	t.Run("PushLFSTracking", func(t *testing.T) {
		// Push initial commit with .gitattributes
		cmd := exec.Command("git", "push", "-u", "origin", "HEAD")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push LFS tracking: %v\nOutput: %s", err, output)
		}
	})

	// Test uploading a small binary file
	t.Run("UploadSmallBinaryFile", func(t *testing.T) {
		// Create a small binary file (1KB)
		binFile := filepath.Join(workDir, "small.bin")
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if err := os.WriteFile(binFile, data, 0644); err != nil {
			t.Fatalf("Failed to create binary file: %v", err)
		}

		// Add and commit
		runGitCommand(t, workDir, "add", "small.bin")
		runGitCommand(t, workDir, "commit", "-m", "Add small binary file")

		// Push with LFS
		cmd := exec.Command("git", "push")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push LFS file: %v\nOutput: %s", err, output)
		}

		// Verify LFS is tracking the file
		cmd = exec.Command("git", "lfs", "ls-files")
		cmd.Dir = workDir
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to list LFS files: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "small.bin") {
			t.Errorf("small.bin should be tracked by LFS, got: %s", output)
		}
	})

	// Test uploading a larger binary file
	t.Run("UploadLargeBinaryFile", func(t *testing.T) {
		// Create a larger binary file (100KB)
		binFile := filepath.Join(workDir, "large.dat")
		data := make([]byte, 100*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		if err := os.WriteFile(binFile, data, 0644); err != nil {
			t.Fatalf("Failed to create large binary file: %v", err)
		}

		// Add and commit
		runGitCommand(t, workDir, "add", "large.dat")
		runGitCommand(t, workDir, "commit", "-m", "Add large binary file")

		// Push with LFS
		cmd := exec.Command("git", "push")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push large LFS file: %v\nOutput: %s", err, output)
		}
	})

	// Test cloning with LFS content
	t.Run("CloneWithLFSContent", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "lfs-clone")

		// Clone the repository
		cmd := exec.Command("git", "clone", repoURL, cloneDir)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
		}

		// Pull LFS content
		runGitLFSCommand(t, cloneDir, "pull")

		// Verify small.bin content
		smallBin := filepath.Join(cloneDir, "small.bin")
		content, err := os.ReadFile(smallBin)
		if err != nil {
			t.Fatalf("Failed to read small.bin: %v", err)
		}
		if len(content) != 1024 {
			t.Errorf("Expected 1024 bytes, got %d", len(content))
		}
		// Verify content is correct
		for i, b := range content {
			if b != byte(i%256) {
				t.Errorf("Content mismatch at byte %d: expected %d, got %d", i, i%256, b)
				break
			}
		}

		// Verify large.dat content
		largeDat := filepath.Join(cloneDir, "large.dat")
		content, err = os.ReadFile(largeDat)
		if err != nil {
			t.Fatalf("Failed to read large.dat: %v", err)
		}
		if len(content) != 100*1024 {
			t.Errorf("Expected %d bytes, got %d", 100*1024, len(content))
		}
	})

	// Test LFS fetch
	t.Run("LFSFetch", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "lfs-clone")

		cmd := exec.Command("git", "lfs", "fetch", "--all")
		cmd.Dir = cloneDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to fetch LFS objects: %v\nOutput: %s", err, output)
		}
	})

	// Test updating LFS file
	t.Run("UpdateLFSFile", func(t *testing.T) {
		// Update the small binary file
		binFile := filepath.Join(workDir, "small.bin")
		data := make([]byte, 2048) // Double the size
		for i := range data {
			data[i] = byte((i * 2) % 256)
		}
		if err := os.WriteFile(binFile, data, 0644); err != nil {
			t.Fatalf("Failed to update binary file: %v", err)
		}

		// Add and commit
		runGitCommand(t, workDir, "add", "small.bin")
		runGitCommand(t, workDir, "commit", "-m", "Update small binary file")

		// Push
		cmd := exec.Command("git", "push")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push updated LFS file: %v\nOutput: %s", err, output)
		}
	})

	// Test pulling updated LFS content
	t.Run("PullUpdatedLFSContent", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "lfs-clone")

		// Pull changes
		cmd := exec.Command("git", "pull")
		cmd.Dir = cloneDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to pull: %v\nOutput: %s", err, output)
		}

		// Verify updated content
		smallBin := filepath.Join(cloneDir, "small.bin")
		content, err := os.ReadFile(smallBin)
		if err != nil {
			t.Fatalf("Failed to read small.bin: %v", err)
		}
		if len(content) != 2048 {
			t.Errorf("Expected 2048 bytes, got %d", len(content))
		}
	})

	// Test LFS status
	t.Run("LFSStatus", func(t *testing.T) {
		cmd := exec.Command("git", "lfs", "status")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get LFS status: %v\nOutput: %s", err, output)
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
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", resp.StatusCode)
		}
	})
}

// runGitLFSCommand runs a git-lfs command in the specified directory.
func runGitLFSCommand(t *testing.T, dir string, args ...string) {
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
