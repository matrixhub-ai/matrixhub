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
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	hfdlfs "github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func lfsPointer(oid string, size int64) []byte {
	return fmt.Appendf(nil, "version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, size)
}

func TestListRevisionFiles(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := NewGitDB(store, nil)

	const (
		project = "proj"
		name    = "model"
		oidA    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		oidB    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	if err := repo.CreateRepository(ctx, "models", project, name); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	commit := &git.Commit{Message: "add files", AuthorName: "test", AuthorEmail: "test@example.com"}
	first, err := repo.CreateCommit(ctx, "models", project, name, "main", commit, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "README.md", Content: []byte("v1")},
		{Type: git.CommitOperationAdd, Path: "model.safetensors", Content: lfsPointer(oidA, 1024)},
		{Type: git.CommitOperationAdd, Path: "nested/weights.bin", Content: lfsPointer(oidB, 2048)},
	})
	if err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}
	if _, err := repo.CreateCommit(ctx, "models", project, name, "main", commit, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "README.md", Content: []byte("version two")},
	}); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	head, err := repo.ListRevisionFiles(ctx, "models", project, name, "")
	if err != nil {
		t.Fatalf("ListRevisionFiles() error = %v", err)
	}
	if head.Revision != "main" || head.Commit == "" || head.Commit == first {
		t.Fatalf("head = %q@%q, want main at the second commit", head.Revision, head.Commit)
	}
	byPath := map[string]git.RevisionFile{}
	for _, f := range head.Files {
		byPath[f.Path] = f
	}
	want := map[string]git.RevisionFile{
		".gitattributes":     {Path: ".gitattributes", Size: int64(len(repository.GitattributesText))},
		"README.md":          {Path: "README.md", Size: int64(len("version two"))},
		"model.safetensors":  {Path: "model.safetensors", Size: 1024, IsLFS: true, OID: oidA},
		"nested/weights.bin": {Path: "nested/weights.bin", Size: 2048, IsLFS: true, OID: oidB},
	}
	for p, w := range want {
		if got, ok := byPath[p]; !ok || got != w {
			t.Errorf("file %s = %+v, want %+v", p, got, w)
		}
	}
	if len(head.Files) != len(want) {
		t.Errorf("got %d files, want %d", len(head.Files), len(want))
	}

	byBranch, err := repo.ListRevisionFiles(ctx, "models", project, name, "main")
	if err != nil || byBranch.Commit != head.Commit {
		t.Fatalf("ListRevisionFiles(main) = %v, %v; want commit %s", byBranch, err, head.Commit)
	}

	old, err := repo.ListRevisionFiles(ctx, "models", project, name, first)
	if err != nil {
		t.Fatalf("ListRevisionFiles(first) error = %v", err)
	}
	for _, f := range old.Files {
		if f.Path == "README.md" && f.Size != int64(len("v1")) {
			t.Errorf("old README size = %d, want %d", f.Size, len("v1"))
		}
	}

	if _, err := repo.ListRevisionFiles(ctx, "models", project, name, "no-such-branch"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("unknown revision error = %v, want a not found error", err)
	}
	if _, err := repo.ListRevisionFiles(ctx, "models", project, "missing", ""); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("missing repository error = %v, want does not exist", err)
	}
}

func TestLFSObjectExists(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := NewGitDB(store, nil)

	content := []byte("weights")
	sum := sha256.Sum256(content)
	oid := hex.EncodeToString(sum[:])

	if repo.LFSObjectExists(ctx, oid) {
		t.Fatal("LFSObjectExists() = true before the object is stored")
	}
	if err := hfdlfs.NewLocal(store.LFSDir()).Put(oid, bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if !repo.LFSObjectExists(ctx, oid) {
		t.Fatal("LFSObjectExists() = false after the object is stored")
	}
	if repo.LFSObjectExists(ctx, "../../etc/passwd") {
		t.Fatal("LFSObjectExists() accepted an invalid oid")
	}
}

func TestFetchLFSObjectWithoutMirror(t *testing.T) {
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil)

	if d := repo.FetchLFSObject(context.Background(), strings.Repeat("a", 64)); d != nil {
		t.Fatalf("FetchLFSObject() = %v, want nil without a mirror", d)
	}
}
