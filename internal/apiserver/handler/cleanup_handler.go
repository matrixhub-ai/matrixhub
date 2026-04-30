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

package handler

import (
	"context"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/cleanup"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// CleanupHandler handles cleanup API requests.
type CleanupHandler struct {
	cleanupService cleanup.ICleanupService
}

// NewCleanupHandler creates a new CleanupHandler instance.
func NewCleanupHandler(cleanupService cleanup.ICleanupService) IHandler {
	return &CleanupHandler{
		cleanupService: cleanupService,
	}
}

// RegisterToServer registers the handler to gRPC and HTTP gateway.
func (h *CleanupHandler) RegisterToServer(options *ServerOptions) {
	v1alpha1.RegisterCleanupServer(options.GRPCServer, h)
	if err := v1alpha1.RegisterCleanupHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
		log.Errorf("register cleanup handler error: %s", err.Error())
	}
}

// PreviewCleanup previews orphaned data without deleting.
func (h *CleanupHandler) PreviewCleanup(ctx context.Context, req *v1alpha1.PreviewCleanupRequest) (*v1alpha1.CleanupPreview, error) {
	preview, err := h.cleanupService.PreviewCleanup(ctx, req.IncludeOrphanedRepos, req.IncludeOrphanedLfs)
	if err != nil {
		return nil, err
	}

	// Convert domain model to proto
	repos := make([]*v1alpha1.OrphanedRepo, len(preview.OrphanedRepos))
	for i, r := range preview.OrphanedRepos {
		repos[i] = &v1alpha1.OrphanedRepo{
			Path:         r.Path,
			Type:         r.Type,
			ProjectName:  r.ProjectName,
			ResourceName: r.ResourceName,
			SizeBytes:    r.SizeBytes,
		}
	}

	lfs := make([]*v1alpha1.OrphanedLFS, len(preview.OrphanedLFSObjects))
	for i, o := range preview.OrphanedLFSObjects {
		lfs[i] = &v1alpha1.OrphanedLFS{
			Oid:       o.OID,
			SizeBytes: o.SizeBytes,
		}
	}

	return &v1alpha1.CleanupPreview{
		OrphanedRepos:         repos,
		OrphanedLfsObjects:    lfs,
		TotalReclaimableBytes: preview.TotalReclaimable,
	}, nil
}

// ExecuteCleanup executes cleanup based on options.
func (h *CleanupHandler) ExecuteCleanup(ctx context.Context, req *v1alpha1.ExecuteCleanupRequest) (*v1alpha1.CleanupResult, error) {
	result, err := h.cleanupService.ExecuteCleanup(ctx, req.CleanOrphanedRepos, req.CleanOrphanedLfs, req.DryRun)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.CleanupResult{
		ReposDeleted:        int32(result.ReposDeleted),
		LfsObjectsDeleted:   int32(result.LFSObjectsDeleted),
		SpaceReclaimedBytes: result.SpaceReclaimed,
		Errors:              result.Errors,
	}, nil
}

// GetStorageStats returns storage statistics.
func (h *CleanupHandler) GetStorageStats(ctx context.Context, req *v1alpha1.GetStorageStatsRequest) (*v1alpha1.StorageStats, error) {
	stats, err := h.cleanupService.GetStorageStats(ctx)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.StorageStats{
		TotalSizeBytes:        stats.TotalSizeBytes,
		RepositoriesSizeBytes: stats.RepositoriesSizeBytes,
		LfsSizeBytes:          stats.LFSSizeBytes,
		OrphanedSizeBytes:     stats.OrphanedSizeBytes,
	}, nil
}
