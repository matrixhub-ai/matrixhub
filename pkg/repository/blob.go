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
	"fmt"
	"io"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
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
	if ref == "" {
		ref = r.DefaultBranch()
	}

	hash, err := r.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve revision: %w", err)
	}

	commit, err := r.repo.CommitObject(*hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	file, err := commit.File(path)
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
