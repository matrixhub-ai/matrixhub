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
	"errors"
	"fmt"
	"io/fs"
	stdpath "path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	hfdgc "github.com/matrixhub-ai/hfd/pkg/gc"
	"github.com/matrixhub-ai/hfd/pkg/repository"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// dirFS is the subset of billy.Filesystem needed for directory walks, declared
// locally to keep go-billy out of matrixhub's direct dependencies.
type dirFS interface {
	Stat(filename string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
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

	reposFS := g.storage.RepositoriesFS()
	orphaned := []*git.OrphanedRepo{}

	err := walkFS(reposFS, "/", func(path string, info fs.FileInfo) error {
		if !info.IsDir() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !repository.IsRepository(reposFS, path) {
			return nil
		}

		relPath := strings.TrimPrefix(path, "/")
		if validPaths[relPath] {
			return fs.SkipDir
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
			SizeBytes:    fsDirSize(reposFS, path),
		})

		return fs.SkipDir
	})

	return orphaned, err
}

// CollectLFS runs the xet LFS garbage collector; dryRun only lists unreferenced objects.
func (g *gitRepo) CollectLFS(ctx context.Context, dryRun bool) (*git.LFSCollectResult, error) {
	if g.gc == nil {
		return nil, errors.New("lfs gc: xet store not configured")
	}
	res, err := g.gc.Collect(ctx, hfdgc.Options{Grace: g.gcGrace, DryRun: dryRun})
	if res == nil {
		return nil, err
	}
	result := &git.LFSCollectResult{Orphaned: make([]*git.OrphanedLFS, 0, len(res.Unlinked))}
	for _, oid := range res.Unlinked {
		result.Orphaned = append(result.Orphaned, &git.OrphanedLFS{OID: oid})
	}
	if res.Sweep != nil {
		result.ReclaimedBytes = res.Sweep.ReclaimedBytes
	}
	// A partial run (failure after unlinking began) returns both the mapped result and err.
	return result, err
}

// DeleteRepositoryAtRelPath deletes an orphaned repository by relative path.
func (g *gitRepo) DeleteRepositoryAtRelPath(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repoPath, err := confinedRepoPath(path)
	if err != nil {
		return err
	}
	repo, err := g.openRepo(repoPath)
	if err != nil {
		return err
	}
	return repo.Remove()
}

// RepositoriesSize returns the size of all repositories on disk.
func (g *gitRepo) RepositoriesSize(ctx context.Context) int64 {
	if ctx.Err() != nil {
		return 0
	}
	return fsDirSize(g.storage.RepositoriesFS(), "/")
}

// LFSSize returns the size of all LFS objects on disk.
// xet deduplicates content into shards and xorbs, so their stored sizes are summed.
func (g *gitRepo) LFSSize(ctx context.Context) int64 {
	if g.xetStore == nil || ctx.Err() != nil {
		return 0
	}
	var size int64
	add := func(_ string, n int64, _ time.Time) error {
		size += n
		return nil
	}
	if err := g.xetStore.WalkShards(ctx, add); err != nil {
		return 0
	}
	if err := g.xetStore.WalkXorbs(ctx, add); err != nil {
		return 0
	}
	return size
}

// walkFS walks fsys from root, calling fn for every file and directory.
// fn may return fs.SkipDir on a directory to skip its contents.
func walkFS(fsys dirFS, root string, fn func(path string, info fs.FileInfo) error) error {
	info, err := fsys.Stat(root)
	if err != nil {
		return nil
	}
	err = walkFSNode(fsys, root, info, fn)
	if errors.Is(err, fs.SkipDir) {
		return nil
	}
	return err
}

func walkFSNode(fsys dirFS, path string, info fs.FileInfo, fn func(path string, info fs.FileInfo) error) error {
	if err := fn(path, info); err != nil || !info.IsDir() {
		return err
	}
	entries, err := fsys.ReadDir(path)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}
		if err := walkFSNode(fsys, stdpath.Join(path, entry.Name()), entryInfo, fn); err != nil {
			if errors.Is(err, fs.SkipDir) {
				continue
			}
			return err
		}
	}
	return nil
}

// fsDirSize returns the total size of regular files under root on fsys.
func fsDirSize(fsys dirFS, root string) int64 {
	var size int64
	err := walkFS(fsys, root, func(_ string, info fs.FileInfo) error {
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

// confinedRepoPath validates that path stays inside the repositories
// filesystem and returns it rooted at "/".
func confinedRepoPath(path string) (string, error) {
	slashPath := filepath.ToSlash(path)
	if slices.Contains(strings.Split(slashPath, "/"), "..") {
		return "", fmt.Errorf("cleanup path %q escapes repositories root", path)
	}
	cleaned := stdpath.Clean("/" + slashPath)
	if cleaned == "/" {
		return "", fmt.Errorf("cleanup path %q does not name a repository", path)
	}
	return cleaned, nil
}
