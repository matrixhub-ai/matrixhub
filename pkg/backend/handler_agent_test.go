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

func TestAgentMetadataAPI(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "matrixhub-agent-api-test")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	handler := handlers.LoggingHandler(os.Stderr, backend.NewHandler(backend.WithRootDir(repoDir)))
	server := httptest.NewServer(handler)
	defer server.Close()

	// Test repository without agent.md
	t.Run("AgentMetadataNotFound", func(t *testing.T) {
		repoName := "no-agent.git"

		// Create and populate repository without agent.md
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-no-agent")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Add a README but no agent.md
		readmePath := filepath.Join(workDir, "README.md")
		if err := os.WriteFile(readmePath, []byte("# Test\n"), 0644); err != nil {
			t.Fatalf("Failed to create README: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "README.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Initial commit")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Request agent metadata for repo without agent.md
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/no-agent/agent", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get agent metadata: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for missing agent.md, got %d", resp.StatusCode)
		}
	})

	// Test repository with agent.md
	t.Run("AgentMetadataFound", func(t *testing.T) {
		repoName := "with-agent.git"

		// Create and populate repository with agent.md
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-agent")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Create agent.md with full metadata
		agentContent := `---
name: "Test Agent"
version: "1.0.0"
type: "assistant"
tags: ["coding", "automation", "testing"]
description: "A comprehensive test agent"
---

# Test Agent

This is a test agent for API validation.

## Capabilities

- Code generation
- Automated testing
- Documentation
`
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "agent.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Add agent.md")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Request agent metadata via API
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/with-agent/agent", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get agent metadata: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify response structure
		var metadata repository.AgentMetadata
		if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Verify metadata fields
		if metadata.Name != "Test Agent" {
			t.Errorf("Expected name 'Test Agent', got '%s'", metadata.Name)
		}
		if metadata.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
		}
		if metadata.Type != "assistant" {
			t.Errorf("Expected type 'assistant', got '%s'", metadata.Type)
		}
		if len(metadata.Tags) != 3 {
			t.Errorf("Expected 3 tags, got %d", len(metadata.Tags))
		}
		if metadata.Description != "A comprehensive test agent" {
			t.Errorf("Expected specific description, got '%s'", metadata.Description)
		}
		if metadata.RawContent == "" {
			t.Error("Expected raw content to be populated")
		}
	})

	// Test agent metadata with revision
	t.Run("AgentMetadataWithRevision", func(t *testing.T) {
		repoName := "revision-agent.git"

		// Create and populate repository
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-revision")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Create initial agent.md
		agentContent := `---
name: "V1 Agent"
version: "1.0.0"
---

Version 1 content
`
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "agent.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Add v1 agent.md")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Update agent.md in a new branch
		runGitCommandAgent(t, workDir, "checkout", "-b", "v2")
		agentContent = `---
name: "V2 Agent"
version: "2.0.0"
---

Version 2 content with new features
`
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to update agent.md: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", "agent.md")
		runGitCommandAgent(t, workDir, "commit", "-m", "Update to v2")
		runGitCommandAgent(t, workDir, "push", "origin", "v2")

		// Request metadata from main branch
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/revision-agent/agent/revision/main", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get agent metadata for main: %v", err)
		}
		defer resp.Body.Close()

		var mainMetadata repository.AgentMetadata
		if err := json.NewDecoder(resp.Body).Decode(&mainMetadata); err != nil {
			t.Fatalf("Failed to decode main metadata: %v", err)
		}

		if mainMetadata.Version != "1.0.0" {
			t.Errorf("Expected v1.0.0 on main branch, got %s", mainMetadata.Version)
		}

		// Request metadata from v2 branch
		req, _ = http.NewRequest(http.MethodGet, server.URL+"/api/models/revision-agent/agent/revision/v2", nil)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get agent metadata for v2: %v", err)
		}
		defer resp.Body.Close()

		var v2Metadata repository.AgentMetadata
		if err := json.NewDecoder(resp.Body).Decode(&v2Metadata); err != nil {
			t.Fatalf("Failed to decode v2 metadata: %v", err)
		}

		if v2Metadata.Version != "2.0.0" {
			t.Errorf("Expected v2.0.0 on v2 branch, got %s", v2Metadata.Version)
		}
	})

	// Test model info includes agent metadata
	t.Run("ModelInfoIncludesAgentMetadata", func(t *testing.T) {
		repoName := "model-with-agent.git"

		// Create and populate repository
		bareRepoPath := filepath.Join(repoDir, "repositories", repoName)
		workDir := filepath.Join(repoDir, "work-model")

		runGitCommandAgent(t, repoDir, "init", "--bare", "--initial-branch=main", bareRepoPath)
		runGitCommandAgent(t, repoDir, "clone", bareRepoPath, workDir)

		// Create agent.md
		agentContent := `---
name: "Model Agent"
version: "1.0.0"
type: "model"
---

Model agent for testing
`
		agentPath := filepath.Join(workDir, "agent.md")
		if err := os.WriteFile(agentPath, []byte(agentContent), 0644); err != nil {
			t.Fatalf("Failed to create agent.md: %v", err)
		}

		// Also add a model file
		modelPath := filepath.Join(workDir, "model.bin")
		if err := os.WriteFile(modelPath, []byte("fake model data"), 0644); err != nil {
			t.Fatalf("Failed to create model file: %v", err)
		}

		runGitCommandAgent(t, workDir, "add", ".")
		runGitCommandAgent(t, workDir, "commit", "-m", "Add agent and model")
		runGitCommandAgent(t, workDir, "push", "origin", "main")

		// Request model info via HF API
		req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/models/model-with-agent", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to get model info: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Verify response includes agent metadata
		var modelInfo map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check that agentMetadata field exists
		agentMeta, ok := modelInfo["agentMetadata"]
		if !ok {
			t.Error("Expected agentMetadata field in model info")
		} else {
			// Verify it's populated
			metaMap, ok := agentMeta.(map[string]interface{})
			if !ok {
				t.Error("Expected agentMetadata to be an object")
			} else {
				if metaMap["name"] != "Model Agent" {
					t.Errorf("Expected name 'Model Agent', got %v", metaMap["name"])
				}
				// RawContent should be empty in API response
				if rawContent, exists := metaMap["rawContent"]; exists && rawContent != "" {
					t.Error("Expected rawContent to be empty in API response")
				}
			}
		}
	})
}
