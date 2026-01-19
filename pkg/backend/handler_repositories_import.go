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
	"io"
	"net/http"
	"strings"

	"github.com/git-lfs/pktline"
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

// getRemoteDefaultBranch discovers the default branch of a remote repository
// by fetching and parsing the git-upload-pack info/refs pkt-line response.
func (h *Handler) getRemoteDefaultBranch(ctx context.Context, sourceURL string) (string, error) {
	// Construct the info/refs URL for git-upload-pack
	infoRefsURL := strings.TrimSuffix(sourceURL, "/") + "/info/refs?service=git-upload-pack"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoRefsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "git/2.0")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch info/refs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return parseDefaultBranchFromPktLine(resp.Body)
}

// parseDefaultBranchFromPktLine parses the git-upload-pack info/refs pkt-line response
// to extract the default branch from the symref capability.
// The response format is:
// - Service announcement: "# service=git-upload-pack\n"
// - Flush packet (0000)
// - First ref line with capabilities: "<sha1> <ref>\0<capabilities>\n"
// - Additional ref lines: "<sha1> <ref>\n"
// - Flush packet (0000)
func parseDefaultBranchFromPktLine(r io.Reader) (string, error) {
	pl := pktline.NewPktline(r, nil)

	// Read and skip the service announcement packet
	// Format: "# service=git-upload-pack"
	_, err := pl.ReadPacketText()
	if err != nil {
		return "", fmt.Errorf("failed to read service packet: %w", err)
	}

	// Read the flush packet after service announcement
	_, pktLen, err := pl.ReadPacketTextWithLength()
	if err != nil {
		return "", fmt.Errorf("failed to read flush packet: %w", err)
	}
	if pktLen != 0 {
		return "", errors.New("expected flush packet after service announcement")
	}

	// Read the first ref packet which contains capabilities
	// Format: "<sha1> <ref>\0<capabilities>\n"
	firstRefPacket, pktLen, err := pl.ReadPacketTextWithLength()
	if err != nil {
		return "", fmt.Errorf("failed to read first ref packet: %w", err)
	}
	if pktLen == 0 {
		return "", errors.New("empty repository: no refs found")
	}

	// Parse capabilities from the first ref line
	// The capabilities are separated from the ref by a NUL byte
	nullIdx := strings.Index(firstRefPacket, "\x00")
	if nullIdx == -1 {
		return "", errors.New("no capabilities found in first ref packet")
	}

	capabilities := firstRefPacket[nullIdx+1:]

	// Look for symref=HEAD:refs/heads/<branch> in capabilities
	for _, cap := range strings.Fields(capabilities) {
		if strings.HasPrefix(cap, "symref=HEAD:") {
			ref := strings.TrimPrefix(cap, "symref=HEAD:")
			if strings.HasPrefix(ref, "refs/heads/") {
				return strings.TrimPrefix(ref, "refs/heads/"), nil
			}
		}
	}

	return "", errors.New("could not determine default branch from symref capability")
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
