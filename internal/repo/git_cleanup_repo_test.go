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
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/osfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	xetclient "github.com/wzshiming/xet/client"
	xetstorage "github.com/wzshiming/xet/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestRepositoriesSize(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil, nil, 0)

	if err := repo.CreateRepository(ctx, "model", "test-project", "valid"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	if size := repo.RepositoriesSize(ctx); size <= 0 {
		t.Fatalf("RepositoriesSize() = %d, want > 0", size)
	}
}

func TestPruneRepos(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil, nil, 0)

	for _, r := range [][3]string{{"model", "test-project", "valid"}, {"dataset", "test-project", "dvalid"}, {"model", "test-project", "orphan"}} {
		if err := repo.CreateRepository(ctx, r[0], r[1], r[2]); err != nil {
			t.Fatalf("CreateRepository(%v) error = %v", r, err)
		}
	}
	models := []string{"test-project/valid"}
	datasets := []string{"test-project/dvalid"}

	orphaned, err := repo.PruneRepos(ctx, models, datasets, true)
	if err != nil {
		t.Fatalf("PruneRepos(dry-run) error = %v", err)
	}
	if len(orphaned) != 1 || orphaned[0].Path != "test-project/orphan.git" {
		t.Fatalf("PruneRepos(dry-run) = %+v, want only test-project/orphan.git", orphaned)
	}
	if orphaned[0].SizeBytes <= 0 {
		t.Fatalf("orphaned repo size = %d, want > 0", orphaned[0].SizeBytes)
	}
	if exists, err := repo.RepositoryExists(ctx, "model", "test-project", "orphan"); err != nil || !exists {
		t.Fatalf("RepositoryExists(orphan) after dry-run = %v, %v; want true", exists, err)
	}

	orphaned, err = repo.PruneRepos(ctx, models, datasets, false)
	if err != nil {
		t.Fatalf("PruneRepos() error = %v", err)
	}
	if len(orphaned) != 1 || orphaned[0].Path != "test-project/orphan.git" {
		t.Fatalf("PruneRepos() = %+v, want only test-project/orphan.git", orphaned)
	}
	if exists, err := repo.RepositoryExists(ctx, "model", "test-project", "orphan"); err != nil || exists {
		t.Fatalf("RepositoryExists(orphan) after prune = %v, %v; want false", exists, err)
	}
	for _, r := range [][2]string{{"model", "valid"}, {"dataset", "dvalid"}} {
		if exists, err := repo.RepositoryExists(ctx, r[0], "test-project", r[1]); err != nil || !exists {
			t.Fatalf("RepositoryExists(%s %s) after prune = %v, %v; want true", r[0], r[1], exists, err)
		}
	}

	if orphaned, err := repo.PruneRepos(ctx, models, datasets, true); err != nil || len(orphaned) != 0 {
		t.Fatalf("PruneRepos(dry-run) after prune = %+v, %v; want none", orphaned, err)
	}
}

func TestPruneReposWithoutRepositoriesDir(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil, nil, 0)

	if orphaned, err := repo.PruneRepos(ctx, nil, nil, false); err != nil || len(orphaned) != 0 {
		t.Fatalf("PruneRepos() = %+v, %v; want none", orphaned, err)
	}
}

func TestPruneReposConfinedToRepositoriesRoot(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(rootDir)), nil, nil, 0)
	if err := repo.CreateRepository(ctx, "model", "test-project", "orphan"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	// A HEAD-bearing .git directory beside the repositories root would be pruned if the walk escaped it.
	sentinel := filepath.Join(rootDir, "stray.git", "HEAD")
	want := []byte("ref: refs/heads/main\n")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sentinel, want, 0600); err != nil {
		t.Fatal(err)
	}

	orphaned, err := repo.PruneRepos(ctx, nil, nil, false)
	if err != nil {
		t.Fatalf("PruneRepos() error = %v", err)
	}
	if len(orphaned) != 1 || orphaned[0].Path != "test-project/orphan.git" {
		t.Fatalf("PruneRepos() = %+v, want only test-project/orphan.git", orphaned)
	}
	if _, err := os.Stat(filepath.Join(rootDir, "repositories", "test-project", "orphan.git", "HEAD")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat(orphan HEAD) after prune error = %v, want not exist", err)
	}
	if got, err := os.ReadFile(sentinel); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("sentinel after prune = %q, %v; want unchanged", got, err)
	}
	if info, err := os.Stat(filepath.Join(rootDir, "repositories")); err != nil || !info.IsDir() {
		t.Fatalf("Stat(repositories) after prune = %v, %v; want directory kept", info, err)
	}
}

type readDirHookFS struct {
	billy.Filesystem
	onReadDir func(path string)
}

func (h *readDirHookFS) ReadDir(path string) ([]fs.DirEntry, error) {
	h.onReadDir(path)
	return h.Filesystem.ReadDir(path)
}

func (h *readDirHookFS) Chroot(path string) (billy.Filesystem, error) {
	sub, err := h.Filesystem.Chroot(path)
	if err != nil {
		return nil, err
	}
	return &readDirHookFS{Filesystem: sub, onReadDir: h.onReadDir}, nil
}

func TestPruneReposCancelledWhileSizing(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%t", dryRun), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rootDir := t.TempDir()
			var sizingArmed bool
			var cancelledAt string
			fsys := &readDirHookFS{Filesystem: osfs.New(rootDir), onReadDir: func(path string) {
				if sizingArmed && cancelledAt == "" && strings.HasSuffix(path, ".git") {
					cancelledAt = path
					cancel()
				}
			}}
			repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithFilesystem(fsys)), nil, nil, 0)
			if err := repo.CreateRepository(ctx, "model", "test-project", "orphan"); err != nil {
				t.Fatalf("CreateRepository() error = %v", err)
			}
			headFile := filepath.Join(rootDir, "repositories", "test-project", "orphan.git", "HEAD")
			if _, err := os.Stat(headFile); err != nil {
				t.Fatalf("Stat(HEAD) before prune error = %v", err)
			}
			if ctx.Err() != nil {
				t.Fatalf("context cancelled during setup: %v", ctx.Err())
			}
			sizingArmed = true

			orphaned, err := repo.PruneRepos(ctx, nil, nil, dryRun)

			if cancelledAt != "/test-project/orphan.git" {
				t.Fatalf("cancelled at %q, want while sizing /test-project/orphan.git", cancelledAt)
			}
			if !errors.Is(err, context.Canceled) {
				t.Errorf("PruneRepos() error = %v, want context.Canceled", err)
			}
			if len(orphaned) != 0 {
				t.Errorf("PruneRepos() = %+v, want none", orphaned)
			}
			if _, err := os.Stat(headFile); err != nil {
				t.Errorf("Stat(HEAD) after cancelled prune error = %v, want repository kept", err)
			}
		})
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

func TestPruneDryRunListsWithoutDeleting(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)
	before := repo.LFSSize(ctx)
	if before <= 0 {
		t.Fatalf("LFSSize() before = %d, want > 0", before)
	}

	res, err := repo.Prune(ctx, true)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if len(res.Unlinked) != 1 || res.Unlinked[0].OID != dead {
		t.Fatalf("Prune() = %+v, want only %s", res.Unlinked, dead)
	}
	if res.Unlinked[0].SizeBytes != lfsTestObjectSize {
		t.Fatalf("orphan size = %d, want %d", res.Unlinked[0].SizeBytes, lfsTestObjectSize)
	}
	if res.ReclaimedBytes != 0 {
		t.Fatalf("dry run ReclaimedBytes = %d, want 0", res.ReclaimedBytes)
	}
	if !lfsStored(t, xs, dead) {
		t.Fatal("listing unlinked the dead object")
	}
	if !lfsStored(t, xs, live) {
		t.Fatal("listing unlinked the live object")
	}
	if after := repo.LFSSize(ctx); after != before {
		t.Fatalf("LFSSize() after listing = %d, want %d", after, before)
	}
}

func TestPruneReclaimsUnreferencedObjects(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)

	before := repo.LFSSize(ctx)
	if before <= 0 {
		t.Fatalf("LFSSize() before = %d, want > 0", before)
	}

	res, err := repo.Prune(ctx, false)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if len(res.Unlinked) != 1 || res.Unlinked[0].OID != dead {
		t.Fatalf("Prune() = %+v, want only %s", res.Unlinked, dead)
	}
	if res.Unlinked[0].SizeBytes != lfsTestObjectSize {
		t.Fatalf("pruned size = %d, want %d", res.Unlinked[0].SizeBytes, lfsTestObjectSize)
	}
	if lfsStored(t, xs, dead) || !lfsStored(t, xs, live) {
		t.Fatalf("dead stored = %v, live stored = %v", lfsStored(t, xs, dead), lfsStored(t, xs, live))
	}
	after := repo.LFSSize(ctx)
	if after <= 0 || after >= before {
		t.Fatalf("LFSSize() after = %d, want in (0, %d)", after, before)
	}
	if res.ReclaimedBytes != before-after {
		t.Fatalf("ReclaimedBytes = %d, want %d", res.ReclaimedBytes, before-after)
	}
	if res, err := repo.Prune(ctx, true); err != nil || len(res.Unlinked) != 0 {
		t.Fatalf("Prune() dry run after delete = %+v, %v; want none", res, err)
	}
	if res, err := repo.Prune(ctx, false); err != nil || len(res.Unlinked) != 0 || res.ReclaimedBytes != 0 {
		t.Fatalf("Prune() after prune = %+v, %v; want none", res, err)
	}
}

func TestPruneDryRunSkipsUnlinkedObjects(t *testing.T) {
	ctx := context.Background()
	repo, store, live, dead := newLFSCollectFixture(t)
	removed, err := store.DeleteSHA256IndexEntry(ctx, dead)
	if err != nil || !removed {
		t.Fatalf("unlink dead object = %v, %v", removed, err)
	}
	before := repo.LFSSize(ctx)
	res, err := repo.Prune(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unlinked) != 0 {
		t.Fatalf("Prune() = %+v, want no candidates", res.Unlinked)
	}
	if repo.LFSSize(ctx) != before || !lfsStored(t, store, live) {
		t.Fatal("listing modified storage")
	}
}

func TestLFSCleanupWithoutXetStore(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil, nil, 0)

	if _, err := repo.Prune(ctx, true); err == nil {
		t.Fatal("expected Prune dry-run to fail without a xet store")
	}
	if _, err := repo.Prune(ctx, false); err == nil {
		t.Fatal("expected Prune to fail without a xet store")
	}
	if size := repo.LFSSize(ctx); size != 0 {
		t.Fatalf("LFSSize() = %d, want 0", size)
	}
}
