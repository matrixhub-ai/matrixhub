// Copyright The MatrixHub Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	ErrRepositoryNotExists = git.ErrRepositoryNotExists
)

type Repository struct {
	repo     *git.Repository
	repoPath string
}

func IsRepository(repoPath string) bool {
	stat, err := os.Stat(filepath.Join(repoPath, "HEAD"))
	if err == nil && stat.Size() != 0 {
		return true
	}
	return false
}

func Init(repoPath string, defaultBranch string) (*Repository, error) {
	repo, err := git.PlainInitWithOptions(repoPath, &git.PlainInitOptions{
		Bare: true,
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName(defaultBranch),
		},
	})
	if err != nil {
		return nil, err
	}
	return &Repository{
		repo:     repo,
		repoPath: repoPath,
	}, nil
}

func Open(repoPath string) (*Repository, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{})
	if err != nil {
		return nil, err
	}
	return &Repository{
		repo:     repo,
		repoPath: repoPath,
	}, nil
}

func (r *Repository) SplitRevisionAndPath(refpath string) (ref string, path string, err error) {
	if refpath == "" {
		return r.DefaultBranch(), "", nil
	}

	branches, err := r.Branches()
	if err != nil {
		return "", "", err
	}

	// Sort branches by length (longest first) to match the most specific branch
	sortedBranches := make([]string, len(branches))
	copy(sortedBranches, branches)
	sort.Slice(sortedBranches, func(i, j int) bool {
		return len(sortedBranches[i]) > len(sortedBranches[j])
	})

	for _, branch := range sortedBranches {
		if refpath == branch {
			return branch, "", nil
		}
		if strings.HasPrefix(refpath, branch+"/") {
			return branch, refpath[len(branch)+1:], nil
		}
	}

	// Fallback: treat first segment as branch
	parts := strings.SplitN(refpath, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return refpath, "", nil
}

func (r *Repository) DefaultBranch() string {
	head := filepath.Join(r.repoPath, "HEAD")
	data, err := os.ReadFile(head)
	if err != nil {
		return "main"
	}

	prefix := "ref: refs/heads/"
	if len(data) > len(prefix) && string(data[:len(prefix)]) == prefix {
		return string(bytes.TrimSpace(data[len(prefix):]))
	}
	return "main"
}

func (r *Repository) Branches() ([]string, error) {
	branchesIter, err := r.repo.Branches()
	if err != nil {
		return nil, err
	}

	var branches []string
	err = branchesIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		branches = append(branches, name)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(branches) == 0 {
		return []string{r.DefaultBranch()}, nil
	}
	return branches, nil
}

func (r *Repository) Remove() error {
	return os.RemoveAll(r.repoPath)
}

func (r *Repository) IsMirror() (bool, string, error) {
	config, err := r.repo.Config()
	if err != nil {
		return false, "", err
	}
	isMirror := false
	sourceURL := ""
	if config != nil {
		if remote, ok := config.Remotes["origin"]; ok {
			if remote.Mirror {
				isMirror = true
			}
			if len(remote.URLs) > 0 {
				sourceURL = remote.URLs[0]
			}
		}
	}
	return isMirror, sourceURL, nil
}
