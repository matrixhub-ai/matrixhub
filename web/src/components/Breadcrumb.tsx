import { Link } from 'react-router-dom';
import './Breadcrumb.css';

interface BreadcrumbProps {
  repo: string;
  branch: string;
  path: string;
  isBlob?: boolean;
}

export function Breadcrumb({ repo, branch, path, isBlob }: BreadcrumbProps) {
  const parts = path ? path.split('/').filter(Boolean) : [];
  
  return (
    <nav className="breadcrumb">
      <Link to={`/${repo}`} className="breadcrumb-item repo-name">
        {repo}
      </Link>
      {parts.length > 0 && <span className="breadcrumb-separator">/</span>}
      {parts.map((part, index) => {
        const isLast = index === parts.length - 1;
        const pathToHere = parts.slice(0, index + 1).join('/');
        const type = isLast && isBlob ? 'blob' : 'tree';
        
        return (
          <span key={pathToHere}>
            {isLast ? (
              <span className="breadcrumb-item current">{part}</span>
            ) : (
              <Link 
                to={`/${repo}/${type}/${branch}/${pathToHere}`}
                className="breadcrumb-item"
              >
                {part}
              </Link>
            )}
            {!isLast && <span className="breadcrumb-separator">/</span>}
          </span>
        );
      })}
    </nav>
  );
}
