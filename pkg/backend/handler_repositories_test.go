package backend_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/handlers"
	"github.com/matrixhub-ai/matrixhub/pkg/backend"
)

// TestRepositoryManagement tests repository creation and deletion.
func TestRepositoryManagement(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "matrixhub-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("CreateAndDeleteRepository", func(t *testing.T) {
		repoName := "new-repo.git"

		// Create repository
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201 for create, got %d", resp.StatusCode)
		}

		// Verify it's a valid git repository
		repoPath := filepath.Join(repoDir, "repositories", repoName)
		headPath := filepath.Join(repoPath, "HEAD")
		if _, err := os.Stat(headPath); os.IsNotExist(err) {
			t.Errorf("HEAD file not found, not a valid git repository")
		}

		// Delete repository
		req, _ = http.NewRequest(http.MethodDelete, server.URL+"/api/repositories/"+repoName, nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to delete repository: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 204 for delete, got %d", resp.StatusCode)
		}

		// Verify deletion
		if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
			t.Errorf("Repository still exists after deletion")
		}
	})

	t.Run("CreateDuplicateRepository", func(t *testing.T) {
		repoName := "duplicate-repo.git"

		// Create first time
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()

		// Create second time
		req, _ = http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected status 409 for duplicate, got %d", resp.StatusCode)
		}
	})

	t.Run("DeleteNonExistentRepository", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/repositories/nonexistent.git", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})

	t.Run("NestedRepository", func(t *testing.T) {
		repoName := "org/team/project.git"

		// Create nested repository
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201 for create, got %d", resp.StatusCode)
		}

		// Verify it exists
		repoPath := filepath.Join(repoDir, "repositories", repoName)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			t.Errorf("Nested repository was not created")
		}
	})
}
