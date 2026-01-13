import { Link } from 'react-router-dom';
import { FaFolder, FaRegFile, FaDownload } from 'react-icons/fa';
import type { TreeEntry } from '../api/client';
import './FileTree.css';

interface FileTreeProps {
  entries: TreeEntry[];
  repo: string;
  branch: string;
  currentPath: string;
}

export function FileTree({ entries, repo, branch, currentPath }: FileTreeProps) {
  // Sort entries: folders first, then files, both alphabetically
  const sortedEntries = [...entries].sort((a, b) => {
    if (a.type !== b.type) {
      return a.type === 'tree' ? -1 : 1;
    }
    return a.name.localeCompare(b.name);
  });

  return (
    <div className="file-tree">
      <table className="file-table">
        <thead>
          <tr>
            <th className="file-name-header">Name</th>
          </tr>
        </thead>
        <tbody>
          {currentPath && (
            <tr className="file-row">
              <td className="file-name">
                <Link 
                  to={`/${repo}/tree/${branch}/${currentPath.split('/').slice(0, -1).join('/')}`}
                  className="file-link"
                >
                  <span className="file-icon"><FaFolder /></span>
                  ..
                </Link>
              </td>
            </tr>
          )}
          {sortedEntries.map((entry) => (
            <tr key={entry.path} className="file-row">
              <td className="file-name">
                <Link
                  to={`/${repo}/${entry.type === 'tree' ? 'tree' : 'blob'}/${branch}/${entry.path}`}
                  className="file-link"
                >
                  <span className="file-icon">
                    {entry.type === 'tree' ? <FaFolder /> : <FaRegFile />}
                  </span>
                  {entry.name}
                </Link>
                {entry.isLfs && <span className="lfs-badge">LFS</span>}
                {entry.type === 'blob' && (
                  <a
                    href={`/api/repositories/${repo}.git/blob/${branch}/${entry.path}`}
                    download={entry.name}
                    className="download-button"
                    title={`Download ${entry.name}`}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <FaDownload />
                  </a>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
