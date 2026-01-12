package backend_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/wzshiming/gitd/pkg/backend"
	"github.com/wzshiming/gitd/pkg/queue"
)

// TestQueueAPI tests the queue management API endpoints.
func TestQueueAPI(t *testing.T) {
	repoDir, err := os.MkdirTemp("", "gitd-test-queue")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer os.RemoveAll(repoDir)

	handler := backend.NewHandler(backend.WithRootDir(repoDir))
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("ListEmptyQueue", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/queue")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var tasks []queue.Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(tasks) != 0 {
			t.Errorf("Expected empty queue, got %d tasks", len(tasks))
		}
	})

	var taskID int64

	t.Run("CreateRepositoryAndImport", func(t *testing.T) {
		// First create a repo to import into
		repoName := "queue-test-repo.git"

		// Create repository first
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", resp.StatusCode)
		}
	})

	t.Run("QueueMirrorSyncTask", func(t *testing.T) {
		// Import a repository (this should create a task in the queue)
		repoName := "mirror-queue-test.git"
		body := strings.NewReader(`{"source_url":"https://github.com/octocat/Hello-World.git"}`)
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName+"/import", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send import request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d", resp.StatusCode)
		}

		// Parse response to get task ID
		var importResp struct {
			TaskID int64 `json:"task_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&importResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if importResp.TaskID == 0 {
			t.Error("Expected non-zero task ID")
		}
		taskID = importResp.TaskID
	})

	t.Run("ListQueueWithTasks", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/queue")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var tasks []queue.Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(tasks) == 0 {
			t.Error("Expected at least one task in queue")
		}
	})

	t.Run("GetTaskByID", func(t *testing.T) {
		if taskID == 0 {
			t.Skip("No task ID from previous test")
		}

		resp, err := http.Get(server.URL + "/api/queue/" + fmt.Sprintf("%d", taskID))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		// Task might be running/completed, just check we get a response
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})

	t.Run("UpdateTaskPriority", func(t *testing.T) {
		if taskID == 0 {
			t.Skip("No task ID from previous test")
		}

		// Add a new pending task first
		repoName := "priority-test.git"
		body := strings.NewReader(`{"source_url":"https://github.com/octocat/Hello-World.git"}`)
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName+"/import", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send import request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d", resp.StatusCode)
		}

		var importResp struct {
			TaskID int64 `json:"task_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&importResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Update priority
		priorityBody := strings.NewReader(`{"priority":100}`)
		priorityReq, _ := http.NewRequest(http.MethodPut,
			server.URL+"/api/queue/"+fmt.Sprintf("%d", importResp.TaskID)+"/priority",
			priorityBody)
		priorityReq.Header.Set("Content-Type", "application/json")
		priorityResp, err := http.DefaultClient.Do(priorityReq)
		if err != nil {
			t.Fatalf("Failed to update priority: %v", err)
		}
		defer priorityResp.Body.Close()

		// Priority update should work for pending tasks
		if priorityResp.StatusCode != http.StatusOK && priorityResp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", priorityResp.StatusCode)
		}
	})

	t.Run("CancelTask", func(t *testing.T) {
		// Add a new task
		repoName := "cancel-test.git"
		body := strings.NewReader(`{"source_url":"https://github.com/octocat/Hello-World.git"}`)
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/repositories/"+repoName+"/import", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send import request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d", resp.StatusCode)
		}

		var importResp struct {
			TaskID int64 `json:"task_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&importResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Cancel the task
		cancelReq, _ := http.NewRequest(http.MethodDelete,
			server.URL+"/api/queue/"+fmt.Sprintf("%d", importResp.TaskID), nil)
		cancelResp, err := http.DefaultClient.Do(cancelReq)
		if err != nil {
			t.Fatalf("Failed to cancel task: %v", err)
		}
		defer cancelResp.Body.Close()

		if cancelResp.StatusCode != http.StatusNoContent && cancelResp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 204 or 404, got %d", cancelResp.StatusCode)
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/queue?status=pending")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var tasks []queue.Task
		if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		for _, task := range tasks {
			if task.Status != queue.TaskStatusPending {
				t.Errorf("Expected all tasks to have status pending, got %s", task.Status)
			}
		}
	})
}
