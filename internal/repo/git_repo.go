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

package repo

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	stdpath "path"
	"strings"
	"time"

	hfdlfs "github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

type gitRepo struct {
	storage *storage.Storage
	mirror  *mirror.Mirror
}

const maxSafetensorsHeaderBytes uint64 = 64 * 1024 * 1024

// safetensorsHeaderReadBudget bounds the time one metadata extraction may spend
// reading safetensors headers, across all files rather than per file. A read
// served by the mirror tee cache blocks until the upstream download reaches the
// header bytes, so extraction must not wait on it indefinitely: it runs inline on
// the proxy sync path that git and Hugging Face clients wait for. A shared budget
// also keeps a sharded model from multiplying the wait by its shard count.
const safetensorsHeaderReadBudget = 10 * time.Second

type safetensorsIndexFiles struct {
	Metadata struct {
		TotalSize int64 `json:"total_size"`
	} `json:"metadata"`
	WeightMap map[string]string `json:"weight_map"`
}

// NewGitDB creates a new GitRepo instance
func NewGitDB(storage *storage.Storage, mirror *mirror.Mirror) git.IGitRepo {
	return &gitRepo{
		storage: storage,
		mirror:  mirror,
	}
}

func repoPrefix(repoType string) string {
	switch repoType {
	case "model", "models":
		return ""
	case "dataset", "datasets":
		return "datasets/"
	case "space", "spaces":
		return "spaces/"
	default:
		return ""
	}
}

func (g *gitRepo) gitPath(repoType string, project, name string) string {
	repoName := repoPrefix(repoType) + project + "/" + name
	repoPath := g.storage.ResolvePath(repoName)
	return repoPath

}

func (g *gitRepo) buildURL(repoType, project, name, revision, path string) string {
	if revision == "" {
		revision = "main"
	} else if strings.Contains(revision, "/") {
		// If revision is a ref like refs/heads/main or refs/tags/v1, keep only the last part
		revision = stdpath.Base(revision)
	}

	return fmt.Sprintf("/%s/%s/resolve/%s/%s", repoPrefix(repoType)+project, name, revision, path)
}

// resolveRef disambiguates a short revision name by trying refs/heads/ before refs/tags/.
// Already-qualified refs (refs/heads/..., refs/tags/...) and 40-char SHAs pass through unchanged.
func resolveRef(repo *repository.Repository, rev string) string {
	if rev == "" || strings.HasPrefix(rev, "refs/") || isCommitSHA(rev) {
		return rev
	}
	if _, err := repo.ResolveRevision("refs/heads/" + rev); err == nil {
		return "refs/heads/" + rev
	}
	if _, err := repo.ResolveRevision("refs/tags/" + rev); err == nil {
		return "refs/tags/" + rev
	}
	return rev
}

// convertCommitOperations converts domain CommitOperation slice to repository CommitOperation slice.
func convertCommitOperations(ops []git.CommitOperation) []repository.CommitOperation {
	result := make([]repository.CommitOperation, len(ops))
	for i, op := range ops {
		var typ repository.CommitOperationType
		switch op.Type {
		case git.CommitOperationAdd:
			typ = repository.CommitOperationAdd
		case git.CommitOperationDelete:
			typ = repository.CommitOperationDelete
		}
		result[i] = repository.CommitOperation{
			Type:    typ,
			Path:    op.Path,
			Content: op.Content,
		}
	}
	return result
}

func isCommitSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// CreateRepository initializes a Git repository
func (g *gitRepo) CreateRepository(ctx context.Context, repoType, project, name string) error {
	gitPath := g.gitPath(repoType, project, name)
	if repository.IsRepository(gitPath) {
		return fmt.Errorf("repository already exists at %s", gitPath)
	}

	defaultBranch := "main"
	repo, err := repository.Init(ctx, gitPath, defaultBranch)
	if err != nil {
		return err
	}

	// TODO(@scyda): use actual user info from context when available
	user := "HuggingFace"
	email := "hf@users.noreply.huggingface.co"

	// Create initial commit with default .gitattributes
	_, err = repo.CreateCommit(ctx, defaultBranch, "Initial commit", user, email, []repository.CommitOperation{
		{
			Type:    repository.CommitOperationAdd,
			Path:    repository.GitattributesFileName,
			Content: repository.GitattributesText,
		},
	}, "")
	if err != nil {
		_ = repo.Remove()
		return err
	}

	return nil
}

// RepositoryExists checks whether a Git repository exists on disk.
func (g *gitRepo) RepositoryExists(ctx context.Context, repoType, project, name string) (bool, error) {
	return repository.IsRepository(g.gitPath(repoType, project, name)), nil
}

// DeleteRepository removes the Git repository
func (g *gitRepo) DeleteRepository(ctx context.Context, repoType, project, name string) error {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return err
	}
	return repo.Remove()
}

// ListRevisions returns all branches and tags for a model
func (g *gitRepo) ListRevisions(ctx context.Context, repoType, project, name string) (*git.Revisions, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return nil, fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, err
	}

	revisions := &git.Revisions{
		Branches: []*git.Revision{},
		Tags:     []*git.Revision{},
	}
	branches, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	for _, branch := range branches {
		revisions.Branches = append(revisions.Branches, &git.Revision{
			Name: branch,
		})
	}

	tags, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		revisions.Tags = append(revisions.Tags, &git.Revision{
			Name: tag,
		})
	}

	return revisions, nil
}

// ListCommits returns the commit history for a model
func (g *gitRepo) ListCommits(ctx context.Context, repoType, project, name, revision string, page, pageSize int) ([]*git.Commit, int64, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return nil, 0, fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, 0, err
	}

	revision = resolveRef(repo, revision)
	commits, err := repo.Commits(revision, nil)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(commits))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(commits) {
		return []*git.Commit{}, total, nil
	}
	if end > len(commits) {
		end = len(commits)
	}
	commits = commits[start:end]

	var gitCommits []*git.Commit
	for _, c := range commits {
		gitCommits = append(gitCommits, &git.Commit{
			ID:             c.Hash().String(),
			Message:        c.Message(),
			AuthorName:     c.Author().Name(),
			AuthorEmail:    c.Author().Email(),
			AuthorDate:     c.Author().When(),
			CommitterName:  c.Committer().Name(),
			CommitterEmail: c.Committer().Email(),
			CommitterDate:  c.Committer().When(),
			CreatedAt:      c.Committer().When(),
		})
	}

	return gitCommits, total, nil
}

// GetCommit returns a specific commit by ID
func (g *gitRepo) GetCommit(ctx context.Context, repoType, project, name, commitID string) (*git.Commit, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return nil, fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, err
	}

	commits, err := repo.Commits(commitID, &repository.CommitsOptions{
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("commit not found: %s", commitID)
	}
	c := commits[0]

	diff, _ := c.Diff()

	return &git.Commit{
		ID:             c.Hash().String(),
		Message:        c.Message(),
		Diff:           diff,
		AuthorName:     c.Author().Name(),
		AuthorEmail:    c.Author().Email(),
		AuthorDate:     c.Author().When(),
		CommitterName:  c.Committer().Name(),
		CommitterEmail: c.Committer().Email(),
		CommitterDate:  c.Committer().When(),
		CreatedAt:      c.Committer().When(),
	}, nil
}

func (g *gitRepo) CreateCommit(ctx context.Context, repoType, project, name, revision string, commit *git.Commit, ops []git.CommitOperation) (string, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return "", fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return "", err
	}
	commitHash, err := repo.CreateCommit(ctx, revision, commit.Message, commit.AuthorName, commit.AuthorEmail, convertCommitOperations(ops), commit.ParentCommit)
	if err != nil {
		return "", err
	}
	return commitHash, nil
}

// GetTree returns the file tree at a specific revision and path
func (g *gitRepo) GetTree(ctx context.Context, repoType, project, name, revision, path string) ([]*git.TreeEntry, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return nil, fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, err
	}

	entries, err := repo.Tree(resolveRef(repo, revision), path, &repository.TreeOptions{
		Recursive: false,
	})
	if err != nil {
		return nil, err
	}

	var treeEntries []*git.TreeEntry
	for _, e := range entries {

		if e.Type() == repository.EntryTypeFile {
			blob, err := e.Blob()
			if err != nil {
				return nil, err
			}
			size := blob.Size()
			lfsPointer, _ := blob.LFSPointer()
			if lfsPointer != nil {
				size = lfsPointer.Size()
			}

			lastCommit := e.LastCommit()

			treeEntries = append(treeEntries, &git.TreeEntry{
				Name:  blob.Name(),
				Path:  e.Path(),
				Hash:  blob.Hash().String(),
				Type:  git.FileTypeFile,
				Size:  size,
				IsLFS: lfsPointer != nil,
				URL:   g.buildURL(repoType, project, name, revision, e.Path()),
				Commit: &git.Commit{
					ID:             lastCommit.Hash().String(),
					Message:        lastCommit.Message(),
					AuthorName:     lastCommit.Author().Name(),
					AuthorEmail:    lastCommit.Author().Email(),
					AuthorDate:     lastCommit.Author().When(),
					CommitterName:  lastCommit.Committer().Name(),
					CommitterEmail: lastCommit.Committer().Email(),
					CommitterDate:  lastCommit.Committer().When(),
					CreatedAt:      lastCommit.Committer().When(),
				},
			})
		} else {
			lastCommit := e.LastCommit()
			treeEntries = append(treeEntries, &git.TreeEntry{
				Name: stdpath.Base(e.Path()),
				Path: e.Path(),
				Type: git.FileTypeDir,
				Commit: &git.Commit{
					ID:             lastCommit.Hash().String(),
					Message:        lastCommit.Message(),
					AuthorName:     lastCommit.Author().Name(),
					AuthorEmail:    lastCommit.Author().Email(),
					AuthorDate:     lastCommit.Author().When(),
					CommitterName:  lastCommit.Committer().Name(),
					CommitterEmail: lastCommit.Committer().Email(),
					CommitterDate:  lastCommit.Committer().When(),
					CreatedAt:      lastCommit.Committer().When(),
				},
			})
		}
	}

	return treeEntries, nil
}

// GetBlob returns the content of a file at a specific revision
func (g *gitRepo) GetBlob(ctx context.Context, repoType, project, name, revision, path string) (*git.TreeEntry, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return nil, fmt.Errorf("repository does not exist at %s", gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, err
	}

	revision = resolveRef(repo, revision)
	blob, err := repo.Blob(revision, path)
	if err != nil {
		// Check if it's a directory
		entries, err := repo.Tree(revision, path, &repository.TreeOptions{
			Recursive: false,
		})
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("file or directory not found at %s", path)
		}
		lastCommit := entries[0].LastCommit()
		return &git.TreeEntry{
			Name: stdpath.Base(path),
			Path: path,
			Type: git.FileTypeDir,
			Commit: &git.Commit{
				ID:             lastCommit.Hash().String(),
				Message:        lastCommit.Message(),
				AuthorName:     lastCommit.Author().Name(),
				AuthorEmail:    lastCommit.Author().Email(),
				AuthorDate:     lastCommit.Author().When(),
				CommitterName:  lastCommit.Committer().Name(),
				CommitterEmail: lastCommit.Committer().Email(),
				CommitterDate:  lastCommit.Committer().When(),
				CreatedAt:      lastCommit.Committer().When(),
			},
		}, nil
	}

	lastCommit, err := repo.Commits(revision, &repository.CommitsOptions{
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	commit := lastCommit[0]

	size := blob.Size()
	lfsPointer, _ := blob.LFSPointer()
	if lfsPointer != nil {
		size = lfsPointer.Size()
	}
	return &git.TreeEntry{
		Name:  blob.Name(),
		Path:  path,
		Hash:  blob.Hash().String(),
		Type:  git.FileTypeFile,
		Size:  size,
		IsLFS: lfsPointer != nil,
		URL:   g.buildURL(repoType, project, name, revision, path),
		Commit: &git.Commit{
			ID:             commit.Hash().String(),
			Message:        commit.Message(),
			AuthorName:     commit.Author().Name(),
			AuthorEmail:    commit.Author().Email(),
			AuthorDate:     commit.Author().When(),
			CommitterName:  commit.Committer().Name(),
			CommitterEmail: commit.Committer().Email(),
			CommitterDate:  commit.Committer().When(),
			CreatedAt:      commit.Committer().When(),
		},
	}, nil
}

func (g *gitRepo) PullFromRemote(ctx context.Context, gitRepository *git.GitRepository) error {
	gitPath := g.gitPath(gitRepository.ResourceType, gitRepository.ProjectName, gitRepository.ResourceName)
	repoName := repoPrefix(gitRepository.ResourceType) + gitRepository.RemoteProjectName + "/" + gitRepository.RemoteResourceName
	sourceURL := strings.TrimSuffix(gitRepository.RemoteRegistryURL, "/") + "/" + repoName
	if !repository.IsRepository(gitPath) {
		_, err := repository.InitMirror(ctx, gitPath, sourceURL)
		if err != nil {
			return err
		}
	}

	logWriter := gitRepository.LogWriter
	if logWriter == nil {
		logWriter = os.Stderr
	}

	syncOptions := []mirror.SyncOption{
		mirror.WithSyncMirrorSourceURL(sourceURL),
		mirror.WithSyncOutput(logWriter),
	}

	if cred := gitRepository.Credential; cred != nil {
		syncOptions = append(syncOptions,
			mirror.WithSyncUserInfo(url.UserPassword(cred.Username, cred.Password)),
		)
	}
	return g.mirror.PullFromRemote(ctx, gitPath, repoName,
		syncOptions...,
	)
}

func (g *gitRepo) readBlobBytes(repo *repository.Repository, rev, path string) ([]byte, error) {
	blob, err := repo.Blob(rev, path)
	if err != nil {
		return nil, err
	}
	rc, err := blob.NewReader()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rc.Close()
	}()
	return io.ReadAll(rc)
}

func (g *gitRepo) openBlobContent(repo *repository.Repository, rev, path string) (io.ReadCloser, error) {
	blob, err := repo.Blob(rev, path)
	if err != nil {
		return nil, err
	}

	ptr, err := blob.LFSPointer()
	if err != nil || ptr == nil {
		return blob.NewReader()
	}

	store := hfdlfs.NewLocal(g.storage.LFSDir())
	if getter, ok := store.(hfdlfs.Getter); ok {
		if rc, _, err := getter.Get(ptr.OID()); err == nil {
			return rc, nil
		}
	}

	// The object is not on disk yet. On proxy repositories PullFromRemote only
	// queues LFS objects for background fetching, so fall back to the mirror tee
	// cache, which serves in-flight content while the download is still running.
	// Callers that only need a file prefix, such as a safetensors header, do not
	// have to wait for the whole object.
	if g.mirror != nil {
		if b := g.mirror.Get(ptr.OID()); b != nil {
			return b.NewReadSeeker(), nil
		}
	}

	return nil, fmt.Errorf("lfs object %s is not available locally", ptr.OID())
}

// collectSafetensorsFile records the header of a safetensors file, falling back
// to the size recorded in its LFS pointer when the header cannot be read. The
// fallback keeps a parameter count derivable for proxy repositories whose
// weights have not been fetched yet.
//
// Once the read budget is spent the header read is skipped entirely: opening a
// tee cache blob promotes it to a foreground download, which is not worth paying
// for a read that is about to time out anyway.
func (g *gitRepo) collectSafetensorsFile(ctx context.Context, repo *repository.Repository, rev, path string, metadata *git.RepoMetadataFiles) {
	if ctx.Err() == nil {
		if header, err := g.readSafetensorsHeader(ctx, repo, rev, path); err == nil {
			metadata.SafetensorsFiles[path] = header
			return
		}
	}
	if size, ok := lfsPointerSize(repo, rev, path); ok {
		metadata.SafetensorsSizes[path] = size
	}
}

// lfsPointerSize returns the object size recorded in the file's LFS pointer.
// The pointer is a regular git blob, so this stays cheap even when the object
// content itself has not been fetched.
func lfsPointerSize(repo *repository.Repository, rev, path string) (int64, bool) {
	blob, err := repo.Blob(rev, path)
	if err != nil {
		return 0, false
	}
	ptr, err := blob.LFSPointer()
	if err != nil || ptr == nil {
		return 0, false
	}
	size := ptr.Size()
	if size <= 0 {
		return 0, false
	}
	return size, true
}

type safetensorsHeaderResult struct {
	header []byte
	err    error
}

// readSafetensorsHeader reads the header of a safetensors file, giving up when ctx
// is done. A tee cache reader blocks until the upstream download produces the
// bytes and observes neither ctx nor Close, so the read runs on its own goroutine
// and is abandoned rather than cancelled. That goroutine owns rc and closes it
// when the read returns, which happens at the latest when the download ends.
func (g *gitRepo) readSafetensorsHeader(ctx context.Context, repo *repository.Repository, rev, path string) ([]byte, error) {
	rc, err := g.openBlobContent(repo, rev, path)
	if err != nil {
		return nil, err
	}

	result := make(chan safetensorsHeaderResult, 1)
	go func() {
		defer func() {
			_ = rc.Close()
		}()
		header, err := readSafetensorsHeaderFrom(rc)
		result <- safetensorsHeaderResult{header: header, err: err}
	}()

	select {
	case res := <-result:
		return res.header, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("read safetensors header %s: %w", path, ctx.Err())
	}
}

func readSafetensorsHeaderFrom(r io.Reader) ([]byte, error) {
	var lengthPrefix [8]byte
	if _, err := io.ReadFull(r, lengthPrefix[:]); err != nil {
		return nil, err
	}

	headerLength := binary.LittleEndian.Uint64(lengthPrefix[:])
	if headerLength > maxSafetensorsHeaderBytes {
		return nil, fmt.Errorf("safetensors header too large: %d bytes", headerLength)
	}

	content := make([]byte, 8+int(headerLength))
	copy(content[:8], lengthPrefix[:])
	if _, err := io.ReadFull(r, content[8:]); err != nil {
		return nil, err
	}
	return content, nil
}

// parseSafetensorsIndex decodes a safetensors index, returning the zero value
// when the JSON cannot be read.
func parseSafetensorsIndex(indexJSON []byte) safetensorsIndexFiles {
	var index safetensorsIndexFiles
	if err := json.Unmarshal(indexJSON, &index); err != nil {
		return safetensorsIndexFiles{}
	}
	return index
}

func (index safetensorsIndexFiles) safetensorsPaths() []string {
	seen := make(map[string]struct{})
	paths := make([]string, 0, len(index.WeightMap))
	for _, path := range index.WeightMap {
		path = stdpath.Clean(path)
		if path == "." || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") || !strings.HasSuffix(path, ".safetensors") {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

// PushToRemote pushes the local repository to the remote registry.
func (g *gitRepo) PushToRemote(ctx context.Context, gitRepository *git.GitRepository) error {
	gitPath := g.gitPath(gitRepository.ResourceType, gitRepository.ProjectName, gitRepository.ResourceName)
	repoName := repoPrefix(gitRepository.ResourceType) + gitRepository.RemoteProjectName + "/" + gitRepository.RemoteResourceName
	destinationURL := strings.TrimSuffix(gitRepository.RemoteRegistryURL, "/") + "/" + repoName
	if !repository.IsRepository(gitPath) {
		return fmt.Errorf("local repository does not exist at %s", gitPath)
	}

	logWriter := gitRepository.LogWriter
	if logWriter == nil {
		logWriter = os.Stderr
	}

	syncOptions := []mirror.SyncOption{
		mirror.WithSyncMirrorDestinationURL(destinationURL),
		mirror.WithSyncOutput(logWriter),
	}

	if cred := gitRepository.Credential; cred != nil {
		syncOptions = append(syncOptions,
			mirror.WithSyncUserInfo(url.UserPassword(cred.Username, cred.Password)),
		)
	}

	return g.mirror.PushToRemote(ctx, gitPath, repoName,
		syncOptions...,
	)
}

// ExtractMetadata reads raw metadata-related files from a Git repository.
func (g *gitRepo) ExtractMetadata(ctx context.Context, repoType, project, name string) (*git.RepoMetadataFiles, error) {
	metadata := &git.RepoMetadataFiles{
		SafetensorsFiles: make(map[string][]byte),
		SafetensorsSizes: make(map[string]int64),
	}

	// Open Git repository
	gitPath := g.gitPath(repoType, project, name)
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repo: %w", err)
	}

	rev := repo.DefaultBranch()

	// Read README.md raw content
	if content, err := g.readBlobBytes(repo, rev, "README.md"); err == nil {
		metadata.ReadmeContent = content
	}

	// Read config.json raw content
	if content, err := g.readBlobBytes(repo, rev, "config.json"); err == nil {
		metadata.ConfigJSON = content
	}

	// Header reads share one deadline: they can block on an upstream download, and
	// this runs inline on the sync path that git and Hugging Face clients wait for.
	headerCtx, cancelHeaderReads := context.WithTimeout(ctx, safetensorsHeaderReadBudget)
	defer cancelHeaderReads()

	// Read model.safetensors.index.json raw content
	if content, err := g.readBlobBytes(repo, rev, "model.safetensors.index.json"); err == nil {
		metadata.SafetensorsIndexJSON = content
		// An index that carries metadata.total_size is enough on its own, and the
		// model domain prefers it over scanning shards. Reading shard headers
		// anyway would promote every shard to a foreground LFS download to
		// produce a result that is never used.
		if index := parseSafetensorsIndex(content); index.Metadata.TotalSize <= 0 {
			for _, path := range index.safetensorsPaths() {
				g.collectSafetensorsFile(headerCtx, repo, rev, path, metadata)
			}
		}
	} else {
		g.collectSafetensorsFile(headerCtx, repo, rev, "model.safetensors", metadata)
	}

	// Compute model content size from the default branch tree. DiskUsage
	// includes bare Git repository storage overhead, which is not user-visible
	// model file size.
	if size, err := repo.TreeSize(rev, ""); err == nil {
		metadata.Size = size
	}

	return metadata, nil
}
