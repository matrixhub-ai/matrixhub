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
	"strings"
	"time"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/util"
	hfdgc "github.com/matrixhub-ai/hfd/pkg/gc"
	"github.com/matrixhub-ai/hfd/pkg/repository"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// PruneRepos removes every repository on disk that neither list names; dryRun only reports them.
func (g *gitRepo) PruneRepos(ctx context.Context, validModelPaths, validDatasetPaths []string, dryRun bool) ([]*git.OrphanedRepo, error) {
	keep := make(map[string]bool, len(validModelPaths)+len(validDatasetPaths))
	for _, p := range validModelPaths {
		keep[repository.ResolvePath(repoPrefix("model")+p)] = true
	}
	for _, p := range validDatasetPaths {
		keep[repository.ResolvePath(repoPrefix("dataset")+p)] = true
	}
	reposFS := g.storage.RepositoriesFS()
	orphaned := []*git.OrphanedRepo{}
	if _, err := reposFS.Stat("/"); errors.Is(err, fs.ErrNotExist) {
		return orphaned, nil
	}
	err := repository.Walk(ctx, reposFS, "/", func(path string) error {
		if keep[path] {
			return nil
		}
		size := fsDirSize(reposFS, path)
		if err := ctx.Err(); err != nil {
			return err
		}
		if !dryRun {
			repo, err := g.openRepo(path)
			if err != nil {
				return fmt.Errorf("open %s: %w", path, err)
			}
			if err := repo.Remove(); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}
		}
		orphaned = append(orphaned, &git.OrphanedRepo{Path: strings.TrimPrefix(path, "/"), SizeBytes: size})
		return nil
	})
	return orphaned, err
}

func (g *gitRepo) Prune(ctx context.Context, dryRun bool) (*git.PruneResult, error) {
	if g.gc == nil {
		return nil, errors.New("lfs gc: xet store not configured")
	}
	objects, err := g.gc.List(ctx)
	if err != nil {
		return nil, err
	}
	sizes := make(map[string]int64, len(objects))
	for _, object := range objects {
		sizes[object.OID] = int64(object.Size)
	}
	res, err := g.gc.Prune(ctx, hfdgc.PruneOptions{Grace: g.gcGrace, DryRun: dryRun})
	if res == nil {
		return nil, err
	}
	result := &git.PruneResult{Unlinked: make([]*git.OrphanedLFS, 0, len(res.Unlinked))}
	for _, oid := range res.Unlinked {
		result.Unlinked = append(result.Unlinked, &git.OrphanedLFS{OID: oid, SizeBytes: sizes[oid]})
	}
	if err != nil || dryRun {
		return result, err
	}
	// Unlinking only drops the index entry; the sweep reclaims the data.
	sweep, err := g.gc.SweepStep(ctx, hfdgc.Options{Grace: g.gcGrace})
	if err != nil {
		return result, err
	}
	result.ReclaimedBytes = sweep.ReclaimedBytes
	return result, nil
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

// fsDirSize returns the total size of regular files under root on fsys.
func fsDirSize(fsys billy.Filesystem, root string) int64 {
	var size int64
	_ = util.Walk(fsys, root, func(_ string, info fs.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
