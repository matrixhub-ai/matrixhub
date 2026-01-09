package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/pkg/repository"
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

func (h *Handler) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	if repository.IsRepository(repoPath) {
		http.Error(w, "Repository already exists", http.StatusConflict)
		return
	}

	_, err := repository.Init(repoPath, "main")
	if err != nil {
		http.Error(w, "Failed to create repository", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
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

	err = repo.Remove()
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

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to read repository config", http.StatusInternalServerError)
		return
	}

	isMirror, _, err := repo.IsMirror()
	if err != nil {
		http.Error(w, "Failed to get mirror config", http.StatusInternalServerError)
		return
	}

	defaultBranch := repo.DefaultBranch()

	info := Repository{
		Name:          repoName,
		IsMirror:      isMirror,
		DefaultBranch: defaultBranch,
		Description:   "", // Description can be implemented later
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
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
		if repository.IsRepository(path) {
			repo, err := repository.Open(path)
			if err != nil {
				return nil
			}

			rel, _ := filepath.Rel(h.rootDir, path)
			name := strings.TrimSuffix(rel, ".git")

			// Check if this is a mirror repository
			isMirror, _, err := repo.IsMirror()
			if err != nil {
				return nil
			}

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
