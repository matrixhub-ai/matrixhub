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
	"fmt"
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
	repoName := vars["repo"]

	if h.queueStore == nil {
		h.JSON(w, fmt.Errorf("queue not initialized"), http.StatusServiceUnavailable)
		return
	}

	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSON(w, fmt.Errorf("invalid request body"), http.StatusBadRequest)
		return
	}

	if req.SourceURL == "" {
		h.JSON(w, fmt.Errorf("source_url is required"), http.StatusBadRequest)
		return
	}

	repoPath := h.resolveRepoPath(repoName)

	if repository.IsRepository(repoPath) {
		h.JSON(w, fmt.Errorf("repository already exists"), http.StatusConflict)
		return
	}

	ctx := context.Background()

	_, err := repository.InitMrror(ctx, repoPath, req.SourceURL)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to create repository"), http.StatusInternalServerError)
		return
	}

	params := map[string]string{"source_url": req.SourceURL}
	taskID, err := h.queueStore.Add(queue.TaskTypeRepositorySync, repoName, 0, params)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to queue import task"), http.StatusInternalServerError)
		return
	}

	h.JSON(w, map[string]any{
		"status":  "accepted",
		"message": "Import queued",
		"task_id": taskID,
	}, http.StatusAccepted)
}

// handleSyncRepository synchronizes a mirror repository with its source.
func (h *Handler) handleSyncRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	if h.queueStore == nil {
		h.JSON(w, fmt.Errorf("queue not initialized"), http.StatusServiceUnavailable)
		return
	}

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		h.JSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			h.JSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
			return
		}
		h.JSON(w, fmt.Errorf("failed to open repository"), http.StatusInternalServerError)
		return
	}

	isMirror, sourceURL, err := repo.IsMirror()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get mirror config"), http.StatusInternalServerError)
		return
	}

	if !isMirror || sourceURL == "" {
		h.JSON(w, fmt.Errorf("repository is not a mirror"), http.StatusBadRequest)
		return
	}

	params := map[string]string{"source_url": sourceURL}
	taskID, err := h.queueStore.Add(queue.TaskTypeRepositorySync, repoName, 0, params)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to queue sync task"), http.StatusInternalServerError)
		return
	}

	h.JSON(w, map[string]any{
		"status":  "accepted",
		"message": "Sync queued",
		"task_id": taskID,
	}, http.StatusAccepted)
}

// handleImportStatus returns the current status of an import operation.
func (h *Handler) handleImportStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	if h.queueStore == nil {
		h.JSON(w, fmt.Errorf("queue not initialized"), http.StatusServiceUnavailable)
		return
	}

	// Get tasks for this repository
	tasks, err := h.queueStore.ListByRepository(repoName)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get import status"), http.StatusInternalServerError)
		return
	}

	if len(tasks) == 0 {
		h.JSON(w, fmt.Errorf("import status not found"), http.StatusNotFound)
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

	h.JSON(w, response, http.StatusOK)
}

// handleMirrorInfo returns information about a mirror repository.
func (h *Handler) handleMirrorInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		h.JSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			h.JSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
			return
		}
		h.JSON(w, fmt.Errorf("failed to read repository config"), http.StatusInternalServerError)
		return
	}

	isMirror, sourceURL, err := repo.IsMirror()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get mirror config"), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"is_mirror":  isMirror,
		"source_url": sourceURL,
	}

	h.JSON(w, response, http.StatusOK)
}
