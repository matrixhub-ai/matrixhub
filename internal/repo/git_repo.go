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

package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

const rootDir = "./data/repositories"

type gitRepo struct {
}

// NewGitDB creates a new GitRepo instance
func NewGitDB() git.IGitRepo {
	return &gitRepo{}
}

func repoPath(rootDir, project, name string) string {
	return filepath.Join(rootDir, fmt.Sprintf("%s:%s.git", project, name))
}

// CreateRepository initializes a Git repository (placeholder)
func (g *gitRepo) CreateRepository(ctx context.Context, project, name string) error {
	_, err := repository.Init(repoPath(rootDir, project, name), "main")
	if err != nil {
		return err
	}

	return nil
}

// DeleteRepository removes the Git repository (placeholder)
func (g *gitRepo) DeleteRepository(ctx context.Context, project, name string) error {
	repo, err := repository.Open(repoPath(rootDir, project, name))
	if err != nil {
		return err
	}

	err = repo.Remove()
	if err != nil {
		return err
	}

	return nil
}

// ListRevisions returns all branches and tags for a model
func (g *gitRepo) ListRevisions(ctx context.Context, project, name string) (*git.Revisions, error) {
	repo, err := repository.Open(repoPath(rootDir, project, name))
	if err != nil {
		return nil, err
	}

	var tags, branches []*git.Revision
	tagrefs, err := repo.Tag()
	if err != nil {
		return nil, err
	}

	for _, t := range tagrefs {
		tags = append(tags, &git.Revision{
			Name: t.Name,
		})
	}

	brs, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	for _, b := range brs {
		branches = append(branches, &git.Revision{
			Name: b,
		})
	}

	return &git.Revisions{
		Branches: branches,
		Tags:     tags,
	}, nil
}

// ListCommits returns the commit history for a model
func (g *gitRepo) ListCommits(ctx context.Context, project, name, revision string, page, pageSize int) ([]*git.Commit, int64, error) {
	repo, err := repository.Open(repoPath(rootDir, project, name))
	if err != nil {
		return nil, 0, err
	}

	// Validate page and pageSize parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// Calculate skip and limit for pagination
	skip := (page - 1) * pageSize
	limit := pageSize

	// Fetch only the commits needed for pagination (skip + limit)
	fetchLimit := skip + limit
	cts, err := repo.Commits(revision, fetchLimit)
	if err != nil {
		return nil, 0, err
	}

	totalCount := int64(len(cts))

	// Apply pagination
	if skip >= len(cts) {
		return []*git.Commit{}, totalCount, nil
	}

	endIndex := skip + limit
	if endIndex > len(cts) {
		endIndex = len(cts)
	}
	pageCommits := cts[skip:endIndex]

	var commits []*git.Commit
	for _, c := range pageCommits {
		date, err := time.Parse("2006-01-02T15:04:05Z", c.Date)
		if err != nil {
			return nil, 0, err
		}

		commits = append(commits, &git.Commit{
			ID:             c.SHA,
			Message:        c.Message,
			AuthorName:     c.Author,
			AuthorEmail:    c.Email,
			AuthorDate:     date,
			CommitterName:  c.CommitterName,
			CommitterEmail: c.CommitterEmail,
			Diffs:          nil,
			CreatedAt:      date,
			UpdatedAt:      date,
		})
	}

	return commits, totalCount, nil
}

// GetCommit returns a specific commit by ID
func (g *gitRepo) GetCommit(ctx context.Context, project, name, commitID string) (*git.Commit, error) {
	repo, err := repository.Open(repoPath(rootDir, project, name))
	if err != nil {
		return nil, err
	}

	c, err := repo.GetCommit(commitID)
	if err != nil {
		return nil, err
	}

	date, err := time.Parse("2006-01-02T15:04:05Z", c.Date)
	if err != nil {
		return nil, err
	}

	// Get diff between commit and its parent
	diffStr, err := repo.GetCommitDiff(commitID)
	if err != nil {
		return nil, err
	}

	var diffs []*git.Diff
	if diffStr != "" {
		diffs = append(diffs, &git.Diff{
			Diff: diffStr,
		})
	}

	return &git.Commit{
		ID:             c.SHA,
		Message:        c.Message,
		AuthorName:     c.Author,
		AuthorEmail:    c.Email,
		AuthorDate:     date,
		CommitterName:  c.CommitterName,
		CommitterEmail: c.CommitterEmail,
		Diffs:          diffs,
		CreatedAt:      date,
		UpdatedAt:      date,
	}, nil
}

// GetTree returns the file tree at a specific revision and path
func (g *gitRepo) GetTree(ctx context.Context, project, name, revision, path string) ([]*git.TreeEntry, error) {
	repo, err := repository.Open(repoPath(rootDir, project, name))
	if err != nil {
		return nil, err
	}

	entries, err := repo.Tree(revision, path)
	if err != nil {
		return nil, err
	}

	var result []*git.TreeEntry
	for _, e := range entries {
		var fileType git.FileType
		if e.Type == "tree" {
			fileType = git.FileTypeDir
		} else {
			fileType = git.FileTypeFile
		}

		// Get file size for blob entries
		var size int64
		if e.Type == "blob" {
			blob, err := repo.Blob(revision, e.Path)
			if err == nil {
				size = blob.Size()
			}
		}

		result = append(result, &git.TreeEntry{
			Name:  e.Name,
			Type:  fileType,
			Size:  size,
			Path:  e.Path,
			Hash:  e.SHA,
			IsLFS: e.IsLFS,
		})
	}

	return result, nil
}

// GetBlob returns the content of a file at a specific revision
func (g *gitRepo) GetBlob(ctx context.Context, project, name, revision, path string) (*git.TreeEntry, error) {
	repo, err := repository.Open(repoPath(rootDir, project, name))
	if err != nil {
		return nil, err
	}

	blob, err := repo.Blob(revision, path)
	if err != nil {
		return nil, err
	}

	reader, err := blob.NewReader()
	if err != nil {
		return nil, err
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
			log.Error(err)
		}
	}(reader)

	// Read blob content
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}

	return &git.TreeEntry{
		Name:    blob.Name(),
		Type:    git.FileTypeFile,
		Size:    blob.Size(),
		Path:    path,
		Hash:    blob.Hash(),
		Content: buf.String(),
	}, nil
}

func (g *gitRepo) Clone(ctx context.Context, gitRepository *git.GitRepository) error {
	panic("not implemented")
}

func (g *gitRepo) Pull(ctx context.Context, gitRepository *git.GitRepository) error {
	panic("not implemented")
}
