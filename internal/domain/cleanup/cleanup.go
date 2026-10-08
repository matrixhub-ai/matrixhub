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

import "github.com/matrixhub-ai/matrixhub/internal/domain/git"

// CleanupOptions selects what ExecuteCleanup acts on; the embedded PruneOptions also carry DryRun for the repository step.
type CleanupOptions struct {
	CleanOrphanedRepos bool
	CleanOrphanedLFS   bool
	git.PruneOptions
}

// CleanupResult contains results from cleanup execution.
type CleanupResult struct {
	OrphanedRepos  []string // repository paths removed, or that a dry run would remove
	SpaceReclaimed int64    // orphaned repository bytes plus the GC's Git shrink and Xet reclaimed (dry run: sweepable) bytes
	GC             *git.GCResult
	Errors         []string
}

// StorageStats contains storage statistics.
type StorageStats struct {
	TotalSizeBytes int64 // every Git and xet category summed
	Git            git.GitUsage
	Xet            git.XetUsage
}
