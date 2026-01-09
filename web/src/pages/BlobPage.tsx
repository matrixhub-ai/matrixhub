import { useEffect, useState, useCallback } from 'react';
import { useLocation, Link } from 'react-router-dom';
import { fetchBlob, fetchBlobMetadata, fetchBranches } from '../api/client';
import type { BlobContent, Branch } from '../api/client';
import { FileViewer } from '../components/FileViewer';
import { BranchSelector } from '../components/BranchSelector';
import { Breadcrumb } from '../components/Breadcrumb';
import { findBranchInPath } from '../utils/branchUtils';
import './BlobPage.css';

interface FetchState {
  content: BlobContent | null;
  branches: Branch[];
  repoName: string;
  branch: string;
  filePath: string;
  loading: boolean;
  error: string | null;
}

export function BlobPage() {
  const location = useLocation();
  const pathname = location.pathname;

  const blobIndex = pathname.indexOf('/blob/');
  let repoPath = '';
  let branchAndPath = '';

  if (blobIndex !== -1) {
    repoPath = pathname.substring(1, blobIndex);
    branchAndPath = pathname.substring(blobIndex + 6);
  }

  const [state, setState] = useState<FetchState>({
    content: null,
    branches: [],
    repoName: '',
    branch: '',
    filePath: '',
    loading: true,
    error: null,
  });

  const loadData = useCallback(async () => {
    if (!repoPath || !branchAndPath) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: 'Invalid URL'
      }));
      return;
    }

    try {
      const branchList = await fetchBranches(repoPath);

      let foundBranch = '';
      let foundPath = '';

      const branchMatch = findBranchInPath(branchAndPath, branchList);
      if (branchMatch) {
        foundBranch = branchMatch.branch;
        foundPath = branchMatch.path;
      } else {
        const segments = branchAndPath.split('/');
        foundBranch = segments[0];
        foundPath = segments.slice(1).join('/');
      }

      if (!foundPath) {
        setState(prev => ({
          ...prev,
          loading: false,
          error: 'File not found'
        }));
        return;
      }

      const blobMetadata = await fetchBlobMetadata(repoPath, foundBranch, foundPath);

      if (blobMetadata.contentType === 'application/octet-stream') {
        setState({
          content: null,
          branches: branchList,
          repoName: repoPath,
          branch: foundBranch,
          filePath: foundPath,
          loading: false,
          error: null,
        });
      } else {
        const blob = await fetchBlob(repoPath, foundBranch, foundPath);
        setState({
          content: blob,
          branches: branchList,
          repoName: repoPath,
          branch: foundBranch,
          filePath: foundPath,
          loading: false,
          error: null,
        });
      }
    } catch (err) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: err instanceof Error ? err.message : 'An error occurred',
        isLargeFile: false
      }));
    }
  }, [repoPath, branchAndPath]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const { content, branches, repoName, branch, filePath, loading, error } = state;
  const filename = filePath.split('/').pop() || '';

  if (loading) {
    return <div className="loading">Loading file...</div>;
  }

  if (error) {
    return <div className="error">Error: {error}</div>;
  }

  if (!repoName) {
    return <div className="error">File not found</div>;
  }

  return (
    <div className="blob-page">
      <header className="repo-header">
        <Link to="/" className="home-link">🗂️ gitd</Link>
      </header>

      <div className="repo-toolbar">
        <BranchSelector
          branches={branches}
          currentBranch={branch}
          repo={repoName}
          currentPath={filePath}
          isBlob
        />
        <Breadcrumb
          repo={repoName}
          branch={branch}
          path={filePath}
          isBlob
        />
      </div>

      <div className="blob-content">
        <FileViewer content={content} filename={filename} />
      </div>
    </div>
  );
}
