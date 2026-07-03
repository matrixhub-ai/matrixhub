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
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	modeldomain "github.com/matrixhub-ai/matrixhub/internal/domain/model"
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

func TestExtractMetadataReadsSingleSafetensorsHeader(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(root))
	repoPath := store.ResolvePath("test-project/test-model")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0750); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	repo, err := repository.Init(ctx, repoPath, "main")
	if err != nil {
		t.Fatalf("repository.Init() error = %v", err)
	}

	fullSafetensors := buildRepoTestSafetensorsFile(t, map[string][]int64{
		"model.embed_tokens.weight": {2, 3},
		"lm_head.weight":            {4, 5},
	}, 1024*1024)

	if _, err := repo.CreateCommit(ctx, "main", "add model", "Test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "config.json", Content: []byte(`{"torch_dtype":"bfloat16"}`)},
		{Type: repository.CommitOperationAdd, Path: "model.safetensors", Content: fullSafetensors},
	}, ""); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	gitRepo := NewGitDB(store, nil)
	files, err := gitRepo.ExtractMetadata(ctx, "models", "test-project", "test-model")
	if err != nil {
		t.Fatalf("ExtractMetadata() error = %v", err)
	}

	header := files.SafetensorsFiles["model.safetensors"]
	if len(header) == 0 {
		t.Fatal("expected model.safetensors header to be extracted")
	}
	if len(header) >= len(fullSafetensors) {
		t.Fatalf("expected only safetensors header, got full file: header=%d full=%d", len(header), len(fullSafetensors))
	}

	metadata, err := modeldomain.AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}
	if metadata.ParameterCount != 26 {
		t.Fatalf("ParameterCount = %d, want 26", metadata.ParameterCount)
	}
}

func buildRepoTestSafetensorsFile(t *testing.T, tensors map[string][]int64, payloadSize int) []byte {
	t.Helper()

	header := map[string]any{
		"__metadata__": map[string]string{"format": "pt"},
	}
	offset := int64(0)
	for name, shape := range tensors {
		count := int64(1)
		for _, dim := range shape {
			count *= dim
		}
		size := count * 2
		header[name] = map[string]any{
			"dtype":        "BF16",
			"shape":        shape,
			"data_offsets": []int64{offset, offset + size},
		}
		offset += size
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	content := make([]byte, 8+len(headerBytes)+payloadSize)
	binary.LittleEndian.PutUint64(content[:8], uint64(len(headerBytes)))
	copy(content[8:], headerBytes)
	return content
}
