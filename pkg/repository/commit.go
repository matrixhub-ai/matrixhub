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
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func (r *Repository) Commits(ref string, limit int) ([]Commit, error) {
	hash, err := r.repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve revision: %w", err)
	}

	commitIter, err := r.repo.Log(&git.LogOptions{From: *hash})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	var commits []Commit
	err = commitIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, Commit{
			SHA:     c.Hash.String(),
			Message: c.Message,
			Author:  c.Author.Name,
			Email:   c.Author.Email,
			Date:    c.Author.When.Format("2006-01-02T15:04:05Z"),
		})
		if len(commits) >= limit {
			return io.EOF // Stop after reaching the limit
		}
		return nil
	})
	if err != nil && err != io.EOF && len(commits) == 0 {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return commits, nil
}

type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Date    string `json:"date"`
}
