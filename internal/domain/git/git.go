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

package git

import (
	"context"
	"time"
)

// Git-related types for model version control

// Revision represents a Git reference (branch or tag).
type Revision struct {
	Name string `json:"name"`
}

// Revisions contains branches and tags.
type Revisions struct {
	Branches []*Revision `json:"branches"`
	Tags     []*Revision `json:"tags"`
}

// Commit represents a Git commit.
type Commit struct {
	ID             string    `json:"id"`
	Message        string    `json:"message"`
	AuthorName     string    `json:"authorName"`
	AuthorEmail    string    `json:"authorEmail"`
	AuthorDate     time.Time `json:"authorDate"`
	CommitterName  string    `json:"committerName"`
	CommitterEmail string    `json:"committerEmail"`
	CommitterDate  time.Time `json:"committerDate"`
	Diff           string    `json:"diff"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Diff represents a file change in a commit.
type Diff struct {
	Diff    string `json:"diff"`
	Deleted bool   `json:"deleted"`
	NewPath string `json:"newPath"`
	OldPath string `json:"oldPath"`
}

// FileType represents the type of file in the Git tree.
type FileType int

const (
	FileTypeDir  FileType = 0
	FileTypeFile FileType = 1
)

// TreeEntry represents a file or directory in the Git tree.
type TreeEntry struct {
	Name   string   `json:"name"`
	Type   FileType `json:"type"`
	Size   int64    `json:"size"`
	Path   string   `json:"path"`
	Hash   string   `json:"hash"`
	IsLFS  bool     `json:"isLFS"`
	URL    string   `json:"url,omitempty"`
	Commit *Commit  `json:"commit,omitempty"`
}

type GitRepository struct {
	RemoteRegistryURL  string
	RemoteProjectName  string
	RemoteResourceName string
	ProjectName        string
	ResourceName       string
	ResourceType       string
}

// IGitRepo defines the repository interface for Git operations on models.
type IGitRepo interface {
	// CreateRepository initializes a Git repository.
	CreateRepository(ctx context.Context, project, name string) error

	// DeleteRepository removes the Git repository.
	DeleteRepository(ctx context.Context, project, name string) error

	// ListRevisions returns all branches and tags for a model.
	ListRevisions(ctx context.Context, project, name string) (*Revisions, error)

	// ListCommits returns the commit history for a model.
	ListCommits(ctx context.Context, project, name, revision string, page, pageSize int) ([]*Commit, int64, error)

	// GetCommit returns a specific commit by ID.
	GetCommit(ctx context.Context, project, name, commitID string) (*Commit, error)

	// GetTree returns the file tree at a specific revision and path.
	GetTree(ctx context.Context, project, name, revision, path string) ([]*TreeEntry, error)

	// GetBlob returns the content of a file at a specific revision.
	GetBlob(ctx context.Context, project, name, revision, path string) (*TreeEntry, error)

	Clone(ctx context.Context, gitRepository *GitRepository) error

	Pull(ctx context.Context, gitRepository *GitRepository) error
}
