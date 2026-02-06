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
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

// registryHuggingFace registers the HuggingFace-compatible API endpoints.
// These endpoints allow using huggingface-cli and huggingface_hub library
// with HF_ENDPOINT pointing to this server.
func (h *Handler) registryHuggingFace(r *mux.Router) {
	// AGENTS.md content endpoint with revision
	r.HandleFunc("/api/models/{repo:.+}/agents/revision/{revision}", h.handleHFAgentsContentRevision).Methods(http.MethodGet)

	// AGENTS.md content endpoint - returns plain markdown content
	r.HandleFunc("/api/models/{repo:.+}/agents", h.handleHFAgentsContent).Methods(http.MethodGet)

	// Model info endpoint with revision - used by huggingface_hub for snapshot_download
	r.HandleFunc("/api/models/{repo:.+}/revision/{revision:.*}", h.handleHFModelInfoRevision).Methods(http.MethodGet)

	// Tree endpoint - used by huggingface_hub to list files in the model repository
	r.HandleFunc("/api/models/{repo:.+}/tree/{refpath:.*}", h.handleHFTree).Methods(http.MethodGet)

	// Model info endpoint - used by huggingface_hub to get model metadata
	r.HandleFunc("/api/models/{repo:.+}", h.handleHFModelInfo).Methods(http.MethodGet)

	// File download endpoint - used by huggingface_hub to download files
	r.HandleFunc("/{repo:.+}/resolve/{refpath:.*}", h.handleHFResolve).Methods(http.MethodGet, http.MethodHead)

}

// HFModelInfo represents the model info response for HuggingFace API
type HFModelInfo struct {
	ID            string                    `json:"id"`
	ModelID       string                    `json:"modelId"`
	SHA           string                    `json:"sha,omitempty"`
	Private       bool                      `json:"private"`
	Disabled      bool                      `json:"disabled"`
	Gated         bool                      `json:"gated"`
	Downloads     int                       `json:"downloads"`
	Likes         int                       `json:"likes"`
	Tags          []string                  `json:"tags"`
	Siblings      []HFSibling               `json:"siblings"`
	CreatedAt     string                    `json:"createdAt,omitempty"`
	LastModified  string                    `json:"lastModified,omitempty"`
	DefaultBranch string                    `json:"defaultBranch,omitempty"`
	AgentsContent *repository.AgentsContent `json:"agentsContent,omitempty"`
}

// HFSibling represents a file in the model repository
type HFSibling struct {
	RFilename string `json:"rfilename"`
}

// handleHFModelInfo handles the /api/models/{repo_id} endpoint
// This is used by huggingface_hub to get model metadata
func (h *Handler) handleHFModelInfo(w http.ResponseWriter, r *http.Request) {
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

	defaultBranch := repo.DefaultBranch()

	// Get list of files in the repository
	entries, err := repo.Tree(defaultBranch, "")
	if err != nil {
		// Return empty siblings if we can't get the tree
		entries = nil
	}

	var siblings []HFSibling
	for _, entry := range entries {
		// Only include blob entries (files) to keep behavior consistent with handleHFModelInfoRevision.
		if entry.Type == "blob" {
			siblings = append(siblings, HFSibling{
				RFilename: entry.Path,
			})
		}
	}

	// Get the latest commit SHA if available
	sha := ""
	commits, err := repo.Commits(defaultBranch, 1)
	if err == nil && len(commits) > 0 {
		sha = commits[0].SHA
	}

	// Try to get AGENTS.md content if available
	var agentsContent *repository.AgentsContent
	if content, err := repo.GetAgentsContent(defaultBranch); err == nil {
		agentsContent = content
	}

	modelInfo := HFModelInfo{
		ID:            repoName,
		ModelID:       repoName,
		SHA:           sha,
		Private:       false,
		Disabled:      false,
		Gated:         false,
		Downloads:     0,
		Likes:         0,
		Tags:          []string{},
		Siblings:      siblings,
		DefaultBranch: defaultBranch,
		AgentsContent: agentsContent,
	}

	h.JSON(w, modelInfo, http.StatusOK)
}

func (h *Handler) handleHFTree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	repoName := vars["repo"]
	refpath := vars["refpath"]

	query := r.URL.Query()
	recursive, _ := strconv.ParseBool(query.Get("recursive"))
	expand, _ := strconv.ParseBool(query.Get("expand"))

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

	ref, path, err := repo.SplitRevisionAndPath(refpath)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to parse ref and path for repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	entries, err := repo.HFTree(ref, path, &repository.HFTreeOptions{
		Recursive: recursive,
		Expand:    expand,
	})
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get tree for repo %q at ref %q and path %q: %v", repoName, ref, path, err), http.StatusInternalServerError)
		return
	}

	h.JSON(w, entries, http.StatusOK)
}

// handleHFModelInfoRevision handles the /api/models/{repo_id}/revision/{revision} endpoint
// This is used by huggingface_hub for snapshot_download to get model info at specific revision
func (h *Handler) handleHFModelInfoRevision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	repoName := vars["repo"]
	ref := vars["revision"]

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

	// Get list of files in the repository at the specified revision
	entries, err := repo.Tree(ref, "")
	if err != nil {
		// Return empty siblings if we can't get the tree
		entries = nil
	}

	var siblings []HFSibling
	for _, entry := range entries {
		if entry.Type == "blob" {
			siblings = append(siblings, HFSibling{
				RFilename: entry.Path,
			})
		}
	}

	// Get the commit SHA for this revision
	sha := ""
	commits, err := repo.Commits(ref, 1)
	if err == nil && len(commits) > 0 {
		sha = commits[0].SHA
	}

	// Try to get AGENTS.md content if available
	var agentsContent *repository.AgentsContent
	if content, err := repo.GetAgentsContent(ref); err == nil {
		agentsContent = content
	}

	modelInfo := HFModelInfo{
		ID:            repoName,
		ModelID:       repoName,
		SHA:           sha,
		Private:       false,
		Disabled:      false,
		Gated:         false,
		Downloads:     0,
		Likes:         0,
		Tags:          []string{},
		Siblings:      siblings,
		DefaultBranch: repo.DefaultBranch(),
		AgentsContent: agentsContent,
	}

	h.JSON(w, modelInfo, http.StatusOK)
}

// handleHFResolve handles the /{repo_id}/resolve/{revision}/{path} endpoint
// This is used by huggingface_hub to download files
func (h *Handler) handleHFResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	repoName := vars["repo"]
	refpath := vars["refpath"]

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

	ref, path, err := repo.SplitRevisionAndPath(refpath)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to parse ref and path for repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	// Get commit hash for the HuggingFace client requirements
	commits, err := repo.Commits(ref, 1)
	commitHash := ""
	if err == nil && len(commits) > 0 {
		commitHash = commits[0].SHA
	}

	blob, err := repo.Blob(ref, path)
	if err != nil {
		h.JSON(w, fmt.Errorf("file %q not found in repository %q at revision %q", path, repoName, ref), http.StatusNotFound)
		return
	}

	// Check if this is an LFS pointer file
	if blob.Size() <= lfs.MaxLFSPointerSize {
		reader, err := blob.NewReader()
		if err == nil {
			defer func() {
				_ = reader.Close()
			}()
			ptr, err := lfs.DecodePointer(reader)
			if err == nil && ptr != nil {
				// This is an LFS file, redirect to the LFS object
				// Set HuggingFace-required headers before redirect
				w.Header().Set("X-Repo-Commit", commitHash)
				w.Header().Set("ETag", fmt.Sprintf("\"%s\"", ptr.Oid))
				if h.s3Store != nil {
					url, err := h.s3Store.SignGet(ptr.Oid)
					if err != nil {
						h.JSON(w, fmt.Errorf("failed to sign S3 URL for LFS object %q: %v", ptr.Oid, err), http.StatusInternalServerError)
						return
					}
					http.Redirect(w, r, url, http.StatusTemporaryRedirect)
					return
				}
				content, stat, err := h.contentStore.Get(ptr.Oid)
				if err != nil {
					h.JSON(w, fmt.Errorf("LFS object %q not found for file %q in repository %q at revision %q", ptr.Oid, path, repoName, ref), http.StatusNotFound)
					return
				}
				defer func() {
					_ = content.Close()
				}()

				http.ServeContent(w, r, ptr.Oid, stat.ModTime(), content)
				return
			}
		}
	}

	// Set HuggingFace-required headers
	// X-Repo-Commit is required by huggingface_hub to identify the commit
	w.Header().Set("X-Repo-Commit", commitHash)

	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", blob.Hash()))

	// Serve regular file content
	w.Header().Set("Content-Length", strconv.FormatInt(blob.Size(), 10))
	w.Header().Set("Last-Modified", blob.ModTime().UTC().Format(http.TimeFormat))

	// Handle HEAD request
	if r.Method == http.MethodHead {
		return
	}

	reader, err := blob.NewReader()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get blob reader for file %q in repository %q at revision %q: %v", path, repoName, ref, err), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = reader.Close()
	}()

	_, err = io.Copy(w, reader)
	if err != nil {
		// Log but don't send error - we may have already written partial content
		return
	}
}

// handleHFAgentsContent handles the /api/models/{repo_id}/agents endpoint
// Returns AGENTS.md content as JSON
func (h *Handler) handleHFAgentsContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoID := vars["repo"]

	// Convert repo ID to internal repo name (add .git suffix)
	repoName := repoID + ".git"

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

	defaultBranch := repo.DefaultBranch()
	if defaultBranch == "" {
		// Empty repository with no commits
		http.NotFound(w, r)
		return
	}

	// Get AGENTS.md content
	content, err := repo.GetAgentsContent(defaultBranch)
	if err != nil {
		if errors.Is(err, repository.ErrAgentsFileNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get AGENTS.md: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("failed to encode AGENTS.md content for repo %q: %v", repoID, err)
	}
}

// handleHFAgentsContentRevision handles the /api/models/{repo_id}/agents/revision/{revision} endpoint
// Returns AGENTS.md content as JSON for a specific revision
func (h *Handler) handleHFAgentsContentRevision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoID := vars["repo"]
	revision := vars["revision"]

	// Convert repo ID to internal repo name (add .git suffix)
	repoName := repoID + ".git"

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

	// Use the specified revision or fall back to default branch
	ref := revision
	if ref == "" {
		ref = repo.DefaultBranch()
	}

	// Get AGENTS.md content
	content, err := repo.GetAgentsContent(ref)
	if err != nil {
		if errors.Is(err, repository.ErrAgentsFileNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get AGENTS.md: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.Printf("failed to encode AGENTS.md content for repo %q (revision %q): %v", repoID, revision, err)
	}
}
