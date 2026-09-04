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
	"path/filepath"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/mirror"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	xetclient "github.com/wzshiming/xet/client"
	xetstorage "github.com/wzshiming/xet/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestConfinedRepoPathAcceptsRelPath(t *testing.T) {
	got, err := confinedRepoPath("test-project/orphan.git")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "/test-project/orphan.git" {
		t.Fatalf("expected %q, got %q", "/test-project/orphan.git", got)
	}
}

func TestConfinedRepoPathRejectsEscapedPath(t *testing.T) {
	if _, err := confinedRepoPath("../lfs/object"); err == nil {
		t.Fatal("expected escaped path error")
	}
}

func TestConfinedRepoPathRejectsRoot(t *testing.T) {
	if _, err := confinedRepoPath(""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := confinedRepoPath("/"); err == nil {
		t.Fatal("expected error for root path")
	}
}

func TestFindOrphanedReposAndDelete(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil, nil, 0)

	if err := repo.CreateRepository(ctx, "model", "test-project", "valid"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	if err := repo.CreateRepository(ctx, "model", "test-project", "orphan"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	orphaned, err := repo.FindOrphanedRepos(ctx, []string{"test-project/valid"}, nil)
	if err != nil {
		t.Fatalf("FindOrphanedRepos() error = %v", err)
	}
	if len(orphaned) != 1 {
		t.Fatalf("expected 1 orphaned repo, got %d", len(orphaned))
	}
	if orphaned[0].Path != "test-project/orphan.git" {
		t.Fatalf("orphaned path = %q, want %q", orphaned[0].Path, "test-project/orphan.git")
	}
	if orphaned[0].SizeBytes <= 0 {
		t.Fatalf("orphaned repo size = %d, want > 0", orphaned[0].SizeBytes)
	}
	if size := repo.RepositoriesSize(ctx); size <= 0 {
		t.Fatalf("RepositoriesSize() = %d, want > 0", size)
	}

	if err := repo.DeleteRepositoryAtRelPath(ctx, orphaned[0].Path); err != nil {
		t.Fatalf("DeleteRepositoryAtRelPath() error = %v", err)
	}
	exists, err := repo.RepositoryExists(ctx, "model", "test-project", "orphan")
	if err != nil {
		t.Fatalf("RepositoryExists() error = %v", err)
	}
	if exists {
		t.Fatal("expected orphaned repo to be deleted")
	}
}

const lfsTestObjectSize = 64 * 1024

// putLFSObject stores a 64 KiB payload derived from seed through the mirror and returns its OID.
func putLFSObject(t *testing.T, m *mirror.Mirror, seed string) string {
	t.Helper()
	data := bytes.Repeat([]byte(seed), lfsTestObjectSize/len(seed)+1)[:lfsTestObjectSize]
	sum := sha256.Sum256(data)
	oid := hex.EncodeToString(sum[:])
	if err := m.PutObject(context.Background(), oid, bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("PutObject(%q) error = %v", seed, err)
	}
	return oid
}

// lfsStored reports whether the xet store still resolves oid.
func lfsStored(t *testing.T, xs *xetstorage.FileStorage, oid string) bool {
	t.Helper()
	raw, err := hex.DecodeString(oid)
	if err != nil {
		t.Fatalf("decode oid: %v", err)
	}
	_, err = xs.GetFileHashBySHA256(context.Background(), "default", [32]byte(raw))
	return err == nil
}

// newLFSCollectFixture seeds a xet store with a live object referenced by a repo pointer and an unreferenced dead one.
func newLFSCollectFixture(t *testing.T) (git.IGitRepo, *xetstorage.FileStorage, string, string) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	xs, err := xetstorage.NewFileStorage(xetstorage.WithBasePath(filepath.Join(dir, "storage")))
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}
	client, err := xetclient.NewClient(xetclient.WithCacheDir(filepath.Join(dir, "chunks")))
	if err != nil {
		t.Fatalf("xetclient.NewClient() error = %v", err)
	}
	m, err := mirror.NewMirror(mirror.WithXETStorage(xs), mirror.WithXETClient(client), mirror.WithDataDir(dir))
	if err != nil {
		t.Fatalf("NewMirror() error = %v", err)
	}
	t.Cleanup(m.Wait)
	live, dead := putLFSObject(t, m, "live "), putLFSObject(t, m, "dead ")

	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), m, xs, -1)
	if err := repo.CreateRepository(ctx, "model", "p", "live"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	pointer := fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", live, lfsTestObjectSize)
	if _, err := repo.CreateCommit(ctx, "model", "p", "live", "main", &git.Commit{
		Message:     "add model",
		AuthorName:  "test",
		AuthorEmail: "test@example.com",
	}, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "model.bin", Content: []byte(pointer)},
	}); err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}
	return repo, xs, live, dead
}

func TestCollectLFSDryRunListsWithoutDeleting(t *testing.T) {
	ctx := context.Background()
	repo, xs, _, dead := newLFSCollectFixture(t)

	res, err := repo.CollectLFS(ctx, true)
	if err != nil {
		t.Fatalf("CollectLFS() error = %v", err)
	}
	if len(res.Orphaned) != 1 || res.Orphaned[0].OID != dead || res.ReclaimedBytes != 0 {
		t.Fatalf("CollectLFS() = %+v, want only %s and 0 reclaimed", res, dead)
	}
	if !lfsStored(t, xs, dead) {
		t.Fatal("dry run unlinked the dead object")
	}
}

func TestCollectLFSReclaimsUnreferencedObjects(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)

	before := repo.LFSSize(ctx)
	if before <= 0 {
		t.Fatalf("LFSSize() before = %d, want > 0", before)
	}

	res, err := repo.CollectLFS(ctx, false)
	if err != nil {
		t.Fatalf("CollectLFS() error = %v", err)
	}
	if len(res.Orphaned) != 1 || res.Orphaned[0].OID != dead || res.ReclaimedBytes <= 0 {
		t.Fatalf("CollectLFS() = %+v, want only %s and reclaimed > 0", res, dead)
	}
	if lfsStored(t, xs, dead) || !lfsStored(t, xs, live) {
		t.Fatalf("dead stored = %v, live stored = %v", lfsStored(t, xs, dead), lfsStored(t, xs, live))
	}
	if after := repo.LFSSize(ctx); after <= 0 || after >= before {
		t.Fatalf("LFSSize() after = %d, want in (0, %d)", after, before)
	}
}

func TestCollectLFSWithoutXetStore(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil, nil, 0)

	if _, err := repo.CollectLFS(ctx, true); err == nil {
		t.Fatal("expected CollectLFS to fail without a xet store")
	}
	if size := repo.LFSSize(ctx); size != 0 {
		t.Fatalf("LFSSize() = %d, want 0", size)
	}
}
