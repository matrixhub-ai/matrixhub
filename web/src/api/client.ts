// API client for interacting with the gitd backend

export interface TreeEntry {
  name: string;
  path: string;
  type: 'blob' | 'tree';
  mode: string;
  sha: string;
  isLfs?: boolean;
  blobSha256?: string;
}

export interface BlobContent {
  content: string;
  size: number;
  contentType: string;
}

export interface Commit {
  sha: string;
  message: string;
  author: string;
  email: string;
  date: string;
}

export interface Branch {
  name: string;
  current: boolean;
}

export interface Repository {
  name: string;
  is_mirror?: boolean;
  default_branch: string;
  description: string;
}

export interface RepositoryItem {
  name: string;
  is_mirror?: boolean;
}

const API_BASE = '/api';

export async function fetchTree(repo: string, ref: string, path: string = ''): Promise<TreeEntry[]> {
  repo = correctRepoName(repo);
  const url = path 
    ? `${API_BASE}/repositories/${repo}/tree/${ref}/${path}`
    : `${API_BASE}/repositories/${repo}/tree/${ref}`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch tree');
  }
  return response.json();
}

export async function fetchBlobMetadata(repo: string, ref: string, path: string): Promise<{ size: number; contentType: string }> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}/blob/${ref}/${path}`, {
    method: 'HEAD',
  });
  if (!response.ok) {
    throw new Error('Failed to fetch blob metadata');
  }
  const size = parseInt(response.headers.get('Content-Length') || '0', 10);
  const contentType = response.headers.get('Content-Type') || 'text/plain';
  return { size, contentType };
}

export async function fetchBlob(repo: string, ref: string, path: string): Promise<BlobContent> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}/blob/${ref}/${path}`);
  if (!response.ok) {
    throw new Error('Failed to fetch blob');
  }
  const content = await response.text();
  const contentType = response.headers.get('Content-Type') || 'text/plain';
  const size = parseInt(response.headers.get('Content-Length') || '0', 10);
  return { content, size, contentType };
}

export async function fetchCommits(repo: string, ref: string): Promise<Commit[]> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}/commits/${ref}`);
  if (!response.ok) {
    throw new Error('Failed to fetch commits');
  }
  return response.json();
}

export async function fetchBranches(repo: string): Promise<Branch[]> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}/branches`);
  if (!response.ok) {
    throw new Error('Failed to fetch branches');
  }
  return response.json();
}

export async function fetchRepoInfo(repo: string): Promise<Repository> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}`);
  if (!response.ok) {
    throw new Error('Failed to fetch repo info');
  }
  return response.json();
}

export async function fetchRepositories(): Promise<RepositoryItem[]> {
  const response = await fetch(`${API_BASE}/repositories`);
  if (!response.ok) {
    throw new Error('Failed to fetch repositories');
  }
  return response.json();
}

export interface ImportStatus {
  status: string;
  step: string;
  error?: string;
  task_id?: number;
}

export interface Task {
  id: number;
  type: 'mirror_sync' | 'lfs_sync';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  priority: number;
  repository: string;
  params?: Record<string, string>;
  progress: number;
  progress_msg?: string;
  total_bytes?: number;
  done_bytes?: number;
  error?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export async function fetchTasks(status?: string, limit?: number): Promise<Task[]> {
  const params = new URLSearchParams();
  if (status) params.set('status', status);
  if (limit) params.set('limit', limit.toString());
  const url = `${API_BASE}/queue${params.toString() ? '?' + params.toString() : ''}`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch tasks');
  }
  return response.json();
}

export async function fetchTask(id: number): Promise<Task> {
  const response = await fetch(`${API_BASE}/queue/${id}`);
  if (!response.ok) {
    throw new Error('Failed to fetch task');
  }
  return response.json();
}

export async function updateTaskPriority(id: number, priority: number): Promise<Task> {
  const response = await fetch(`${API_BASE}/queue/${id}/priority`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ priority }),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Failed to update task priority');
  }
  return response.json();
}

export async function cancelTask(id: number): Promise<void> {
  const response = await fetch(`${API_BASE}/queue/${id}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Failed to cancel task');
  }
}

export async function createRepository(repo: string): Promise<void> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}`, {
    method: 'POST',
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Failed to create repository');
  }
}

export async function deleteRepository(repo: string): Promise<void> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Failed to delete repository');
  }
}

export async function importRepository(repo: string, sourceUrl: string): Promise<void> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}/import`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ source_url: sourceUrl }),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Failed to start import');
  }
}

export async function syncRepository(repo: string): Promise<void> {
  repo = correctRepoName(repo);
  const response = await fetch(`${API_BASE}/repositories/${repo}/sync`, {
    method: 'POST',
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Failed to start sync');
  }
}

function correctRepoName(repo: string): string {
  return repo.endsWith('.git') ? repo : `${repo}.git`;
}
