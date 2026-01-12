import { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { fetchTasks, cancelTask, updateTaskPriority } from '../api/client';
import type { Task } from '../api/client';
import './QueuePage.css';

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

function getStatusIcon(status: Task['status']): string {
  switch (status) {
    case 'pending': return '⏳';
    case 'running': return '🔄';
    case 'completed': return '✅';
    case 'failed': return '❌';
    case 'cancelled': return '🚫';
    default: return '❓';
  }
}

function getTaskTypeLabel(type: Task['type']): string {
  switch (type) {
    case 'mirror_sync': return 'Mirror Sync';
    case 'lfs_sync': return 'LFS Sync';
    default: return type;
  }
}

export function QueuePage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<string>('all');

  const loadTasks = useCallback(async () => {
    try {
      const status = filter === 'all' ? undefined : filter;
      const data = await fetchTasks(status, 100);
      setTasks(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tasks');
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    loadTasks();
    // Auto-refresh every 2 seconds for active tasks
    const interval = setInterval(loadTasks, 2000);
    return () => clearInterval(interval);
  }, [loadTasks]);

  const handleCancel = async (id: number) => {
    if (!confirm('Are you sure you want to cancel this task?')) return;
    try {
      await cancelTask(id);
      loadTasks();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to cancel task');
    }
  };

  const handlePriorityChange = async (id: number, delta: number) => {
    const task = tasks.find(t => t.id === id);
    if (!task) return;
    try {
      await updateTaskPriority(id, task.priority + delta);
      loadTasks();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to update priority');
    }
  };

  const activeCount = tasks.filter(t => t.status === 'running' || t.status === 'pending').length;

  return (
    <div className="queue-page">
      <header className="header">
        <h1>
          <Link to="/" className="back-link">←</Link>
          <span className="logo">📋</span>
          Task Queue
        </h1>
      </header>

      <main className="main-content">
        <div className="queue-header">
          <div className="queue-stats">
            <span className="stat">
              <strong>{activeCount}</strong> active tasks
            </span>
            <span className="stat">
              <strong>{tasks.length}</strong> total
            </span>
          </div>
          <div className="queue-filters">
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="all">All Tasks</option>
              <option value="pending">Pending</option>
              <option value="running">Running</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
              <option value="cancelled">Cancelled</option>
            </select>
            <button className="btn btn-secondary" onClick={loadTasks}>
              🔄 Refresh
            </button>
          </div>
        </div>

        {loading && tasks.length === 0 ? (
          <div className="loading">Loading tasks...</div>
        ) : error ? (
          <div className="error">Error: {error}</div>
        ) : tasks.length === 0 ? (
          <div className="no-tasks">
            <p>No tasks in the queue.</p>
          </div>
        ) : (
          <div className="task-list">
            {tasks.map((task) => (
              <div key={task.id} className={`task-card status-${task.status}`}>
                <div className="task-header">
                  <div className="task-info">
                    <span className="task-status-icon">{getStatusIcon(task.status)}</span>
                    <span className="task-type">{getTaskTypeLabel(task.type)}</span>
                    <Link to={`/${task.repository}`} className="task-repo">
                      {task.repository}
                    </Link>
                  </div>
                  <div className="task-actions">
                    {task.status === 'pending' && (
                      <>
                        <button
                          className="btn btn-icon"
                          onClick={() => handlePriorityChange(task.id, 10)}
                          title="Increase priority"
                        >
                          ⬆️
                        </button>
                        <button
                          className="btn btn-icon"
                          onClick={() => handlePriorityChange(task.id, -10)}
                          title="Decrease priority"
                        >
                          ⬇️
                        </button>
                      </>
                    )}
                    {(task.status === 'pending' || task.status === 'running') && (
                      <button
                        className="btn btn-danger btn-icon"
                        onClick={() => handleCancel(task.id)}
                        title="Cancel task"
                      >
                        ✖️
                      </button>
                    )}
                  </div>
                </div>

                {(task.status === 'running' || task.status === 'pending') && (
                  <div className="task-progress">
                    <div className="progress-bar">
                      <div
                        className="progress-fill"
                        style={{ width: `${task.progress}%` }}
                      />
                    </div>
                    <span className="progress-text">
                      {task.progress}%
                      {task.progress_msg && ` - ${task.progress_msg}`}
                    </span>
                    {task.total_bytes && task.total_bytes > 0 && (
                      <span className="progress-bytes">
                        {formatBytes(task.done_bytes || 0)} / {formatBytes(task.total_bytes)}
                      </span>
                    )}
                  </div>
                )}

                <div className="task-meta">
                  <span className="task-id">#{task.id}</span>
                  <span className="task-priority" title="Priority">
                    P{task.priority}
                  </span>
                  <span className="task-created">
                    Created: {formatDate(task.created_at)}
                  </span>
                  {task.started_at && (
                    <span className="task-started">
                      Started: {formatDate(task.started_at)}
                    </span>
                  )}
                  {task.completed_at && (
                    <span className="task-completed">
                      Completed: {formatDate(task.completed_at)}
                    </span>
                  )}
                </div>

                {task.error && (
                  <div className="task-error">
                    ⚠️ {task.error}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
