import type { Commit } from '../api/client';
import './CommitList.css';

interface CommitListProps {
  commits: Commit[];
}

export function CommitList({ commits }: CommitListProps) {
  return (
    <div className="commit-list">
      <h3 className="commit-list-header">Recent Commits</h3>
      {commits.map((commit) => (
        <div key={commit.sha} className="commit-item">
          <div className="commit-message">{commit.message}</div>
          <div className="commit-meta">
            <span className="commit-author">{commit.author}</span>
            <span className="commit-date">{formatDate(commit.date)}</span>
          </div>
          <div className="commit-sha">{commit.sha.substring(0, 7)}</div>
        </div>
      ))}
    </div>
  );
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  
  if (days === 0) return 'today';
  if (days === 1) return 'yesterday';
  if (days < 7) return `${days} days ago`;
  if (days < 30) return `${Math.floor(days / 7)} weeks ago`;
  if (days < 365) return `${Math.floor(days / 30)} months ago`;
  return `${Math.floor(days / 365)} years ago`;
}
