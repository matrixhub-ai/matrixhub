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

			result, err := service.ExecuteCleanup(ctx, false, false, dryRun)
			require.NoError(t, err)
			require.Equal(t, &CleanupResult{}, result)
		})
	}
}

func TestCleanupServiceExecuteDelegatesDeletionToStoragePort(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	orphanedLFS := &git.OrphanedLFS{OID: "12345678", SizeBytes: 20}

	gitRepo.EXPECT().Prune(ctx, false).Return(&git.PruneResult{Unlinked: []*git.OrphanedLFS{orphanedLFS}, ReclaimedBytes: 15}, nil)

	service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitRepo)

	result, err := service.ExecuteCleanup(ctx, false, true, false)
	require.NoError(t, err)
	require.Equal(t, 1, result.LFSObjectsDeleted)
	require.EqualValues(t, 15, result.SpaceReclaimed)
	require.Empty(t, result.Errors)
}

func TestCleanupServiceExecuteDryRun(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	orphanedLFS := &git.OrphanedLFS{OID: "12345678", SizeBytes: 20}
	gitRepo.EXPECT().Prune(ctx, true).Return(&git.PruneResult{Unlinked: []*git.OrphanedLFS{orphanedLFS}}, nil)

	service := NewCleanupService(modelmocks.NewMockIModelRepo(ctrl), datasetmocks.NewMockIDatasetRepo(ctrl), gitRepo)

	result, err := service.ExecuteCleanup(ctx, false, true, true)
	require.NoError(t, err)
	require.Equal(t, 1, result.LFSObjectsDeleted)
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
	gitRepo.EXPECT().Prune(ctx, false).Return(&git.PruneResult{Unlinked: []*git.OrphanedLFS{{OID: "12345678", SizeBytes: 20}}, ReclaimedBytes: 15}, nil)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	result, err := service.ExecuteCleanup(ctx, true, true, false)
	require.NoError(t, err)
	require.Equal(t, 1, result.ReposDeleted)
	require.Equal(t, 1, result.LFSObjectsDeleted)
	require.EqualValues(t, 115, result.SpaceReclaimed)
	require.Empty(t, result.Errors)
}

// Preview is ExecuteCleanup with dryRun; the port results must be forwarded unchanged and only summed.
func TestCleanupServicePreviewUsesDomainPorts(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	modelPaths := []string{"project/model"}
	datasetPaths := []string{"project/dataset"}
	orphanedRepos := []*git.OrphanedRepo{{Path: "orphan/model.git", SizeBytes: 10}}
	orphanedLFS := []*git.OrphanedLFS{{OID: "12345678", SizeBytes: 20}}
	gomock.InOrder(
		modelRepo.EXPECT().ListAllPaths(ctx).Return(modelPaths, nil),
		datasetRepo.EXPECT().ListAllPaths(ctx).Return(datasetPaths, nil),
		gitRepo.EXPECT().PruneRepos(ctx, modelPaths, datasetPaths, true).Return(orphanedRepos, nil),
		gitRepo.EXPECT().Prune(ctx, true).Return(&git.PruneResult{Unlinked: orphanedLFS}, nil),
	)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	result, err := service.ExecuteCleanup(ctx, true, true, true)
	require.NoError(t, err)
	require.Equal(t, 1, result.ReposDeleted)
	require.Equal(t, 1, result.LFSObjectsDeleted)
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

			result, err := service.ExecuteCleanup(ctx, true, false, dryRun)
			require.NoError(t, err)
			require.Equal(t, 1, result.ReposDeleted)
			require.Zero(t, result.LFSObjectsDeleted)
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
		gitRepo.EXPECT().Prune(ctx, false).Return(&git.PruneResult{Unlinked: []*git.OrphanedLFS{{OID: "12345678", SizeBytes: 20}}, ReclaimedBytes: 15}, nil),
	)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	result, err := service.ExecuteCleanup(ctx, true, true, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ReposDeleted)
	require.Equal(t, 1, result.LFSObjectsDeleted)
	require.EqualValues(t, 115, result.SpaceReclaimed)
	require.Equal(t, []string{"walk failed"}, result.Errors)
}

func TestCleanupServiceExecuteFailsClosedWhenPathsUnavailable(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	modelRepo.EXPECT().ListAllPaths(ctx).Return(nil, errors.New("db down"))

	service := NewCleanupService(modelRepo, datasetmocks.NewMockIDatasetRepo(ctrl), gitmocks.NewMockIGitRepo(ctrl))

	result, err := service.ExecuteCleanup(ctx, true, true, false)
	require.Error(t, err)
	require.Nil(t, result)
}

func TestCleanupServiceGetStorageStats(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	datasetRepo := datasetmocks.NewMockIDatasetRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)

	gitRepo.EXPECT().RepositoriesSize(ctx).Return(int64(1000))
	gitRepo.EXPECT().LFSSize(ctx).Return(int64(500))
	modelRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/m"}, nil)
	datasetRepo.EXPECT().ListAllPaths(ctx).Return([]string{"p/d"}, nil)
	gitRepo.EXPECT().PruneRepos(ctx, []string{"p/m"}, []string{"p/d"}, true).Return([]*git.OrphanedRepo{{Path: "p/orphan.git", SizeBytes: 100}}, nil)
	gitRepo.EXPECT().Prune(ctx, true).Return(&git.PruneResult{Unlinked: []*git.OrphanedLFS{{OID: "12345678", SizeBytes: 20}}}, nil)

	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	stats, err := service.GetStorageStats(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1000, stats.RepositoriesSizeBytes)
	require.EqualValues(t, 500, stats.LFSSizeBytes)
	require.EqualValues(t, 1500, stats.TotalSizeBytes)
	require.EqualValues(t, 120, stats.OrphanedSizeBytes)
}
