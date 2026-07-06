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
	"context"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestExtractMetadataUsesTreeSize(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil)

	const (
		project = "test-project"
		name    = "test-model"
		readme  = "# Test\n"
		lfsSize = int64(1024)
	)

	if err := repo.CreateRepository(ctx, "model", project, name); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	lfsPointer := []byte("version https://git-lfs.github.com/spec/v1\n" +
		"oid sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n" +
		"size 1024\n")

	if _, err := repo.CreateCommit(ctx, "model", project, name, "main", &git.Commit{
		Message:     "Add model files",
		AuthorName:  "test",
		AuthorEmail: "test@example.com",
	}, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "README.md", Content: []byte(readme)},
		{Type: git.CommitOperationAdd, Path: "model.bin", Content: lfsPointer},
	}); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	metadata, err := repo.ExtractMetadata(ctx, "model", project, name)
	if err != nil {
		t.Fatalf("ExtractMetadata() error = %v", err)
	}

	expectedSize := int64(len(repository.GitattributesText)+len(readme)) + lfsSize
	if metadata.Size != expectedSize {
		t.Fatalf("metadata.Size = %d, want tree size %d", metadata.Size, expectedSize)
	}
}
