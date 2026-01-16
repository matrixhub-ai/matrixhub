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

func TestGetAgentMetadata(t *testing.T) {
	// Create a temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "agent-test-")
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

	t.Run("NoAgentFile", func(t *testing.T) {
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

		// Try to get agent metadata - should return error
		_, err = repo.GetAgentMetadata("main")
		if err != repository.ErrAgentFileNotFound {
			t.Errorf("Expected ErrAgentFileNotFound, got: %v", err)
		}
	})

	t.Run("AgentFileWithFrontmatter", func(t *testing.T) {
		// Create agent.md with YAML frontmatter
		agentContent := `---
name: "Test Agent"
version: "1.0.0"
type: "assistant"
tags: ["coding", "automation"]
description: "A test agent for unit testing"
---

# Test Agent

This is a comprehensive test agent for validating agent.md parsing.

## Features

- Feature 1
- Feature 2
`
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "agent.md")
		runGitCommand(t, workDir, "commit", "-m", "Add agent.md")
		runGitCommand(t, workDir, "push", "origin", "main")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get agent metadata
		metadata, err := repo.GetAgentMetadata("main")
		if err != nil {
			t.Fatalf("Failed to get agent metadata: %v", err)
		}

		// Verify metadata
		if metadata.Name != "Test Agent" {
			t.Errorf("Expected name 'Test Agent', got '%s'", metadata.Name)
		}
		if metadata.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
		}
		if metadata.Type != "assistant" {
			t.Errorf("Expected type 'assistant', got '%s'", metadata.Type)
		}
		if len(metadata.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(metadata.Tags))
		} else {
			if metadata.Tags[0] != "coding" || metadata.Tags[1] != "automation" {
				t.Errorf("Expected tags [coding, automation], got %v", metadata.Tags)
			}
		}
		if metadata.Description != "A test agent for unit testing" {
			t.Errorf("Expected description from frontmatter, got '%s'", metadata.Description)
		}
		if !strings.Contains(metadata.RawContent, "Test Agent") {
			t.Errorf("RawContent should contain the full file content")
		}
	})

	t.Run("AgentFileWithoutFrontmatter", func(t *testing.T) {
		// Create a new branch for this test
		runGitCommand(t, workDir, "checkout", "-b", "no-frontmatter")

		// Create agent.md without YAML frontmatter
		agentContent := `# Simple Agent

This is a simple agent without YAML frontmatter.
It should still be parsed successfully.

The description should be extracted from the content.
`
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "agent.md")
		runGitCommand(t, workDir, "commit", "-m", "Add simple agent.md")
		runGitCommand(t, workDir, "push", "origin", "no-frontmatter")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get agent metadata
		metadata, err := repo.GetAgentMetadata("no-frontmatter")
		if err != nil {
			t.Fatalf("Failed to get agent metadata: %v", err)
		}

		// Verify metadata - should have description extracted from content
		if metadata.Name != "" {
			t.Errorf("Expected empty name, got '%s'", metadata.Name)
		}
		if metadata.Description == "" {
			t.Errorf("Expected non-empty description extracted from content")
		}
		if !strings.Contains(metadata.Description, "simple agent") {
			t.Errorf("Description should contain 'simple agent', got: '%s'", metadata.Description)
		}
	})

	t.Run("AgentFileWithInvalidYAML", func(t *testing.T) {
		// Create a new branch for this test
		runGitCommand(t, workDir, "checkout", "-b", "invalid-yaml")

		// Create agent.md with invalid YAML frontmatter
		agentContent := `---
name: "Test Agent
this is invalid yaml: [unclosed
---

# Content after invalid YAML

This should still work, just without the frontmatter data.
`
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "agent.md")
		runGitCommand(t, workDir, "commit", "-m", "Add agent.md with invalid YAML")
		runGitCommand(t, workDir, "push", "origin", "invalid-yaml")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get agent metadata - should not fail even with invalid YAML
		metadata, err := repo.GetAgentMetadata("invalid-yaml")
		if err != nil {
			t.Fatalf("Should handle invalid YAML gracefully, got error: %v", err)
		}

		// Should have some content even if YAML parsing failed
		if metadata == nil {
			t.Error("Expected metadata to be non-nil")
		}
	})

	t.Run("EmptyAgentFile", func(t *testing.T) {
		// Create a new branch for this test
		runGitCommand(t, workDir, "checkout", "-b", "empty-agent")

		// Create empty agent.md
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}
		runGitCommand(t, workDir, "add", "agent.md")
		runGitCommand(t, workDir, "commit", "-m", "Add empty agent.md")
		runGitCommand(t, workDir, "push", "origin", "empty-agent")

		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Get agent metadata - should work with empty file
		metadata, err := repo.GetAgentMetadata("empty-agent")
		if err != nil {
			t.Fatalf("Should handle empty file gracefully, got error: %v", err)
		}

		if metadata == nil {
			t.Error("Expected metadata to be non-nil")
		}
	})

	t.Run("HasAgentMetadata", func(t *testing.T) {
		// Open repository
		repo, err := repository.Open(repoPath)
		if err != nil {
			t.Fatalf("Failed to open repository: %v", err)
		}

		// Check main branch (has agent.md)
		if !repo.HasAgentMetadata("main") {
			t.Error("Expected main branch to have agent metadata")
		}

		// Create a branch without agent.md
		runGitCommand(t, workDir, "checkout", "main")
		runGitCommand(t, workDir, "checkout", "-b", "no-agent")
		runGitCommand(t, workDir, "rm", "agent.md")
		runGitCommand(t, workDir, "commit", "-m", "Remove agent.md")
		runGitCommand(t, workDir, "push", "origin", "no-agent")

		// Check no-agent branch (no agent.md)
		if repo.HasAgentMetadata("no-agent") {
			t.Error("Expected no-agent branch to not have agent metadata")
		}
	})
}
