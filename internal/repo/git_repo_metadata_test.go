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
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/mirror"
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
	if size := files.SafetensorsSizes["model.safetensors"]; size != int64(len(fullSafetensors)) {
		t.Fatalf("SafetensorsSizes[model.safetensors] = %d, want %d", size, len(fullSafetensors))
	}

	metadata, err := modeldomain.AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}
	if metadata.ParameterCount != 26 {
		t.Fatalf("ParameterCount = %d, want 26", metadata.ParameterCount)
	}
}

func TestCollectSafetensorsFileSkipsReadWhenBudgetSpent(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)

	lfsPointer := []byte("version https://git-lfs.github.com/spec/v1\n" +
		"oid sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n" +
		"size 1024\n")

	if _, err := repo.CreateCommit(ctx, "main", "add model", "Test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "model.safetensors", Content: lfsPointer},
	}, ""); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	// A spent budget means an earlier read already blocked on an upstream
	// download. Opening another tee cache blob would promote it to a foreground
	// download for a read that cannot finish, so only the pointer size is taken.
	spentCtx, cancel := context.WithCancel(ctx)
	cancel()

	metadata := &git.RepoMetadataFiles{
		SafetensorsFiles: make(map[string][]byte),
		SafetensorsSizes: make(map[string]int64),
	}
	NewGitDB(store, nil).(*gitRepo).collectSafetensorsFile(spentCtx, repo, "main", "model.safetensors", metadata)

	if len(metadata.SafetensorsFiles) != 0 {
		t.Fatalf("SafetensorsFiles = %v, want no header read", metadata.SafetensorsFiles)
	}
	if got := metadata.SafetensorsSizes["model.safetensors"]; got != 1024 {
		t.Fatalf("SafetensorsSizes[model.safetensors] = %d, want 1024", got)
	}
}

func TestExtractMetadataFallsBackToLFSPointerSize(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)

	// The pointer stands in for a proxied repository whose weights have not been
	// fetched yet: the blob is a pointer, and no LFS object exists on disk.
	lfsPointer := []byte("version https://git-lfs.github.com/spec/v1\n" +
		"oid sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n" +
		"size 1024\n")

	if _, err := repo.CreateCommit(ctx, "main", "add model", "Test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "config.json", Content: []byte(`{"torch_dtype":"bfloat16"}`)},
		{Type: repository.CommitOperationAdd, Path: "model.safetensors", Content: lfsPointer},
	}, ""); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	files, err := NewGitDB(store, nil).ExtractMetadata(ctx, "models", "test-project", "test-model")
	if err != nil {
		t.Fatalf("ExtractMetadata() error = %v", err)
	}

	if len(files.SafetensorsFiles) != 0 {
		t.Fatalf("SafetensorsFiles = %v, want no readable headers", files.SafetensorsFiles)
	}
	if got := files.SafetensorsSizes["model.safetensors"]; got != 1024 {
		t.Fatalf("SafetensorsSizes[model.safetensors] = %d, want 1024", got)
	}

	metadata, err := modeldomain.AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}
	// 1024 bytes of bfloat16 weights.
	if metadata.ParameterCount != 512 {
		t.Fatalf("ParameterCount = %d, want 512", metadata.ParameterCount)
	}
}

func TestExtractMetadataReadsLocalShardHeadersWhenIndexHasTotalSize(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)

	index, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"total_size": 2048},
		"weight_map": map[string]string{
			"model.embed_tokens.weight": "model-00001-of-00002.safetensors",
			"lm_head.weight":            "model-00002-of-00002.safetensors",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	shard := buildRepoTestSafetensorsFile(t, map[string][]int64{"lm_head.weight": {4, 5}}, 1024)
	if _, err := repo.CreateCommit(ctx, "main", "add sharded model", "Test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "config.json", Content: []byte(`{"torch_dtype":"bfloat16"}`)},
		{Type: repository.CommitOperationAdd, Path: "model.safetensors.index.json", Content: index},
		{Type: repository.CommitOperationAdd, Path: "model-00001-of-00002.safetensors", Content: shard},
		{Type: repository.CommitOperationAdd, Path: "model-00002-of-00002.safetensors", Content: shard},
	}, ""); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	files, err := NewGitDB(store, &mirror.Mirror{}).ExtractMetadata(ctx, "models", "test-project", "test-model")
	if err != nil {
		t.Fatalf("ExtractMetadata() error = %v", err)
	}

	if len(files.SafetensorsFiles) != 2 {
		t.Fatalf("SafetensorsFiles contains %d headers, want 2", len(files.SafetensorsFiles))
	}
	if len(files.SafetensorsSizes) != 2 {
		t.Fatalf("SafetensorsSizes contains %d sizes, want 2", len(files.SafetensorsSizes))
	}
	for path, size := range files.SafetensorsSizes {
		if size != int64(len(shard)) {
			t.Fatalf("SafetensorsSizes[%s] = %d, want %d", path, size, len(shard))
		}
	}

	metadata, err := modeldomain.AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}
	if metadata.ParameterCount != 40 {
		t.Fatalf("ParameterCount = %d, want 40 from local headers", metadata.ParameterCount)
	}
}

func TestExtractMetadataIndexedHeadersNeverAccessMirror(t *testing.T) {
	for _, tt := range []struct {
		name       string
		localLFS   bool
		corrupt    bool
		cancel     bool
		wantHeader int
	}{
		{"all LFS shards local", true, false, false, 2},
		{"one LFS shard missing", false, false, false, 0},
		{"one local header unreadable", true, true, false, 0},
		{"spent budget", true, false, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
			repo := initRepoTestRepository(t, ctx, store)
			first := buildRepoTestSafetensorsFile(t, map[string][]int64{"a": {2, 3}}, 64)
			second := buildRepoTestSafetensorsFile(t, map[string][]int64{"b": {4, 5}}, 64)
			if tt.corrupt {
				second = []byte("invalid")
			}
			firstPointer := repoTestLFSPointer(t, store, first, true)
			secondPointer := repoTestLFSPointer(t, store, second, tt.localLFS)
			index := []byte(`{"metadata":{"total_size":52},"weight_map":{"a":"a.safetensors","b":"b.safetensors"}}`)
			if _, err := repo.CreateCommit(ctx, "main", "add indexed LFS model", "Test", "test@example.com", []repository.CommitOperation{
				{Type: repository.CommitOperationAdd, Path: "config.json", Content: []byte(`{"torch_dtype":"bfloat16"}`)},
				{Type: repository.CommitOperationAdd, Path: "model.safetensors.index.json", Content: index},
				{Type: repository.CommitOperationAdd, Path: "a.safetensors", Content: firstPointer},
				{Type: repository.CommitOperationAdd, Path: "b.safetensors", Content: secondPointer},
			}, ""); err != nil {
				t.Fatal(err)
			}
			if tt.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			// An uninitialized mirror panics on Get, detecting any tee-cache access.
			files, err := NewGitDB(store, &mirror.Mirror{}).ExtractMetadata(ctx, "models", "test-project", "test-model")
			if err != nil {
				t.Fatal(err)
			}
			if len(files.SafetensorsFiles) != tt.wantHeader {
				t.Fatalf("got %d headers, want %d", len(files.SafetensorsFiles), tt.wantHeader)
			}
			if files.SafetensorsSizes["a.safetensors"] != int64(len(first)) ||
				files.SafetensorsSizes["b.safetensors"] != int64(len(second)) {
				t.Fatalf("full LFS sizes not retained: %v", files.SafetensorsSizes)
			}
			metadata, err := modeldomain.AnalyzeRepoMetadata(files)
			if err != nil {
				t.Fatal(err)
			}
			if metadata.ParameterCount != 26 {
				t.Fatalf("ParameterCount = %d, want 26", metadata.ParameterCount)
			}
		})
	}
}

func TestExtractMetadataUnindexedLocalLFSHeader(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)
	shard := buildRepoTestSafetensorsFile(t, map[string][]int64{"weight": {2, 3}}, 64)
	pointer := repoTestLFSPointer(t, store, shard, true)
	if _, err := repo.CreateCommit(ctx, "main", "add local LFS model", "Test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "model.safetensors", Content: pointer},
	}, ""); err != nil {
		t.Fatal(err)
	}
	files, err := NewGitDB(store, &mirror.Mirror{}).ExtractMetadata(ctx, "models", "test-project", "test-model")
	if err != nil {
		t.Fatal(err)
	}
	if len(files.SafetensorsFiles) != 1 || files.SafetensorsSizes["model.safetensors"] != int64(len(shard)) {
		t.Fatalf("expected local header and full size, got %d headers and %v", len(files.SafetensorsFiles), files.SafetensorsSizes)
	}
	metadata, err := modeldomain.AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ParameterCount != 6 {
		t.Fatalf("ParameterCount = %d, want 6", metadata.ParameterCount)
	}
}

func repoTestLFSPointer(t *testing.T, store *hfdstorage.Storage, content []byte, local bool) []byte {
	t.Helper()
	oid := fmt.Sprintf("%x", sha256.Sum256(content))
	if local {
		objectPath := filepath.Join(store.LFSDir(), oid[:2], oid[2:4], oid[4:])
		if err := os.MkdirAll(filepath.Dir(objectPath), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(objectPath, content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return []byte(fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, len(content)))
}

func TestReadSafetensorsHeaderRejectsOversizedHeaders(t *testing.T) {
	var prefix [8]byte
	binary.LittleEndian.PutUint64(prefix[:], maxSafetensorsHeaderBytes+1)
	if _, err := readSafetensorsHeaderFrom(bytes.NewReader(prefix[:])); err == nil {
		t.Fatal("expected an oversized header to be rejected before allocating its body")
	}
}

func TestIndexedSafetensorsPathsRequireEveryReference(t *testing.T) {
	index := parseSafetensorsIndex([]byte(`{"metadata":{"total_size":100},"weight_map":{"a":"a.safetensors","b":"../b.safetensors"}}`))
	paths, complete := index.safetensorsPaths()
	if complete || len(paths) != 1 || paths[0] != "a.safetensors" {
		t.Fatalf("safetensorsPaths = %v, %v; want the valid subset but incomplete", paths, complete)
	}
}

func initRepoTestRepository(t *testing.T, ctx context.Context, store *hfdstorage.Storage) *repository.Repository {
	t.Helper()

	repoPath := store.ResolvePath("test-project/test-model")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0750); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	repo, err := repository.Init(ctx, repoPath, "main")
	if err != nil {
		t.Fatalf("repository.Init() error = %v", err)
	}
	return repo
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
