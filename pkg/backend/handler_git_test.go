package backend_test

import (
	"bytes"
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

// TestGitServer tests the git server using the git binary.
func TestGitServer(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "matrixhub-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "matrixhub-test-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer os.RemoveAll(clientDir)

	// Create handler and test server
	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	repoName := "test-repo.git"
	repoURL := server.URL + "/" + repoName

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

		// Verify repository was created
		repoPath := filepath.Join(repoDir, "repositories", repoName)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			t.Errorf("Repository was not created at %s", repoPath)
		}
	})

	t.Run("CloneEmptyRepository", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "clone-empty")

		cmd := exec.Command("git", "clone", repoURL, cloneDir)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
		}

		// Verify .git directory exists
		matrixhubir := filepath.Join(cloneDir, ".git")
		if _, err := os.Stat(matrixhubir); os.IsNotExist(err) {
			t.Errorf(".git directory not found in cloned repository")
		}
	})

	t.Run("PushToRepository", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-empty")

		// Configure git user for commits
		runGitCommand(t, workDir, "config", "user.email", "test@test.com")
		runGitCommand(t, workDir, "config", "user.name", "Test User")

		// Create a test file
		testFile := filepath.Join(workDir, "README.md")
		if err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Add and commit
		runGitCommand(t, workDir, "add", "README.md")
		runGitCommand(t, workDir, "commit", "-m", "Initial commit")

		// Push to remote
		cmd := exec.Command("git", "push", "-u", "origin", "master")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		// Try main branch if master fails
		if err != nil {
			cmd = exec.Command("git", "push", "-u", "origin", "main")
			cmd.Dir = workDir
			cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
			output, err = cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to push to repository: %v\nOutput: %s", err, output)
			}
		}
	})

	t.Run("CloneWithContent", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "clone-with-content")

		cmd := exec.Command("git", "clone", repoURL, cloneDir)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
		}

		// Verify README.md exists
		readmePath := filepath.Join(cloneDir, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			t.Errorf("README.md not found in cloned repository")
		}

		// Verify content
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("Failed to read README.md: %v", err)
		}
		if string(content) != "# Test Repository\n" {
			t.Errorf("Unexpected content: %s", content)
		}
	})

	t.Run("FetchFromRepository", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-with-content")

		cmd := exec.Command("git", "fetch", "origin")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to fetch from repository: %v\nOutput: %s", err, output)
		}
	})

	t.Run("PushMoreCommits", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-with-content")

		// Create another file
		testFile := filepath.Join(workDir, "file2.txt")
		if err := os.WriteFile(testFile, []byte("Second file\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Configure git user
		runGitCommand(t, workDir, "config", "user.email", "test@test.com")
		runGitCommand(t, workDir, "config", "user.name", "Test User")

		// Add and commit
		runGitCommand(t, workDir, "add", "file2.txt")
		runGitCommand(t, workDir, "commit", "-m", "Add second file")

		// Push
		cmd := exec.Command("git", "push")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to push to repository: %v\nOutput: %s", err, output)
		}
	})

	t.Run("PullChanges", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-empty")

		// Configure git user
		runGitCommand(t, workDir, "config", "user.email", "test@test.com")
		runGitCommand(t, workDir, "config", "user.name", "Test User")

		// Pull changes from another clone
		cmd := exec.Command("git", "pull")
		cmd.Dir = workDir
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to pull from repository: %v\nOutput: %s", err, output)
		}

		// Verify file2.txt exists
		file2Path := filepath.Join(workDir, "file2.txt")
		if _, err := os.Stat(file2Path); os.IsNotExist(err) {
			t.Errorf("file2.txt not found after pull")
		}
	})

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

		// Verify repository was deleted
		repoPath := filepath.Join(repoDir, "repositories", repoName)
		if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
			t.Errorf("Repository was not deleted at %s", repoPath)
		}
	})
}

// TestInfoRefs tests the /info/refs endpoint.
func TestInfoRefs(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "matrixhub-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	// Create a bare repository
	repoName := "test.git"
	repoPath := filepath.Join(repoDir, "repositories", repoName)
	cmd := exec.Command("git", "init", "--bare", repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create bare repository: %v\nOutput: %s", err, output)
	}

	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("UploadPackService", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/" + repoName + "/info/refs?service=git-upload-pack")
		if err != nil {
			t.Fatalf("Failed to get info/refs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/x-git-upload-pack-advertisement" {
			t.Errorf("Unexpected Content-Type: %s", contentType)
		}

		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		body := buf.String()

		if !strings.Contains(body, "# service=git-upload-pack") {
			t.Errorf("Response body should contain service header")
		}
	})

	t.Run("ReceivePackService", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/" + repoName + "/info/refs?service=git-receive-pack")
		if err != nil {
			t.Fatalf("Failed to get info/refs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/x-git-receive-pack-advertisement" {
			t.Errorf("Unexpected Content-Type: %s", contentType)
		}
	})

	t.Run("MissingServiceParam", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/" + repoName + "/info/refs")
		if err != nil {
			t.Fatalf("Failed to get info/refs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("UnsupportedService", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/" + repoName + "/info/refs?service=invalid")
		if err != nil {
			t.Fatalf("Failed to get info/refs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", resp.StatusCode)
		}
	})

	t.Run("NonExistentRepository", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/nonexistent.git/info/refs?service=git-upload-pack")
		if err != nil {
			t.Fatalf("Failed to get info/refs: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
}

// runGitCommand runs a git command in the specified directory.
func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git command failed: git %s\nError: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
}
