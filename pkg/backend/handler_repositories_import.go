package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/internal/utils"
	"github.com/wzshiming/gitd/pkg/queue"
	"github.com/wzshiming/gitd/pkg/repository"
)

// importRequest represents a request to import a repository from a source URL.
type importRequest struct {
	SourceURL string `json:"source_url"`
}

func (h *Handler) registryRepositoriesImport(r *mux.Router) {
	r.HandleFunc("/api/repositories/{repo:.+}.git/import", h.requireAuth(h.handleImportRepository)).Methods(http.MethodPost)
	r.HandleFunc("/api/repositories/{repo:.+}.git/import/status", h.requireAuth(h.handleImportStatus)).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/sync", h.requireAuth(h.handleSyncRepository)).Methods(http.MethodPost)
	r.HandleFunc("/api/repositories/{repo:.+}.git/mirror", h.requireAuth(h.handleMirrorInfo)).Methods(http.MethodGet)
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

	defaultBranch, err := h.getRemoteDefaultBranch(ctx, req.SourceURL)
	if err != nil {
		http.Error(w, "Failed to get default branch from source", http.StatusInternalServerError)
		return
	}

	repo, err := repository.Init(repoPath, defaultBranch)
	if err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
		return
	}

	err = repo.SetMirrorRemote(req.SourceURL)
	if err != nil {
		http.Error(w, "Failed to set mirror remote", http.StatusInternalServerError)
		return
	}

	// Add import task to queue
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	params := map[string]string{"source_url": req.SourceURL}
	task, err := h.queueStore.Add(queue.TaskTypeMirrorSync, repoName, 0, params)
	if err != nil {
		http.Error(w, "Failed to queue import task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "accepted",
		"message": "Import queued",
		"task_id": task.ID,
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
	task, err := h.queueStore.Add(queue.TaskTypeMirrorSync, repoName, 0, params)
	if err != nil {
		http.Error(w, "Failed to queue sync task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "accepted",
		"message": "Sync queued",
		"task_id": task.ID,
	})
}

// getRemoteDefaultBranch discovers the default branch of a remote repository
func (h *Handler) getRemoteDefaultBranch(ctx context.Context, sourceURL string) (string, error) {
	cmd := utils.Command(ctx, "git", "ls-remote", "--symref", sourceURL, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output to find the default branch
	// Format: ref: refs/heads/main	HEAD
	lines := string(output)
	for _, line := range splitLines(lines) {
		if len(line) > 5 && line[:5] == "ref: " {
			// Extract the ref part after "ref: "
			remaining := line[5:]
			// Find the end of the ref (before whitespace/tab)
			refEnd := len(remaining)
			for i := 0; i < len(remaining); i++ {
				if remaining[i] == ' ' || remaining[i] == '\t' {
					refEnd = i
					break
				}
			}
			ref := remaining[:refEnd]
			// Extract branch name from refs/heads/xxx
			if len(ref) > 11 && ref[:11] == "refs/heads/" {
				return ref[11:], nil
			}
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
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
	response := map[string]interface{}{
		"status":   task.Status,
		"progress": task.Progress,
		"step":     task.ProgressMsg,
		"task_id":  task.ID,
	}
	if task.Error != "" {
		response["error"] = task.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	response := map[string]interface{}{
		"is_mirror":  isMirror,
		"source_url": sourceURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
