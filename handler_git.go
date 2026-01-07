package gitd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

func (h *Handler) registryGit(r *mux.Router) {
	r.HandleFunc("/{repo:.+}.git/info/refs", h.requireAuth(h.handleInfoRefs)).Methods(http.MethodGet)
	r.HandleFunc("/{repo:.+}.git/git-upload-pack", h.requireAuth(h.handleUploadPack)).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}.git/git-receive-pack", h.requireAuth(h.handleReceivePack)).Methods(http.MethodPost)
}

// handleInfoRefs handles the /info/refs endpoint for git service discovery.
func (h *Handler) handleInfoRefs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "service parameter is required", http.StatusBadRequest)
		return
	}

	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "unsupported service", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", service))
	w.Header().Set("Cache-Control", "no-cache")

	// Write packet-line formatted header
	if _, err := w.Write(packetLine(fmt.Sprintf("# service=%s\n", service))); err != nil {
		return
	}
	if _, err := w.Write([]byte("0000")); err != nil {
		return
	}

	base, dir := filepath.Split(repoPath)

	cmd := command(r.Context(), service, "--stateless-rpc", "--advertise-refs", dir)
	// Execute git command
	cmd.Dir = base
	cmd.Stdout = w
	err := cmd.Run()
	if err != nil {
		http.Error(w, "Failed to execute git command", http.StatusInternalServerError)
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
	repoName := vars["repo"] + ".git"

	repoPath := h.resolveRepoPath(repoName)
	if repoPath == "" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))
	w.Header().Set("Cache-Control", "no-cache")

	base, dir := filepath.Split(repoPath)

	cmd := command(r.Context(), service, "--stateless-rpc", dir)
	cmd.Dir = base
	cmd.Stdin = r.Body
	cmd.Stdout = w
	if err := cmd.Run(); err != nil {
		http.Error(w, "Failed to execute git command", http.StatusInternalServerError)
		return
	}
}

// resolveRepoPath resolves and validates a repository path.
func (h *Handler) resolveRepoPath(urlPath string) string {
	// Clean the path
	urlPath = strings.TrimPrefix(urlPath, "/")
	if urlPath == "" {
		return ""
	}

	// Construct the full path
	fullPath := filepath.Join(h.rootDir, urlPath)

	// Clean and verify the path is within RepoDir using filepath.Rel
	fullPath = filepath.Clean(fullPath)
	absRepoDir, err := filepath.Abs(h.rootDir)
	if err != nil {
		return ""
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return ""
	}
	// Use filepath.Rel to safely check if absFullPath is within absRepoDir
	relPath, err := filepath.Rel(absRepoDir, absFullPath)
	if err != nil {
		return ""
	}
	// Reject if the relative path starts with ".." (meaning it's outside RepoDir)
	if strings.HasPrefix(relPath, "..") {
		return ""
	}

	// Check if the path is a git repository
	if isGitRepository(fullPath) {
		return fullPath
	}

	// Try with .git extension
	gitPath := fullPath + ".git"
	if isGitRepository(gitPath) {
		return gitPath
	}

	return ""
}

// isGitRepository checks if the given path is a git repository.
func isGitRepository(path string) bool {
	// Check for bare repository (has HEAD file directly)
	if _, err := os.Stat(filepath.Join(path, "HEAD")); err == nil {
		return true
	}
	// Check for non-bare repository (has .git directory)
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return true
	}
	return false
}

// packetLine formats a string as a git packet-line.
func packetLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}
