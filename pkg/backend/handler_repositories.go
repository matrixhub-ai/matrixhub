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
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

func (h *Handler) registryRepositories(r *mux.Router) {
	r.HandleFunc("/api/repositories", h.handleListRepositories).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git", h.handleGetRepository).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git", h.handleCreateRepository).Methods(http.MethodPost)
	r.HandleFunc("/api/repositories/{repo:.+}.git", h.handleDeleteRepository).Methods(http.MethodDelete)
}

func (h *Handler) handleCreateRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		h.JSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
		return
	}

	if repository.IsRepository(repoPath) {
		h.JSON(w, fmt.Errorf("repository %q already exists", repoName), http.StatusConflict)
		return
	}

	_, err := repository.Init(repoPath, "main")
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to create repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}
	h.JSON(w, nil, http.StatusCreated)
}

func (h *Handler) handleDeleteRepository(w http.ResponseWriter, r *http.Request) {
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
		h.JSON(w, fmt.Errorf("failed to open repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	err = repo.Remove()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to delete repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}
	h.JSON(w, nil, http.StatusNoContent)
}

type Repository struct {
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
	IsMirror      bool   `json:"is_mirror"`
}

func (h *Handler) handleGetRepository(w http.ResponseWriter, r *http.Request) {
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
		h.JSON(w, fmt.Errorf("failed to read repository config for %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	isMirror, _, err := repo.IsMirror()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get mirror config for repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	defaultBranch := repo.DefaultBranch()

	info := Repository{
		Name:          repoName,
		IsMirror:      isMirror,
		DefaultBranch: defaultBranch,
		Description:   "", // Description can be implemented later
	}

	h.JSON(w, info, http.StatusOK)
}

type RepositoryItem struct {
	Name     string `json:"name"`
	IsMirror bool   `json:"is_mirror"`
}

// handleListRepositories handles requests to list all repositories
func (h *Handler) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	var repos []RepositoryItem

	// Walk through rootDir to find all git repositories at any depth
	err := filepath.Walk(h.repositoriesDir, func(path string, info os.FileInfo, err error) error {
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

			rel, _ := filepath.Rel(h.repositoriesDir, path)
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
		h.JSON(w, fmt.Errorf("failed to walk repositories directory: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSON(w, repos, http.StatusOK)
}
