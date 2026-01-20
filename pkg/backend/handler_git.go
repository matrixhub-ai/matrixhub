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
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

func (h *Handler) registryGit(r *mux.Router) {
	r.HandleFunc("/{repo:.+}.git/info/refs", h.handleInfoRefs).Methods(http.MethodGet)
	r.HandleFunc("/{repo:.+}/info/refs", h.handleInfoRefs).Methods(http.MethodGet)
	r.HandleFunc("/{repo:.+}.git/git-upload-pack", h.handleUploadPack).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}/git-upload-pack", h.handleUploadPack).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}.git/git-receive-pack", h.handleReceivePack).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}/git-receive-pack", h.handleReceivePack).Methods(http.MethodPost)
}

// handleInfoRefs handles the /info/refs endpoint for git service discovery.
func (h *Handler) handleInfoRefs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	service := r.URL.Query().Get("service")
	if service == "" {
		h.Text(w, "service parameter is required", http.StatusBadRequest)
		return
	}

	if service != "git-upload-pack" && service != "git-receive-pack" {
		h.Text(w, "unsupported service", http.StatusForbidden)
		return
	}

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		h.Text(w, fmt.Sprintf("repository %q not found", repoName), http.StatusNotFound)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			h.Text(w, fmt.Sprintf("repository %q not found", repoName), http.StatusNotFound)
			return
		}
		h.Text(w, fmt.Sprintf("Failed to open repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}
	if service == "git-receive-pack" {
		isMirror, _, err := repo.IsMirror()
		if err != nil {
			h.Text(w, fmt.Sprintf("Failed to check repository type for %q: %v", repoName, err), http.StatusInternalServerError)
			return
		}
		if isMirror {
			h.Text(w, fmt.Sprintf("push to mirror repository %q is not allowed", repoName), http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")

	err = repo.Stateless(r.Context(), w, nil, service, true)
	if err != nil {
		h.Text(w, fmt.Sprintf("Failed to get info refs for %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}
}

// handleUploadPack handles the git-upload-pack endpoint (fetch/clone).
func (h *Handler) handleUploadPack(w http.ResponseWriter, r *http.Request) {
	h.handleService(w, r, "git-upload-pack")
}

// handleReceivePack handles the git-receive-pack endpoint (push).
func (h *Handler) handleReceivePack(w http.ResponseWriter, r *http.Request) {
	h.handleService(w, r, "git-receive-pack")
}

// handleService handles a git service request.
func (h *Handler) handleService(w http.ResponseWriter, r *http.Request, service string) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		h.Text(w, fmt.Sprintf("repository %q not found", repoName), http.StatusNotFound)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			h.Text(w, fmt.Sprintf("repository %q not found", repoName), http.StatusNotFound)
			return
		}
		h.Text(w, fmt.Sprintf("Failed to open repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}
	if service == "git-receive-pack" {
		isMirror, _, err := repo.IsMirror()
		if err != nil {
			h.Text(w, fmt.Sprintf("Failed to check repository type for %q: %v", repoName, err), http.StatusInternalServerError)
			return
		}
		if isMirror {
			h.Text(w, fmt.Sprintf("push to mirror repository %q is not allowed", repoName), http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))
	w.Header().Set("Cache-Control", "no-cache")

	err = repo.Stateless(r.Context(), w, r.Body, service, false)
	if err != nil {
		h.Text(w, fmt.Sprintf("Failed to get info refs for %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}
}

// resolveRepoPath resolves and validates a repository path.
func (h *Handler) resolveRepoPath(urlPath string) string {
	if urlPath == "" {
		return ""
	}

	if !strings.HasSuffix(urlPath, ".git") {
		urlPath += ".git"
	}

	// Construct the full path
	fullPath := filepath.Join(h.repositoriesDir, urlPath)

	return filepath.Clean(fullPath)
}
