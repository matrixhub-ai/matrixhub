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
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

type Blob struct {
	name        string
	size        int64
	contentType string
	modTime     time.Time
	newReader   func() (io.ReadCloser, error)
	hash        string
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

func (b *Blob) ContentType() string {
	return b.contentType
}

func (b *Blob) NewReader() (io.ReadCloser, error) {
	return b.newReader()
}

func (b *Blob) Hash() string {
	return b.hash
}

func (r *Repository) Blob(ref string, path string) (b *Blob, err error) {
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

	contentType := getContentType(file.Name)
	return &Blob{
		name:        file.Name,
		size:        file.Size,
		contentType: contentType,
		modTime:     commit.Committer.When,
		newReader:   file.Reader,
		hash:        file.Hash.String(),
	}, nil
}

// getContentType returns the MIME type based on file extension
func getContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".go":
		return "text/x-go; charset=utf-8"
	case ".py":
		return "text/x-python; charset=utf-8"
	case ".java":
		return "text/x-java; charset=utf-8"
	case ".c", ".h":
		return "text/x-c; charset=utf-8"
	case ".cpp", ".hpp", ".cc":
		return "text/x-c++; charset=utf-8"
	case ".rs":
		return "text/x-rust; charset=utf-8"
	case ".ts":
		return "text/typescript; charset=utf-8"
	case ".tsx":
		return "text/tsx; charset=utf-8"
	case ".jsx":
		return "text/jsx; charset=utf-8"
	case ".yaml", ".yml":
		return "text/yaml; charset=utf-8"
	case ".sh", ".bash":
		return "text/x-shellscript; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "text/plain; charset=utf-8"
	}
}
