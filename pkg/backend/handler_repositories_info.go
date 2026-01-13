package backend

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

func (h *Handler) registryRepositoriesInfo(r *mux.Router) {
	// Use {refpath:.*} to capture both ref and path since branches can contain '/'
	r.HandleFunc("/api/repositories/{repo:.+}.git/commits/{refpath:.*}", h.requireAuth(h.handleCommits)).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/tree/{refpath:.*}", h.requireAuth(h.handleTree)).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/tree", h.requireAuth(h.handleTree)).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/blob/{refpath:.+}", h.requireAuth(h.handleBlob)).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/api/repositories/{repo:.+}.git/branches", h.requireAuth(h.handleBranches)).Methods(http.MethodGet)
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

// parseRefPath parses a combined ref/path string using the branch list
// Branches can contain '/', so we need to match against the branch list
// Returns the ref (branch) and the remaining path
func (h *Handler) parseRefPath(refpath string, branches []string) (ref string, path string) {
	if refpath == "" {
		return "", ""
	}

	// Sort branches by length (longest first) to match the most specific branch
	sortedBranches := make([]string, len(branches))
	copy(sortedBranches, branches)
	sort.Slice(sortedBranches, func(i, j int) bool {
		return len(sortedBranches[i]) > len(sortedBranches[j])
	})

	for _, branch := range sortedBranches {
		if refpath == branch {
			return branch, ""
		}
		if strings.HasPrefix(refpath, branch+"/") {
			return branch, refpath[len(branch)+1:]
		}
	}

	// Fallback: treat first segment as branch
	parts := strings.SplitN(refpath, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return refpath, ""
}

// handleTree handles requests to list directory contents
func (h *Handler) handleTree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"
	refpath := vars["refpath"]

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
	// Get branches to properly parse ref and path
	branches, err := repo.Branches()
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, path := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		ref = repo.DefaultBranch()
	}

	entries, err := repo.Tree(ref, path)
	if err != nil {
		if errors.Is(err, repository.ErrObjectNotFound) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entries)
			return
		}
		http.Error(w, "Failed to get tree", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handleBlob handles requests to get file content
func (h *Handler) handleBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"
	refpath := vars["refpath"]

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

	branches, err := repo.Branches()
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, path := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		ref = repo.DefaultBranch()
	}

	blob, err := repo.Blob(ref, path)
	if err != nil {
		http.Error(w, "Failed to get blob", http.StatusInternalServerError)
		return
	}

	if blob.Size() <= lfs.MaxLFSPointerSize {
		reader, err := blob.NewReader()
		if err == nil {
			defer reader.Close()
			ptr, err := lfs.DecodePointer(reader)
			if err == nil && ptr != nil {
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

	w.Header().Set("Content-Type", blob.ContentType())
	w.Header().Set("Content-Length", strconv.FormatInt(blob.Size(), 10))
	w.Header().Set("Last-Modified", blob.ModTime().UTC().Format(http.TimeFormat))

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
		http.Error(w, "Failed to read blob content", http.StatusInternalServerError)
		return
	}
}

// handleCommits handles requests to list commits
func (h *Handler) handleCommits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"
	refpath := vars["refpath"]

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

	branches, err := repo.Branches()
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, _ := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		ref = repo.DefaultBranch()
	}

	commits, err := repo.Commits(ref, 1)
	if err != nil {
		if errors.Is(err, repository.ErrObjectNotFound) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(commits)
			return
		}
		http.Error(w, "Failed to get commits", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}

// Branch represents a git branch
type Branch struct {
	Name string `json:"name"`
}

// handleBranches handles requests to list branches
func (h *Handler) handleBranches(w http.ResponseWriter, r *http.Request) {
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

	branches, err := repo.Branches()
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	var branchList []Branch
	for _, b := range branches {
		branchList = append(branchList, Branch{Name: b})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(branchList)
}
