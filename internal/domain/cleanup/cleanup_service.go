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
		preview.OrphanedRepos = make([]*OrphanedRepo, len(orphanedRepos))
		for i, repo := range orphanedRepos {
			preview.OrphanedRepos[i] = &OrphanedRepo{
				Path:         repo.Path,
				Type:         repo.Type,
				ProjectName:  repo.ProjectName,
				ResourceName: repo.ResourceName,
				SizeBytes:    repo.SizeBytes,
			}
			preview.TotalReclaimable += preview.OrphanedRepos[i].SizeBytes
		}
	}

	if includeLFS {
		orphanedLFS, err := s.gitRepo.FindOrphanedLFS(ctx)
		if err != nil {
			return nil, err
		}
		preview.OrphanedLFSObjects = make([]*OrphanedLFS, len(orphanedLFS))
		for i, obj := range orphanedLFS {
			preview.OrphanedLFSObjects[i] = &OrphanedLFS{
				OID:       obj.OID,
				SizeBytes: obj.SizeBytes,
				Path:      obj.Path,
			}
			preview.TotalReclaimable += preview.OrphanedLFSObjects[i].SizeBytes
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
				if err := s.gitRepo.DeleteRepo(ctx, repo.Path); err != nil {
					result.Errors = append(result.Errors, err.Error())
				} else {
					result.ReposDeleted++
					result.SpaceReclaimed += repo.SizeBytes
				}
			}
		}
	}

	if cleanLFS {
		preview, err := s.PreviewCleanup(ctx, false, true)
		if err != nil {
			return nil, err
		}
		for _, obj := range preview.OrphanedLFSObjects {
			if dryRun {
				result.LFSObjectsDeleted++
				result.SpaceReclaimed += obj.SizeBytes
			} else {
				if err := s.gitRepo.DeleteLFSObject(ctx, &git.OrphanedLFS{
					OID:       obj.OID,
					SizeBytes: obj.SizeBytes,
					Path:      obj.Path,
				}); err != nil {
					result.Errors = append(result.Errors, err.Error())
				} else {
					result.LFSObjectsDeleted++
					result.SpaceReclaimed += obj.SizeBytes
				}
			}
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
	preview, err := s.PreviewCleanup(ctx, true, true)
	if err != nil {
		return nil, err
	}
	stats.OrphanedSizeBytes = preview.TotalReclaimable

	// Total
	stats.TotalSizeBytes = stats.RepositoriesSizeBytes + stats.LFSSizeBytes

	return stats, nil
}
