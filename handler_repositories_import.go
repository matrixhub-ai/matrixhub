package gitd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/gorilla/mux"
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

	// Check if the repository directory already exists
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		http.Error(w, "Repository already exists", http.StatusConflict)
		return
	}

	ctx := context.Background()

	defaultBranch, err := h.getRemoteDefaultBranch(ctx, req.SourceURL)
	if err != nil {
		http.Error(w, "Failed to get default branch from source", http.StatusInternalServerError)
		return
	}

	err = h.initMirrorRepo(repoPath, req.SourceURL, defaultBranch)
	if err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
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
		err := h.doImport(context.Background(), repoPath, defaultBranch, repoName)
		importMutex.Lock()
		defer importMutex.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Import failed for %s: %v\n", repoName, err)
			importStatuses[repoName] = &importStatus{
				Status: "failed",
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

// doImport performs the actual import operation in steps.
func (h *Handler) doImport(ctx context.Context, repoPath string, branch string, repoName string) error {
	importMutex.Lock()
	importStatuses[repoName] = &importStatus{Status: "in_progress", Step: "fetching initial branch"}
	importMutex.Unlock()

	err := h.fetchWithOptions(ctx, repoPath, branch, false, 1)
	if err != nil {
		return fmt.Errorf("failed to import history: %w", err)
	}

	err = h.fetchWithOptions(ctx, repoPath, branch, false, 100)
	if err != nil {
		return fmt.Errorf("failed to import history: %w", err)
	}

	err = h.fetchWithOptions(ctx, repoPath, branch, false, 0)
	if err != nil {
		return fmt.Errorf("failed to import history: %w", err)
	}

	importMutex.Lock()
	importStatuses[repoName] = &importStatus{Status: "in_progress", Step: "fetching full history"}
	importMutex.Unlock()

	err = h.fetchFull(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("failed to import full history: %w", err)
	}

	return nil
}

// fetchWithOptions fetches from remote with specified options
func (h *Handler) fetchWithOptions(ctx context.Context, repoPath, branch string, tags bool, depth int) error {
	args := []string{"fetch"}

	if depth > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", depth))
	}

	if branch != "" {
		args = append(args, "origin", fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, branch))
	}

	if tags {
		args = append(args, "--tags")
	}

	cmd := command(ctx, "git", args...)
	cmd.Dir = repoPath
	return cmd.Run()
}

// fetchFull fetches the complete history from the source.
func (h *Handler) fetchFull(ctx context.Context, repoPath string) error {
	// Check if repository is already a complete (non-shallow) clone
	cmd := command(ctx, "git", "rev-parse", "--is-shallow-repository")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil {
		isShallow := strings.TrimSpace(string(output))
		if isShallow == "false" {
			// Repository is already complete, just fetch updates
			cmd = command(ctx, "git", "fetch", "--prune", "origin", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*")
			cmd.Dir = repoPath
			return cmd.Run()
		}
	}

	// Fetch full history with unshallow
	cmd = command(ctx, "git", "fetch", "--unshallow", "--prune", "origin", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*")
	cmd.Dir = repoPath
	return cmd.Run()
}

func (h *Handler) isMirrorRepository(repo *git.Repository) (bool, string, error) {
	config, err := repo.Config()
	if err != nil {
		return false, "", err
	}
	isMirror := false
	sourceURL := ""
	if config != nil {
		if remote, ok := config.Remotes["origin"]; ok {
			if remote.Mirror {
				isMirror = true
			}
			if len(remote.URLs) > 0 {
				sourceURL = remote.URLs[0]
			}
		}
	}
	return isMirror, sourceURL, nil
}

func (h *Handler) initMirrorRepo(repoPath string, sourceURL string, defaultBranch string) error {
	repo, err := git.PlainInit(repoPath, true)
	if err != nil {
		return err
	}

	cfg, err := repo.Config()
	if err != nil {
		return err
	}
	cfg.Init.DefaultBranch = defaultBranch
	cfg.Remotes = map[string]*config.RemoteConfig{
		"origin": {
			Name:   "origin",
			URLs:   []string{sourceURL},
			Mirror: true,
			Fetch: []config.RefSpec{
				"+refs/heads/*:refs/heads/*",
				"+refs/tags/*:refs/tags/*",
			},
		},
	}

	err = repo.Storer.SetConfig(cfg)
	if err != nil {
		return err
	}
	return nil
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

	repository, err := h.openRepository(repoPath)
	if err != nil {
		http.Error(w, "Failed to read repository config", http.StatusInternalServerError)
		return
	}

	isMirror, sourceURL, err := h.isMirrorRepository(repository)
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
		err := h.doImport(context.Background(), repoPath, "", repoName)
		importMutex.Lock()
		defer importMutex.Unlock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Import failed for %s: %v\n", repoName, err)
			importStatuses[repoName] = &importStatus{
				Status: "failed",
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
	cmd := command(ctx, "git", "ls-remote", "--symref", sourceURL, "HEAD")
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

	repository, err := h.openRepository(repoPath)
	if err != nil {
		http.Error(w, "Failed to read repository config", http.StatusInternalServerError)
		return
	}

	isMirror, sourceURL, err := h.isMirrorRepository(repository)
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
