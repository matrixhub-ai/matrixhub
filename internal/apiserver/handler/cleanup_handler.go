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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/cleanup"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
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

// ExecuteCleanup executes cleanup based on options.
func (h *CleanupHandler) ExecuteCleanup(ctx context.Context, req *v1alpha1.ExecuteCleanupRequest) (*v1alpha1.CleanupResult, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	opts := cleanup.CleanupOptions{
		CleanOrphanedRepos: req.CleanOrphanedRepos,
		CleanOrphanedLFS:   req.CleanOrphanedLfs,
		PruneOptions: git.PruneOptions{
			DryRun:     req.DryRun,
			MaxDeletes: int(req.MaxDeletes),
			Budget:     req.Budget.AsDuration(),
		},
	}
	if req.Grace != nil {
		grace := req.Grace.AsDuration()
		if grace == 0 {
			grace = -1 // an explicit "0s" disables the grace window; zero would select the library default
		}
		opts.Grace = &grace
	}
	result, err := h.cleanupService.ExecuteCleanup(ctx, opts)
	if err != nil {
		return nil, err
	}

	res := &v1alpha1.CleanupResult{
		SpaceReclaimedBytes: result.SpaceReclaimed,
		Errors:              result.Errors,
		OrphanedRepos:       result.OrphanedRepos,
	}
	if gc := result.GC; gc != nil {
		res.Gc = &v1alpha1.GCResult{
			DryRun:              gc.DryRun,
			Repositories:        int32(gc.Repositories),
			DeletedGitObjects:   int32(gc.DeletedGitObjects),
			DeletedGitBytes:     gc.DeletedGitBytes,
			GitReclaimedBytes:   gc.GitReclaimedBytes,
			Failed:              gc.Failed,
			LiveObjects:         int32(gc.LiveObjects),
			Unlinked:            gc.Unlinked,
			PruneSkippedInGrace: int32(gc.PruneSkippedInGrace),
			SweptShards:         int32(gc.SweptShards),
			SweptXorbs:          int32(gc.SweptXorbs),
			XetReclaimedBytes:   gc.XetReclaimedBytes,
			SweepSkippedInGrace: int32(gc.SweepSkippedInGrace),
			Dangling:            gc.Dangling,
			UnreadableShards:    gc.UnreadableShards,
			RemainingShards:     int32(gc.RemainingShards),
			RemainingXorbs:      int32(gc.RemainingXorbs),
		}
		if gc.SweepDone != nil {
			res.Gc.SweepDone = wrapperspb.Bool(*gc.SweepDone)
		}
	}
	return res, nil
}

// GetStorageStats returns storage statistics.
func (h *CleanupHandler) GetStorageStats(ctx context.Context, req *v1alpha1.GetStorageStatsRequest) (*v1alpha1.StorageStats, error) {
	stats, err := h.cleanupService.GetStorageStats(ctx)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.StorageStats{
		TotalSizeBytes: stats.TotalSizeBytes,
		Git: &v1alpha1.GitStorageUsage{
			Objects: storageObjectUsage(stats.Git.Objects),
			Other:   storageObjectUsage(stats.Git.Other),
		},
		Xet: &v1alpha1.XetStorageUsage{
			Xorbs:       storageObjectUsage(stats.Xet.Xorbs),
			Shards:      storageObjectUsage(stats.Xet.Shards),
			FileIndex:   storageObjectUsage(stats.Xet.FileIndex),
			ChunkIndex:  storageObjectUsage(stats.Xet.ChunkIndex),
			Sha256Index: storageObjectUsage(stats.Xet.SHA256Index),
		},
	}, nil
}

func storageObjectUsage(u git.ObjectUsage) *v1alpha1.StorageObjectUsage {
	return &v1alpha1.StorageObjectUsage{Count: u.Count, Bytes: u.Bytes}
}
