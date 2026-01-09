package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/internal/utils"
	"github.com/wzshiming/gitd/pkg/lfs"
	"github.com/wzshiming/gitd/pkg/repository"
)

// importRequest represents a request to import a repository from a source URL.
type importRequest struct {
	SourceURL string `json:"source_url"`
}

// importStatus tracks the status of an import operation.
type importStatus struct {
	Status string `json:"status"` // "in_progress", "completed", "failed"
	Step   string `json:"step"`
	Error  string `json:"error,omitempty"`
}

var (
	importStatuses = make(map[string]*importStatus)
	importMutex    sync.RWMutex
)

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

	// Initialize import status
	importMutex.Lock()
	importStatuses[repoName] = &importStatus{
		Status: "in_progress",
		Step:   "starting",
	}
	importMutex.Unlock()

	// Run import in background
	go func() {
		err := repo.SyncMirror(context.Background())
		if err != nil {
			importMutex.Lock()
			fmt.Fprintf(os.Stderr, "Import failed for %s: %v\n", repoName, err)
			importStatuses[repoName] = &importStatus{
				Status: "failed",
				Error:  err.Error(),
			}
			importMutex.Unlock()
			return
		}

		// Sync LFS objects after git fetch
		importMutex.Lock()
		importStatuses[repoName].Step = "syncing_lfs"
		importMutex.Unlock()

		err = h.syncLFSObjects(context.Background(), repo, req.SourceURL)
		importMutex.Lock()
		defer importMutex.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "LFS sync failed for %s: %v\n", repoName, err)
			// LFS sync failure is not fatal, mark as completed with warning
			importStatuses[repoName] = &importStatus{
				Status: "completed",
				Step:   "lfs_sync_failed",
				Error:  err.Error(),
			}
		} else {
			importStatuses[repoName] = &importStatus{
				Status: "completed",
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "Import started",
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

	// Initialize import status
	importMutex.Lock()
	importStatuses[repoName] = &importStatus{
		Status: "in_progress",
		Step:   "syncing",
	}
	importMutex.Unlock()

	// Run sync in background
	go func() {
		err := repo.SyncMirror(context.Background())
		if err != nil {
			importMutex.Lock()
			fmt.Fprintf(os.Stderr, "Import failed for %s: %v\n", repoName, err)
			importStatuses[repoName] = &importStatus{
				Status: "failed",
				Error:  err.Error(),
			}
			importMutex.Unlock()
			return
		}

		// Sync LFS objects after git fetch
		importMutex.Lock()
		importStatuses[repoName].Step = "syncing_lfs"
		importMutex.Unlock()

		err = h.syncLFSObjects(context.Background(), repo, sourceURL)
		importMutex.Lock()
		defer importMutex.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "LFS sync failed for %s: %v\n", repoName, err)
			// LFS sync failure is not fatal, mark as completed with warning
			importStatuses[repoName] = &importStatus{
				Status: "completed",
				Step:   "lfs_sync_failed",
				Error:  err.Error(),
			}
		} else {
			importStatuses[repoName] = &importStatus{
				Status: "completed",
			}
		}
	}()

	w.WriteHeader(http.StatusAccepted)
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

	importMutex.RLock()
	status, exists := importStatuses[repoName]
	importMutex.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
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

// syncLFSObjects scans the repository for LFS pointers and fetches missing objects
func (h *Handler) syncLFSObjects(ctx context.Context, repo *repository.Repository, sourceURL string) error {
	// Scan repository for LFS pointers
	pointers, err := repo.ScanLFSPointers()
	if err != nil {
		return fmt.Errorf("failed to scan LFS pointers: %w", err)
	}

	if len(pointers) == 0 {
		return nil
	}

	// Convert to LFS objects
	objects := make([]lfs.LFSObject, len(pointers))
	for i, ptr := range pointers {
		objects[i] = lfs.LFSObject{
			Oid:  ptr.Oid,
			Size: ptr.Size,
		}
	}

	// Get LFS endpoint from source URL
	lfsEndpoint := lfs.GetLFSEndpoint(sourceURL)

	// Create remote client and fetch missing objects
	remoteClient := lfs.NewRemoteClient()
	return remoteClient.FetchAndStore(ctx, lfsEndpoint, objects, h.contentStore)
}
