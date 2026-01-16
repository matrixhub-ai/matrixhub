package repository

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
)

// TreeEntry represents a file or directory in the repository
type HFTreeEntry struct {
	OID  string     `json:"oid"`
	Path string     `json:"path"`
	Type string     `json:"type"` // "file" or "directory"
	Size int64      `json:"size"`
	LFS  *HFTreeLFS `json:"lfs,omitempty"`
}

type HFTreeLFS struct {
	OID         string `json:"oid"`
	Size        int64  `json:"size"`
	PointerSize int64  `json:"pointerSize"`
}

func (r *Repository) HFTree(ref string, path string) ([]HFTreeEntry, error) {
	var commit *object.Commit

	// First try to resolve as a branch reference
	refObj, err := r.repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err == nil {
		commit, err = r.repo.CommitObject(refObj.Hash())
		if err != nil {
			return nil, fmt.Errorf("failed to get commit object: %w", err)
		}
	} else if err == plumbing.ErrReferenceNotFound {
		// If not a branch, try to resolve as a commit SHA
		if !isValidSHA(ref) {
			// Neither a branch nor a valid commit SHA format
			return []HFTreeEntry{}, nil
		}
		hash := plumbing.NewHash(ref)
		commit, err = r.repo.CommitObject(hash)
		if err != nil {
			// Valid SHA format but commit not found
			return []HFTreeEntry{}, nil
		}
	} else {
		return nil, err
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

	var entries []HFTreeEntry
	for _, entry := range tree.Entries {
		if entry.Mode.IsFile() {
			hfentry := HFTreeEntry{
				OID:  entry.Hash.String(),
				Path: filepath.Join(path, entry.Name),
				Type: "file",
			}

			hash := plumbing.NewHash(hfentry.OID)
			blob, err := r.repo.BlobObject(hash)
			if err == nil {
				hfentry.Size = blob.Size
				if blob.Size <= lfs.MaxLFSPointerSize {
					reader, err := blob.Reader()
					if err == nil {
						ptr, err := lfs.DecodePointer(reader)
						reader.Close()
						hfentry.Size = blob.Size
						if err == nil && ptr != nil {
							hfentry.LFS = &HFTreeLFS{
								OID:         ptr.Oid,
								Size:        ptr.Size,
								PointerSize: blob.Size,
							}
						}
					}
				}
			}
			entries = append(entries, hfentry)
		} else {
			entries = append(entries, HFTreeEntry{
				OID:  entry.Hash.String(),
				Path: filepath.Join(path, entry.Name),
				Type: "directory",
				Size: 0,
			})
		}
	}
	return entries, nil
}
