import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Branch } from '../api/client';
import './BranchSelector.css';
import { FaCodeBranch, FaAngleDown, FaCheck } from 'react-icons/fa';

interface BranchSelectorProps {
  branches: Branch[];
  currentBranch: string;
  repo: string;
  currentPath: string;
  isBlob?: boolean;
}

export function BranchSelector({ branches, currentBranch, repo, currentPath, isBlob }: BranchSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const navigate = useNavigate();

  const handleBranchSelect = (branch: string) => {
    setIsOpen(false);
    const type = isBlob ? 'blob' : 'tree';
    if (currentPath) {
      navigate(`/${repo}/${type}/${branch}/${currentPath}`);
    } else {
      navigate(`/${repo}/tree/${branch}`);
    }
  };

  return (
    <div className="branch-selector">
      <button 
        className="branch-button"
        onClick={() => setIsOpen(!isOpen)}
      >
        <FaCodeBranch />
        <span className="branch-name">{currentBranch}</span>
        <FaAngleDown />
      </button>
      {isOpen && (
        <div className="branch-dropdown">
          <div className="branch-dropdown-header">Switch branches</div>
          {branches.map((branch) => (
            <div
              key={branch.name}
              className={`branch-option ${branch.name === currentBranch ? 'active' : ''}`}
              onClick={() => handleBranchSelect(branch.name)}
            >
              {branch.name === currentBranch && <FaCheck />}
              {branch.name}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
