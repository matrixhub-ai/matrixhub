package backend

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/wzshiming/gitd/pkg/lfs"
	"github.com/wzshiming/gitd/pkg/queue"
	"github.com/wzshiming/gitd/pkg/repository"
)

// registerTaskHandlers registers all task handlers with the queue worker
func (h *Handler) registerTaskHandlers() {
	h.queueWorker.RegisterHandler(queue.TaskTypeRepositorySync, h.handleRepositorySyncTask)
	h.queueWorker.RegisterHandler(queue.TaskTypeLFSSync, h.handleLFSSyncObjectTask)
}

// handleRepositorySyncTask handles a mirror sync task
func (h *Handler) handleRepositorySyncTask(ctx context.Context, task *queue.Task, progressFn queue.ProgressFunc) error {
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

	// Scan repository for LFS pointers
	pointers, err := repo.ScanLFSPointers()
	if err != nil {
		return fmt.Errorf("failed to scan LFS pointers: %w", err)
	}

	if len(pointers) == 0 {
		progressFn(100, "No LFS objects to sync", 0, 0)
		return nil
	}

	// Queue each LFS object as a separate task
	for _, ptr := range pointers {
		params := map[string]string{
			"source_url": sourceURL,
			"oid":        ptr.Oid,
			"size":       fmt.Sprintf("%d", ptr.Size),
		}
		_, err := h.queueStore.Add(queue.TaskTypeLFSSync, task.Repository, task.Priority, params)
		if err != nil {
			log.Printf("Failed to queue LFS object sync for %s (OID: %s): %v\n", task.Repository, ptr.Oid, err)
		}
	}
	progressFn(100, "Completed", 0, 0)
	return nil
}

// handleLFSSyncObjectTask handles the sync of a single LFS object
func (h *Handler) handleLFSSyncObjectTask(ctx context.Context, task *queue.Task, progressFn queue.ProgressFunc) error {
	repoPath := h.resolveRepoPath(task.Repository)
	if repoPath == "" {
		return fmt.Errorf("repository not found: %s", task.Repository)
	}

	sourceURL := task.Params["source_url"]
	oid := task.Params["oid"]
	sizeStr := task.Params["size"]

	if sourceURL == "" || oid == "" || sizeStr == "" {
		return fmt.Errorf("invalid task parameters")
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid size parameter: %w", err)
	}

	// Check if the object already exists
	if h.contentStore.Exists(oid) {
		progressFn(100, fmt.Sprintf("LFS object %s already exists", oid), size, size)
		return nil
	}

	// Get LFS endpoint from source URL
	lfsEndpoint := lfs.GetLFSEndpoint(sourceURL)

	// Create remote client
	remoteClient := lfs.NewRemoteClient()

	// Request download URL for the object
	batchResp, err := remoteClient.BatchDownload(ctx, lfsEndpoint, []lfs.LFSObject{{Oid: oid, Size: size}})
	if err != nil {
		return fmt.Errorf("batch download request failed: %w", err)
	}

	if len(batchResp.Objects) == 0 || batchResp.Objects[0].Error != nil {
		return fmt.Errorf("failed to fetch LFS object %s: %v", oid, batchResp.Objects[0].Error)
	}

	downloadAction, ok := batchResp.Objects[0].Actions["download"]
	if !ok {
		return fmt.Errorf("no download action for LFS object %s", oid)
	}

	// Download the object
	body, err := remoteClient.DownloadObject(ctx, &downloadAction)
	if err != nil {
		return fmt.Errorf("failed to download LFS object %s: %w", oid, err)
	}
	defer body.Close()

	rp := &readerProgress{
		reader: body,
		size:   size,
		callFunc: func(n int64, size int64) {
			progressFn(int(float64(n)*100/float64(size)), oid, n, size)
		},
	}

	// Store the object
	err = h.contentStore.Put(oid, rp, size)
	if err != nil {
		return fmt.Errorf("failed to store LFS object %s: %w", oid, err)
	}

	progressFn(100, fmt.Sprintf("LFS object %s synced successfully", oid), size, size)
	return nil
}

type readerProgress struct {
	reader   io.Reader
	n        int64
	size     int64
	last     time.Time
	callFunc func(n int64, size int64)
}

func (r *readerProgress) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.n += int64(n)
	if time.Since(r.last) > time.Second && r.callFunc != nil {
		r.callFunc(r.n, r.size)
		r.last = time.Now()
	}
	return n, err
}
