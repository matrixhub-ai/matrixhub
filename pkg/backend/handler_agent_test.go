package backend_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gorilla/handlers"
	"github.com/matrixhub-ai/matrixhub/pkg/backend"
	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

func runGitCommandAgent(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}

func TestAgentsAPI(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "matrixhub-agents-api-test")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test repository without AGENTS.md
	t.Run("AgentsContentNotFound", func(t *testing.T) {
		repoName := "no-agents.git"

		// Create and populate repository without AGENTS.md
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-no-agents")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Add a README but no AGENTS.md
		readmePath := filepath.Join(workDir, "README.md")
		if err := os.WriteFile(readmePath, []byte("# Test\n"), 0644); err != nil {
			t.Fatalf("Failed to create README: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "README.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Initial commit")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Request AGENTS.md for repo without it
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/no-agents/agents", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for missing AGENTS.md, got %d", resp.StatusCode)
		}
	})

	// Test repository with AGENTS.md
	t.Run("AgentsContentFound", func(t *testing.T) {
		repoName := "with-agents.git"

		// Create and populate repository with AGENTS.md
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-agents")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Create AGENTS.md with plain markdown content
		agentsContent := "# AGENTS Guidelines for This Repository\n\n" +
			"## Dev environment tips\n" +
			"- Use `pnpm dlx turbo run where <project_name>` to jump to a package\n" +
			"- Run `pnpm install --filter <project_name>` to add the package\n\n" +
			"## Testing instructions\n" +
			"- Find the CI plan in the .github/workflows folder\n" +
			"- Run `pnpm turbo run test --filter <project_name>` to run checks\n\n" +
			"## PR instructions\n" +
			"- Title format: [<project_name>] <Title>\n" +
			"- Always run `pnpm lint` and `pnpm test` before committing\n"
		agentsPath := filepath.Join(workDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "AGENTS.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Add AGENTS.md")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Request AGENTS.md via API
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/with-agents/agents", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify response structure
		var content repository.AgentsContent
		if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify content
		if content.Content == "" {
			t.Error("Expected non-empty content")
		}
		if len(content.Content) < 50 {
			t.Errorf("Content seems too short: %s", content.Content)
		}
	})

	// Test AGENTS.md with revision
	t.Run("AgentsContentWithRevision", func(t *testing.T) {
		repoName := "revision-agents.git"

		// Create and populate repository
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-revision")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Create initial AGENTS.md
		agentsContent := `# AGENTS v1

Main branch content
`
		agentsPath := filepath.Join(workDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "AGENTS.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Add v1 AGENTS.md")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Update AGENTS.md in a new branch
		runGitCommandAgent(t, workDir, "checkout", "-b", "v2")
		agentsContent = `# AGENTS v2

Version 2 content with new instructions
`
		if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
			t.Fatalf("Failed to update AGENTS.md: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "AGENTS.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Update to v2")
		runGitCommandAgent(t, workDir, "push", "origin", "v2")

		// Request content from main branch
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/revision-agents/agents/revision/main", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md for main: %v", err)
		}
		defer resp.Body.Close()

		var mainContent repository.AgentsContent
		if err := json.NewDecoder(resp.Body).Decode(&mainContent); err != nil {
			t.Fatalf("Failed to decode main content: %v", err)
		}

		// Should contain v1 content
		if len(mainContent.Content) == 0 {
			t.Error("Expected main branch to have content")
		}

		// Request content from v2 branch
		req, _ = http.NewRequest(http.MethodGet, server.URL+"/api/models/revision-agents/agents/revision/v2", nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md for v2: %v", err)
		}
		defer resp.Body.Close()

		var v2Content repository.AgentsContent
		if err := json.NewDecoder(resp.Body).Decode(&v2Content); err != nil {
			t.Fatalf("Failed to decode v2 content: %v", err)
		}

		// Should contain v2 content
		if len(v2Content.Content) == 0 {
			t.Error("Expected v2 branch to have content")
		}
	})

	// Test model info includes AGENTS.md content
	t.Run("ModelInfoIncludesAgentsContent", func(t *testing.T) {
		repoName := "model-with-agents.git"

		// Create and populate repository
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-model")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Create AGENTS.md
		agentsContent := `# AGENTS for Model

Model-specific agent guidelines
`
		agentsPath := filepath.Join(workDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}

		// Also add a model file
		modelPath := filepath.Join(workDir, "model.bin")
		if err := os.WriteFile(modelPath, []byte("fake model data"), 0644); err != nil {
			t.Fatalf("Failed to create model file: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", ".")
		runGitCommandAgent(t, workDir, "commit", "-m", "Add AGENTS.md and model")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Request model info via HF API
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/model-with-agents", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get model info: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify response includes AGENTS.md content
		var modelInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check that agentsContent field exists
		agentsContentField, ok := modelInfo["agentsContent"]
		if !ok {
			t.Error("Expected agentsContent field in model info")
		} else {
			// Verify it's populated
			contentMap, ok := agentsContentField.(map[string]interface{})
			if !ok {
				t.Error("Expected agentsContent to be an object")
			} else {
				if contentMap["content"] == nil || contentMap["content"] == "" {
					t.Error("Expected content field to be populated")
				}
			}
		}
	})
}
