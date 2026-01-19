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
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
)

type HFEntryType string

const (
	HFEntryTypeFile      HFEntryType = "file"
	HFEntryTypeDirectory HFEntryType = "directory"
)

// HFTreeEntry represents a file or directory in the repository
type HFTreeEntry struct {
	OID        string            `json:"oid"`
	Path       string            `json:"path"`
	Type       HFEntryType       `json:"type"`
	Size       int64             `json:"size"`
	LFS        *HFTreeLFS        `json:"lfs,omitempty"`
	LastCommit *HFTreeLastCommit `json:"lastCommit,omitempty"`
}

type HFTreeLFS struct {
	OID         string `json:"oid"`
	Size        int64  `json:"size"`
	PointerSize int64  `json:"pointerSize"`
}

// HFTreeLastCommit represents the last commit that modified a file or directory
type HFTreeLastCommit struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
}

// HFTreeOptions provides options for the HFTree method.
type HFTreeOptions struct {
	// Recursive enables recursive traversal of subdirectories.
	Recursive bool
	// Expand enables fetching additional metadata such as lastCommit info.
	Expand bool
}

// hfBlobMetadata populates the Size and LFS fields for a file entry.
func (r *Repository) hfBlobMetadata(hfentry *HFTreeEntry) {
	hash := plumbing.NewHash(hfentry.OID)
	blob, err := r.repo.BlobObject(hash)
	if err != nil {
		return
	}
	hfentry.Size = blob.Size
	if blob.Size <= lfs.MaxLFSPointerSize {
		reader, err := blob.Reader()
		if err != nil {
			return
		}
		ptr, err := lfs.DecodePointer(reader)
		_ = reader.Close()
		if err == nil && ptr != nil {
			hfentry.LFS = &HFTreeLFS{
				OID:         ptr.Oid,
				Size:        ptr.Size,
				PointerSize: blob.Size,
			}

			hfentry.Size = ptr.Size
		}
	}
}

// hfLastCommit fetches the last commit that modified the given path.
func (r *Repository) hfLastCommit(commit *object.Commit) (*HFTreeLastCommit, error) {
	// Get the commit log for this specific file
	iter, err := r.repo.Log(&git.LogOptions{
		From: commit.Hash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer iter.Close()

	// Get the first (most recent) commit
	c, err := iter.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to get last commit: %w", err)
	}

	// Extract just the first line of the commit message as the title
	title := c.Message
	if idx := strings.Index(title, "\n"); idx >= 0 {
		title = title[:idx]
	}

	return &HFTreeLastCommit{
		ID:    c.Hash.String(),
		Title: title,
		Date:  c.Author.When.UTC().Format("2006-01-02T15:04:05.000Z"),
	}, nil
}

func (r *Repository) HFTree(ref string, path string, opts *HFTreeOptions) ([]HFTreeEntry, error) {
	if opts == nil {
		opts = &HFTreeOptions{}
	}

	hash, err := r.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve revision: %w", err)
	}

	commit, err := r.repo.CommitObject(*hash)
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

	var entries []HFTreeEntry
	err = r.hfWalkTree(commit, tree, path, opts, func(entry *HFTreeEntry) error {
		entries = append(entries, *entry)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// hfWalkTree recursively walks a tree and returns all entries.
func (r *Repository) hfWalkTree(commit *object.Commit, tree *object.Tree, basePath string, opts *HFTreeOptions, cb func(entry *HFTreeEntry) error) error {
	for _, entry := range tree.Entries {
		entryPath := path.Join(basePath, entry.Name)
		if entry.Mode.IsFile() {
			hfentry := HFTreeEntry{
				OID:  entry.Hash.String(),
				Path: entryPath,
				Type: HFEntryTypeFile,
			}

			r.hfBlobMetadata(&hfentry)
			if opts.Expand {
				lastCommit, err := r.hfLastCommit(commit)
				if err != nil {
					return err
				}
				hfentry.LastCommit = lastCommit
			}

			if err := cb(&hfentry); err != nil {
				return err
			}
		} else {
			hfentry := HFTreeEntry{
				OID:  entry.Hash.String(),
				Path: entryPath,
				Type: HFEntryTypeDirectory,
				Size: 0,
			}
			if opts.Expand {
				lastCommit, err := r.hfLastCommit(commit)
				if err != nil {
					return err
				}
				hfentry.LastCommit = lastCommit
			}

			if err := cb(&hfentry); err != nil {
				return err
			}

			if opts.Recursive {
				subTree, err := r.repo.TreeObject(entry.Hash)
				if err == nil {
					err = r.hfWalkTree(commit, subTree, entryPath, opts, cb)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
