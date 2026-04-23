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
	"os"
	"path/filepath"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
)

// CleanupService implements the cleanup service.
type CleanupService struct {
	cleanupRepo ICleanupRepo
	storage     *gitstorage.Storage
	dataDir     string
}

// NewCleanupService creates a new CleanupService instance.
func NewCleanupService(repo ICleanupRepo, storage *gitstorage.Storage, dataDir string) ICleanupService {
	return &CleanupService{
		cleanupRepo: repo,
		storage:     storage,
		dataDir:     dataDir,
	}
}

// PreviewCleanup previews orphaned data without deleting.
func (s *CleanupService) PreviewCleanup(ctx context.Context, includeRepos, includeLFS bool) (*CleanupPreview, error) {
	preview := &CleanupPreview{}

	if includeRepos {
		orphanedRepos, err := s.findOrphanedRepos(ctx)
		if err != nil {
			return nil, err
		}
		preview.OrphanedRepos = orphanedRepos
		for _, repo := range orphanedRepos {
			preview.TotalReclaimable += repo.SizeBytes
		}
	}

	if includeLFS {
		orphanedLFS, err := s.findOrphanedLFS(ctx)
		if err != nil {
			return nil, err
		}
		preview.OrphanedLFSObjects = orphanedLFS
		for _, obj := range orphanedLFS {
			preview.TotalReclaimable += obj.SizeBytes
		}
	}

	return preview, nil
}

// findOrphanedRepos finds orphaned Git repositories on disk.
func (s *CleanupService) findOrphanedRepos(ctx context.Context) ([]*OrphanedRepo, error) {
	// Get all valid paths from database
	validModelPaths, err := s.cleanupRepo.ListAllModelPaths(ctx)
	if err != nil {
		return nil, err
	}
	validDatasetPaths, err := s.cleanupRepo.ListAllDatasetPaths(ctx)
	if err != nil {
		return nil, err
	}

	// Build valid paths map
	// Note: Disk paths have ".git" suffix (bare repository naming convention),
	// so we add ".git" suffix to database paths for correct comparison.
	validPaths := make(map[string]bool)
	for _, p := range validModelPaths {
		validPaths[p+".git"] = true
	}
	for _, p := range validDatasetPaths {
		validPaths["datasets/"+p+".git"] = true
	}

	// Scan repositories directory
	reposDir := filepath.Join(s.dataDir, "repositories")
	orphaned := []*OrphanedRepo{}

	err = filepath.Walk(reposDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}

		// Check if it's a Git repository
		if !repository.IsRepository(path) {
			return nil
		}

		// Parse relative path
		relPath := strings.TrimPrefix(path, reposDir+"/")

		// Skip this repository's subdirectories (like logs/, objects/, refs/, etc.)
		// to avoid misidentifying internal Git directories as separate repositories.
		// Return filepath.SkipDir AFTER checking if it's a repository.
		if validPaths[relPath] {
			// Valid repository, skip it and all its subdirectories
			return filepath.SkipDir
		}

		// Parse type and names
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

		// Calculate directory size
		size := s.calculateDirSize(path)

		orphaned = append(orphaned, &OrphanedRepo{
			Path:         relPath,
			Type:         repoType,
			ProjectName:  projectName,
			ResourceName: resourceName,
			SizeBytes:    size,
		})

		// Skip subdirectories of this orphaned repository
		return filepath.SkipDir
	})

	return orphaned, err
}

// calculateDirSize calculates the total size of a directory.
func (s *CleanupService) calculateDirSize(path string) int64 {
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

// findOrphanedLFS finds orphaned LFS objects on disk.
func (s *CleanupService) findOrphanedLFS(ctx context.Context) ([]*OrphanedLFS, error) {
	detector := NewLFSOrphanDetector(s.dataDir, s.storage.LFSDir())
	return detector.DetectOrphanedLFS(ctx)
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
				fullPath := filepath.Join(s.dataDir, "repositories", repo.Path)
				if err := os.RemoveAll(fullPath); err != nil {
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
				if err := os.Remove(obj.Path); err != nil {
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

	// Calculate repositories size
	reposDir := filepath.Join(s.dataDir, "repositories")
	stats.RepositoriesSizeBytes = s.calculateDirSize(reposDir)

	// Calculate LFS size
	lfsDir := s.storage.LFSDir()
	stats.LFSSizeBytes = s.calculateDirSize(lfsDir)

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
