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
)

// OrphanedRepo represents an orphaned Git repository on disk
// that has no corresponding record in the database.
type OrphanedRepo struct {
	Path         string // Relative path from repositories directory
	Type         string // "model" or "dataset"
	ProjectName  string
	ResourceName string
	SizeBytes    int64
}

// OrphanedLFS represents an orphaned LFS object on disk
// that is not referenced by any Git repository.
type OrphanedLFS struct {
	OID       string // SHA256 hash
	SizeBytes int64
	Path      string // File system path
}

// CleanupPreview contains preview results for orphaned data.
type CleanupPreview struct {
	OrphanedRepos      []*OrphanedRepo
	OrphanedLFSObjects []*OrphanedLFS
	TotalReclaimable   int64
}

// CleanupResult contains results from cleanup execution.
type CleanupResult struct {
	ReposDeleted      int
	LFSObjectsDeleted int
	SpaceReclaimed    int64
	Errors            []string
}

// StorageStats contains storage statistics.
type StorageStats struct {
	TotalSizeBytes        int64
	RepositoriesSizeBytes int64
	LFSSizeBytes          int64
	OrphanedSizeBytes     int64
}

// ICleanupRepo defines the repository interface for cleanup operations.
type ICleanupRepo interface {
	// ListAllModelPaths returns all valid model paths (project/name format) from database.
	ListAllModelPaths(ctx context.Context) ([]string, error)
	// ListAllDatasetPaths returns all valid dataset paths from database.
	ListAllDatasetPaths(ctx context.Context) ([]string, error)
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
