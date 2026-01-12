// Package queue provides a SQLite-backed task queue for background processing.
package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TaskType represents the type of task to be processed
type TaskType string

const (
	TaskTypeMirrorSync TaskType = "mirror_sync"
	TaskTypeLFSSync    TaskType = "lfs_sync"
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// Task represents a queued task
type Task struct {
	ID          int64             `json:"id"`
	Type        TaskType          `json:"type"`
	Status      TaskStatus        `json:"status"`
	Priority    int               `json:"priority"` // Higher value = higher priority
	Repository  string            `json:"repository"`
	Params      map[string]string `json:"params,omitempty"`
	Progress    int               `json:"progress"`    // 0-100
	ProgressMsg string            `json:"progress_msg,omitempty"`
	TotalBytes  int64             `json:"total_bytes,omitempty"`
	DoneBytes   int64             `json:"done_bytes,omitempty"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// Store provides SQLite-backed storage for the task queue
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewStore creates a new queue store with SQLite backend
func NewStore(dbPath string) (*Store, error) {
	err := os.MkdirAll(filepath.Dir(dbPath), 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory for database: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tasks table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER NOT NULL DEFAULT 0,
			repository TEXT NOT NULL,
			params TEXT,
			progress INTEGER NOT NULL DEFAULT 0,
			progress_msg TEXT,
			total_bytes INTEGER DEFAULT 0,
			done_bytes INTEGER DEFAULT 0,
			error TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority DESC);
		CREATE INDEX IF NOT EXISTS idx_tasks_repository ON tasks(repository);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Add adds a new task to the queue
func (s *Store) Add(taskType TaskType, repository string, priority int, params map[string]string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	result, err := s.db.Exec(`
		INSERT INTO tasks (type, repository, priority, params, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, taskType, repository, priority, string(paramsJSON), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.getByID(id)
}

// Get retrieves a task by ID
func (s *Store) Get(id int64) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getByID(id)
}

func (s *Store) getByID(id int64) (*Task, error) {
	row := s.db.QueryRow(`
		SELECT id, type, status, priority, repository, params, progress, progress_msg, 
		       total_bytes, done_bytes, error, created_at, started_at, completed_at
		FROM tasks WHERE id = ?
	`, id)

	return s.scanTask(row)
}

// List returns all tasks, optionally filtered by status
func (s *Store) List(status *TaskStatus, limit int) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var query string
	var args []interface{}

	if status != nil {
		query = `
			SELECT id, type, status, priority, repository, params, progress, progress_msg,
			       total_bytes, done_bytes, error, created_at, started_at, completed_at
			FROM tasks WHERE status = ?
			ORDER BY priority DESC, created_at ASC
			LIMIT ?
		`
		args = []interface{}{*status, limit}
	} else {
		query = `
			SELECT id, type, status, priority, repository, params, progress, progress_msg,
			       total_bytes, done_bytes, error, created_at, started_at, completed_at
			FROM tasks
			ORDER BY 
				CASE status 
					WHEN 'running' THEN 0 
					WHEN 'pending' THEN 1 
					ELSE 2 
				END,
				priority DESC, created_at ASC
			LIMIT ?
		`
		args = []interface{}{limit}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task, err := s.scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// ListByRepository returns tasks for a specific repository
func (s *Store) ListByRepository(repository string) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, type, status, priority, repository, params, progress, progress_msg,
		       total_bytes, done_bytes, error, created_at, started_at, completed_at
		FROM tasks WHERE repository = ?
		ORDER BY created_at DESC
		LIMIT 100
	`, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task, err := s.scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// GetNext returns the next pending task to process (highest priority, oldest first)
func (s *Store) GetNext() (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row := s.db.QueryRow(`
		SELECT id, type, status, priority, repository, params, progress, progress_msg,
		       total_bytes, done_bytes, error, created_at, started_at, completed_at
		FROM tasks 
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
	`)

	task, err := s.scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return task, err
}

// UpdateStatus updates the status of a task
func (s *Store) UpdateStatus(id int64, status TaskStatus, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	now := time.Now()

	switch status {
	case TaskStatusRunning:
		_, err = s.db.Exec(`
			UPDATE tasks SET status = ?, started_at = ?, error = NULL
			WHERE id = ?
		`, status, now, id)
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		_, err = s.db.Exec(`
			UPDATE tasks SET status = ?, completed_at = ?, error = ?
			WHERE id = ?
		`, status, now, errorMsg, id)
	default:
		_, err = s.db.Exec(`
			UPDATE tasks SET status = ?, error = ?
			WHERE id = ?
		`, status, errorMsg, id)
	}

	return err
}

// UpdateProgress updates the progress of a task
func (s *Store) UpdateProgress(id int64, progress int, progressMsg string, doneBytes, totalBytes int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE tasks SET progress = ?, progress_msg = ?, done_bytes = ?, total_bytes = ?
		WHERE id = ?
	`, progress, progressMsg, doneBytes, totalBytes, id)
	return err
}

// UpdatePriority updates the priority of a task
func (s *Store) UpdatePriority(id int64, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE tasks SET priority = ?
		WHERE id = ? AND status = 'pending'
	`, priority, id)
	return err
}

// Cancel cancels a pending task
func (s *Store) Cancel(id int64) error {
	return s.UpdateStatus(id, TaskStatusCancelled, "Cancelled by user")
}

// Delete removes a task from the queue
func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// CleanupOld removes completed/failed tasks older than the specified duration
func (s *Store) CleanupOld(olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Exec(`
		DELETE FROM tasks 
		WHERE status IN ('completed', 'failed', 'cancelled') 
		AND completed_at < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetPendingCount returns the number of pending tasks
func (s *Store) GetPendingCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status = 'pending'`).Scan(&count)
	return count, err
}

// GetRunningCount returns the number of running tasks
func (s *Store) GetRunningCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status = 'running'`).Scan(&count)
	return count, err
}

func (s *Store) scanTask(row *sql.Row) (*Task, error) {
	var task Task
	var paramsJSON string
	var startedAt, completedAt sql.NullTime
	var progressMsg, errorStr sql.NullString

	err := row.Scan(
		&task.ID, &task.Type, &task.Status, &task.Priority, &task.Repository,
		&paramsJSON, &task.Progress, &progressMsg, &task.TotalBytes, &task.DoneBytes,
		&errorStr, &task.CreatedAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &task.Params); err != nil {
			return nil, fmt.Errorf("failed to parse task params: %w", err)
		}
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if progressMsg.Valid {
		task.ProgressMsg = progressMsg.String
	}
	if errorStr.Valid {
		task.Error = errorStr.String
	}

	return &task, nil
}

func (s *Store) scanTaskRows(rows *sql.Rows) (*Task, error) {
	var task Task
	var paramsJSON string
	var startedAt, completedAt sql.NullTime
	var progressMsg, errorStr sql.NullString

	err := rows.Scan(
		&task.ID, &task.Type, &task.Status, &task.Priority, &task.Repository,
		&paramsJSON, &task.Progress, &progressMsg, &task.TotalBytes, &task.DoneBytes,
		&errorStr, &task.CreatedAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if paramsJSON != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &task.Params); err != nil {
			return nil, fmt.Errorf("failed to parse task params: %w", err)
		}
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if progressMsg.Valid {
		task.ProgressMsg = progressMsg.String
	}
	if errorStr.Valid {
		task.Error = errorStr.String
	}

	return &task, nil
}
