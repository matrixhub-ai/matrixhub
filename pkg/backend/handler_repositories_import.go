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

package backend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/pkg/queue"
	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

// importRequest represents a request to import a repository from a source URL.
type importRequest struct {
	SourceURL string `json:"source_url"`
}

func (h *Handler) registryRepositoriesImport(r *mux.Router) {
	r.HandleFunc("/api/repositories/{repo:.+}.git/import", h.handleImportRepository).Methods(http.MethodPost)
	r.HandleFunc("/api/repositories/{repo:.+}.git/import/status", h.handleImportStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/sync", h.handleSyncRepository).Methods(http.MethodPost)
	r.HandleFunc("/api/repositories/{repo:.+}.git/mirror", h.handleMirrorInfo).Methods(http.MethodGet)
}

// handleImportRepository handles the import of a repository from a source URL.
// The import process follows these steps for fast imports and intermittent transfers:
func (h *Handler) handleImportRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceURL == "" {
		http.Error(w, "source_url is required", http.StatusBadRequest)
		return
	}

	// Validate and construct the repository path using the same logic as resolveRepoPath
	repoPath, err := h.validateRepoPath(repoName)
	if err != nil {
		http.Error(w, "Invalid repository path", http.StatusBadRequest)
		return
	}

	if repository.IsRepository(repoPath) {
		http.Error(w, "Repository already exists", http.StatusConflict)
		return
	}

	ctx := context.Background()

	_, err = repository.InitMrror(ctx, repoPath, req.SourceURL)
	if err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
		return
	}

	// Add import task to queue
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	params := map[string]string{"source_url": req.SourceURL}
	taskID, err := h.queueStore.Add(queue.TaskTypeRepositorySync, repoName, 0, params)
	if err != nil {
		http.Error(w, "Failed to queue import task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "accepted",
		"message": "Import queued",
		"task_id": taskID,
	})
}

// handleSyncRepository synchronizes a mirror repository with its source.
func (h *Handler) handleSyncRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to open repository", http.StatusInternalServerError)
		return
	}

	isMirror, sourceURL, err := repo.IsMirror()
	if err != nil {
		http.Error(w, "Failed to get mirror config", http.StatusInternalServerError)
		return
	}

	if !isMirror || sourceURL == "" {
		http.Error(w, "Repository is not a mirror", http.StatusBadRequest)
		return
	}

	// Add sync task to queue
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	params := map[string]string{"source_url": sourceURL}
	taskID, err := h.queueStore.Add(queue.TaskTypeRepositorySync, repoName, 0, params)
	if err != nil {
		http.Error(w, "Failed to queue sync task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "accepted",
		"message": "Sync queued",
		"task_id": taskID,
	})
}

// handleImportStatus returns the current status of an import operation.
func (h *Handler) handleImportStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get tasks for this repository
	tasks, err := h.queueStore.ListByRepository(repoName)
	if err != nil {
		http.Error(w, "Failed to get import status", http.StatusInternalServerError)
		return
	}

	if len(tasks) == 0 {
		http.NotFound(w, r)
		return
	}

	// Return the most recent task status
	task := tasks[0]
	response := map[string]any{
		"status":   task.Status,
		"progress": task.Progress,
		"step":     task.ProgressMsg,
		"task_id":  task.ID,
	}
	if task.Error != "" {
		response["error"] = task.Error
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleMirrorInfo returns information about a mirror repository.
func (h *Handler) handleMirrorInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to read repository config", http.StatusInternalServerError)
		return
	}

	isMirror, sourceURL, err := repo.IsMirror()
	if err != nil {
		http.Error(w, "Failed to get mirror config", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"is_mirror":  isMirror,
		"source_url": sourceURL,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
