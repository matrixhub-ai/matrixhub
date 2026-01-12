import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { fetchRepositories, createRepository, deleteRepository, importRepository, syncRepository } from '../api/client';
import type { RepositoryItem } from '../api/client';
import './HomePage.css';

export function HomePage() {
  const [repos, setRepos] = useState<RepositoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);
  const [newRepoName, setNewRepoName] = useState('');
  const [importRepoName, setImportRepoName] = useState('');
  const [importSourceUrl, setImportSourceUrl] = useState('');
  const [actionLoading, setActionLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [syncingRepos, setSyncingRepos] = useState<Set<string>>(new Set());

  const loadRepos = () => {
    setLoading(true);
    fetchRepositories()
      .then(setRepos)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadRepos();
  }, []);

  const handleCreateRepository = async () => {
    if (!newRepoName.trim()) return;

    setActionLoading(true);
    setActionError(null);
    try {
      const repoName = newRepoName;
      await createRepository(repoName);
      setShowCreateModal(false);
      setNewRepoName('');
      loadRepos();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to create repository');
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeleteRepository = async (name: string) => {
    if (!confirm(`Are you sure you want to delete "${name}"? This action cannot be undone.`)) {
      return;
    }

    try {
      const repoName = name;
      await deleteRepository(repoName);
      loadRepos();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete repository');
    }
  };

  const handleSyncRepository = async (name: string) => {
    const repoName = name;

    setSyncingRepos(prev => new Set(prev).add(name));

    try {
      await syncRepository(repoName);
      loadRepos();
    } catch (err) {
      setSyncingRepos(prev => {
        const next = new Set(prev);
        next.delete(name);
        return next;
      });
      alert(err instanceof Error ? err.message : 'Failed to start sync');
    }
  };

  const handleImportRepository = async () => {
    if (!importRepoName.trim() || !importSourceUrl.trim()) return;

    setActionLoading(true);
    setActionError(null);
    try {
      const repoName = importRepoName;
      await importRepository(repoName, importSourceUrl);

      // Close modal immediately and run import in background
      setShowImportModal(false);
      setImportRepoName('');
      setImportSourceUrl('');
      setActionLoading(false);

      // Refresh repo list to show the new importing repository
      loadRepos();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to start import');
      setActionLoading(false);
    }
  };

  const closeImportModal = () => {
    setShowImportModal(false);
    setImportRepoName('');
    setImportSourceUrl('');
    setActionError(null);
    setActionLoading(false);
  };

  if (loading) {
    return <div className="loading">Loading repositories...</div>;
  }

  if (error) {
    return <div className="error">Error: {error}</div>;
  }

  return (
    <div className="home-page">
      <header className="header">
        <h1>
          <span className="logo">🗂️</span>
          gitd
        </h1>
        <nav className="header-nav">
          <Link to="/queue" className="nav-link">📋 Queue</Link>
        </nav>
      </header>
      <main className="main-content">
        <div className="repo-header">
          <h2>Repositories</h2>
          <div className="repo-actions">
            <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
              ➕ New Repository
            </button>
            <button className="btn btn-secondary" onClick={() => setShowImportModal(true)}>
              📥 Import Repository
            </button>
          </div>
        </div>
        {repos.length === 0 ? (
          <p className="no-repos">No repositories found.</p>
        ) : (
          <ul className="repo-list">
            {repos.map((repo) => (
              <li key={repo.name} className="repo-item">
                <Link to={`/${repo.name}`} className="repo-link">
                  <span className="repo-icon">📁</span>
                  <span className="repo-full-name">
                    <strong>{repo.name}</strong>
                    {repo.is_mirror && <span className="mirror-badge">Mirror</span>}
                  </span>
                </Link>
                <div className="repo-item-actions">
                  {repo.is_mirror && (
                    <button
                      className="btn btn-secondary btn-small"
                      onClick={(e) => {
                        e.preventDefault();
                        handleSyncRepository(repo.name);
                      }}
                      disabled={syncingRepos.has(repo.name)}
                      title="Synchronize with source"
                    >
                      {syncingRepos.has(repo.name) ? '⏳' : '🔄'}
                    </button>
                  )}
                  <button
                    className="btn btn-danger btn-small"
                    onClick={(e) => {
                      e.preventDefault();
                      handleDeleteRepository(repo.name);
                    }}
                    title="Delete repository"
                  >
                    🗑️
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </main>

      {/* Create Repository Modal */}
      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Create New Repository</h3>
            <div className="form-group">
              <label htmlFor="repoName">Repository Name</label>
              <input
                type="text"
                id="repoName"
                value={newRepoName}
                onChange={(e) => setNewRepoName(e.target.value)}
                placeholder="my-repository"
                disabled={actionLoading}
              />
              <small>.git extension will be added automatically</small>
            </div>
            {actionError && <div className="error-message">{actionError}</div>}
            <div className="modal-actions">
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)} disabled={actionLoading}>
                Cancel
              </button>
              <button className="btn btn-primary" onClick={handleCreateRepository} disabled={actionLoading || !newRepoName.trim()}>
                {actionLoading ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Import Repository Modal */}
      {showImportModal && (
        <div className="modal-overlay" onClick={closeImportModal}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Import Repository</h3>
            <div className="form-group">
              <label htmlFor="importRepoName">Repository Name</label>
              <input
                type="text"
                id="importRepoName"
                value={importRepoName}
                onChange={(e) => setImportRepoName(e.target.value)}
                placeholder="my-repository"
                disabled={actionLoading}
              />
              <small>.git extension will be added automatically</small>
            </div>
            <div className="form-group">
              <label htmlFor="sourceUrl">Source URL</label>
              <input
                type="text"
                id="sourceUrl"
                value={importSourceUrl}
                onChange={(e) => setImportSourceUrl(e.target.value)}
                placeholder="https://github.com/user/repo.git"
                disabled={actionLoading}
              />
            </div>
            {actionError && <div className="error-message">{actionError}</div>}
            <div className="modal-actions">
              <button className="btn btn-secondary" onClick={closeImportModal} disabled={actionLoading}>
                Cancel
              </button>
              <button
                className="btn btn-primary"
                onClick={handleImportRepository}
                disabled={actionLoading || !importRepoName.trim() || !importSourceUrl.trim()}
              >
                {actionLoading ? 'Starting...' : 'Import'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
