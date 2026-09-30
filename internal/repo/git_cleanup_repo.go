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
		repo, err := g.openRepo(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		usage, err := repo.Usage(ctx)
		if err != nil {
			return fmt.Errorf("usage %s: %w", path, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !dryRun {
			if err := repo.Remove(); err != nil {
				return fmt.Errorf("remove %s: %w", path, err)
			}
		}
		size := usage.Objects.Bytes + usage.Other.Bytes
		orphaned = append(orphaned, &git.OrphanedRepo{Path: strings.TrimPrefix(path, "/"), SizeBytes: size})
		return nil
	})
	return orphaned, err
}

// Prune maps the hfd prune and sweep steps into one GCResult; SweepDone stays nil until the sweep step succeeded.
func (g *gitRepo) Prune(ctx context.Context, opts git.PruneOptions) (*git.GCResult, error) {
	if g.xetStore == nil {
		return nil, errors.New("lfs gc: xet store not configured")
	}
	grace := g.gcGrace
	if opts.Grace != nil {
		grace = *opts.Grace
	}
	res, err := g.gc.Prune(ctx, hfdgc.PruneOptions{Grace: grace, DryRun: opts.DryRun})
	if res == nil {
		return nil, err
	}
	result := &git.GCResult{
		DryRun:              res.DryRun,
		Repositories:        res.Repositories,
		DeletedGitObjects:   res.DeletedGitObjects,
		DeletedGitBytes:     res.DeletedGitBytes,
		GitReclaimedBytes:   res.ReclaimedBytes,
		Failed:              res.Failed,
		LiveObjects:         res.LiveObjects,
		Unlinked:            res.Unlinked,
		PruneSkippedInGrace: res.SkippedInGrace,
	}
	if err != nil {
		return result, err
	}
	// Unlinking only drops the index entry; the sweep reclaims (or, dry, estimates) already-unlinked data.
	sweep, err := g.gc.SweepStep(ctx, hfdgc.Options{Grace: grace, DryRun: opts.DryRun, MaxDeletes: opts.MaxDeletes, Budget: opts.Budget})
	if err != nil {
		return result, err
	}
	result.SweptShards = sweep.SweptShards
	result.SweptXorbs = sweep.SweptXorbs
	result.XetReclaimedBytes = sweep.ReclaimedBytes
	result.SweepSkippedInGrace = sweep.SkippedInGrace
	result.Dangling = sweep.Dangling
	result.UnreadableShards = sweep.UnreadableShards
	result.SweepDone = &sweep.Done
	result.RemainingShards = sweep.RemainingShards
	result.RemainingXorbs = sweep.RemainingXorbs
	return result, nil
}

// Usage reports Git and xet storage by kind; Xet stays zero without a xet store.
func (g *gitRepo) Usage(ctx context.Context) (*git.StorageUsage, error) {
	usage, err := g.gc.Usage(ctx)
	if err != nil {
		return nil, err
	}
	return &git.StorageUsage{
		Git: git.GitUsage{
			Objects: git.ObjectUsage(usage.Objects),
			Other:   git.ObjectUsage(usage.Other),
		},
		Xet: git.XetUsage{
			Xorbs:       git.ObjectUsage(usage.Xet.Xorbs),
			Shards:      git.ObjectUsage(usage.Xet.Shards),
			FileIndex:   git.ObjectUsage(usage.Xet.FileIndex),
			ChunkIndex:  git.ObjectUsage(usage.Xet.ChunkIndex),
			SHA256Index: git.ObjectUsage(usage.Xet.SHA256Index),
		},
	}, nil
}
