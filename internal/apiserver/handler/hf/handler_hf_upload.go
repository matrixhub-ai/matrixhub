package hf

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	"github.com/matrixhub-ai/hfd/pkg/repository"
)

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
		scheme = fwdProto
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// handleValidateYAML handles POST /api/validate-yaml
// This endpoint is called by huggingface_hub to validate YAML front matter in files like README.md.
// https://github.com/huggingface/huggingface_hub/blob/8708631a463d9a6dc8ec7b046e748cc14844474e/src/huggingface_hub/repocard.py#L189-L224
func (h *Handler) handleValidateYAML(w http.ResponseWriter, r *http.Request) {
	// Return a successful validation response
	responseJSON(w, struct {
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	}{
		Errors:   []string{},
		Warnings: []string{},
	}, http.StatusOK)
}

func repoTypePrefix(repoType string) string {
	switch repoType {
	case "dataset":
		return "datasets"
	case "space":
		return "spaces"
	default:
		return ""
	}
}

// handleCreateRepo handles POST /api/repos/create
func (h *Handler) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	repoName := req.Name
	if req.Organization != "" {
		repoName = req.Organization + "/" + repoName
	}

	storageName := repoName
	prefix := repoTypePrefix(req.Type)
	if prefix != "" {
		storageName = prefix + "/" + repoName
	}

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationCreateRepo, storageName, permission.Context{}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	user, ok := authenticate.GetUserInfo(r.Context())
	if !ok {
		user = authenticate.UserInfo{
			User:  "HuggingFace",
			Email: "hf@users.noreply.huggingface.co",
		}
	}

	urlName := "/" + storageName

	repoPath := h.storage.ResolvePath(storageName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("invalid repository name: %q", repoName), http.StatusBadRequest)
		return
	}

	// Check if repository already exists
	if repository.IsRepository(repoPath) {
		resp := createRepoResponse{
			URL: fmt.Sprintf("%s%s", requestOrigin(r), urlName),
		}
		responseJSON(w, resp, http.StatusOK)
		return
	}

	// Create repository directory
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		responseJSON(w, fmt.Errorf("failed to create repository directory: %v", err), http.StatusInternalServerError)
		return
	}

	defaultBranch := "main"

	// Initialize bare repository
	repo, err := repository.Init(r.Context(), repoPath, defaultBranch)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to initialize repository: %v", err), http.StatusInternalServerError)
		return
	}

	// Create initial commit with default .gitattributes
	_, err = repo.CreateCommit(context.Background(), defaultBranch, "Initial commit", user.User, user.Email, []repository.CommitOperation{
		{
			Type:    repository.CommitOperationAdd,
			Path:    repository.GitattributesFileName,
			Content: repository.GitattributesText,
		},
	}, "")
	if err != nil {
		_ = repo.Remove()
		responseJSON(w, fmt.Errorf("failed to create initial commit: %v", err), http.StatusInternalServerError)
		return
	}

	resp := createRepoResponse{
		URL: fmt.Sprintf("%s%s", requestOrigin(r), urlName),
	}
	responseJSON(w, resp, http.StatusOK)
}

// handlePreupload handles POST /api/{repoType}/{repo}/preupload/{rev}
func (h *Handler) handlePreupload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ri := getRepoInformation(r)
	rev := vars["rev"]

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationUpdateRepo, ri.RepoName, permission.Context{Ref: rev}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	var req preuploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository not found"), http.StatusNotFound)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	gitAttrs, err := repo.GitAttributes(rev)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to read .gitattributes for repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	var respFiles []preuploadResponseFile
	for _, file := range req.Files {
		uploadMode := "regular"
		if gitAttrs.IsLFS(file.Path) {
			uploadMode = "lfs"
		}

		respFiles = append(respFiles, preuploadResponseFile{
			Path:       file.Path,
			UploadMode: uploadMode,
		})
	}

	resp := preuploadResponse{
		Files: respFiles,
	}
	responseJSON(w, resp, http.StatusOK)
}

// handleCommit handles POST /api/{repoType}/{repo}/commit/{rev}
func (h *Handler) handleCommit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ri := getRepoInformation(r)
	rev := vars["rev"]

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationUpdateRepo, ri.RepoName, permission.Context{
			Ref: rev,
		}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	user, ok := authenticate.GetUserInfo(r.Context())
	if !ok {
		user = authenticate.UserInfo{
			User:  "HuggingFace",
			Email: "hf@users.noreply.huggingface.co",
		}
	}

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
		return
	}

	// Parse NDJSON body
	scanner := bufio.NewScanner(r.Body)
	scanner.Buffer(make([]byte, 1024*1024), 100*1024*1024) // Allow up to 100MB lines

	var header commitHeader
	var ops []repository.CommitOperation

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var op commitOperation
		if err := json.Unmarshal([]byte(line), &op); err != nil {
			responseJSON(w, fmt.Errorf("invalid NDJSON line: %v", err), http.StatusBadRequest)
			return
		}

		switch op.Key {
		case "header":
			if err := json.Unmarshal(op.Value, &header); err != nil {
				responseJSON(w, fmt.Errorf("invalid header: %v", err), http.StatusBadRequest)
				return
			}

		case "file":
			var file commitFile
			if err := json.Unmarshal(op.Value, &file); err != nil {
				responseJSON(w, fmt.Errorf("invalid file operation: %v", err), http.StatusBadRequest)
				return
			}

			content := []byte(file.Content)
			if file.Encoding == "base64" {
				decoded, err := base64.StdEncoding.DecodeString(file.Content)
				if err != nil {
					responseJSON(w, fmt.Errorf("failed to decode base64 content for %s: %v", file.Path, err), http.StatusBadRequest)
					return
				}
				content = decoded
			}

			ops = append(ops, repository.CommitOperation{
				Type:    repository.CommitOperationAdd,
				Path:    file.Path,
				Content: content,
			})

		case "lfsFile":
			var lfsFile commitLFSFile
			if err := json.Unmarshal(op.Value, &lfsFile); err != nil {
				responseJSON(w, fmt.Errorf("invalid LFS file operation: %v", err), http.StatusBadRequest)
				return
			}

			// Create an LFS pointer content
			pointerContent := fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", lfsFile.OID, lfsFile.Size)
			ops = append(ops, repository.CommitOperation{
				Type:    repository.CommitOperationAdd,
				Path:    lfsFile.Path,
				Content: []byte(pointerContent),
			})

		case "deletedFile":
			var deleted commitDeletedFile
			if err := json.Unmarshal(op.Value, &deleted); err != nil {
				responseJSON(w, fmt.Errorf("invalid delete operation: %v", err), http.StatusBadRequest)
				return
			}

			ops = append(ops, repository.CommitOperation{
				Type: repository.CommitOperationDelete,
				Path: deleted.Path,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		responseJSON(w, fmt.Errorf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	message := header.Summary
	if message == "" {
		message = "Upload files"
	}
	if header.Description != "" {
		message += "\n\n" + header.Description
	}

	// Open the repository
	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	// Mock pre-receive hook with current branch head as OldRev
	if h.preReceiveHookFunc != nil {
		oldRev := header.ParentCommit
		if oldRev == "" {
			oldRev, _ = repo.RefHash(plumbing.NewBranchReferenceName(rev))
			if oldRev == "" {
				oldRev = receive.ZeroHash
			}
		}
		if ok, err := h.preReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(oldRev, receive.ZeroHash, "refs/heads/"+rev, ri.RepoName),
		}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "pre-receive hook denied the commit", http.StatusForbidden)
			return
		}
	}

	commitHash, err := repo.CreateCommit(r.Context(), rev, message, user.User, user.Email, ops, header.ParentCommit)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to create commit in repository %q: %v", ri.RepoName, err), http.StatusInternalServerError)
		return
	}

	if h.postReceiveHookFunc != nil {
		oldRev := header.ParentCommit
		if oldRev == "" {
			oldRev = receive.ZeroHash
		}
		if hookErr := h.postReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(oldRev, commitHash, "refs/heads/"+rev, ri.RepoName),
		}); hookErr != nil {
			slog.WarnContext(r.Context(), "post-receive hook error", "repo", ri.RepoName, "error", hookErr)
		}
	}

	resp := commitResponse{
		CommitURL:     fmt.Sprintf("%s/%s/commit/%s", requestOrigin(r), ri.RepoName, commitHash),
		CommitOid:     commitHash,
		CommitMessage: message,
	}
	responseJSON(w, resp, http.StatusOK)
}
