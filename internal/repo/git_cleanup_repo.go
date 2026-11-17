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

	hfdgc "github.com/matrixhub-ai/hfd/pkg/gc"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	xetmirror "github.com/wzshiming/xet/mirror"
	"github.com/wzshiming/xet/mirror/spool"
	xetstorage "github.com/wzshiming/xet/storage"

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

// Prune maps the hfd prune, the xet CAS sweep step and, once that finished, the spool and mirror-index sweeps into one
// GCResult; SweepDone stays nil until the CAS sweep step succeeded.
func (g *gitRepo) Prune(ctx context.Context, opts git.PruneOptions) (*git.GCResult, error) {
	if g.xetStore == nil {
		return nil, errors.New("lfs gc: xet store not configured")
	}
	if !g.gcMu.TryLock() {
		return nil, git.ErrGCBusy
	}
	defer g.gcMu.Unlock()
	grace := g.gcGrace
	if opts.Grace != nil {
		grace = *opts.Grace
	}
	res, err := g.gc.Prune(ctx, hfdgc.PruneOptions{Grace: grace, DryRun: opts.DryRun})
	if errors.Is(err, hfdgc.ErrBusy) {
		err = fmt.Errorf("%w: %v", git.ErrGCBusy, err)
	}
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
	var deadline time.Time
	if opts.Budget > 0 && !opts.DryRun {
		deadline = g.now().Add(opts.Budget)
	}
	// Unlinking only drops the index entry; the sweep reclaims (or, dry, estimates) already-unlinked data.
	// LFS objects are addressed by SHA-256, so only sha256 entries anchor a shard.
	sweep, err := g.xgc.SweepStep(ctx, xetstorage.SweepOptions{
		Anchor: xetstorage.AnchorSHA256, Grace: grace, DryRun: opts.DryRun, MaxDeletes: opts.MaxDeletes, Budget: opts.Budget,
	})
	if errors.Is(err, xetstorage.ErrGCBusy) {
		return result, fmt.Errorf("%w: %v", git.ErrGCBusy, err)
	}
	if err != nil {
		return result, err
	}
	result.SweptShards = len(sweep.SweptShards)
	result.SweptXorbs = len(sweep.SweptXorbs)
	result.XetReclaimedBytes = sweep.ReclaimedBytes
	result.SweepSkippedInGrace = sweep.SkippedInGrace
	result.Dangling = sweep.DanglingSHA256Entries
	result.UnreadableShards = sweep.UnreadableShards
	result.SweepDone = &sweep.Done
	result.RemainingShards = sweep.RemainingShards
	result.RemainingXorbs = sweep.RemainingXorbs
	if !sweep.Done {
		return result, nil
	}

	swept := len(sweep.SweptShards) + len(sweep.SweptXorbs)
	// handDown returns the budget left for the next sweep; false defers it because this run already swept and spent its budget.
	handDown := func() (time.Duration, bool) {
		if deadline.IsZero() {
			return 0, true
		}
		if left := deadline.Sub(g.now()); left > 0 {
			return left, true
		}
		if swept > 0 {
			return 0, false
		}
		return time.Nanosecond, true
	}
	if b, ok := handDown(); ok && g.spool != nil {
		sp, err := g.spool.Sweep(ctx, spool.SweepOptions{Grace: grace, DryRun: opts.DryRun, Budget: b})
		if err != nil {
			return result, fmt.Errorf("spool sweep: %w", err)
		}
		result.SpoolSweep = &git.SpoolSweepResult{
			SweptSpools:     sp.SweptSpools,
			ReclaimedBytes:  sp.ReclaimedBytes,
			Done:            sp.Done,
			RemainingSpools: sp.RemainingSpools,
		}
		swept += sp.SweptSpools
	}
	if b, ok := handDown(); ok && g.xmirror != nil {
		mr, err := g.xmirror.Sweep(ctx, xetmirror.SweepOptions{Grace: grace, DryRun: opts.DryRun, Budget: b})
		if err != nil {
			return result, fmt.Errorf("mirror index sweep: %w", err)
		}
		result.MirrorSweep = &git.MirrorSweepResult{
			DroppedEntries:     mr.DroppedEntries,
			RemovedManifests:   mr.RemovedManifests,
			RemovedTempFiles:   mr.RemovedTempFiles,
			Done:               mr.Done,
			RemainingTempFiles: mr.RemainingTempFiles,
			RemainingManifests: mr.RemainingManifests,
		}
	}
	return result, nil
}

// Usage reports Git, xet, spool and mirror-index storage by kind; a category stays zero without its component.
func (g *gitRepo) Usage(ctx context.Context) (*git.StorageUsage, error) {
	usage, err := g.gc.Usage(ctx)
	if err != nil {
		return nil, err
	}
	out := &git.StorageUsage{
		Git: git.GitUsage{
			Objects: git.ObjectUsage(usage.Objects),
			Other:   git.ObjectUsage(usage.Other),
		},
	}
	if g.xetStore != nil {
		xu, err := g.xetStore.Usage(ctx)
		if err != nil {
			return nil, fmt.Errorf("xet usage: %w", err)
		}
		out.Xet = git.XetUsage{
			Xorbs:       git.ObjectUsage(xu.Xorbs),
			Shards:      git.ObjectUsage(xu.Shards),
			FileIndex:   git.ObjectUsage(xu.FileIndex),
			ChunkIndex:  git.ObjectUsage(xu.ChunkIndex),
			SHA256Index: git.ObjectUsage(xu.SHA256Index),
		}
	}
	if g.spool != nil {
		su, err := g.spool.Usage(ctx)
		if err != nil {
			return nil, fmt.Errorf("spool usage: %w", err)
		}
		out.Spool = git.ObjectUsage(su)
	}
	if g.xmirror != nil {
		mu, err := g.xmirror.Usage(ctx)
		if err != nil {
			return nil, fmt.Errorf("mirror usage: %w", err)
		}
		out.Mirror = git.ObjectUsage(mu)
	}
	return out, nil
}
