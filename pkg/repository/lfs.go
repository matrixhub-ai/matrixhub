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
	"bufio"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// LFSPointer represents a Git LFS pointer found in the repository
type LFSPointer struct {
	Oid  string // SHA256 hash of the LFS object
	Size int64  // Size of the LFS object in bytes
}

// ScanLFSPointers scans all branches in the repository for LFS pointer files
// and returns a list of unique LFS pointers
func (r *Repository) ScanLFSPointers() ([]LFSPointer, error) {
	seen := make(map[string]LFSPointer)

	// Get all branches
	branches, err := r.repo.Branches()
	if err != nil {
		return nil, err
	}

	err = branches.ForEach(func(ref *plumbing.Reference) error {
		commit, err := r.repo.CommitObject(ref.Hash())
		if err != nil {
			// Skip branches with inaccessible commits, continue with others
			return nil
		}

		tree, err := commit.Tree()
		if err != nil {
			// Skip branches with inaccessible trees, continue with others
			return nil
		}

		// Walk all files in the tree
		walker := object.NewTreeWalker(tree, true, nil)
		defer walker.Close()

		for {
			_, entry, err := walker.Next()
			if err != nil {
				break
			}

			if !entry.Mode.IsFile() {
				continue
			}

			blob, err := r.repo.BlobObject(entry.Hash)
			if err != nil {
				continue
			}

			ptr, err := parseLFSPointerFromBlob(blob)
			if err != nil || ptr == nil {
				continue
			}

			seen[ptr.Oid] = *ptr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	result := make([]LFSPointer, 0, len(seen))
	for _, ptr := range seen {
		result = append(result, ptr)
	}

	return result, nil
}

// parseLFSPointerFromBlob parses an LFS pointer from a git blob
// Returns nil if the blob is not an LFS pointer
func parseLFSPointerFromBlob(blob *object.Blob) (*LFSPointer, error) {
	// LFS pointers are small (typically < 200 bytes)
	if blob.Size > 1024 {
		return nil, nil
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	scanner := bufio.NewScanner(reader)
	var isLFS bool
	var oid string
	var size int64

	for scanner.Scan() {
		line := scanner.Text()

		// Check for LFS version header
		if strings.HasPrefix(line, "version https://git-lfs.github.com/spec/v") {
			isLFS = true
		}

		// Extract OID from oid line
		if after, ok := strings.CutPrefix(line, "oid sha256:"); ok {
			oid = after
		}

		// Extract size
		if after, ok := strings.CutPrefix(line, "size "); ok {
			sizeStr := after
			size = parseSize(sizeStr)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if isLFS && oid != "" {
		return &LFSPointer{
			Oid:  oid,
			Size: size,
		}, nil
	}

	return nil, nil
}

// parseSize parses a size string into an int64
func parseSize(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			break
		}
	}
	return n
}
