package repository

import (
	"fmt"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func (r *Repository) Commits(ref string, limit int) ([]Commit, error) {
	var fromHash plumbing.Hash

	// First try to resolve as a branch reference
	refObj, err := r.repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true)
	if err == nil {
		fromHash = refObj.Hash()
	} else if err == plumbing.ErrReferenceNotFound {
		// If not a branch, try to resolve as a commit SHA
		if !isValidSHA(ref) {
			// Neither a branch nor a valid commit SHA format
			return []Commit{}, nil
		}
		hash := plumbing.NewHash(ref)
		// Verify the commit exists
		_, err := r.repo.CommitObject(hash)
		if err != nil {
			// Valid SHA format but commit not found
			return []Commit{}, nil
		}
		fromHash = hash
	} else {
		return nil, err
	}

	commitIter, err := r.repo.Log(&git.LogOptions{From: fromHash})
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
