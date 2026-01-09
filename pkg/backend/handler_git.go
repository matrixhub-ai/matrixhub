package backend

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/internal/utils"
	"github.com/wzshiming/gitd/pkg/repository"
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

	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "service parameter is required", http.StatusBadRequest)
		return
	}

	if service != "git-upload-pack" && service != "git-receive-pack" {
		http.Error(w, "unsupported service", http.StatusForbidden)
		return
	}

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
	isMirror, _, err := repo.IsMirror()
	if err != nil {
		http.Error(w, "Failed to check repository type", http.StatusInternalServerError)
		return
	}
	if isMirror {
		if service == "git-receive-pack" {
			http.Error(w, "push to mirror repository is not allowed", http.StatusForbidden)
			return
		}
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

	cmd := utils.Command(r.Context(), service, "--stateless-rpc", "--advertise-refs", dir)
	// Execute git command
	cmd.Dir = base
	cmd.Stdout = w
	err = cmd.Run()
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

	cmd := utils.Command(r.Context(), service, "--stateless-rpc", dir)
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
	if urlPath == "" {
		return ""
	}

	// Construct the full path
	fullPath := filepath.Join(h.rootDir, urlPath)

	return filepath.Clean(fullPath)
}

// packetLine formats a string as a git packet-line.
func packetLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}
