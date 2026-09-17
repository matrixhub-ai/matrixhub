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
	// ExecuteCleanup removes orphaned repositories when cleanRepos is set, then prunes unreferenced LFS objects when cleanLFS is set; dryRun only counts them.
	ExecuteCleanup(ctx context.Context, cleanRepos, cleanLFS, dryRun bool) (*CleanupResult, error)
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

// ExecuteCleanup removes orphaned repositories when cleanRepos is set, then prunes unreferenced LFS objects when cleanLFS is set; dryRun only counts them.
func (s *CleanupService) ExecuteCleanup(ctx context.Context, cleanRepos, cleanLFS, dryRun bool) (*CleanupResult, error) {
	result := &CleanupResult{}
	if cleanRepos {
		modelPaths, err := s.modelRepo.ListAllPaths(ctx)
		if err != nil {
			return nil, err
		}
		datasetPaths, err := s.datasetRepo.ListAllPaths(ctx)
		if err != nil {
			return nil, err
		}
		repos, err := s.gitRepo.PruneRepos(ctx, modelPaths, datasetPaths, dryRun)
		if err != nil && dryRun {
			return nil, err
		}
		result.ReposDeleted = len(repos)
		for _, repo := range repos {
			result.SpaceReclaimed += repo.SizeBytes
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	if cleanLFS {
		res, err := s.gitRepo.Prune(ctx, dryRun)
		if err != nil && dryRun {
			return nil, err
		}
		if res != nil {
			result.LFSObjectsDeleted = len(res.Unlinked)
			result.SpaceReclaimed += res.ReclaimedBytes
			if dryRun {
				// Nothing is swept in a dry run; report the objects' own sizes instead.
				for _, object := range res.Unlinked {
					result.SpaceReclaimed += object.SizeBytes
				}
			}
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result, nil
}

// GetStorageStats returns storage statistics.
func (s *CleanupService) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	stats := &StorageStats{}

	stats.RepositoriesSizeBytes = s.gitRepo.RepositoriesSize(ctx)
	stats.LFSSizeBytes = s.gitRepo.LFSSize(ctx)

	// Calculate orphaned size
	result, err := s.ExecuteCleanup(ctx, true, true, true)
	if err != nil {
		return nil, err
	}
	stats.OrphanedSizeBytes = result.SpaceReclaimed

	// Total
	stats.TotalSizeBytes = stats.RepositoriesSizeBytes + stats.LFSSizeBytes

	return stats, nil
}
