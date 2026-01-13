package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	// Model info endpoint with revision - used by huggingface_hub for snapshot_download
	r.HandleFunc("/api/models/{repo:.+}/revision/{revision}", h.handleHFModelInfoRevision).Methods(http.MethodGet)

	// Model info endpoint - used by huggingface_hub to get model metadata
	r.HandleFunc("/api/models/{repo:.+}", h.handleHFModelInfo).Methods(http.MethodGet)

	// File download endpoint - used by huggingface_hub to download files
	// Pattern: /{repo_id}/resolve/{revision}/{path}
	r.HandleFunc("/{repo:.+}/resolve/{revision}/{path:.*}", h.handleHFResolve).Methods(http.MethodGet, http.MethodHead)
}

// HFModelInfo represents the model info response for HuggingFace API
type HFModelInfo struct {
	ID            string      `json:"id"`
	ModelID       string      `json:"modelId"`
	SHA           string      `json:"sha,omitempty"`
	Private       bool        `json:"private"`
	Disabled      bool        `json:"disabled"`
	Gated         bool        `json:"gated"`
	Downloads     int         `json:"downloads"`
	Likes         int         `json:"likes"`
	Tags          []string    `json:"tags"`
	Siblings      []HFSibling `json:"siblings"`
	CreatedAt     string      `json:"createdAt,omitempty"`
	LastModified  string      `json:"lastModified,omitempty"`
	DefaultBranch string      `json:"defaultBranch,omitempty"`
}

// HFSibling represents a file in the model repository
type HFSibling struct {
	RFilename string `json:"rfilename"`
}

// handleHFModelInfo handles the /api/models/{repo_id} endpoint
// This is used by huggingface_hub to get model metadata
func (h *Handler) handleHFModelInfo(w http.ResponseWriter, r *http.Request) {
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

	// Get list of files in the repository
	entries, err := repo.Tree(defaultBranch, "")
	if err != nil {
		// Return empty siblings if we can't get the tree
		entries = nil
	}

	var siblings []HFSibling
	if entries != nil {
		for _, entry := range entries {
			// Only include blob entries (files) to keep behavior consistent with handleHFModelInfoRevision.
			if entry.Type == "blob" {
				siblings = append(siblings, HFSibling{
					RFilename: entry.Path,
				})
			}
		}
	}

	// Get the latest commit SHA if available
	sha := ""
	commits, err := repo.Commits(defaultBranch, 1)
	if err == nil && len(commits) > 0 {
		sha = commits[0].SHA
	}

	modelInfo := HFModelInfo{
		ID:            repoID,
		ModelID:       repoID,
		SHA:           sha,
		Private:       false,
		Disabled:      false,
		Gated:         false,
		Downloads:     0,
		Likes:         0,
		Tags:          []string{},
		Siblings:      siblings,
		DefaultBranch: defaultBranch,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelInfo)
}

// handleHFModelInfoRevision handles the /api/models/{repo_id}/revision/{revision} endpoint
// This is used by huggingface_hub for snapshot_download to get model info at specific revision
func (h *Handler) handleHFModelInfoRevision(w http.ResponseWriter, r *http.Request) {
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

	// Get list of files in the repository at the specified revision
	entries, err := repo.Tree(ref, "")
	if err != nil {
		// Return empty siblings if we can't get the tree
		entries = nil
	}

	var siblings []HFSibling
	if entries != nil {
		for _, entry := range entries {
			if entry.Type == "blob" {
				siblings = append(siblings, HFSibling{
					RFilename: entry.Path,
				})
			}
		}
	}

	// Get the commit SHA for this revision
	sha := ""
	commits, err := repo.Commits(ref, 1)
	if err == nil && len(commits) > 0 {
		sha = commits[0].SHA
	}

	modelInfo := HFModelInfo{
		ID:            repoID,
		ModelID:       repoID,
		SHA:           sha,
		Private:       false,
		Disabled:      false,
		Gated:         false,
		Downloads:     0,
		Likes:         0,
		Tags:          []string{},
		Siblings:      siblings,
		DefaultBranch: repo.DefaultBranch(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(modelInfo); err != nil {
		log.Printf("failed to encode HF model info for repo %q (revision %q): %v", repoID, revision, err)
	}
}

// handleHFResolve handles the /{repo_id}/resolve/{revision}/{path} endpoint
// This is used by huggingface_hub to download files
func (h *Handler) handleHFResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoID := vars["repo"]
	revision := vars["revision"]
	filePath := vars["path"]

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

	// Use default branch if revision is "main" or empty
	ref := revision
	if ref == "" {
		ref = repo.DefaultBranch()
	}

	// Get commit hash for the HuggingFace client requirements
	commits, err := repo.Commits(ref, 1)
	commitHash := ""
	if err == nil && len(commits) > 0 {
		commitHash = commits[0].SHA
	}

	blob, err := repo.Blob(ref, filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Check if this is an LFS pointer file
	if blob.Size() <= lfs.MaxLFSPointerSize {
		reader, err := blob.NewReader()
		if err == nil {
			defer reader.Close()
			ptr, err := lfs.DecodePointer(reader)
			if err == nil && ptr != nil {
				// This is an LFS file, redirect to the LFS object
				// Set HuggingFace-required headers before redirect
				w.Header().Set("X-Repo-Commit", commitHash)
				w.Header().Set("ETag", fmt.Sprintf("\"%s\"", ptr.Oid))
				content, stat, err := h.contentStore.Get(ptr.Oid)
				if err != nil {
					log.Println("Error getting LFS object:", err)
					http.NotFound(w, r)
					return
				}
				defer content.Close()

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
	w.Header().Set("Content-Type", blob.ContentType())
	w.Header().Set("Content-Length", strconv.FormatInt(blob.Size(), 10))
	w.Header().Set("Last-Modified", blob.ModTime().UTC().Format(http.TimeFormat))

	// Handle HEAD request
	if r.Method == http.MethodHead {
		return
	}

	reader, err := blob.NewReader()
	if err != nil {
		http.Error(w, "Failed to get blob reader", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	_, err = io.Copy(w, reader)
	if err != nil {
		// Log but don't send error - we may have already written partial content
		return
	}
}
