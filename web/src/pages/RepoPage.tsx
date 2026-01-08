import { useEffect, useState, useCallback } from 'react';
import { useLocation, Link } from 'react-router-dom';
import { fetchTree, fetchBranches, fetchCommits, fetchBlob, fetchRepoInfo } from '../api/client';
import type { TreeEntry, Branch, Commit, Repository } from '../api/client';
import { FileTree } from '../components/FileTree';
import { BranchSelector } from '../components/BranchSelector';
import { Breadcrumb } from '../components/Breadcrumb';
import { CommitList } from '../components/CommitList';
import { ReadmeViewer } from '../components/ReadmeViewer';
import { findBranchInPath } from '../utils/branchUtils';
import './RepoPage.css';

interface FetchState {
  repoInfo: Repository | null;
  branches: Branch[];
  entries: TreeEntry[];
  commits: Commit[];
  readme: string | null;
  currentBranch: string;
  currentPath: string;
  loading: boolean;
  error: string | null;
}

export function RepoPage() {
  const location = useLocation();
  const pathname = location.pathname;
  
  // Parse URL: /{repo}/tree/{branch}/{path} or just /{repo}
  // Find /tree/ in the path to split repo from branch/path
  const treeIndex = pathname.indexOf('/tree/');
  let repoPath = '';
  let branchAndPath = '';
  
  if (treeIndex !== -1) {
    repoPath = pathname.substring(1, treeIndex); // Remove leading /
    branchAndPath = pathname.substring(treeIndex + 6); // After /tree/
  } else {
    // No /tree/ in URL, treat entire path as repo
    repoPath = pathname.substring(1); // Remove leading /
  }
  
  const [state, setState] = useState<FetchState>({
    repoInfo: null,
    branches: [],
    entries: [],
    commits: [],
    readme: null,
    currentBranch: 'main',
    currentPath: '',
    loading: true,
    error: null
  });

  const loadData = useCallback(async () => {
    if (!repoPath) return;

    try {
      const [info, branchList] = await Promise.all([
        fetchRepoInfo(repoPath),
        fetchBranches(repoPath),
      ]);

      // Parse branch and path from branchAndPath
      let effectiveBranch = info.default_branch || 'main';
      let foundPath = '';
      
      if (branchAndPath) {
        const branchMatch = findBranchInPath(branchAndPath, branchList);
        if (branchMatch) {
          effectiveBranch = branchMatch.branch;
          foundPath = branchMatch.path;
        } else {
          // No branch matched, treat first segment as branch (fallback)
          const segments = branchAndPath.split('/');
          effectiveBranch = segments[0];
          foundPath = segments.slice(1).join('/');
        }
      }

      const [treeEntries, commitList] = await Promise.all([
        fetchTree(repoPath, effectiveBranch, foundPath),
        fetchCommits(repoPath, effectiveBranch),
      ]);

      // Check for README.md
      let readmeContent: string | null = null;
      const readmeEntry = (treeEntries || []).find(
        (e) => e.name.toLowerCase() === 'readme.md' && e.type === 'blob'
      );
      if (readmeEntry) {
        try {
          const blob = await fetchBlob(repoPath, effectiveBranch, readmeEntry.path);
          readmeContent = blob.content;
        } catch {
          // Ignore README fetch errors
        }
      }

      setState({
        repoInfo: { ...info, name: repoPath },
        branches: branchList,
        entries: treeEntries || [],
        commits: commitList || [],
        readme: readmeContent,
        currentBranch: effectiveBranch,
        currentPath: foundPath,
        loading: false,
        error: null
      });
    } catch (err) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'An error occurred'
      }));
    }
  }, [repoPath, branchAndPath]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const { repoInfo, branches, entries, commits, readme, currentBranch, currentPath, loading, error } = state;

  if (loading) {
    return <div className="loading">Loading repository...</div>;
  }

  if (error) {
    return <div className="error">Error: {error}</div>;
  }

  if (!repoInfo) {
    return <div className="error">Invalid repository</div>;
  }

  return (
    <div className="repo-page">
      <header className="repo-header">
        <Link to="/" className="home-link">🗂️ gitd</Link>
      </header>
      
      <div className="repo-toolbar">
        <BranchSelector 
          branches={branches} 
          currentBranch={currentBranch} 
          repo={repoInfo.name}
          currentPath={currentPath}
        />
        <Breadcrumb 
          repo={repoInfo.name} 
          branch={currentBranch} 
          path={currentPath}
        />
      </div>

      <div className="repo-content">
        <div className="file-section">
          <FileTree 
            entries={entries} 
            repo={repoInfo.name} 
            branch={currentBranch}
            currentPath={currentPath}
          />
          {readme && !currentPath && <ReadmeViewer content={readme} />}
        </div>
        
        <aside className="sidebar">
          <CommitList commits={commits.slice(0, 5)} />
        </aside>
      </div>
    </div>
  );
}
