package gitd

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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

// getBranches returns a list of branch names for a repository
func (h *Handler) getBranches(repoPath string) ([]string, error) {
	base, dir := filepath.Split(repoPath)
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = filepath.Join(base, dir)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		name := strings.TrimLeft(line, "* ")
		name = strings.TrimSpace(name)
		if strings.Contains(name, "->") {
			continue
		}
		branches = append(branches, name)
	}
	return branches, nil
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

	// Get branches to properly parse ref and path
	branches, err := h.getBranches(repoPath)
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, path := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		base, dir := filepath.Split(repoPath)
		cmd := command(r.Context(), "git", "symbolic-ref", "--short", "HEAD")
		cmd.Dir = filepath.Join(base, dir)
		output, err := cmd.Output()
		if err == nil {
			ref = strings.TrimSpace(string(output))
		} else {
			ref = "main"
		}
	}

	base, dir := filepath.Split(repoPath)

	// Build the tree path
	treePath := ref
	if path != "" {
		treePath = ref + ":" + path
	}

	cmd := command(r.Context(),
		"git", "ls-tree", treePath)
	cmd.Dir = filepath.Join(base, dir)

	output, err := cmd.Output()
	if err != nil {
		if cmd.ProcessState.ExitCode() != 128 {
			http.Error(w, "Failed to list tree", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}

	entries := parseTreeOutput(string(output), path)

	cmd = command(r.Context(),
		"git", "lfs", "ls-files", "--long", treePath)
	cmd.Dir = filepath.Join(base, dir)
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

// parseTreeOutput parses the output of git ls-tree
func parseTreeOutput(output string, basePath string) []TreeEntry {
	var entries []TreeEntry
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Format: <mode> <type> <sha>\t<name>
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		meta := strings.Fields(parts[0])
		if len(meta) < 3 {
			continue
		}

		name := parts[1]
		fullPath := name
		if basePath != "" {
			fullPath = basePath + "/" + name
		}

		entries = append(entries, TreeEntry{
			Name: name,
			Path: fullPath,
			Mode: meta[0],
			Type: meta[1],
			SHA:  meta[2],
		})
	}

	return entries
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

	// Get branches to properly parse ref and path
	branches, err := h.getBranches(repoPath)
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, path := h.parseRefPath(refpath, branches)
	if ref == "" || path == "" {
		http.Error(w, "Invalid ref or path", http.StatusBadRequest)
		return
	}

	base, dir := filepath.Split(repoPath)

	cmd := command(r.Context(),
		"git", "show", ref+":"+path)
	cmd.Dir = filepath.Join(base, dir)
	cmd.Stdout = w

	contentType := getContentType(path)
	w.Header().Set("Content-Type", contentType)

	err = cmd.Run()
	if err != nil {
		http.Error(w, "Failed to get blob content", http.StatusNotFound)
		return
	}
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

	// Get branches to properly parse ref
	branches, err := h.getBranches(repoPath)
	if err != nil {
		http.Error(w, "Failed to get branches", http.StatusInternalServerError)
		return
	}

	ref, _ := h.parseRefPath(refpath, branches)
	if ref == "" {
		// Use default branch
		base, dir := filepath.Split(repoPath)
		cmd := command(r.Context(), "git", "symbolic-ref", "--short", "HEAD")
		cmd.Dir = filepath.Join(base, dir)
		output, err := cmd.Output()
		if err == nil {
			ref = strings.TrimSpace(string(output))
		} else {
			ref = "main"
		}
	}

	base, dir := filepath.Split(repoPath)

	cmd := command(r.Context(),
		"git", "log", "--format=%H|%s|%an|%ae|%ai", "-n", "20", ref)
	cmd.Dir = filepath.Join(base, dir)

	output, err := cmd.Output()
	if err != nil {
		if cmd.ProcessState.ExitCode() != 128 {
			http.Error(w, "Failed to get commits", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}

	commits := parseCommitsOutput(string(output))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commits)
}

// parseCommitsOutput parses the output of git log
func parseCommitsOutput(output string) []Commit {
	var commits []Commit
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}

		commits = append(commits, Commit{
			SHA:     parts[0],
			Message: parts[1],
			Author:  parts[2],
			Email:   parts[3],
			Date:    parts[4],
		})
	}

	return commits
}

// Branch represents a git branch
type Branch struct {
	Name    string `json:"name"`
	Current bool   `json:"current"`
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

	base, dir := filepath.Split(repoPath)

	cmd := command(r.Context(),
		"git", "branch", "-a")
	cmd.Dir = filepath.Join(base, dir)

	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to list branches", http.StatusInternalServerError)
		return
	}

	branches := parseBranchesOutput(string(output))

	if len(branches) == 0 {
		// Extract the default branch using git symbolic-ref HEAD
		cmd = command(r.Context(),
			"git", "symbolic-ref", "--short", "HEAD")
		cmd.Dir = filepath.Join(base, dir)

		output, err = cmd.Output()
		if err == nil {
			defaultBranch := strings.TrimSpace(string(output))
			branches = append(branches, Branch{
				Name:    defaultBranch,
				Current: true,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(branches)
}

// parseBranchesOutput parses the output of git branch
func parseBranchesOutput(output string) []Branch {
	var branches []Branch
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		current := strings.HasPrefix(line, "* ")
		name := strings.TrimPrefix(strings.TrimPrefix(line, "* "), "  ")
		name = strings.TrimSpace(name)

		// Skip HEAD references
		if strings.Contains(name, "->") {
			continue
		}

		branches = append(branches, Branch{
			Name:    name,
			Current: current,
		})
	}

	return branches
}
