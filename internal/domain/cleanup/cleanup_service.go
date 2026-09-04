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
	// PreviewCleanup previews orphaned data without deleting.
	PreviewCleanup(ctx context.Context, includeRepos, includeLFS bool) (*CleanupPreview, error)
	// ExecuteCleanup executes cleanup based on options.
	ExecuteCleanup(ctx context.Context, cleanRepos, cleanLFS bool, dryRun bool) (*CleanupResult, error)
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

// PreviewCleanup previews orphaned data without deleting.
func (s *CleanupService) PreviewCleanup(ctx context.Context, includeRepos, includeLFS bool) (*CleanupPreview, error) {
	preview := &CleanupPreview{}

	if includeRepos {
		validModelPaths, err := s.modelRepo.ListAllPaths(ctx)
		if err != nil {
			return nil, err
		}
		validDatasetPaths, err := s.datasetRepo.ListAllPaths(ctx)
		if err != nil {
			return nil, err
		}
		orphanedRepos, err := s.gitRepo.FindOrphanedRepos(ctx, validModelPaths, validDatasetPaths)
		if err != nil {
			return nil, err
		}
		preview.OrphanedRepos = orphanedRepos
		for _, repo := range orphanedRepos {
			preview.TotalReclaimable += repo.SizeBytes
		}
	}

	if includeLFS {
		res, err := s.gitRepo.CollectLFS(ctx, true)
		if err != nil {
			return nil, err
		}
		preview.OrphanedLFSObjects = res.Orphaned
		for _, obj := range res.Orphaned {
			preview.TotalReclaimable += obj.SizeBytes
		}
	}

	return preview, nil
}

// ExecuteCleanup executes cleanup based on options.
func (s *CleanupService) ExecuteCleanup(ctx context.Context, cleanRepos, cleanLFS bool, dryRun bool) (*CleanupResult, error) {
	result := &CleanupResult{}

	if cleanRepos {
		preview, err := s.PreviewCleanup(ctx, true, false)
		if err != nil {
			return nil, err
		}
		for _, repo := range preview.OrphanedRepos {
			if dryRun {
				result.ReposDeleted++
				result.SpaceReclaimed += repo.SizeBytes
			} else {
				if err := s.gitRepo.DeleteRepositoryAtRelPath(ctx, repo.Path); err != nil {
					result.Errors = append(result.Errors, err.Error())
				} else {
					result.ReposDeleted++
					result.SpaceReclaimed += repo.SizeBytes
				}
			}
		}
	}

	if cleanLFS {
		res, err := s.gitRepo.CollectLFS(ctx, dryRun)
		if res == nil {
			return nil, err
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
		result.LFSObjectsDeleted += len(res.Orphaned)
		result.SpaceReclaimed += res.ReclaimedBytes
	}

	return result, nil
}

// GetStorageStats returns storage statistics.
func (s *CleanupService) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	stats := &StorageStats{}

	stats.RepositoriesSizeBytes = s.gitRepo.RepositoriesSize(ctx)
	stats.LFSSizeBytes = s.gitRepo.LFSSize(ctx)

	// Calculate orphaned size
	preview, err := s.PreviewCleanup(ctx, true, true)
	if err != nil {
		return nil, err
	}
	stats.OrphanedSizeBytes = preview.TotalReclaimable

	// Total
	stats.TotalSizeBytes = stats.RepositoriesSizeBytes + stats.LFSSizeBytes

	return stats, nil
}
