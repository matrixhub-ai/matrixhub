package backend

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/pkg/queue"
)

func (h *Handler) registryQueue(r *mux.Router) {
	r.HandleFunc("/api/queue", h.handleListTasks).Methods(http.MethodGet)
	r.HandleFunc("/api/queue/{id:[0-9]+}", h.handleGetTask).Methods(http.MethodGet)
	r.HandleFunc("/api/queue/{id:[0-9]+}/priority", h.handleUpdateTaskPriority).Methods(http.MethodPut)
	r.HandleFunc("/api/queue/{id:[0-9]+}", h.handleCancelTask).Methods(http.MethodDelete)
}

// handleListTasks returns all tasks in the queue
func (h *Handler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	statusStr := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	repoFilter := r.URL.Query().Get("repository")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	var tasks []*queue.Task
	var err error

	if repoFilter != "" {
		tasks, err = h.queueStore.ListByRepository(repoFilter)
	} else if statusStr != "" {
		status := queue.TaskStatus(statusStr)
		tasks, err = h.queueStore.List(&status, limit)
	} else {
		tasks, err = h.queueStore.List(nil, limit)
	}

	if err != nil {
		http.Error(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}

	// Handle nil slice
	if tasks == nil {
		tasks = []*queue.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handleGetTask returns a specific task by ID
func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := h.queueStore.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// updatePriorityRequest represents a request to update task priority
type updatePriorityRequest struct {
	Priority int `json:"priority"`
}

// handleUpdateTaskPriority updates the priority of a task
func (h *Handler) handleUpdateTaskPriority(w http.ResponseWriter, r *http.Request) {
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var req updatePriorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.queueStore.UpdatePriority(id, req.Priority)
	if err != nil {
		http.Error(w, "Failed to update priority", http.StatusInternalServerError)
		return
	}

	task, err := h.queueStore.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// handleCancelTask cancels a task
func (h *Handler) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	if h.queueStore == nil {
		http.Error(w, "Queue not initialized", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Check if task is running and cancel it
	if h.queueWorker != nil && h.queueWorker.IsTaskRunning(id) {
		h.queueWorker.CancelTask(id)
	}

	err = h.queueStore.Cancel(id)
	if err != nil {
		http.Error(w, "Failed to cancel task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
