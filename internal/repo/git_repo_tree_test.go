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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	xetlocal "github.com/wzshiming/xet/storage/local"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// storedXetHash returns the xet file hash the store recorded for oid.
func storedXetHash(t *testing.T, xs *xetlocal.Storage, oid string) string {
	t.Helper()
	raw, err := hex.DecodeString(oid)
	if err != nil {
		t.Fatalf("decode oid: %v", err)
	}
	fh, err := xs.GetFileHashBySHA256(context.Background(), "default", [32]byte(raw))
	if err != nil {
		t.Fatalf("GetFileHashBySHA256(%s) error = %v", oid, err)
	}
	return fh.String()
}

// lfsPointerText returns an LFS pointer for oid in the same format newLFSCollectFixture writes.
func lfsPointerText(oid string) string {
	return fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, lfsTestObjectSize)
}

func TestXetHashOnTreeEntries(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, _ := newLFSCollectFixture(t)
	var unknown [32]byte
	if _, err := rand.Read(unknown[:]); err != nil {
		t.Fatal(err)
	}
	unknownOID := hex.EncodeToString(unknown[:])
	pointer := lfsPointerText(unknownOID)
	if _, err := repo.CreateCommit(ctx, "model", "p", "live", "main", &git.Commit{
		Message: "add files", AuthorName: "test", AuthorEmail: "test@example.com",
	}, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "missing.bin", Content: []byte(pointer)},
		{Type: git.CommitOperationAdd, Path: "README.md", Content: []byte("# live\n")},
	}); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	entries, err := repo.GetTree(ctx, "model", "p", "live", "main", "")
	if err != nil {
		t.Fatalf("GetTree() error = %v", err)
	}
	byName := map[string]*git.TreeEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	pointerSize := int64(len(pointer))
	type treeWant struct {
		isLFS       bool
		xetHash     string
		sha256      string
		pointerSize int64
		size        int64
	}
	want := map[string]treeWant{
		"model.bin":   {true, storedXetHash(t, xs, live), live, pointerSize, lfsTestObjectSize},
		"missing.bin": {true, "", unknownOID, pointerSize, lfsTestObjectSize},
		"README.md":   {false, "", "", 0, int64(len("# live\n"))},
	}
	matches := func(e *git.TreeEntry, w treeWant) bool {
		return e != nil && e.IsLFS == w.isLFS && e.XetHash == w.xetHash &&
			e.Sha256 == w.sha256 && e.PointerSize == w.pointerSize && e.Size == w.size
	}
	for name, w := range want {
		if e := byName[name]; !matches(e, w) {
			t.Fatalf("GetTree() %s = %+v, want %+v", name, e, w)
		}
		blob, err := repo.GetBlob(ctx, "model", "p", "live", "main", name)
		if err != nil {
			t.Fatalf("GetBlob(%s) error = %v", name, err)
		}
		if !matches(blob, w) {
			t.Fatalf("GetBlob(%s) = %+v, want %+v", name, blob, w)
		}
	}
}

func TestXetHashWithoutMirror(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, nil, 0)
	if err := repo.CreateRepository(ctx, "model", "p", "plain"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	oid := hex.EncodeToString(make([]byte, 32))
	pointer := lfsPointerText(oid)
	if _, err := repo.CreateCommit(ctx, "model", "p", "plain", "main", &git.Commit{
		Message: "add model", AuthorName: "test", AuthorEmail: "test@example.com",
	}, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "model.bin", Content: []byte(pointer)},
	}); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}

	entries, err := repo.GetTree(ctx, "model", "p", "plain", "main", "")
	if err != nil {
		t.Fatalf("GetTree() error = %v", err)
	}
	var found *git.TreeEntry
	for _, e := range entries {
		if e.Name == "model.bin" {
			found = e
		}
	}
	ok := func(e *git.TreeEntry) bool {
		return e != nil && e.IsLFS && e.XetHash == "" && e.Sha256 == oid && e.PointerSize == int64(len(pointer))
	}
	if !ok(found) {
		t.Fatalf("GetTree() model.bin = %+v, want IsLFS=true XetHash=\"\" Sha256=%s PointerSize=%d", found, oid, len(pointer))
	}
	entry, err := repo.GetBlob(ctx, "model", "p", "plain", "main", "model.bin")
	if err != nil {
		t.Fatalf("GetBlob() error = %v", err)
	}
	if !ok(entry) {
		t.Fatalf("GetBlob() = %+v, want IsLFS=true XetHash=\"\" Sha256=%s PointerSize=%d", entry, oid, len(pointer))
	}
}
