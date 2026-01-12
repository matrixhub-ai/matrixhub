package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// TaskHandler is a function that processes a task
type TaskHandler func(ctx context.Context, task *Task, progressFn ProgressFunc) error

// ProgressFunc is called to report progress of a task
type ProgressFunc func(progress int, message string, doneBytes, totalBytes int64)

// Worker processes tasks from the queue
type Worker struct {
	store      *Store
	handlers   map[TaskType]TaskHandler
	maxWorkers int
	pollDelay  time.Duration

	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    map[int64]context.CancelFunc // task ID -> cancel function
}

// NewWorker creates a new queue worker
func NewWorker(store *Store, maxWorkers int) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		store:      store,
		handlers:   make(map[TaskType]TaskHandler),
		maxWorkers: maxWorkers,
		pollDelay:  time.Second,
		ctx:        ctx,
		cancel:     cancel,
		running:    make(map[int64]context.CancelFunc),
	}
}

// RegisterHandler registers a handler for a specific task type
func (w *Worker) RegisterHandler(taskType TaskType, handler TaskHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[taskType] = handler
}

// Start begins processing tasks from the queue
func (w *Worker) Start() {
	w.wg.Add(1)
	go w.pollLoop()
}

// Stop stops the worker and waits for all tasks to complete
func (w *Worker) Stop() {
	w.cancel()
	w.wg.Wait()
}

// CancelTask cancels a running task
func (w *Worker) CancelTask(taskID int64) bool {
	w.mu.Lock()
	cancelFn, exists := w.running[taskID]
	w.mu.Unlock()

	if exists {
		cancelFn()
		return true
	}
	return false
}

// IsTaskRunning checks if a task is currently running
func (w *Worker) IsTaskRunning(taskID int64) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	_, exists := w.running[taskID]
	return exists
}

func (w *Worker) pollLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		// Check if we have capacity to run more tasks
		runningCount, err := w.store.GetRunningCount()
		if err != nil {
			log.Printf("Error getting running count: %v\n", err)
			time.Sleep(w.pollDelay)
			continue
		}

		if runningCount >= w.maxWorkers {
			time.Sleep(w.pollDelay)
			continue
		}

		// Get next pending task
		task, err := w.store.GetNext()
		if err != nil {
			log.Printf("Error getting next task: %v\n", err)
			time.Sleep(w.pollDelay)
			continue
		}

		if task == nil {
			time.Sleep(w.pollDelay)
			continue
		}

		// Start processing the task
		w.wg.Add(1)
		go w.processTask(task)
	}
}

func (w *Worker) processTask(task *Task) {
	defer w.wg.Done()

	// Get handler for this task type
	w.mu.RLock()
	handler, exists := w.handlers[task.Type]
	w.mu.RUnlock()

	if !exists {
		log.Printf("No handler registered for task type: %s\n", task.Type)
		w.store.UpdateStatus(task.ID, TaskStatusFailed, fmt.Sprintf("No handler for task type: %s", task.Type))
		return
	}

	// Create cancellable context for this task
	taskCtx, taskCancel := context.WithCancel(w.ctx)
	defer taskCancel()

	// Register running task
	w.mu.Lock()
	w.running[task.ID] = taskCancel
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.running, task.ID)
		w.mu.Unlock()
	}()

	// Update task status to running
	err := w.store.UpdateStatus(task.ID, TaskStatusRunning, "")
	if err != nil {
		log.Printf("Failed to update task status: %v\n", err)
		return
	}

	// Create progress callback
	progressFn := func(progress int, message string, doneBytes, totalBytes int64) {
		w.store.UpdateProgress(task.ID, progress, message, doneBytes, totalBytes)
	}

	// Execute the handler
	log.Printf("Starting task %d: %s for %s\n", task.ID, task.Type, task.Repository)
	err = handler(taskCtx, task, progressFn)

	if err != nil {
		if taskCtx.Err() == context.Canceled {
			log.Printf("Task %d cancelled\n", task.ID)
			w.store.UpdateStatus(task.ID, TaskStatusCancelled, "Cancelled")
		} else {
			log.Printf("Task %d failed: %v\n", task.ID, err)
			w.store.UpdateStatus(task.ID, TaskStatusFailed, err.Error())
		}
		return
	}

	log.Printf("Task %d completed successfully\n", task.ID)
	w.store.UpdateProgress(task.ID, 100, "Completed", 0, 0)
	w.store.UpdateStatus(task.ID, TaskStatusCompleted, "")
}
