package gitd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

func (h *Handler) registryRepositories(r *mux.Router) {
	r.HandleFunc("/api/repositories", h.requireAuth(h.handleListRepositories)).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git", h.requireAuth(h.handleGetRepository)).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git", h.requireAuth(h.handleCreateRepository)).Methods(http.MethodPost)
	r.HandleFunc("/api/repositories/{repo:.+}.git", h.requireAuth(h.handleDeleteRepository)).Methods(http.MethodDelete)
}

// validateRepoPath validates and constructs a repository path, ensuring it's within the root directory.
func (h *Handler) validateRepoPath(urlPath string) (string, error) {
	// Clean the path
	urlPath = strings.TrimPrefix(urlPath, "/")
	if urlPath == "" {
		return "", fmt.Errorf("empty path")
	}

	// Construct the full path
	fullPath := filepath.Join(h.rootDir, urlPath)

	// Clean and verify the path is within RepoDir using filepath.Rel
	fullPath = filepath.Clean(fullPath)
	absRepoDir, err := filepath.Abs(h.rootDir)
	if err != nil {
		return "", err
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	// Use filepath.Rel to safely check if absFullPath is within absRepoDir
	relPath, err := filepath.Rel(absRepoDir, absFullPath)
	if err != nil {
		return "", err
	}
	// Reject if the relative path starts with ".." (meaning it's outside RepoDir)
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path outside repository directory")
	}

	return fullPath, nil
}

// createBareRepo creates a bare git repository at the given path.
func (h *Handler) createBareRepo(ctx context.Context, repoPath string) error {
	if _, err := os.Stat(filepath.Join(repoPath, "HEAD")); err == nil {
		return fmt.Errorf("repository already exists")
	}

	// Create all parent directories
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return err
	}

	// Run git init --bare in the repository directory itself
	cmd := command(ctx, "git", "init", "--bare")
	cmd.Dir = repoPath
	return cmd.Run()
}

func (h *Handler) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath != "" {
		http.Error(w, "Repository already exists", http.StatusConflict)
		return
	}

	repoPath = filepath.Join(h.rootDir, repoName)

	err := h.createBareRepo(r.Context(), repoPath)
	if err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	err := os.RemoveAll(repoPath)
	if err != nil {
		http.Error(w, "Failed to delete repository", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type Repository struct {
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
	IsMirror      bool   `json:"is_mirror"`
}

func (h *Handler) handleGetRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}
	base, dir := filepath.Split(repoPath)
	// Check if this is a mirror repository
	_, isMirror, _ := h.getMirrorInfo(repoPath)

	cmd := command(r.Context(),
		"git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = filepath.Join(base, dir)

	output, err := cmd.Output()
	defaultBranch := "main"
	if err == nil {
		defaultBranch = strings.TrimSpace(string(output))
	}

	repo := Repository{
		Name:          repoName,
		IsMirror:      isMirror,
		DefaultBranch: defaultBranch,
		Description:   "", // Description can be implemented later
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repo)
}

type RepositoryItem struct {
	Name     string `json:"name"`
	IsMirror bool   `json:"is_mirror"`
}

// handleListRepositories handles requests to list all repositories
func (h *Handler) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	var repos []RepositoryItem

	// Walk through rootDir to find all git repositories at any depth
	err := filepath.Walk(h.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if isGitRepository(path) {
			rel, _ := filepath.Rel(h.rootDir, path)
			name := strings.TrimSuffix(rel, ".git")

			// Check if this is a mirror repository
			_, isMirror, _ := h.getMirrorInfo(path)

			repos = append(repos, RepositoryItem{
				Name:     name,
				IsMirror: isMirror,
			})
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		http.Error(w, "Failed to list repos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}
