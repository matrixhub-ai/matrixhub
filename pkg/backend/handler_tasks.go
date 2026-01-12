package backend

import (
	"context"
	"fmt"
	"log"

	"github.com/wzshiming/gitd/pkg/lfs"
	"github.com/wzshiming/gitd/pkg/queue"
	"github.com/wzshiming/gitd/pkg/repository"
)

// registerTaskHandlers registers all task handlers with the queue worker
func (h *Handler) registerTaskHandlers() {
	h.queueWorker.RegisterHandler(queue.TaskTypeMirrorSync, h.handleMirrorSyncTask)
	h.queueWorker.RegisterHandler(queue.TaskTypeLFSSync, h.handleLFSSyncTask)
}

// handleMirrorSyncTask handles a mirror sync task
func (h *Handler) handleMirrorSyncTask(ctx context.Context, task *queue.Task, progressFn queue.ProgressFunc) error {
	repoPath := h.resolveRepoPath(task.Repository)
	if repoPath == "" {
		return fmt.Errorf("repository not found: %s", task.Repository)
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	progressFn(10, "Starting git fetch...", 0, 0)

	// Perform the sync
	err = repo.SyncMirror(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync mirror: %w", err)
	}

	progressFn(50, "Git fetch completed", 0, 0)

	// Get source URL for LFS sync
	_, sourceURL, err := repo.IsMirror()
	if err != nil {
		return fmt.Errorf("failed to get mirror config: %w", err)
	}

	// Check if there are LFS objects to sync
	pointers, err := repo.ScanLFSPointers()
	if err != nil {
		log.Printf("Failed to scan LFS pointers for %s: %v\n", task.Repository, err)
		progressFn(100, "Completed (LFS scan failed)", 0, 0)
		return nil
	}

	if len(pointers) == 0 {
		progressFn(100, "Completed (no LFS objects)", 0, 0)
		return nil
	}

	// Queue LFS sync as a separate task if there are LFS objects
	if h.queueStore != nil && sourceURL != "" {
		params := map[string]string{"source_url": sourceURL}
		_, err := h.queueStore.Add(queue.TaskTypeLFSSync, task.Repository, task.Priority, params)
		if err != nil {
			log.Printf("Failed to queue LFS sync for %s: %v\n", task.Repository, err)
		}
	}

	progressFn(100, "Completed", 0, 0)
	return nil
}

// handleLFSSyncTask handles an LFS sync task
func (h *Handler) handleLFSSyncTask(ctx context.Context, task *queue.Task, progressFn queue.ProgressFunc) error {
	repoPath := h.resolveRepoPath(task.Repository)
	if repoPath == "" {
		return fmt.Errorf("repository not found: %s", task.Repository)
	}

	repo, err := repository.Open(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	sourceURL := task.Params["source_url"]
	if sourceURL == "" {
		// Try to get from repository config
		_, sourceURL, err = repo.IsMirror()
		if err != nil || sourceURL == "" {
			return fmt.Errorf("source URL not found")
		}
	}

	progressFn(10, "Scanning for LFS pointers...", 0, 0)

	// Scan repository for LFS pointers
	pointers, err := repo.ScanLFSPointers()
	if err != nil {
		return fmt.Errorf("failed to scan LFS pointers: %w", err)
	}

	if len(pointers) == 0 {
		progressFn(100, "No LFS objects to sync", 0, 0)
		return nil
	}

	// Convert to LFS objects
	objects := make([]lfs.LFSObject, len(pointers))
	var totalSize int64
	for i, ptr := range pointers {
		objects[i] = lfs.LFSObject{
			Oid:  ptr.Oid,
			Size: ptr.Size,
		}
		totalSize += ptr.Size
	}

	progressFn(20, fmt.Sprintf("Found %d LFS objects to sync", len(objects)), 0, totalSize)

	// Get LFS endpoint from source URL
	lfsEndpoint := lfs.GetLFSEndpoint(sourceURL)

	// Create remote client with progress tracking
	remoteClient := lfs.NewRemoteClient()
	err = h.fetchLFSObjectsWithProgress(ctx, remoteClient, lfsEndpoint, objects, progressFn, totalSize)
	if err != nil {
		return fmt.Errorf("failed to fetch LFS objects: %w", err)
	}

	progressFn(100, "LFS sync completed", totalSize, totalSize)
	return nil
}

// fetchLFSObjectsWithProgress fetches LFS objects with progress reporting
func (h *Handler) fetchLFSObjectsWithProgress(ctx context.Context, client *lfs.RemoteClient, lfsEndpoint string, objects []lfs.LFSObject, progressFn queue.ProgressFunc, totalSize int64) error {
	// Filter out objects that already exist
	var missing []lfs.LFSObject
	for _, obj := range objects {
		if !h.contentStore.Exists(obj.Oid) {
			missing = append(missing, obj)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	// Request download URLs for missing objects
	batchResp, err := client.BatchDownload(ctx, lfsEndpoint, missing)
	if err != nil {
		return fmt.Errorf("batch download request failed: %w", err)
	}

	// Download and store each object
	var downloadedBytes int64
	downloadedCount := 0
	for _, obj := range batchResp.Objects {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if obj.Error != nil {
			log.Printf("Skipping object %s due to error: %s\n", obj.Oid, obj.Error.Message)
			continue
		}

		downloadAction, ok := obj.Actions["download"]
		if !ok {
			continue
		}

		progressFn(20+int(80*downloadedBytes/totalSize), fmt.Sprintf("Downloading %d/%d objects", downloadedCount+1, len(missing)), downloadedBytes, totalSize)

		body, err := client.DownloadObject(ctx, &downloadAction)
		if err != nil {
			log.Printf("Failed to download LFS object %s: %v\n", obj.Oid, err)
			continue
		}

		err = h.contentStore.Put(obj.Oid, body, obj.Size)
		body.Close()
		if err != nil {
			log.Printf("Failed to store LFS object %s: %v\n", obj.Oid, err)
			continue
		}

		downloadedBytes += obj.Size
		downloadedCount++
		progressFn(20+int(80*downloadedBytes/totalSize), fmt.Sprintf("Downloaded %d/%d objects", downloadedCount, len(missing)), downloadedBytes, totalSize)
	}

	return nil
}
