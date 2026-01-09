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
			return nil
		}

		tree, err := commit.Tree()
		if err != nil {
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
	defer reader.Close()

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
		if strings.HasPrefix(line, "oid sha256:") {
			oid = strings.TrimPrefix(line, "oid sha256:")
		}

		// Extract size
		if strings.HasPrefix(line, "size ") {
			sizeStr := strings.TrimPrefix(line, "size ")
			_, err := parseSize(sizeStr, &size)
			if err != nil {
				continue
			}
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
func parseSize(s string, size *int64) (bool, error) {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			break
		}
	}
	*size = n
	return true, nil
}
