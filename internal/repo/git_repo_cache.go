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
	"fmt"
	"strings"

	hfdlfs "github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/repository"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// ListRevisionFiles resolves revision in the local repository and lists every
// file of that commit recursively, with LFS pointer details. The revision
// resolves as the Hugging Face API of hfd resolves it, with git's rules: a
// commit id first, then a tag before a branch of the same name. A pull then
// caches the commit that hf download asks MatrixHub for. The model pages use
// resolveRef, which prefers the branch; the two differ only for a name that
// is both.
func (g *gitRepo) ListRevisionFiles(ctx context.Context, repoType, project, name, revision string) (*git.RevisionFiles, error) {
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return nil, fmt.Errorf("%w: repository does not exist at %s", git.ErrRevisionNotFound, gitPath)
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return nil, err
	}

	rev := revision
	if rev == "" {
		rev = repo.DefaultBranch()
	}
	commit, err := repo.ResolveRevision(rev)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %w", git.ErrRevisionNotFound, rev, err)
	}

	// List the resolved commit rather than the ref so the files match the
	// commit reported to the caller even if the ref moves meanwhile.
	entries, err := repo.Tree(commit, "", &repository.TreeOptions{Recursive: true})
	if err != nil {
		return nil, err
	}

	files := make([]git.RevisionFile, 0, len(entries))
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if e.Type() != repository.EntryTypeFile {
			continue
		}
		blob, err := e.Blob()
		if err != nil {
			return nil, err
		}
		file := git.RevisionFile{Path: e.Path(), Size: blob.Size()}
		if ptr, _ := blob.LFSPointer(); ptr != nil {
			file.IsLFS = true
			file.OID = ptr.OID()
			file.Size = ptr.Size()
		}
		files = append(files, file)
	}

	return &git.RevisionFiles{Revision: rev, Commit: commit, Files: files}, nil
}

// HasCommit reports whether the local repository has commit. Unlike
// ListRevisionFiles, it reads no tree or blob.
func (g *gitRepo) HasCommit(_ context.Context, repoType, project, name, commit string) bool {
	if !isCommitSHA(commit) {
		return false
	}
	gitPath := g.gitPath(repoType, project, name)
	if !repository.IsRepository(gitPath) {
		return false
	}
	repo, err := repository.Open(gitPath)
	if err != nil {
		return false
	}
	resolved, err := repo.ResolveRevision(commit)
	return err == nil && strings.EqualFold(resolved, commit)
}

// LFSObjectExists reports whether an LFS object is fully stored locally.
func (g *gitRepo) LFSObjectExists(_ context.Context, oid string) bool {
	if !lfsOIDPattern.MatchString(oid) {
		return false
	}
	return hfdlfs.NewLocal(g.storage.LFSDir()).Exists(oid)
}

// FetchLFSObject returns the in-flight mirror download of oid, promoting it to
// a foreground download when it is queued. Callers only observe progress: no
// reader is opened, because an open reader keeps the finished download from
// being moved into LFS storage.
func (g *gitRepo) FetchLFSObject(_ context.Context, oid string) git.LFSDownload {
	if g.mirror == nil || !lfsOIDPattern.MatchString(oid) {
		return nil
	}
	b := g.mirror.Get(oid)
	if b == nil {
		return nil
	}
	return mirrorLFSDownload{blob: b}
}

type mirrorLFSDownload struct {
	blob *mirror.Blob
}

func (d mirrorLFSDownload) Downloaded() int64 { return d.blob.Progress() }

func (d mirrorLFSDownload) Total() int64 { return d.blob.Total() }
