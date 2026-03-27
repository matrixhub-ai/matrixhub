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

package hf

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"
)

func (h *Handler) handleTree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	ri := getRepoInformation(r)
	revpath := vars["revpath"]

	query := r.URL.Query()
	recursive, _ := strconv.ParseBool(query.Get("recursive"))
	expand, _ := strconv.ParseBool(query.Get("expand"))

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationReadRepo, ri.RepoName, permission.Context{}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
		return
	}

	repo, err := h.openRepo(r.Context(), repoPath, ri.RepoName, repository.GitUploadPack)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	rev, path, err := repo.SplitRevisionAndPath(revpath)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to parse rev and path for repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	entries, err := repo.Tree(rev, path, &repository.TreeOptions{
		Recursive: recursive,
	})
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to get tree for repo %q at rev %q and path %q: %v", ri.RepoName, rev, path, err), http.StatusInternalServerError)
		return
	}

	responseJSON(w, toHFTreeEntries(r.Context(), entries, expand), http.StatusOK)
}

func toHFTreeEntries(ctx context.Context, entries []*repository.TreeEntry, expand bool) []treeEntry {
	result := make([]treeEntry, len(entries))
	for i, e := range entries {

		blob, err := e.Blob()
		if err != nil {
			slog.WarnContext(ctx, "failed to get blob for tree entry, skipping", "path", e.Path(), "error", err)
			continue
		}

		result[i] = treeEntry{
			OID:  e.Hash().String(),
			Path: e.Path(),
			Type: e.Type(),
			Size: blob.Size(),
		}
		if ptr, _ := blob.LFSPointer(); ptr != nil {
			result[i].LFS = &lfsPointer{
				OID:         ptr.OID(),
				Size:        ptr.Size(),
				PointerSize: blob.Size(),
			}
			result[i].Size = ptr.Size()
		}
		if lastCommit := e.LastCommit(); expand && lastCommit != nil {
			result[i].LastCommit = &treeLastCommit{
				ID:    lastCommit.Hash().String(),
				Title: lastCommit.Title(),
				Date:  lastCommit.Author().When().UTC().Format(repository.TimeFormat),
			}
		}
	}
	return result
}

// handleTreeSize handles GET /api/{repoType}/{namespace}/{repo}/treesize/{revpath}
func (h *Handler) handleTreeSize(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	ri := getRepoInformation(r)
	revpath := vars["revpath"]

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationReadRepo, ri.RepoName, permission.Context{}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
		return
	}

	repo, err := h.openRepo(r.Context(), repoPath, ri.RepoName, repository.GitUploadPack)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	rev, path, err := repo.SplitRevisionAndPath(revpath)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to parse rev and path for repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	size, err := repo.TreeSize(rev, path)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to get tree size for repo %q at rev %q and path %q: %v", ri.RepoName, rev, path, err), http.StatusInternalServerError)
		return
	}

	responseJSON(w, treeSize{
		Path: "/" + path,
		Size: size,
	}, http.StatusOK)
}

// handleResolve handles the /{repo_id}/resolve/{revision}/{path} endpoint
// This is used by huggingface_hub to download files
func (h *Handler) handleResolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	ri := getRepoInformation(r)
	revpath := vars["revpath"]

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationReadRepo, ri.RepoName, permission.Context{}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
		return
	}

	repo, err := h.openRepo(r.Context(), repoPath, ri.RepoName, repository.GitUploadPack)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	rev, path, err := repo.SplitRevisionAndPath(revpath)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to parse rev and path for repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	// Get commit hash for the HuggingFace client requirements
	commits, err := repo.Commits(rev, &repository.CommitsOptions{Limit: 1})
	commitHash := ""
	if err == nil && len(commits) > 0 {
		commitHash = commits[0].Hash().String()
	}

	blob, err := repo.Blob(rev, path)
	if err != nil {
		responseJSON(w, fmt.Errorf("file %q not found in repository %q at revision %q", path, ri.RepoName, rev), http.StatusNotFound)
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
				w.Header().Set("ETag", fmt.Sprintf("\"%s\"", ptr.OID()))

				if h.mirror != nil && !h.lfsStorage.Exists(ptr.OID()) {
					// Try tee cache fetch if configured
					pf := h.mirror.Get(ptr.OID())
					if pf != nil {
						rs := pf.NewReadSeeker()
						defer func() {
							_ = rs.Close()
						}()
						http.ServeContent(w, r, ptr.OID(), pf.ModTime(), rs)
						return
					}
					responseJSON(w, fmt.Errorf("LFS object %q not found for file %q in repository %q at revision %q", ptr.OID(), path, ri.RepoName, rev), http.StatusNotFound)
					return
				}
				if signer, ok := h.lfsStorage.(lfs.SignGetter); ok {
					url, err := signer.SignGet(ptr.OID())
					if err != nil {
						responseJSON(w, fmt.Errorf("failed to sign URL for LFS object %q: %v", ptr.OID(), err), http.StatusInternalServerError)
						return
					}
					http.Redirect(w, r, url, http.StatusTemporaryRedirect)
					return
				}
				if getter, ok := h.lfsStorage.(lfs.Getter); ok {
					content, stat, err := getter.Get(ptr.OID())
					if err != nil {
						if os.IsNotExist(err) {
							responseJSON(w, fmt.Errorf("LFS object %q not found for file %q in repository %q at revision %q", ptr.OID(), path, ri.RepoName, rev), http.StatusNotFound)
							return
						}
						responseJSON(w, fmt.Errorf("failed to get LFS object %q: %v", ptr.OID(), err), http.StatusInternalServerError)
						return
					}
					defer func() {
						_ = content.Close()
					}()

					http.ServeContent(w, r, ptr.OID(), stat.ModTime(), content)
					return
				}
				responseJSON(w, fmt.Errorf("LFS storage does not support direct content retrieval for object %q", ptr.OID()), http.StatusNotImplemented)
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
		responseJSON(w, fmt.Errorf("failed to get blob reader for file %q in repository %q at revision %q: %v", path, ri.RepoName, rev, err), http.StatusInternalServerError)
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
