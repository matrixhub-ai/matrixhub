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

package repository

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Blob struct {
	name      string
	size      int64
	modTime   time.Time
	newReader func() (io.ReadCloser, error)
	hash      string
}

func (b *Blob) Name() string {
	return b.name
}

func (b *Blob) Size() int64 {
	return b.size
}

func (b *Blob) ModTime() (t time.Time) {
	return b.modTime
}

func (b *Blob) NewReader() (io.ReadCloser, error) {
	return b.newReader()
}

func (b *Blob) Hash() string {
	return b.hash
}

func (r *Repository) Blob(ref string, path string) (b *Blob, err error) {
	var commit *object.Commit

	// First try to resolve as a branch reference
	refObj, err := r.repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err == nil {
		commit, err = r.repo.CommitObject(refObj.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get commit object: %w", err)
		}
	} else {
		// If not a branch, try to resolve as a commit SHA
		if !isValidSHA(ref) {
			return nil, errors.New("failed to resolve reference: not a valid branch or commit SHA")
		}
		hash := plumbing.NewHash(ref)
		commit, err = r.repo.CommitObject(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve reference or commit: %w", err)
		}
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree object: %w", err)
	}

	dir, filename := filepath.Split(path)
	var file *object.File
	dir = strings.TrimSuffix(dir, "/")
	if dir != "" {
		entry, err := tree.FindEntry(strings.TrimSuffix(dir, "/"))
		if err != nil {
			return nil, fmt.Errorf("path %s not found in tree: %w", dir, err)
		}

		if entry.Mode.IsFile() {
			return nil, errors.New("path is not a directory")
		}

		tree, err = r.repo.TreeObject(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtree object: %w", err)
		}
	}

	file, err = tree.File(filename)
	if err != nil {
		return nil, fmt.Errorf("file not found in tree: %w", err)
	}

	return &Blob{
		name:      file.Name,
		size:      file.Size,
		modTime:   commit.Committer.When,
		newReader: file.Reader,
		hash:      file.Hash.String(),
	}, nil
}
