package hf

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	"github.com/matrixhub-ai/hfd/pkg/repository"
)

// handleInfoRevision handles the /api/{repoType}/{repo_id}/revision/{rev} and /api/{repoType}/{repo_id} endpoint
func (h *Handler) handleInfoRevision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	ri := getRepoInformation(r)
	rev := vars["rev"]

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationReadRepo, ri.RepoName, permission.Context{Ref: rev}); err != nil {
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

	if rev == "" {
		rev = repo.DefaultBranch()
	}

	// Get list of files in the repository at the specified revision (recursive to include files in subdirectories)
	// An empty repository (no commits yet) is a valid state; treat it as having no files.
	hfEntries, err := repo.Tree(rev, "", &repository.TreeOptions{Recursive: true})
	if err != nil && !errors.Is(err, repository.ErrRevisionNotFound) {
		responseJSON(w, fmt.Errorf("failed to get tree for repo %q at rev %q: %v", ri.RepoName, rev, err), http.StatusInternalServerError)
		return
	}

	var siblings []sibling
	for _, entry := range hfEntries {
		if entry.Type() == repository.EntryTypeFile {
			siblings = append(siblings, sibling{
				RFilename: entry.Path(),
			})
		}
	}

	usedStorage, _ := repo.DiskUsage()

	// Get the commit SHA for this revision
	commitHash := ""
	commits, err := repo.Commits(rev, &repository.CommitsOptions{Limit: 1})
	if err == nil && len(commits) > 0 {
		commitHash = commits[0].Hash().String()
	}

	// Collect metadata (tags, cardData, pipeline_tag, etc.) from README.md and config.json.
	meta := collectRepoMetadata(repo, rev)

	tags := meta.tags
	if tags == nil {
		tags = []string{}
	}

	hfInfo := repoInfo{
		ID:          ri.FullName,
		SHA:         commitHash,
		Private:     false,
		Disabled:    false,
		Gated:       false,
		Downloads:   0,
		Likes:       0,
		Tags:        tags,
		Siblings:    siblings,
		CardData:    meta.cardData,
		UsedStorage: usedStorage,
	}

	// For models, also set the modelId field which is required by some HuggingFace clients. For datasets and spaces, the client doesn't require it and it can be confusing to have it be different from the ID, so we leave it empty.
	if ri.RepoType == "models" {
		hfInfo.ModelID = hfInfo.ID
	}

	responseJSON(w, hfInfo, http.StatusOK)
}

// handleDeleteRepo handles DELETE /api/repos/delete
func (h *Handler) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	var req deleteRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	repoName := req.Name
	if req.Organization != "" {
		repoName = req.Organization + "/" + repoName
	}

	prefix := repoTypePrefix(req.Type)
	storageName := repoName
	if prefix != "" {
		storageName = prefix + "/" + repoName
	}

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationDeleteRepo, storageName, permission.Context{}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	repoPath := h.storage.ResolvePath(storageName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
		return
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", repoName), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	if err := repo.Remove(); err != nil {
		responseJSON(w, fmt.Errorf("failed to delete repository %q: %v", repoName, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleMoveRepo handles POST /api/repos/move
func (h *Handler) handleMoveRepo(w http.ResponseWriter, r *http.Request) {
	var req moveRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	prefix := repoTypePrefix(req.Type)

	fromName := req.FromRepo
	if prefix != "" {
		fromName = prefix + "/" + fromName
	}
	toName := req.ToRepo
	if prefix != "" {
		toName = prefix + "/" + toName
	}

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationUpdateRepo, fromName, permission.Context{DestRepo: toName}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	fromPath := h.storage.ResolvePath(fromName)
	if fromPath == "" {
		responseJSON(w, fmt.Errorf("invalid source repository: %q", req.FromRepo), http.StatusBadRequest)
		return
	}

	toPath := h.storage.ResolvePath(toName)
	if toPath == "" {
		responseJSON(w, fmt.Errorf("invalid destination repository: %q", req.ToRepo), http.StatusBadRequest)
		return
	}

	repo, err := repository.Open(fromPath)
	if err != nil {
		if errors.Is(err, repository.ErrRepositoryNotExists) {
			responseJSON(w, fmt.Errorf("repository %q not found", req.FromRepo), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to open repository %q: %v", req.FromRepo, err), http.StatusInternalServerError)
		return
	}

	// Check that destination doesn't already exist
	if repository.IsRepository(toPath) {
		responseJSON(w, fmt.Errorf("destination repository %q already exists", req.ToRepo), http.StatusConflict)
		return
	}

	if err := repo.Move(toPath); err != nil {
		responseJSON(w, fmt.Errorf("failed to move repository: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleRepoSettings handles PUT /api/{repoType}/{repo}/settings
func (h *Handler) handleRepoSettings(w http.ResponseWriter, r *http.Request) {
	ri := getRepoInformation(r)

	if h.permissionHookFunc != nil {
		if ok, err := h.permissionHookFunc(r.Context(), permission.OperationUpdateRepo, ri.RepoName, permission.Context{}); err != nil {
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

	if !repository.IsRepository(repoPath) {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
		return
	}

	// Accept the settings payload but don't enforce private/gated in this server
	var req repoSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleCreateBranch handles POST /api/{repoType}/{repo}/branch/{rev}
func (h *Handler) handleCreateBranch(w http.ResponseWriter, r *http.Request) {
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

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
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

	// Check if branch already exists
	exists, err := repo.BranchExists(rev)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to check branch %q: %v", rev, err), http.StatusInternalServerError)
		return
	}
	if exists {
		responseJSON(w, fmt.Errorf("branch %q already exists", rev), http.StatusConflict)
		return
	}

	var req createBranchRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}

	if h.preReceiveHookFunc != nil {
		// Resolve the starting point to a hash so the hook has the target commit
		newRev, _ := repo.ResolveRevision(req.StartingPoint)
		if newRev == "" {
			newRev, _ = repo.RefHash(plumbing.NewBranchReferenceName(repo.DefaultBranch()))
		}
		if ok, err := h.preReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(receive.ZeroHash, newRev, "refs/heads/"+rev, repo.RepoPath()),
		}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "pre-receive hook denied the branch creation", http.StatusForbidden)
			return
		}
	}

	revision := req.StartingPoint
	if err := repo.CreateBranch(rev, revision); err != nil {
		responseJSON(w, fmt.Errorf("failed to create branch %q: %v", rev, err), http.StatusInternalServerError)
		return
	}

	if h.postReceiveHookFunc != nil {
		hash, _ := repo.RefHash(plumbing.NewBranchReferenceName(rev))
		if hookErr := h.postReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(receive.ZeroHash, hash, "refs/heads/"+rev, repo.RepoPath()),
		}); hookErr != nil {
			slog.WarnContext(r.Context(), "post-receive hook error", "repo", ri.RepoName, "error", hookErr)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteBranch handles DELETE /api/{repoType}/{repo}/branch/{rev}
func (h *Handler) handleDeleteBranch(w http.ResponseWriter, r *http.Request) {
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

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
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

	// Prevent deleting the default branch
	if rev == repo.DefaultBranch() {
		responseJSON(w, fmt.Errorf("cannot delete default branch %q", rev), http.StatusForbidden)
		return
	}

	exists, err := repo.BranchExists(rev)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to check branch %q: %v", rev, err), http.StatusInternalServerError)
		return
	}
	if !exists {
		responseJSON(w, fmt.Errorf("branch %q not found", rev), http.StatusNotFound)
		return
	}

	// Capture hash before deletion for pre/post hooks
	oldHash, _ := repo.RefHash(plumbing.NewBranchReferenceName(rev))

	updates := []receive.RefUpdate{
		receive.NewRefUpdate(oldHash, receive.ZeroHash, "refs/heads/"+rev, repo.RepoPath()),
	}

	if h.preReceiveHookFunc != nil {
		if ok, err := h.preReceiveHookFunc(r.Context(), ri.RepoName, updates); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "pre-receive hook denied the branch deletion", http.StatusForbidden)
			return
		}
	}

	if err := repo.DeleteBranch(rev); err != nil {
		responseJSON(w, fmt.Errorf("failed to delete branch %q: %v", rev, err), http.StatusInternalServerError)
		return
	}

	if h.postReceiveHookFunc != nil {
		if hookErr := h.postReceiveHookFunc(r.Context(), ri.RepoName, updates); hookErr != nil {
			slog.WarnContext(r.Context(), "post-receive hook error", "repo", ri.RepoName, "error", hookErr)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleCreateTag handles POST /api/{repoType}/{repo}/tag/{rev}
func (h *Handler) handleCreateTag(w http.ResponseWriter, r *http.Request) {
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

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
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

	var req createTagRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}

	if req.Tag == "" {
		responseJSON(w, fmt.Errorf("tag name is required"), http.StatusBadRequest)
		return
	}

	// Check if tag already exists
	exists, err := repo.TagExists(req.Tag)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to check tag %q: %v", req.Tag, err), http.StatusInternalServerError)
		return
	}
	if exists {
		responseJSON(w, fmt.Errorf("tag %q already exists", req.Tag), http.StatusConflict)
		return
	}

	if h.preReceiveHookFunc != nil {
		// Resolve the revision to a hash so the hook has the target commit
		newRev, _ := repo.ResolveRevision(rev)
		if ok, err := h.preReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(receive.ZeroHash, newRev, "refs/tags/"+req.Tag, repo.RepoPath()),
		}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "pre-receive hook denied the tag creation", http.StatusForbidden)
			return
		}
	}

	if err := repo.CreateTag(req.Tag, rev); err != nil {
		responseJSON(w, fmt.Errorf("failed to create tag %q: %v", req.Tag, err), http.StatusInternalServerError)
		return
	}

	if h.postReceiveHookFunc != nil {
		hash, _ := repo.RefHash(plumbing.NewTagReferenceName(req.Tag))
		if hookErr := h.postReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(receive.ZeroHash, hash, "refs/tags/"+req.Tag, repo.RepoPath()),
		}); hookErr != nil {
			slog.WarnContext(r.Context(), "post-receive hook error", "repo", ri.RepoName, "error", hookErr)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleDeleteTag handles DELETE /api/{repoType}/{repo}/tag/{rev}
func (h *Handler) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
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

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
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

	exists, err := repo.TagExists(rev)
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to check tag %q: %v", rev, err), http.StatusInternalServerError)
		return
	}
	if !exists {
		responseJSON(w, fmt.Errorf("tag %q not found", rev), http.StatusNotFound)
		return
	}

	// Capture hash before deletion for pre/post hooks
	oldHash, _ := repo.RefHash(plumbing.NewTagReferenceName(rev))

	updates := []receive.RefUpdate{
		receive.NewRefUpdate(oldHash, receive.ZeroHash, "refs/tags/"+rev, repo.RepoPath()),
	}

	if h.preReceiveHookFunc != nil {
		if ok, err := h.preReceiveHookFunc(r.Context(), ri.RepoName, updates); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "pre-receive hook denied the tag deletion", http.StatusForbidden)
			return
		}
	}

	if err := repo.DeleteTag(rev); err != nil {
		responseJSON(w, fmt.Errorf("failed to delete tag %q: %v", rev, err), http.StatusInternalServerError)
		return
	}

	if h.postReceiveHookFunc != nil {
		if hookErr := h.postReceiveHookFunc(r.Context(), ri.RepoName, updates); hookErr != nil {
			slog.WarnContext(r.Context(), "post-receive hook error", "repo", ri.RepoName, "error", hookErr)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleListRefs handles GET /api/{repoType}/{repo}/refs
func (h *Handler) handleListRefs(w http.ResponseWriter, r *http.Request) {
	ri := getRepoInformation(r)

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

	// List branches
	branchNames, err := repo.Branches()
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to list branches: %v", err), http.StatusInternalServerError)
		return
	}

	var branches []gitRefInfo
	for _, name := range branchNames {
		refName := plumbing.NewBranchReferenceName(name)
		hash, err := repo.RefHash(refName)
		if err != nil {
			continue
		}
		branches = append(branches, gitRefInfo{
			Name:         name,
			Ref:          refName.String(),
			TargetCommit: hash,
		})
	}

	// List tags
	tagNames, err := repo.Tags()
	if err != nil {
		tagNames = nil
	}

	var tags []gitRefInfo
	for _, name := range tagNames {
		refName := plumbing.NewTagReferenceName(name)
		hash, err := repo.RefHash(refName)
		if err != nil {
			continue
		}
		tags = append(tags, gitRefInfo{
			Name:         name,
			Ref:          refName.String(),
			TargetCommit: hash,
		})
	}

	if branches == nil {
		branches = []gitRefInfo{}
	}
	if tags == nil {
		tags = []gitRefInfo{}
	}

	refs := gitRefs{
		Branches: branches,
		Converts: []gitRefInfo{},
		Tags:     tags,
	}
	responseJSON(w, refs, http.StatusOK)
}

// handleListCommits handles GET /api/{repoType}/{repo}/commits/{rev}
// It returns a paginated list of commits for the given revision.
func (h *Handler) handleListCommits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ri := getRepoInformation(r)
	rev := vars["rev"]

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

	query := r.URL.Query()

	limit := 50
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	var page int
	var offset int
	if pageStr := query.Get("p"); pageStr != "" {
		if page, err = strconv.Atoi(pageStr); err == nil && page >= 0 {
			offset = page * limit
		}
	}

	// Fetch one extra commit so we can detect whether a next page exists.
	rawCommits, err := repo.Commits(rev, &repository.CommitsOptions{Limit: limit + 1, Offset: offset})
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to list commits for %q: %v", rev, err), http.StatusInternalServerError)
		return
	}

	if len(rawCommits) > limit {
		rawCommits = rawCommits[:limit]
		nextURL := buildNextCommitPageURL(r, page+1, limit)
		w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL))
	}

	commitInfos := make([]commitInfo, 0, len(rawCommits))
	for _, c := range rawCommits {
		commitInfos = append(commitInfos, commitInfo{
			ID:      c.Hash().String(),
			Title:   c.Title(),
			Message: c.Message(),
			Authors: []commitAuthor{{User: c.Author().Name()}},
			Date:    c.Author().When().UTC().Format(repository.TimeFormat),
		})
	}

	responseJSON(w, commitInfos, http.StatusOK)
}

// buildNextCommitPageURL constructs the URL for the next commits page,
// replacing the p parameter with the given page number.
func buildNextCommitPageURL(r *http.Request, nextPage, limit int) string {
	origin := requestOrigin(r)
	q := r.URL.Query()
	q.Set("p", strconv.Itoa(nextPage))
	q.Set("limit", strconv.Itoa(limit))
	return origin + r.URL.Path + "?" + q.Encode()
}

// handleCompare handles GET /api/{repoType}/{repo}/compare/{compare}
func (h *Handler) handleCompare(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ri := getRepoInformation(r)
	compare := vars["compare"]

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

	base, head, found := strings.Cut(compare, "..")
	if !found || base == "" || head == "" ||
		strings.HasPrefix(head, ".") || strings.Contains(head, "..") {
		responseJSON(w, fmt.Errorf("invalid compare format %q, expected base..head", compare), http.StatusBadRequest)
		return
	}

	changes, err := repo.Compare(r.Context(), base, head)
	if err != nil {
		if errors.Is(err, repository.ErrRevisionNotFound) || errors.Is(err, plumbing.ErrReferenceNotFound) {
			responseJSON(w, fmt.Errorf("failed to resolve compare revisions %q: %v", compare, err), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Errorf("failed to compare %q: %v", compare, err), http.StatusInternalServerError)
		return
	}

	patch, err := changes.PatchContext(r.Context())
	if err != nil {
		responseJSON(w, fmt.Errorf("failed to generate patch for compare %q: %v", compare, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = patch.Encode(w)
}

// handleSuperSquash handles POST /api/{repoType}/{repo}/super-squash/{rev}
// It squashes all commits in the current rev into a single commit with the given message.
// The action is irreversible.
func (h *Handler) handleSuperSquash(w http.ResponseWriter, r *http.Request) {
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

	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		responseJSON(w, fmt.Errorf("repository %q not found", ri.RepoName), http.StatusNotFound)
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

	var req superSquashRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responseJSON(w, fmt.Errorf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}

	if h.preReceiveHookFunc != nil {
		if ok, err := h.preReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(receive.BreakHash, receive.BreakHash, "refs/heads/"+rev, repo.RepoPath()),
		}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "pre-receive hook denied the super-squash operation", http.StatusForbidden)
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

	message := req.Message
	if message == "" {
		message = "Super-squash branch '" + rev + "'"
	}

	if _, err := repo.SuperSquash(r.Context(), rev, message, user.User, user.Email); err != nil {
		responseJSON(w, fmt.Errorf("failed to squash repository %q rev %q: %v", ri.RepoName, rev, err), http.StatusInternalServerError)
		return
	}

	if h.postReceiveHookFunc != nil {
		if hookErr := h.postReceiveHookFunc(r.Context(), ri.RepoName, []receive.RefUpdate{
			receive.NewRefUpdate(receive.BreakHash, receive.BreakHash, "refs/heads/"+rev, repo.RepoPath()),
		}); hookErr != nil {
			slog.WarnContext(r.Context(), "post-receive hook error", "repo", ri.RepoName, "error", hookErr)
		}
	}

	w.WriteHeader(http.StatusOK)
}
