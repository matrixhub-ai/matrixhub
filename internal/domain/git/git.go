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

type CommitOperationType string

const (
	// CommitOperationAdd adds or updates a file.
	CommitOperationAdd CommitOperationType = "add"
	// CommitOperationDelete deletes a file.
	CommitOperationDelete CommitOperationType = "delete"
)

// CommitOperation represents a single operation in a commit.
type CommitOperation struct {
	Type    CommitOperationType
	Path    string
	Content []byte // file content for add operations
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
	ParentCommit   string
	CreatedAt      time.Time `json:"createdAt"`
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

// RepoMetadataFiles holds raw metadata-related files loaded from a git repo.
// File interpretation stays in the model domain; git only loads bytes.
type RepoMetadataFiles struct {
	ReadmeContent        []byte
	ConfigJSON           []byte
	SafetensorsIndexJSON []byte
	Size                 int64
}

// BasicCredential holds username/password for remote git authentication.
type BasicCredential struct {
	Username string
	Password string
}

type GitRepository struct {
	RemoteRegistryURL  string
	RemoteProjectName  string
	RemoteResourceName string
	ProjectName        string
	ResourceName       string
	ResourceType       string
	Credential         *BasicCredential // optional; used for push auth
}

// IGitRepo defines the repository interface for Git operations on models.
type IGitRepo interface {
	// CreateRepository initializes a Git repository.
	// repoType: "models" or "datasets"
	CreateRepository(ctx context.Context, repoType, project, name string) error

	// DeleteRepository removes the Git repository.
	// repoType: "models" or "datasets"
	DeleteRepository(ctx context.Context, repoType, project, name string) error

	// ListRevisions returns all branches and tags for a model.
	// repoType: "models" or "datasets"
	ListRevisions(ctx context.Context, repoType, project, name string) (*Revisions, error)

	// ListCommits returns the commit history for a model.
	// repoType: "models" or "datasets"
	ListCommits(ctx context.Context, repoType, project, name, revision string, page, pageSize int) ([]*Commit, int64, error)

	// GetCommit returns a specific commit by ID.
	// repoType: "models" or "datasets"
	GetCommit(ctx context.Context, repoType, project, name, commitID string) (*Commit, error)

	// CreateCommit returns a specific commit by ID.
	// repoType: "models" or "datasets"
	CreateCommit(ctx context.Context, repoType, project, name, revision string, commit *Commit, ops []CommitOperation) (string, error)

	// GetTree returns the file tree at a specific revision and path.
	// repoType: "models" or "datasets"
	GetTree(ctx context.Context, repoType, project, name, revision, path string) ([]*TreeEntry, error)

	// GetBlob returns the content of a file at a specific revision.
	// repoType: "models" or "datasets"
	GetBlob(ctx context.Context, repoType, project, name, revision, path string) (*TreeEntry, error)

	CloneFromRemote(ctx context.Context, gitRepository *GitRepository) error

	PullFromRemote(ctx context.Context, gitRepository *GitRepository) error

	PushToRemote(ctx context.Context, gitRepository *GitRepository) error

	// ExtractMetadata reads metadata-related raw files from a Git repository.
	// repoType: "models" or "datasets"
	ExtractMetadata(ctx context.Context, repoType, project, name string) (*RepoMetadataFiles, error)
}
