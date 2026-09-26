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
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/scan"
)

// ScanTreeReader adapts the git/LFS storage to the scan domain's TreeReader
// port: it lists and opens files at a revision, transparently dereferencing
// LFS pointer files to the underlying object content.
type ScanTreeReader struct {
	storage    *gitstorage.Storage
	lfsStorage lfs.Storage
}

func NewScanTreeReader(storage *gitstorage.Storage, lfsStorage lfs.Storage) *ScanTreeReader {
	return &ScanTreeReader{storage: storage, lfsStorage: lfsStorage}
}

// repoName maps a RepoKey to the on-disk repository name: models live at
// {project}/{name}, datasets and spaces carry their type prefix.
func (s *ScanTreeReader) repoName(key scan.RepoKey) string {
	full := key.Project + "/" + key.Name
	if key.RepoType == "datasets" || key.RepoType == "spaces" {
		return key.RepoType + "/" + full
	}
	return full
}

func (s *ScanTreeReader) open(key scan.RepoKey) (*repository.Repository, error) {
	repoPath := s.storage.ResolvePath(s.repoName(key))
	if repoPath == "" {
		return nil, repository.ErrRepositoryNotExists
	}
	return repository.Open(repoPath)
}

func (s *ScanTreeReader) Files(ctx context.Context, key scan.RepoKey, revision string) ([]scan.FileDesc, error) {
	repo, err := s.open(key)
	if err != nil {
		return nil, err
	}
	entries, err := repo.Tree(revision, "", &repository.TreeOptions{Recursive: true})
	if err != nil {
		return nil, err
	}
	out := make([]scan.FileDesc, 0, len(entries))
	for _, e := range entries {
		if e.Type() != repository.EntryTypeFile {
			continue
		}
		size := int64(0)
		if b, err := e.Blob(); err == nil {
			size = b.Size()
		}
		out = append(out, scan.FileDesc{
			Path:     e.Path(),
			Size:     size,
			BlobHash: e.Hash().String(),
		})
	}
	return out, nil
}

func (s *ScanTreeReader) Open(ctx context.Context, key scan.RepoKey, revision, path string) (io.ReadCloser, error) {
	repo, err := s.open(key)
	if err != nil {
		return nil, err
	}
	blob, err := repo.Blob(revision, path)
	if err != nil {
		return nil, err
	}
	// LFS pointer files resolve to the stored object so scanners always see
	// real content. Pointer files are tiny, so buffering them is cheap.
	if rc, rerr := blob.NewReader(); rerr == nil && blob.Size() <= lfs.MaxLFSPointerSize {
		buf, err := io.ReadAll(io.LimitReader(rc, lfs.MaxLFSPointerSize+1))
		_ = rc.Close()
		if err == nil {
			ptr, derr := lfs.DecodePointer(bytes.NewReader(buf))
			if derr == nil && ptr != nil {
				if s.lfsStorage != nil && s.lfsStorage.Exists(ptr.OID()) {
					if getter, ok := s.lfsStorage.(lfs.Getter); ok {
						obj, _, gerr := getter.Get(ptr.OID())
						if gerr != nil {
							return nil, gerr
						}
						return obj, nil
					}
				}
				return nil, errors.New("lfs object not available locally for scanning")
			}
			return io.NopCloser(bytes.NewReader(buf)), nil
		}
	}
	return blob.NewReader()
}

func (s *ScanTreeReader) ResolveRevision(ctx context.Context, key scan.RepoKey, revision string) (string, error) {
	repo, err := s.open(key)
	if err != nil {
		return "", err
	}
	if revision == "" {
		revision = repo.DefaultBranch()
	}
	commits, err := repo.Commits(revision, &repository.CommitsOptions{Limit: 1})
	if err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", errors.New("revision has no commits")
	}
	return commits[0].Hash().String(), nil
}

var _ scan.TreeReader = (*ScanTreeReader)(nil)
