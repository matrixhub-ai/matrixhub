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

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

// CleanupService implements the cleanup service.
type CleanupService struct {
	modelRepo   model.IModelRepo
	datasetRepo dataset.IDatasetRepo
	gitRepo     git.IGitRepo
}

// ICleanupService defines the service interface for cleanup operations.
type ICleanupService interface {
	// ExecuteCleanup removes orphaned repositories when CleanOrphanedRepos is set, then prunes unreferenced LFS objects when CleanOrphanedLFS is set; DryRun only reports them.
	ExecuteCleanup(ctx context.Context, opts CleanupOptions) (*CleanupResult, error)
	// GetStorageStats returns storage statistics.
	GetStorageStats(ctx context.Context) (*StorageStats, error)
}

// NewCleanupService creates a new CleanupService instance.
func NewCleanupService(modelRepo model.IModelRepo, datasetRepo dataset.IDatasetRepo, gitRepo git.IGitRepo) ICleanupService {
	return &CleanupService{
		modelRepo:   modelRepo,
		datasetRepo: datasetRepo,
		gitRepo:     gitRepo,
	}
}

// ExecuteCleanup removes orphaned repositories when CleanOrphanedRepos is set, then prunes unreferenced LFS objects when CleanOrphanedLFS is set; DryRun only reports them.
func (s *CleanupService) ExecuteCleanup(ctx context.Context, opts CleanupOptions) (*CleanupResult, error) {
	result := &CleanupResult{}
	if opts.CleanOrphanedRepos {
		modelPaths, err := s.modelRepo.ListAllPaths(ctx)
		if err != nil {
			return nil, err
		}
		datasetPaths, err := s.datasetRepo.ListAllPaths(ctx)
		if err != nil {
			return nil, err
		}
		repos, err := s.gitRepo.PruneRepos(ctx, modelPaths, datasetPaths, opts.DryRun)
		if err != nil && opts.DryRun {
			return nil, err
		}
		for _, repo := range repos {
			result.OrphanedRepos = append(result.OrphanedRepos, repo.Path)
			result.SpaceReclaimed += repo.SizeBytes
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	if opts.CleanOrphanedLFS {
		res, err := s.gitRepo.Prune(ctx, opts.PruneOptions)
		if err != nil && opts.DryRun {
			return nil, err
		}
		result.GC = res
		if res != nil {
			result.SpaceReclaimed += res.GitReclaimedBytes + res.XetReclaimedBytes
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result, nil
}

// GetStorageStats returns storage statistics.
func (s *CleanupService) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	usage, err := s.gitRepo.Usage(ctx)
	if err != nil {
		return nil, err
	}
	stats := &StorageStats{Git: usage.Git, Xet: usage.Xet}
	for _, u := range []git.ObjectUsage{
		usage.Git.Objects, usage.Git.Other,
		usage.Xet.Xorbs, usage.Xet.Shards, usage.Xet.FileIndex, usage.Xet.ChunkIndex, usage.Xet.SHA256Index,
	} {
		stats.TotalSizeBytes += u.Bytes
	}
	return stats, nil
}
