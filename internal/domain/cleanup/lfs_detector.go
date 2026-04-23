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

package cleanup

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	"golang.org/x/sync/errgroup"
)

// LFSOrphanDetector detects orphaned LFS objects.
type LFSOrphanDetector struct {
	dataDir string
	lfsDir  string
}

// lfsObjectInfo holds information about an LFS object on disk.
type lfsObjectInfo struct {
	size int64
	path string
}

// NewLFSOrphanDetector creates a new LFSOrphanDetector instance.
func NewLFSOrphanDetector(dataDir, lfsDir string) *LFSOrphanDetector {
	return &LFSOrphanDetector{
		dataDir: dataDir,
		lfsDir:  lfsDir,
	}
}

// DetectOrphanedLFS detects orphaned LFS objects on disk.
func (d *LFSOrphanDetector) DetectOrphanedLFS(ctx context.Context) ([]*OrphanedLFS, error) {
	// Step 1: Scan LFS directory to get all OIDs
	allOIDs, err := d.scanLFSObjects()
	if err != nil {
		return nil, err
	}

	// Step 2: Collect all referenced OIDs from Git repositories
	referencedOIDs, err := d.collectReferencedOIDs(ctx)
	if err != nil {
		return nil, err
	}

	// Step 3: Calculate difference (orphaned = all - referenced)
	orphaned := make([]*OrphanedLFS, 0)
	for oid, info := range allOIDs {
		if !referencedOIDs[oid] {
			orphaned = append(orphaned, &OrphanedLFS{
				OID:       oid,
				SizeBytes: info.size,
				Path:      info.path,
			})
		}
	}

	return orphaned, nil
}

// scanLFSObjects scans the LFS directory and returns all objects.
func (d *LFSOrphanDetector) scanLFSObjects() (map[string]*lfsObjectInfo, error) {
	objects := make(map[string]*lfsObjectInfo)

	// LFS files are stored in: lfs/<oid[:2]>/<oid[2:4]>/<oid>
	// No "objects" subdirectory - we scan the lfsDir directly
	if _, err := os.Stat(d.lfsDir); os.IsNotExist(err) {
		return objects, nil // No LFS directory
	}

	err := filepath.Walk(d.lfsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// File name is the OID
		// OID can be various lengths (SHA256 is 64 chars, but other formats exist)
		oid := info.Name()
		// Minimum reasonable OID length (e.g., at least 8 chars for a valid hash)
		if len(oid) >= 8 {
			objects[oid] = &lfsObjectInfo{
				size: info.Size(),
				path: path,
			}
		}
		return nil
	})

	return objects, err
}

// collectReferencedOIDs collects all OIDs referenced by Git repositories.
func (d *LFSOrphanDetector) collectReferencedOIDs(ctx context.Context) (map[string]bool, error) {
	referencedOIDs := make(map[string]bool)

	reposDir := filepath.Join(d.dataDir, "repositories")
	if _, err := os.Stat(reposDir); os.IsNotExist(err) {
		return referencedOIDs, nil // No repositories directory
	}

	// Collect all repository paths first
	repoPaths := d.listAllRepoPaths(reposDir)

	// Use errgroup for parallel processing
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrent processing

	var mu sync.Mutex

	for _, repoPath := range repoPaths {
		g.Go(func() error {
			oids, err := d.collectRepoLFSOIDs(repoPath)
			if err != nil {
				return nil // Skip errors, continue processing
			}
			mu.Lock()
			for oid := range oids {
				referencedOIDs[oid] = true
			}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return referencedOIDs, nil
}

// listAllRepoPaths lists all Git repository paths.
func (d *LFSOrphanDetector) listAllRepoPaths(reposDir string) []string {
	repoPaths := []string{}
	err := filepath.Walk(reposDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if repository.IsRepository(path) {
			repoPaths = append(repoPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return repoPaths
}

// collectRepoLFSOIDs collects all LFS OIDs from a single repository.
// It scans all commits in all branches and tags to ensure we don't
// miss LFS objects referenced in history.
func (d *LFSOrphanDetector) collectRepoLFSOIDs(repoPath string) (map[string]bool, error) {
	oids := make(map[string]bool)

	repo, err := repository.Open(repoPath)
	if err != nil {
		return oids, err
	}

	// Process all branches
	branches, err := repo.Branches()
	if err != nil {
		return oids, nil
	}
	for _, branch := range branches {
		d.collectOIDsFromRevision(repo, "refs/heads/"+branch, oids)
	}

	// Process all tags
	tags, err := repo.Tags()
	if err != nil {
		return oids, nil
	}
	for _, tag := range tags {
		d.collectOIDsFromRevision(repo, "refs/tags/"+tag, oids)
	}

	return oids, nil
}

// collectOIDsFromRevision collects LFS OIDs from all commits in a revision.
// This is critical: we must scan all commits, not just HEAD, because
// LFS objects may be referenced in history even if deleted in current HEAD.
func (d *LFSOrphanDetector) collectOIDsFromRevision(repo *repository.Repository, rev string, oids map[string]bool) {
	commits, err := repo.Commits(rev, nil)
	if err != nil {
		return
	}

	for _, commit := range commits {
		entries, err := repo.Tree(commit.Hash().String(), "", &repository.TreeOptions{Recursive: true})
		if err != nil {
			continue
		}

		for _, entry := range entries {
			blob, err := entry.Blob()
			if err != nil {
				continue
			}

			// Check if this is an LFS pointer
			ptr, _ := blob.LFSPointer()
			if ptr != nil {
				oids[ptr.OID()] = true
			}
		}
	}
}
