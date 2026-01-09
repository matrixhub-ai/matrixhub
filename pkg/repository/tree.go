package repository

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/wzshiming/gitd/pkg/lfs"
)

var (
	ErrObjectNotFound = plumbing.ErrObjectNotFound
)

// TreeEntry represents a file or directory in the repository
type TreeEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"` // "blob" or "tree"
	Mode       string `json:"mode"`
	SHA        string `json:"sha"`
	IsLFS      bool   `json:"isLfs,omitempty"`
	BlobSha256 string `json:"blobSha256,omitempty"`
}

func (r *Repository) Tree(ref string, path string) ([]TreeEntry, error) {
	refObj, err := r.repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return []TreeEntry{}, nil
		}
		return nil, err
	}

	commit, err := r.repo.CommitObject(refObj.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree object: %w", err)
	}

	if path != "" {
		entry, err := tree.FindEntry(path)
		if err != nil {
			return nil, fmt.Errorf("path not found: %w", err)
		}

		if entry.Mode.IsFile() {
			return nil, fmt.Errorf("path is not a directory")
		}

		tree, err = r.repo.TreeObject(entry.Hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get subtree object: %w", err)
		}
	}

	var entries []TreeEntry
	for _, entry := range tree.Entries {
		if entry.Mode.IsFile() {
			entry := TreeEntry{
				Name: entry.Name,
				Path: filepath.Join(path, entry.Name),
				Type: "blob",
				Mode: formatMode(entry.Mode),
				SHA:  entry.Hash.String(),
			}

			hash := plumbing.NewHash(entry.SHA)
			blob, err := r.repo.BlobObject(hash)
			if err == nil && blob.Size <= lfs.MaxLFSPointerSize {
				reader, err := blob.Reader()
				if err == nil {
					ptr, err := lfs.DecodePointer(reader)
					reader.Close()
					if err == nil && ptr != nil {
						entry.IsLFS = true
						entry.BlobSha256 = ptr.Oid
					}
				}
			}
			entries = append(entries, entry)
		} else {
			entries = append(entries, TreeEntry{
				Name: entry.Name,
				Path: filepath.Join(path, entry.Name),
				Type: "tree",
				Mode: formatMode(entry.Mode),
				SHA:  entry.Hash.String(),
			})
		}
	}
	return entries, nil
}

func formatMode(mode filemode.FileMode) string {
	switch mode {
	case filemode.Dir:
		return "dir"
	case filemode.Regular:
		return "regular"
	case filemode.Executable:
		return "executable"
	case filemode.Symlink:
		return "symlink"
	case filemode.Submodule:
		return "submodule"
	default:
		return fmt.Sprintf("unknown(%07o)", uint32(mode))
	}
}
