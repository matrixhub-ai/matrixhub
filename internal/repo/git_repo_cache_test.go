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
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	if _, err := repo.ListRevisionFiles(ctx, "models", project, name, "no-such-branch"); !errors.Is(err, git.ErrRevisionNotFound) {
		t.Errorf("unknown revision error = %v, want ErrRevisionNotFound", err)
	}
	if _, err := repo.ListRevisionFiles(ctx, "models", project, "missing", ""); !errors.Is(err, git.ErrRevisionNotFound) {
		t.Errorf("missing repository error = %v, want ErrRevisionNotFound", err)
	}
}

func TestHasCommit(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil)
	if err := repo.CreateRepository(ctx, "models", "proj", "model"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	head, err := repo.ListRevisionFiles(ctx, "models", "proj", "model", "")
	if err != nil {
		t.Fatalf("ListRevisionFiles() error = %v", err)
	}

	tests := []struct {
		name, model, commit string
		want                bool
	}{
		{"commit of the repository", "model", head.Commit, true},
		{"upper case commit", "model", strings.ToUpper(head.Commit), true},
		{"unknown commit", "model", strings.Repeat("0", 40), false},
		{"branch name", "model", "main", false},
		{"short commit", "model", head.Commit[:7], false},
		{"missing repository", "missing", head.Commit, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repo.HasCommit(ctx, "models", "proj", tt.model, tt.commit); got != tt.want {
				t.Errorf("HasCommit(%s) = %v, want %v", tt.commit, got, tt.want)
			}
		})
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
	// The store maps an oid to <lfs>/oid[0:2]/oid[2:4]/oid[4:], so without
	// the oid check this one would name the file next to the LFS directory.
	escape := filepath.Join(store.LFSDir(), "..", "escape")
	if err := os.WriteFile(escape, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if repo.LFSObjectExists(ctx, "../../escape") {
		t.Fatal("LFSObjectExists() accepted an oid that leaves the LFS directory")
	}
}

func TestFetchLFSObjectWithoutMirror(t *testing.T) {
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil)

	if d := repo.FetchLFSObject(context.Background(), strings.Repeat("a", 64)); d != nil {
		t.Fatalf("FetchLFSObject() = %v, want nil without a mirror", d)
	}
}
