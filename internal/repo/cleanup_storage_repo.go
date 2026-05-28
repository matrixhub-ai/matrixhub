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
	"golang.org/x/sync/errgroup"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

type lfsObjectInfo struct {
	size int64
	path string
}

// FindOrphanedRepos finds orphaned Git repositories on disk.
func (g *gitRepo) FindOrphanedRepos(ctx context.Context, validModelPaths, validDatasetPaths []string) ([]*git.OrphanedRepo, error) {
	validPaths := make(map[string]bool)
	for _, p := range validModelPaths {
		validPaths[p+".git"] = true
	}
	for _, p := range validDatasetPaths {
		validPaths["datasets/"+p+".git"] = true
	}

	reposDir := g.storage.RepositoriesDir()
	orphaned := []*git.OrphanedRepo{}

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

		orphaned = append(orphaned, &git.OrphanedRepo{
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
func (g *gitRepo) FindOrphanedLFS(ctx context.Context) ([]*git.OrphanedLFS, error) {
	allOIDs, err := g.scanLFSObjects(ctx)
	if err != nil {
		return nil, err
	}

	referencedOIDs, err := g.collectReferencedOIDs(ctx)
	if err != nil {
		return nil, err
	}

	orphaned := make([]*git.OrphanedLFS, 0)
	for oid, info := range allOIDs {
		if !referencedOIDs[oid] {
			orphaned = append(orphaned, &git.OrphanedLFS{
				OID:       oid,
				SizeBytes: info.size,
				Path:      info.path,
			})
		}
	}

	return orphaned, nil
}

// DeleteRepo deletes an orphaned repository by relative path.
func (g *gitRepo) DeleteRepo(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fullPath, err := confinedPath(g.storage.RepositoriesDir(), path)
	if err != nil {
		return err
	}
	return os.RemoveAll(fullPath)
}

// DeleteLFSObject deletes an orphaned LFS object.
func (g *gitRepo) DeleteLFSObject(ctx context.Context, object *git.OrphanedLFS) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fullPath, err := confinedPath(g.storage.LFSDir(), object.Path)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

// RepositoriesSize returns the size of all repositories on disk.
func (g *gitRepo) RepositoriesSize(ctx context.Context) int64 {
	if ctx.Err() != nil {
		return 0
	}
	return calculateDirSize(g.storage.RepositoriesDir())
}

// LFSSize returns the size of all LFS objects on disk.
func (g *gitRepo) LFSSize(ctx context.Context) int64 {
	if ctx.Err() != nil {
		return 0
	}
	return calculateDirSize(g.storage.LFSDir())
}

func (g *gitRepo) scanLFSObjects(ctx context.Context) (map[string]*lfsObjectInfo, error) {
	objects := make(map[string]*lfsObjectInfo)
	lfsDir, err := filepath.Abs(g.storage.LFSDir())
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

func (g *gitRepo) collectReferencedOIDs(ctx context.Context) (map[string]bool, error) {
	referencedOIDs := make(map[string]bool)

	reposDir := g.storage.RepositoriesDir()
	if _, err := os.Stat(reposDir); os.IsNotExist(err) {
		return referencedOIDs, nil
	}

	repoPaths := g.listAllRepoPaths(ctx, reposDir)

	group, _ := errgroup.WithContext(ctx)
	group.SetLimit(10)

	var mu sync.Mutex

	for _, repoPath := range repoPaths {
		group.Go(func() error {
			oids, err := g.collectRepoLFSOIDs(repoPath)
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

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return referencedOIDs, nil
}

func (g *gitRepo) listAllRepoPaths(ctx context.Context, reposDir string) []string {
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

func (g *gitRepo) collectRepoLFSOIDs(repoPath string) (map[string]bool, error) {
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
