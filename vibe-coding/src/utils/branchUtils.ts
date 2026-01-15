import type { Branch } from '../api/client';

/**
 * Find which branch matches the start of the remaining path
 * Branches can contain '/', so we need to match the longest branch name first
 */
export function findBranchInPath(remainingPath: string, branches: Branch[]): { branch: string; path: string } | null {
  // Sort branches by length (longest first) to match the most specific branch
  const sortedBranches = [...branches].sort((a, b) => b.name.length - a.name.length);
  
  for (const branch of sortedBranches) {
    if (remainingPath === branch.name) {
      return { branch: branch.name, path: '' };
    }
    if (remainingPath.startsWith(branch.name + '/')) {
      return { branch: branch.name, path: remainingPath.slice(branch.name.length + 1) };
    }
  }
  return null;
}
