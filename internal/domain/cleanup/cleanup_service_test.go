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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	datasetmocks "github.com/matrixhub-ai/matrixhub/internal/domain/dataset/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
)

func TestCleanupServiceExecuteNoSelectionIsNoOp(t *testing.T) {
	ctx := context.Background()
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%t", dryRun), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitmocks.NewMockIGitRepo(ctrl))

			result, err := service.ExecuteCleanup(ctx, CleanupOptions{PruneOptions: git.PruneOptions{DryRun: dryRun}})
			require.NoError(t, err)
			require.Equal(t, &CleanupResult{}, result)
		})
	}
}

func TestCleanupServiceExecuteDelegatesDeletionToStoragePort(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	hour := time.Hour
	opts := git.PruneOptions{Grace: &hour, MaxDeletes: 3, Budget: time.Minute}
	done := true
	pruned := &git.GCResult{Unlinked: []string{"12345678"}, GitReclaimedBytes: 12, XetReclaimedBytes: 15, SweepDone: &done}

	gitRepo.EXPECT().Prune(ctx, opts).DoAndReturn(func(_ context.Context, got git.PruneOptions) (*git.GCResult, error) {
		require.Same(t, &hour, got.Grace)
		return pruned, nil
	})

	service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitRepo)

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedLFS: true, PruneOptions: opts})
	require.NoError(t, err)
	require.Same(t, pruned, result.GC)
	require.Empty(t, result.OrphanedRepos)
	require.EqualValues(t, 27, result.SpaceReclaimed)
	require.Empty(t, result.Errors)
}

func TestCleanupServiceExecuteDryRun(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	done := true
	pruned := &git.GCResult{DryRun: true, Unlinked: []string{"12345678"}, XetReclaimedBytes: 20, SweepDone: &done}
	gitRepo.EXPECT().Prune(ctx, git.PruneOptions{DryRun: true}).Return(pruned, nil)

	service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitRepo)

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{DryRun: true}})
	require.NoError(t, err)
	require.Same(t, pruned, result.GC)
	require.EqualValues(t, 20, result.SpaceReclaimed)
	require.Empty(t, result.Errors)
}

func TestCleanupServiceExecuteCleansOrphanedRepos(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	modelRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/m"}, nil)
	datasetRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/d"}, nil)
	gitRepo.EXPECT().PruneRepos(ctx, []string{"p/m"}, []string{"p/d"}, false).Return([]*git.OrphanedRepo{{Path: "p/orphan.git", SizeBytes: 100}}, nil)
	gitRepo.EXPECT().Prune(ctx, git.PruneOptions{}).Return(&git.GCResult{Unlinked: []string{"12345678"}, XetReclaimedBytes: 15}, nil)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedRepos: true, CleanOrphanedLFS: true})
	require.NoError(t, err)
	require.Equal(t, []string{"p/orphan.git"}, result.OrphanedRepos)
	require.Equal(t, []string{"12345678"}, result.GC.Unlinked)
	require.EqualValues(t, 115, result.SpaceReclaimed)
	require.Empty(t, result.Errors)
}

// Preview is ExecuteCleanup with DryRun; the port results must be forwarded unchanged and only summed.
func TestCleanupServicePreviewUsesDomainPorts(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	modelPaths := []string{"project/model"}
	datasetPaths := []string{"project/dataset"}
	orphanedRepos := []*git.OrphanedRepo{{Path: "orphan/model.git", SizeBytes: 10}}
	done := true
	pruned := &git.GCResult{DryRun: true, Unlinked: []string{"12345678"}, XetReclaimedBytes: 20, SweepDone: &done}
	gomock.InOrder(
		modelRepo.EXPECT().ListAllPaths(ctx).Return(modelPaths, nil),
		datasetRepo.EXPECT().ListAllPaths(ctx).Return(datasetPaths, nil),
		gitRepo.EXPECT().PruneRepos(ctx, modelPaths, datasetPaths, true).Return(orphanedRepos, nil),
		gitRepo.EXPECT().Prune(ctx, git.PruneOptions{DryRun: true}).Return(pruned, nil),
	)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedRepos: true, CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{DryRun: true}})
	require.NoError(t, err)
	require.Equal(t, []string{"orphan/model.git"}, result.OrphanedRepos)
	require.Same(t, pruned, result.GC)
	require.EqualValues(t, 30, result.SpaceReclaimed)
	require.Empty(t, result.Errors)
}

func TestCleanupServiceExecuteReposOnlySkipsLFS(t *testing.T) {
	ctx := context.Background()
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%t", dryRun), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			modelRepo := modelmocks.NewMockIModelRepo(ctrl)
			datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
			gitRepo := gitmocks.NewMockIGitRepo(ctrl)

			modelRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/m"}, nil)
			datasetRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/d"}, nil)
			gitRepo.EXPECT().PruneRepos(ctx, []string{"p/m"}, []string{"p/d"}, dryRun).Return([]*git.OrphanedRepo{{Path: "p/orphan.git", SizeBytes: 100}}, nil)

			service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

			result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedRepos: true, PruneOptions: git.PruneOptions{DryRun: dryRun}})
			require.NoError(t, err)
			require.Equal(t, []string{"p/orphan.git"}, result.OrphanedRepos)
			require.Nil(t, result.GC)
			require.EqualValues(t, 100, result.SpaceReclaimed)
			require.Empty(t, result.Errors)
		})
	}
}

func TestCleanupServiceExecuteContinuesAfterRepoPruneError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	gomock.InOrder(
		modelRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/m"}, nil),
		datasetRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/d"}, nil),
		gitRepo.EXPECT().PruneRepos(ctx, []string{"p/m"}, []string{"p/d"}, false).Return([]*git.OrphanedRepo{{Path: "p/orphan.git", SizeBytes: 100}}, errors.New("walk failed")),
		gitRepo.EXPECT().Prune(ctx, git.PruneOptions{}).Return(&git.GCResult{Unlinked: []string{"12345678"}, XetReclaimedBytes: 15}, nil),
	)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedRepos: true, CleanOrphanedLFS: true})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"p/orphan.git"}, result.OrphanedRepos)
	require.Equal(t, []string{"12345678"}, result.GC.Unlinked)
	require.EqualValues(t, 115, result.SpaceReclaimed)
	require.Equal(t, []string{"walk failed"}, result.Errors)
}

// A prune that fails after unlinking returns its partial result with no sweep result; a dry run fails hard instead.
func TestCleanupServiceExecuteReportsPartialLFSPrune(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	partial := &git.GCResult{Unlinked: []string{"12345678"}, GitReclaimedBytes: 9}
	gitRepo.EXPECT().Prune(ctx, git.PruneOptions{}).Return(partial, errors.New("sweep failed"))
	gitRepo.EXPECT().Prune(ctx, git.PruneOptions{DryRun: true}).Return(nil, errors.New("gc busy"))

	service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitRepo)

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedLFS: true})
	require.NoError(t, err)
	require.Same(t, partial, result.GC)
	require.Nil(t, result.GC.SweepDone)
	require.EqualValues(t, 9, result.SpaceReclaimed)
	require.Equal(t, []string{"sweep failed"}, result.Errors)

	result, err = service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedLFS: true, PruneOptions: git.PruneOptions{DryRun: true}})
	require.EqualError(t, err, "gc busy")
	require.Nil(t, result)
}

func TestCleanupServiceExecuteFailsClosedWhenPathsUnavailable(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	modelRepo.EXPECT().ListAllPaths(ctx).Return(nil, errors.New("db down"))

	service := NewCleanupService(modelRepo, datasetmocks.NewMockIDatasetRepo(ctrl), gitmocks.NewMockIGitRepo(ctrl))

	result, err := service.ExecuteCleanup(ctx, CleanupOptions{CleanOrphanedRepos: true, CleanOrphanedLFS: true})
	require.Error(t, err)
	require.Nil(t, result)
}

// Stats come from one Usage call; every category, index entries included, counts toward the total.
func TestCleanupServiceGetStorageStatsDoesNotRunCleanup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	usage := &git.StorageUsage{
		Git: git.GitUsage{Objects: git.ObjectUsage{Count: 1, Bytes: 10}, Other: git.ObjectUsage{Count: 2, Bytes: 20}},
		Xet: git.XetUsage{
			Xorbs:       git.ObjectUsage{Count: 3, Bytes: 30},
			Shards:      git.ObjectUsage{Count: 4, Bytes: 40},
			FileIndex:   git.ObjectUsage{Count: 5, Bytes: 50},
			ChunkIndex:  git.ObjectUsage{Count: 6, Bytes: 60},
			SHA256Index: git.ObjectUsage{Count: 7, Bytes: 70},
		},
	}
	gitRepo.EXPECT().Usage(ctx).Return(usage, nil)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	stats, err := service.GetStorageStats(ctx)
	require.NoError(t, err)
	require.Equal(t, &StorageStats{TotalSizeBytes: 280, Git: usage.Git, Xet: usage.Xet}, stats)
}

func TestCleanupServiceGetStorageStatsPropagatesUsageError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	gitRepo.EXPECT().Usage(ctx).Return(nil, errors.New("xet usage: store offline"))

	service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitRepo)

	stats, err := service.GetStorageStats(ctx)
	require.EqualError(t, err, "xet usage: store offline")
	require.Nil(t, stats)
}
