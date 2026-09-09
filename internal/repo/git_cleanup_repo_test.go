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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"
)

func TestFindOrphanedLFSShardedObjects(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	referencedOID := strings.Repeat("a", 64)
	orphanedOID := strings.Repeat("b", 64)
	for _, oid := range []string{referencedOID, orphanedOID} {
		objectPath := filepath.Join(store.LFSDir(), oid[:2], oid[2:4], oid[4:])
		if err := os.MkdirAll(filepath.Dir(objectPath), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(objectPath, []byte("object"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(store.LFSDir(), "bb", "bb", "lfsd_tmp_x"), []byte("pending"), 0600); err != nil {
		t.Fatal(err)
	}

	repoPath := store.ResolvePath("test-project/test-model")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0750); err != nil {
		t.Fatal(err)
	}
	repo, err := repository.Init(ctx, repoPath, "main")
	if err != nil {
		t.Fatal(err)
	}
	pointer := "version https://git-lfs.github.com/spec/v1\noid sha256:" + referencedOID + "\nsize 1048576\n"
	if _, err := repo.CreateCommit(ctx, "main", "add model", "Test", "test@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "model.bin", Content: []byte(pointer)},
	}, ""); err != nil {
		t.Fatal(err)
	}

	gitRepo := NewGitDB(store, nil)
	orphaned, err := gitRepo.FindOrphanedLFS(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(orphaned) != 1 {
		oids := make([]string, 0, len(orphaned))
		for _, object := range orphaned {
			oids = append(oids, object.OID)
		}
		t.Fatalf("expected one orphaned LFS object, got %d: %q", len(orphaned), oids)
	}
	wantPath := filepath.Join(store.LFSDir(), orphanedOID[:2], orphanedOID[2:4], orphanedOID[4:])
	if orphaned[0].OID != orphanedOID || orphaned[0].Path != wantPath {
		t.Fatalf("expected OID %q at %q, got %+v", orphanedOID, wantPath, orphaned[0])
	}
	if err := gitRepo.DeleteLFSObject(ctx, orphaned[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wantPath); !os.IsNotExist(err) {
		t.Fatalf("expected orphaned LFS file removed, got %v", err)
	}
}

func TestConfinedPathAcceptsRelativePathAlreadyUnderRoot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	root := filepath.Join("data", "lfs")
	path := filepath.Join("data", "lfs", "fd", "f7", "object")
	got, err := confinedPath(root, path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := filepath.Join(cwd, path)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestConfinedPathJoinsObjectRelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	got, err := confinedPath(filepath.Join("data", "lfs"), filepath.Join("fd", "f7", "object"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := filepath.Join(cwd, "data", "lfs", "fd", "f7", "object")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestConfinedPathRejectsEscapedPath(t *testing.T) {
	if _, err := confinedPath(filepath.Join("data", "lfs"), filepath.Join("..", "repositories", "repo.git")); err == nil {
		t.Fatal("expected escaped path error")
	}
}
