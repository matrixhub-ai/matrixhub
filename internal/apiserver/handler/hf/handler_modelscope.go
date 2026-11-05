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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// registerModelScope shares storage, authorization, proxy synchronization and
// file serving with the HF endpoints. Both clients see the same Git/LFS data.
func (h *Handler) registerModelScope(r *mux.Router) {
	base := "/api/v1/models/{namespace}/{repo}"
	r.HandleFunc(base+"/revisions", h.handleMSRevisions).Methods(http.MethodGet)
	r.HandleFunc(base+"/repo/files", h.handleMSFiles).Methods(http.MethodGet)
	r.HandleFunc(base+"/repo", h.handleMSDownload).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc(base, h.handleMSInfo).Methods(http.MethodGet)
}

func msResponse(w http.ResponseWriter, status int, data any, message string) {
	responseJSON(w, map[string]any{"Code": status, "Success": status < 400, "Data": data, "Message": message}, status)
}

func (h *Handler) openMSRepo(w http.ResponseWriter, r *http.Request) (*repository.Repository, *git.GitRepository, bool) {
	ri := getRepoInformation(r)
	if !isSafePathComponent(ri.Namespace) || !isSafePathComponent(ri.Name) {
		msResponse(w, http.StatusBadRequest, nil, "invalid repository path")
		return nil, nil, false
	}
	if h.permissionHookFunc != nil {
		allowed, err := h.permissionHookFunc(r.Context(), permission.OperationReadRepo, ri.RepoName, permission.Context{})
		if err != nil {
			msResponse(w, http.StatusInternalServerError, nil, "authorization failed")
			return nil, nil, false
		}
		if !allowed {
			msResponse(w, http.StatusForbidden, nil, "permission denied")
			return nil, nil, false
		}
	}
	repoPath := h.storage.ResolvePath(ri.RepoName)
	if repoPath == "" {
		msResponse(w, http.StatusNotFound, nil, "repository not found")
		return nil, nil, false
	}
	var remote *git.GitRepository
	var repo *repository.Repository
	var err error
	if h.modelService != nil {
		remote, err = h.modelService.PrepareModelScopeRead(r.Context(), ri.Namespace, ri.Name)
		if err != nil {
			log.Warnf("failed to prepare ModelScope repository %s: %v", ri.RepoName, err)
			msResponse(w, http.StatusBadGateway, nil, "upstream repository unavailable")
			return nil, nil, false
		}
	}
	if remote == nil {
		repo, err = h.openRepo(r.Context(), ri, repoPath, repository.GitUploadPack)
	} else {
		repo, err = repository.Open(repoPath)
	}
	if err != nil {
		msResponse(w, http.StatusNotFound, nil, "repository unavailable")
		return nil, nil, false
	}
	return repo, remote, true
}

// ModelScope defaults to master, while locally created repositories use main.
// Only the absent master name aliases the actual default; existing branches,
// tags and commit hashes retain their normal Git meaning.
func msRevision(repo *repository.Repository, revision string) (string, error) {
	if revision == "" {
		revision = "HEAD"
	}
	sha, err := repo.ResolveRevision(revision)
	if err != nil && revision == "master" {
		return repo.ResolveRevision("HEAD")
	}
	return sha, err
}

type msRevisionInfo struct {
	Revision  string
	CreatedAt int64
}

func (h *Handler) handleMSRevisions(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := h.openMSRepo(w, r)
	if !ok {
		return
	}
	branches, err := repo.Branches()
	if err != nil {
		msResponse(w, 500, nil, "cannot list branches")
		return
	}
	tags, err := repo.Tags()
	if err != nil {
		msResponse(w, 500, nil, "cannot list tags")
		return
	}
	hasMaster := false
	for _, branch := range branches {
		if branch == "master" {
			hasMaster = true
		}
	}
	if !hasMaster && len(branches) > 0 {
		branches = append(branches, "master")
	}
	convert := func(names []string) []msRevisionInfo {
		result := make([]msRevisionInfo, 0, len(names))
		for _, name := range names {
			sha, err := msRevision(repo, name)
			if err != nil {
				continue
			}
			commits, err := repo.Commits(sha, &repository.CommitsOptions{Limit: 1})
			if err != nil || len(commits) == 0 {
				continue
			}
			result = append(result, msRevisionInfo{Revision: name, CreatedAt: commits[0].Committer().When().Unix()})
		}
		sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt > result[j].CreatedAt })
		return result
	}
	msResponse(w, 200, map[string]any{"RevisionMap": map[string]any{"Branches": convert(branches), "Tags": convert(tags)}}, "")
}

func (h *Handler) handleMSInfo(w http.ResponseWriter, r *http.Request) {
	_, _, ok := h.openMSRepo(w, r)
	if !ok {
		return
	}
	ri := getRepoInformation(r)
	msResponse(w, 200, map[string]any{"Name": ri.Name, "Path": ri.Namespace, "ModelId": ri.FullName}, "")
}

type msFile struct {
	Name, Path, Type, Revision, Sha256 string
	Size                               int64
	IsLFS                              bool
}

func safeMSPath(value string) bool {
	if value == "" {
		return true
	}
	return !strings.ContainsAny(value, "\\\x00") && !strings.HasPrefix(value, "/") && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func (h *Handler) handleMSFiles(w http.ResponseWriter, r *http.Request) {
	root := strings.TrimSuffix(r.URL.Query().Get("Root"), "/")
	if !safeMSPath(root) {
		msResponse(w, 400, nil, "invalid file path")
		return
	}
	repo, _, ok := h.openMSRepo(w, r)
	if !ok {
		return
	}
	sha, err := msRevision(repo, r.URL.Query().Get("Revision"))
	if err != nil {
		msResponse(w, 404, nil, "revision not found")
		return
	}
	recursive, _ := strconv.ParseBool(r.URL.Query().Get("Recursive"))
	entries, err := repo.Tree(sha, root, &repository.TreeOptions{Recursive: recursive})
	if err != nil {
		msResponse(w, 404, nil, "file tree not found")
		return
	}
	files := make([]msFile, 0, len(entries))
	for _, entry := range entries {
		if err := r.Context().Err(); err != nil {
			return
		}
		file := msFile{Name: path.Base(entry.Path()), Path: entry.Path(), Type: "tree", Revision: sha}
		if entry.Type() == repository.EntryTypeFile {
			file.Type = "blob"
			blob, err := entry.Blob()
			if err != nil {
				msResponse(w, 500, nil, "cannot read file metadata")
				return
			}
			file.Size = blob.Size()
			if pointer, _ := blob.LFSPointer(); pointer != nil {
				file.Size, file.Sha256, file.IsLFS = pointer.Size(), pointer.OID(), true
			} else {
				reader, err := blob.NewReader()
				if err != nil {
					msResponse(w, 500, nil, "cannot read file")
					return
				}
				hash := sha256.New()
				_, err = io.Copy(hash, reader)
				_ = reader.Close()
				if err != nil {
					msResponse(w, 500, nil, "cannot hash file")
					return
				}
				file.Sha256 = hex.EncodeToString(hash.Sum(nil))
			}
		}
		files = append(files, file)
	}
	msResponse(w, 200, map[string]any{"Files": files}, "")
}

func (h *Handler) handleMSDownload(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("FilePath")
	if filePath == "" || !safeMSPath(filePath) {
		msResponse(w, 400, nil, "invalid file path")
		return
	}
	repo, remote, ok := h.openMSRepo(w, r)
	if !ok {
		return
	}
	sha, err := msRevision(repo, r.URL.Query().Get("Revision"))
	if err != nil {
		msResponse(w, 404, nil, "revision not found")
		return
	}
	if remote != nil {
		blob, err := repo.Blob(sha, filePath)
		if err != nil {
			msResponse(w, http.StatusNotFound, nil, "file not found")
			return
		}
		if pointer, _ := blob.LFSPointer(); pointer != nil {
			info, err := h.lfsStorage.Info(pointer.OID())
			if err != nil || info.Size() != pointer.Size() {
				h.serveUncachedModelScopeFile(w, r, remote, sha, filePath, pointer)
				return
			}
		}
	}
	h.serveResolvedFile(w, r, repo, getRepoInformation(r), sha, filePath)
}

func (h *Handler) serveUncachedModelScopeFile(w http.ResponseWriter, r *http.Request, remote *git.GitRepository, revision, filePath string, pointer *lfs.Pointer) {
	size := pointer.Size()
	if r.Method == http.MethodHead {
		writeModelScopeFileHeaders(w, revision, filePath, pointer)
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		return
	}

	byteRange := r.Header.Get("Range")
	if match := r.Header.Get("If-Range"); match != "" && match != fmt.Sprintf("%q", pointer.OID()) {
		byteRange = ""
	}
	upstream, err := h.gitRepo.OpenModelScopeFile(r.Context(), remote, revision, filePath, byteRange)
	if err != nil {
		log.Warnf("failed to open ModelScope file %s: %v", filePath, err)
		msResponse(w, http.StatusBadGateway, nil, "upstream file unavailable")
		return
	}
	defer func() { _ = upstream.Body.Close() }()

	length := size
	status := http.StatusOK
	if upstream.Partial {
		var start, end, total int64
		n, err := fmt.Sscanf(upstream.ContentRange, "bytes %d-%d/%d", &start, &end, &total)
		if err != nil || n != 3 || start < 0 || end < start || end >= size || total != size {
			msResponse(w, http.StatusBadGateway, nil, "invalid upstream file range")
			return
		}
		length = end - start + 1
		status = http.StatusPartialContent
	}
	if upstream.ContentLength >= 0 && upstream.ContentLength != length {
		msResponse(w, http.StatusBadGateway, nil, "upstream file size differs from Git metadata")
		return
	}

	writeModelScopeFileHeaders(w, revision, filePath, pointer)
	if upstream.Partial {
		w.Header().Set("Content-Range", upstream.ContentRange)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.WriteHeader(status)
	if upstream.Partial {
		_, err = io.CopyN(w, upstream.Body, length)
	} else {
		// Stream to the client while the LFS store verifies size and SHA256.
		// Partial reads and interrupted transfers are never published in the cache.
		err = h.lfsStorage.Put(pointer.OID(), io.TeeReader(io.LimitReader(upstream.Body, size+1), w), size)
	}
	if err != nil && r.Context().Err() == nil {
		log.Warnf("ModelScope stream failed for %s: %v", filePath, err)
	}
}

func writeModelScopeFileHeaders(w http.ResponseWriter, revision, filePath string, pointer *lfs.Pointer) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": path.Base(filePath)}))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("ETag", fmt.Sprintf("%q", pointer.OID()))
	w.Header().Set("X-Repo-Commit", revision)
}
