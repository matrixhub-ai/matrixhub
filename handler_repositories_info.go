package gitd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

func (h *Handler) registryRepositoriesInfo(r *mux.Router) {
	// Use {refpath:.*} to capture both ref and path since branches can contain '/'
	r.HandleFunc("/api/repositories/{repo:.+}.git/commits/{refpath:.*}", h.handleCommits).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/tree/{refpath:.*}", h.handleTree).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/tree", h.handleTree).Methods(http.MethodGet)
	r.HandleFunc("/api/repositories/{repo:.+}.git/blob/{refpath:.+}", h.handleBlob).Methods(http.MethodGet)
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

func (h *Handler) openRepository(repoPath string) (*git.Repository, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// getBranches returns a list of branch names for a repository
func (h *Handler) getBranches(repo *git.Repository) ([]string, error) {
	branchesIter, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	var branches []string
	err = branchesIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		branches = append(branches, name)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}

func (h *Handler) getDefaultBranch(repo *git.Repository) (string, error) {
	headRef, err := repo.Head()
	if err != nil {
		return "", err
	}

	return headRef.Name().Short(), nil
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
	repo := vars["repo"] + ".git"
	refpath := vars["refpath"]

	repoName := repo
	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	repository, err := h.openRepository(repoPath)
	if err != nil {
		http.Error(w, "Failed to open repository", http.StatusInternalServerError)
		return
	}
	// Get branches to properly parse ref and path
	branches, err := h.getBranches(repository)
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, path := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		defaultBranch, err := h.getDefaultBranch(repository)
		if err != nil {
			http.Error(w, "Failed to get default branch", http.StatusInternalServerError)
			return
		}
		ref = defaultBranch
		if ref == "" {
			ref = "main"
		}
	}

	refObj, err := repository.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err != nil {
		http.Error(w, "Failed to resolve reference", http.StatusNotFound)
		return
	}

	commit, err := repository.CommitObject(refObj.Hash())
	if err != nil {
		http.Error(w, "Failed to get commit object", http.StatusInternalServerError)
		return
	}

	tree, err := commit.Tree()
	if err != nil {
		http.Error(w, "Failed to get tree object", http.StatusInternalServerError)
		return
	}

	if path != "" {
		entry, err := tree.FindEntry(path)
		if err != nil {
			http.Error(w, "Path not found in tree", http.StatusNotFound)
			return
		}

		if entry.Mode.IsFile() {
			http.Error(w, "Path is not a directory", http.StatusBadRequest)
			return
		}

		tree, err = repository.TreeObject(entry.Hash)
		if err != nil {
			http.Error(w, "Failed to get subtree object", http.StatusInternalServerError)
			return
		}
	}

	var entries []TreeEntry
	for _, entry := range tree.Entries {
		entryType := "blob"
		if !entry.Mode.IsFile() {
			entryType = "tree"
		}

		entries = append(entries, TreeEntry{
			Name: entry.Name,
			Path: filepath.Join(path, entry.Name),
			Type: entryType,
			Mode: entry.Mode.String(),
			SHA:  entry.Hash.String(),
		})
	}

	// Build the tree path
	treePath := ref
	if path != "" {
		treePath = ref + ":" + path
	}

	cmd := command(r.Context(),
		"git", "lfs", "ls-files", "--long", treePath)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	lfsOutput, err := cmd.Output()

	if err == nil {
		lfsFiles := parseLFSFilesOutput(string(lfsOutput))
		if len(lfsFiles) != 0 {
			for i := range entries {
				if entries[i].Type == "blob" {
					if sha256, ok := lfsFiles[entries[i].Name]; ok {
						entries[i].IsLFS = true
						entries[i].BlobSha256 = sha256
					}
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func parseLFSFilesOutput(output string) map[string]string {
	lfsFiles := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Format: <sha256> - <file>
		parts := strings.SplitN(line, "-", 2)
		if len(parts) != 2 {
			continue
		}

		sha256 := strings.TrimSpace(parts[0])
		file := strings.TrimSpace(parts[1])

		lfsFiles[file] = sha256
	}

	return lfsFiles
}

// handleBlob handles requests to get file content
func (h *Handler) handleBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"
	refpath := vars["refpath"]

	repoName := repo
	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	repository, err := h.openRepository(repoPath)
	if err != nil {
		http.Error(w, "Failed to open repository", http.StatusInternalServerError)
		return
	}

	branches, err := h.getBranches(repository)
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, path := h.parseRefPath(refpath, branches)
	if ref == "" || path == "" {
		http.Error(w, "Invalid ref or path", http.StatusBadRequest)
		return
	}

	refObj, err := repository.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err != nil {
		http.Error(w, "Failed to resolve reference", http.StatusNotFound)
		return
	}

	commit, err := repository.CommitObject(refObj.Hash())
	if err != nil {
		http.Error(w, "Failed to get commit object", http.StatusInternalServerError)
		return
	}

	tree, err := commit.Tree()
	if err != nil {
		http.Error(w, "Failed to get tree object", http.StatusInternalServerError)
		return
	}

	dir, filename := filepath.Split(path)
	var file *object.File
	dir = strings.TrimSuffix(dir, "/")
	if dir != "" {
		entry, err := tree.FindEntry(strings.TrimSuffix(dir, "/"))
		if err != nil {
			http.Error(w, fmt.Sprintf("Path %s not found in tree", dir), http.StatusNotFound)
			return
		}

		if entry.Mode.IsFile() {
			http.Error(w, "Path is not a directory", http.StatusBadRequest)
			return
		}

		tree, err = repository.TreeObject(entry.Hash)
		if err != nil {
			http.Error(w, "Failed to get subtree object", http.StatusInternalServerError)
			return
		}
	}

	file, err = tree.File(filename)
	if err != nil {
		http.Error(w, "File not found in tree", http.StatusNotFound)
		return
	}

	if file.Size > 100*1024 { // 100 KB limit
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "File too large to display")
		return
	}

	reader, err := file.Blob.Reader()
	if err != nil {
		http.Error(w, "Failed to read blob content", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	_, err = io.Copy(w, reader)
	if err != nil {
		http.Error(w, "Failed to write blob content", http.StatusInternalServerError)
		return
	}

	contentType := getContentType(path)
	w.Header().Set("Content-Type", contentType)
}

// Commit represents a git commit
type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Date    string `json:"date"`
}

// handleCommits handles requests to list commits
func (h *Handler) handleCommits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"
	refpath := vars["refpath"]

	repoName := repo
	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	repository, err := h.openRepository(repoPath)
	if err != nil {
		http.Error(w, "Failed to open repository", http.StatusInternalServerError)
		return
	}

	// Get branches to properly parse ref
	branches, err := h.getBranches(repository)
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, _ := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		defaultBranch, err := h.getDefaultBranch(repository)
		if err != nil {
			http.Error(w, "Failed to get default branch", http.StatusInternalServerError)
			return
		}
		ref = defaultBranch
		if ref == "" {
			ref = "main"
		}
	}

	refObj, err := repository.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err != nil {
		http.Error(w, "Failed to resolve reference", http.StatusNotFound)
		return
	}

	commitIter, err := repository.Log(&git.LogOptions{From: refObj.Hash()})
	if err != nil {
		http.Error(w, "Failed to get commit log", http.StatusInternalServerError)
		return
	}

	var commits []Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, Commit{
			SHA:     c.Hash.String(),
			Message: c.Message,
			Author:  c.Author.Name,
			Email:   c.Author.Email,
			Date:    c.Author.When.Format("2006-01-02T15:04:05Z"),
		})
		if len(commits) >= 20 {
			return io.EOF // Stop after 20 commits
		}
		return nil
	})
	if err != nil && err != io.EOF {
		http.Error(w, "Failed to iterate commits", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}

// Branch represents a git branch
type Branch struct {
	Name string `json:"name"`
	// Current bool   `json:"current"`
}

// handleBranches handles requests to list branches
func (h *Handler) handleBranches(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"

	repoName := repo
	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	repository, err := h.openRepository(repoPath)
	if err != nil {
		http.Error(w, "Failed to open repository", http.StatusInternalServerError)
		return
	}

	branches, err := h.getBranches(repository)
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
