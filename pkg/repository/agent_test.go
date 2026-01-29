package repository_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

func runGitCommand(t *testing.T, dir string, args ...string) {
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

func TestGetAgentsContent(t *testing.T) {
	// Create a temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "agents-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoPath := filepath.Join(tempDir, "test-repo.git")

	// Initialize a bare git repository with main branch
	runGitCommand(t, tempDir, "init", "--bare", "--initial-branch=main", repoPath)

	// Create a working directory to add files
	workDir := filepath.Join(tempDir, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	// Clone the bare repo
	runGitCommand(t, tempDir, "clone", repoPath, workDir)

	t.Run("NoAgentsFile", func(t *testing.T) {
		// Add a simple file to create a commit
		readmePath := filepath.Join(workDir, "README.md")
		if err := os.WriteFile(readmePath, []byte("# Test\n"), 0644); err != nil {
			t.Fatalf("Failed to create README: %v", err)
		}
		runGitCommand(t, workDir, "add", "README.md")
		runGitCommand(t, workDir, "commit", "-m", "Initial commit")
		runGitCommand(t, workDir, "push", "origin", "main")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Try to get AGENTS.md - should return error
		_, err = repo.GetAgentsContent("main")
		if err != repository.ErrAgentsFileNotFound {
			t.Errorf("Expected ErrAgentsFileNotFound, got: %v", err)
		}
	})

	t.Run("AgentsFileWithContent", func(t *testing.T) {
		// Create AGENTS.md with plain markdown content
		agentsContent := "# AGENTS Guidelines for This Repository\n\n" +
			"## Dev environment tips\n" +
			"- Use `pnpm dlx turbo run where <project_name>` to jump to a package\n" +
			"- Run `pnpm install --filter <project_name>` to add the package to your workspace\n\n" +
			"## Testing instructions\n" +
			"- Find the CI plan in the .github/workflows folder.\n" +
			"- Run `pnpm turbo run test --filter <project_name>` to run every check\n\n" +
			"## PR instructions\n" +
			"- Title format: [<project_name>] <Title>\n" +
			"- Always run `pnpm lint` and `pnpm test` before committing.\n"
		agentsPath := filepath.Join(workDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "AGENTS.md")
		runGitCommand(t, workDir, "commit", "-m", "Add AGENTS.md")
		runGitCommand(t, workDir, "push", "origin", "main")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get AGENTS.md content
		content, err := repo.GetAgentsContent("main")
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md content: %v", err)
		}

		// Verify content
		if !strings.Contains(content.Content, "AGENTS Guidelines") {
			t.Errorf("Expected content to contain 'AGENTS Guidelines', got: %s", content.Content)
		}
		if !strings.Contains(content.Content, "Dev environment tips") {
			t.Errorf("Expected content to contain 'Dev environment tips'")
		}
		if !strings.Contains(content.Content, "Testing instructions") {
			t.Errorf("Expected content to contain 'Testing instructions'")
		}
	})

	t.Run("AgentsFileWithDifferentBranches", func(t *testing.T) {
		// Create a new branch for this test
		runGitCommand(t, workDir, "checkout", "-b", "feature-branch")

		// Create different AGENTS.md content
		agentsContent := `# AGENTS for Feature Branch

## Feature-specific instructions
- This is a feature branch
- Different instructions apply here
`
		agentsPath := filepath.Join(workDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte(agentsContent), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "AGENTS.md")
		runGitCommand(t, workDir, "commit", "-m", "Update AGENTS.md for feature")
		runGitCommand(t, workDir, "push", "origin", "feature-branch")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get content from feature branch
		featureContent, err := repo.GetAgentsContent("feature-branch")
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md from feature branch: %v", err)
		}

		if !strings.Contains(featureContent.Content, "Feature-specific instructions") {
			t.Errorf("Expected feature branch content, got: %s", featureContent.Content)
		}

		// Get content from main branch
		mainContent, err := repo.GetAgentsContent("main")
		if err != nil {
			t.Fatalf("Failed to get AGENTS.md from main: %v", err)
		}

		if !strings.Contains(mainContent.Content, "Dev environment tips") {
			t.Errorf("Expected main branch content, got: %s", mainContent.Content)
		}
	})

	t.Run("EmptyAgentsFile", func(t *testing.T) {
		// Create a new branch for this test
		runGitCommand(t, workDir, "checkout", "-b", "empty-agents")

		// Create empty AGENTS.md
		agentsPath := filepath.Join(workDir, "AGENTS.md")
		if err := os.WriteFile(agentsPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create AGENTS.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "AGENTS.md")
		runGitCommand(t, workDir, "commit", "-m", "Add empty AGENTS.md")
		runGitCommand(t, workDir, "push", "origin", "empty-agents")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get AGENTS.md content - should work with empty file
		content, err := repo.GetAgentsContent("empty-agents")
		if err != nil {
			t.Fatalf("Should handle empty file gracefully, got error: %v", err)
		}

		if content.Content != "" {
			t.Errorf("Expected empty content, got: %s", content.Content)
		}
	})

	t.Run("HasAgentsFile", func(t *testing.T) {
		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Check main branch (has AGENTS.md)
		if !repo.HasAgentsFile("main") {
			t.Error("Expected main branch to have AGENTS.md")
		}

		// Create a branch without AGENTS.md
		runGitCommand(t, workDir, "checkout", "main")
		runGitCommand(t, workDir, "checkout", "-b", "no-agents")
		runGitCommand(t, workDir, "rm", "AGENTS.md")
		runGitCommand(t, workDir, "commit", "-m", "Remove AGENTS.md")
		runGitCommand(t, workDir, "push", "origin", "no-agents")

		// Check no-agents branch (no AGENTS.md)
		if repo.HasAgentsFile("no-agents") {
			t.Error("Expected no-agents branch to not have AGENTS.md")
		}
	})
}
