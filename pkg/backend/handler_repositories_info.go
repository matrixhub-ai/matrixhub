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

func (h *Handler) registryRepositoriesInfo(r *mux.Router) {
	// Use {refpath:.*} to capture both ref and path since branches can contain '/'
	r.HandleFunc("/api/repositories/{repo:.+}.git/commits/{refpath:.*}", h.handleCommits).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/tree/{refpath:.*}", h.handleTree).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/tree", h.handleTree).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/blob/{refpath:.+}", h.handleBlob).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/api/repositories/{repo:.+}.git/branches", h.handleBranches).Methods(http.MethodGet)
}

// TreeEntry represents a file or directory in the repository
type TreeEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"` // "blob" or "tree"
	Mode       string `json:"mode"`
	SHA        string `json:"sha"`
	IsLFS      bool   `json:"isLfs,omitempty"`
	BlobSha256 string `json:"blobSha256,omitempty"`
}

// handleTree handles requests to list directory contents
func (h *Handler) handleTree(w http.ResponseWriter, r *http.Request) {
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

	entries, err := repo.Tree(ref, path)
	if err != nil {
		if repository.IsNotFoundError(err) {
			h.JSON(w, []any{}, http.StatusOK)
			return
		}
		h.JSON(w, fmt.Errorf("failed to get tree for repository %q at revision %q and path %q: %v", repoName, ref, path, err), http.StatusInternalServerError)
		return
	}

	h.JSON(w, entries, http.StatusOK)
}

// handleBlob handles requests to get file content
func (h *Handler) handleBlob(w http.ResponseWriter, r *http.Request) {
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

	blob, err := repo.Blob(ref, path)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get blob for repository %q at revision %q and path %q: %v", repoName, ref, path, err), http.StatusInternalServerError)
		return
	}

	if blob.Size() <= lfs.MaxLFSPointerSize {
		reader, err := blob.NewReader()
		if err == nil {
			defer func() {
				_ = reader.Close()
			}()
			ptr, err := lfs.DecodePointer(reader)
			if err == nil && ptr != nil {
				content, stat, err := h.contentStore.Get(ptr.Oid)
				if err != nil {
					h.JSON(w, fmt.Errorf("LFS object %q not found", ptr.Oid), http.StatusNotFound)
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

	w.Header().Set("Content-Length", strconv.FormatInt(blob.Size(), 10))
	w.Header().Set("Last-Modified", blob.ModTime().UTC().Format(http.TimeFormat))

	if r.Method == http.MethodHead {
		return
	}

	reader, err := blob.NewReader()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get blob reader for repository %q at revision %q and path %q: %v", repoName, ref, path, err), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = reader.Close()
	}()
	_, err = io.Copy(w, reader)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to read blob content for repository %q at revision %q and path %q: %v", repoName, ref, path, err), http.StatusInternalServerError)
		return
	}
}

// handleCommits handles requests to list commits
func (h *Handler) handleCommits(w http.ResponseWriter, r *http.Request) {
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

	ref, _, err := repo.SplitRevisionAndPath(refpath)
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to parse ref and path for repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	commits, err := repo.Commits(ref, 1)
	if err != nil {
		if repository.IsNotFoundError(err) {
			h.JSON(w, []any{}, http.StatusOK)
			return
		}
		h.JSON(w, fmt.Errorf("failed to get commits for repository %q at ref %q: %v", repoName, ref, err), http.StatusInternalServerError)
		return
	}

	h.JSON(w, commits, http.StatusOK)
}

// Branch represents a git branch
type Branch struct {
	Name string `json:"name"`
}

// handleBranches handles requests to list branches
func (h *Handler) handleBranches(w http.ResponseWriter, r *http.Request) {
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

	branches, err := repo.Branches()
	if err != nil {
		h.JSON(w, fmt.Errorf("failed to get branches for repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	var branchList []Branch
	for _, b := range branches {
		branchList = append(branchList, Branch{Name: b})
	}

	h.JSON(w, branchList, http.StatusOK)
}
