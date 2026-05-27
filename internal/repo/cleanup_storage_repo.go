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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	"golang.org/x/sync/errgroup"

	"github.com/matrixhub-ai/matrixhub/internal/domain/cleanup"
)

type cleanupStorageRepo struct {
	storage *gitstorage.Storage
	dataDir string
}

type lfsObjectInfo struct {
	size int64
	path string
}

// NewCleanupStorageRepo creates a filesystem-backed cleanup storage adapter.
func NewCleanupStorageRepo(storage *gitstorage.Storage, dataDir string) cleanup.ICleanupStorageRepo {
	return &cleanupStorageRepo{
		storage: storage,
		dataDir: dataDir,
	}
}

// FindOrphanedRepos finds orphaned Git repositories on disk.
func (r *cleanupStorageRepo) FindOrphanedRepos(ctx context.Context, validModelPaths, validDatasetPaths []string) ([]*cleanup.OrphanedRepo, error) {
	validPaths := make(map[string]bool)
	for _, p := range validModelPaths {
		validPaths[p+".git"] = true
	}
	for _, p := range validDatasetPaths {
		validPaths["datasets/"+p+".git"] = true
	}

	reposDir := filepath.Join(r.dataDir, "repositories")
	orphaned := []*cleanup.OrphanedRepo{}

	err := filepath.Walk(reposDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !repository.IsRepository(path) {
			return nil
		}

		relPath := strings.TrimPrefix(path, reposDir+"/")
		if validPaths[relPath] {
			return filepath.SkipDir
		}

		parts := strings.Split(relPath, "/")
		repoType := "model"
		if strings.HasPrefix(relPath, "datasets/") {
			repoType = "dataset"
		}

		var projectName, resourceName string
		if len(parts) >= 2 {
			if repoType == "dataset" && len(parts) >= 3 {
				projectName = parts[1]
				resourceName = strings.TrimSuffix(parts[2], ".git")
			} else {
				projectName = parts[0]
				resourceName = strings.TrimSuffix(parts[1], ".git")
			}
		}

		orphaned = append(orphaned, &cleanup.OrphanedRepo{
			Path:         relPath,
			Type:         repoType,
			ProjectName:  projectName,
			ResourceName: resourceName,
			SizeBytes:    calculateDirSize(path),
		})

		return filepath.SkipDir
	})

	return orphaned, err
}

// FindOrphanedLFS finds orphaned LFS objects on disk.
func (r *cleanupStorageRepo) FindOrphanedLFS(ctx context.Context) ([]*cleanup.OrphanedLFS, error) {
	allOIDs, err := r.scanLFSObjects(ctx)
	if err != nil {
		return nil, err
	}

	referencedOIDs, err := r.collectReferencedOIDs(ctx)
	if err != nil {
		return nil, err
	}

	orphaned := make([]*cleanup.OrphanedLFS, 0)
	for oid, info := range allOIDs {
		if !referencedOIDs[oid] {
			orphaned = append(orphaned, &cleanup.OrphanedLFS{
				OID:       oid,
				SizeBytes: info.size,
				Path:      info.path,
			})
		}
	}

	return orphaned, nil
}

// DeleteRepo deletes an orphaned repository by relative path.
func (r *cleanupStorageRepo) DeleteRepo(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	reposDir := filepath.Join(r.dataDir, "repositories")
	fullPath, err := confinedPath(reposDir, path)
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
}

// DeleteLFSObject deletes an orphaned LFS object.
func (r *cleanupStorageRepo) DeleteLFSObject(ctx context.Context, object *cleanup.OrphanedLFS) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fullPath, err := confinedPath(r.storage.LFSDir(), object.Path)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

// RepositoriesSize returns the size of all repositories on disk.
func (r *cleanupStorageRepo) RepositoriesSize(ctx context.Context) int64 {
	if ctx.Err() != nil {
		return 0
	}
	return calculateDirSize(filepath.Join(r.dataDir, "repositories"))
}

// LFSSize returns the size of all LFS objects on disk.
func (r *cleanupStorageRepo) LFSSize(ctx context.Context) int64 {
	if ctx.Err() != nil {
		return 0
	}
	return calculateDirSize(r.storage.LFSDir())
}

func (r *cleanupStorageRepo) scanLFSObjects(ctx context.Context) (map[string]*lfsObjectInfo, error) {
	objects := make(map[string]*lfsObjectInfo)
	lfsDir, err := filepath.Abs(r.storage.LFSDir())
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(lfsDir); os.IsNotExist(err) {
		return objects, nil
	}

	err = filepath.Walk(lfsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		oid := info.Name()
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

func (r *cleanupStorageRepo) collectReferencedOIDs(ctx context.Context) (map[string]bool, error) {
	referencedOIDs := make(map[string]bool)

	reposDir := filepath.Join(r.dataDir, "repositories")
	if _, err := os.Stat(reposDir); os.IsNotExist(err) {
		return referencedOIDs, nil
	}

	repoPaths := r.listAllRepoPaths(ctx, reposDir)

	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(10)

	var mu sync.Mutex

	for _, repoPath := range repoPaths {
		g.Go(func() error {
			oids, err := r.collectRepoLFSOIDs(repoPath)
			if err != nil {
				return nil
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

func (r *cleanupStorageRepo) listAllRepoPaths(ctx context.Context, reposDir string) []string {
	repoPaths := []string{}
	err := filepath.Walk(reposDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
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

func (r *cleanupStorageRepo) collectRepoLFSOIDs(repoPath string) (map[string]bool, error) {
	oids := make(map[string]bool)

	repo, err := repository.Open(repoPath)
	if err != nil {
		return oids, err
	}

	branches, err := repo.Branches()
	if err != nil {
		return oids, nil
	}
	for _, branch := range branches {
		collectOIDsFromRevision(repo, "refs/heads/"+branch, oids)
	}

	tags, err := repo.Tags()
	if err != nil {
		return oids, nil
	}
	for _, tag := range tags {
		collectOIDsFromRevision(repo, "refs/tags/"+tag, oids)
	}

	return oids, nil
}

func collectOIDsFromRevision(repo *repository.Repository, rev string, oids map[string]bool) {
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

			ptr, _ := blob.LFSPointer()
			if ptr != nil {
				oids[ptr.OID()] = true
			}
		}
	}
}

func calculateDirSize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return size
}

func confinedPath(root, path string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(path) {
		return checkedPath(rootAbs, path, path, root)
	}

	pathAbs, err := filepath.Abs(path)
	if err == nil && isSubpath(rootAbs, pathAbs) {
		return pathAbs, nil
	}

	return checkedPath(rootAbs, filepath.Join(rootAbs, path), path, root)
}

func checkedPath(rootAbs, fullPath, originalPath, originalRoot string) (string, error) {
	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	if !isSubpath(rootAbs, fullPath) {
		return "", fmt.Errorf("cleanup path %q escapes root %q", originalPath, originalRoot)
	}
	return fullPath, nil
}

func isSubpath(rootAbs, pathAbs string) bool {
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
