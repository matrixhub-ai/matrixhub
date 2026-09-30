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
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/mirror"
	xetclient "github.com/wzshiming/xet/client"
	xetstorage "github.com/wzshiming/xet/storage"
	xetlocal "github.com/wzshiming/xet/storage/local"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestUsageWithoutXetStore(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, nil, 0)

	if err := repo.CreateRepository(ctx, "model", "test-project", "valid"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}
	usage, err := repo.Usage(ctx)
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}
	// The initial commit stores a blob, a tree and a commit.
	if usage.Git.Objects.Count != 3 || usage.Git.Objects.Bytes <= 0 || usage.Git.Other.Count <= 0 || usage.Git.Other.Bytes <= 0 {
		t.Fatalf("Usage() Git = %+v, want 3 objects and other files with bytes", usage.Git)
	}
	if usage.Xet != (git.XetUsage{}) {
		t.Fatalf("Usage() Xet = %+v, want zero without a xet store", usage.Xet)
	}
}

func TestUsageExcludesNonRepositoryFiles(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	repo := NewGitDB(newRepoTestStorage(t, rootDir), nil, nil, 0)
	for _, repoType := range []string{"model", "dataset", "space"} {
		if err := repo.CreateRepository(ctx, repoType, "project", "valid"); err != nil {
			t.Fatalf("CreateRepository(%s) error = %v", repoType, err)
		}
	}
	before, err := repo.Usage(ctx)
	if err != nil || before.Git.Objects.Count != 9 {
		t.Fatalf("Usage() = %+v, %v; want the objects of three repositories", before, err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "repositories", "stray"), []byte("not a repository"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := repo.Usage(ctx)
	if err != nil || *after != *before {
		t.Fatalf("Usage() with stray file = %+v, %v; want %+v", after, err, before)
	}
}

func TestUsageEmptyStore(t *testing.T) {
	ctx := context.Background()
	xs, err := xetlocal.NewStorage(xetlocal.WithBasePath(t.TempDir()))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, xs, 0)

	if usage, err := repo.Usage(ctx); err != nil || *usage != (git.StorageUsage{}) {
		t.Fatalf("Usage() = %+v, %v; want all zero", usage, err)
	}
}

// Usage maps every Collector category and follows the data a prune reclaims.
func TestUsageReportsGitAndXetCategories(t *testing.T) {
	ctx := context.Background()
	repo, xs, _, _ := newLFSCollectFixture(t)

	usage, err := repo.Usage(ctx)
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}
	// The initial and the pointer commit store two blobs, two trees and two commits.
	if usage.Git.Objects.Count != 6 || usage.Git.Objects.Bytes <= 0 || usage.Git.Other.Count <= 0 || usage.Git.Other.Bytes <= 0 {
		t.Fatalf("Usage() Git = %+v, want 6 objects and other files with bytes", usage.Git)
	}
	store, err := xs.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := git.XetUsage{
		Xorbs:       git.ObjectUsage(store.Xorbs),
		Shards:      git.ObjectUsage(store.Shards),
		FileIndex:   git.ObjectUsage(store.FileIndex),
		ChunkIndex:  git.ObjectUsage(store.ChunkIndex),
		SHA256Index: git.ObjectUsage(store.SHA256Index),
	}
	if usage.Xet != want {
		t.Fatalf("Usage() Xet = %+v, want the store's %+v", usage.Xet, want)
	}
	// Two distinct objects: one xorb, shard, file entry and sha256 entry each, and at least one chunk each.
	for name, u := range map[string]git.ObjectUsage{"xorbs": usage.Xet.Xorbs, "shards": usage.Xet.Shards, "file index": usage.Xet.FileIndex, "sha256 index": usage.Xet.SHA256Index} {
		if u.Count != 2 || u.Bytes <= 0 {
			t.Fatalf("Usage() %s = %+v, want 2 entries with bytes", name, u)
		}
	}
	if usage.Xet.ChunkIndex.Count < 2 || usage.Xet.ChunkIndex.Bytes <= 0 {
		t.Fatalf("Usage() chunk index = %+v, want >= 2 entries with bytes", usage.Xet.ChunkIndex)
	}

	if _, err := repo.Prune(ctx, git.PruneOptions{}); err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	after, err := repo.Usage(ctx)
	if err != nil {
		t.Fatalf("Usage() after prune error = %v", err)
	}
	for name, u := range map[string]git.ObjectUsage{"xorbs": after.Xet.Xorbs, "shards": after.Xet.Shards, "file index": after.Xet.FileIndex, "sha256 index": after.Xet.SHA256Index} {
		if u.Count != 1 || u.Bytes <= 0 {
			t.Fatalf("Usage() %s after prune = %+v, want 1 entry with bytes", name, u)
		}
	}
	if after.Xet.ChunkIndex.Count >= usage.Xet.ChunkIndex.Count || after.Xet.ChunkIndex.Bytes >= usage.Xet.ChunkIndex.Bytes {
		t.Fatalf("Usage() chunk index after prune = %+v, want fewer than %+v", after.Xet.ChunkIndex, usage.Xet.ChunkIndex)
	}
}

type failingUsageStore struct {
	xetstorage.Storage
	err error
}

func (s failingUsageStore) Usage(context.Context) (xetstorage.Usage, error) {
	return xetstorage.Usage{}, s.err
}

func TestUsagePropagatesErrors(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	repo := NewGitDB(newRepoTestStorage(t, rootDir), nil, nil, 0)
	if err := repo.CreateRepository(ctx, "model", "p", "broken"); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if usage, err := repo.Usage(canceled); !errors.Is(err, context.Canceled) || usage != nil {
		t.Fatalf("Usage(canceled) = %+v, %v; want context.Canceled", usage, err)
	}

	xs, err := xetlocal.NewStorage(xetlocal.WithBasePath(t.TempDir()))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	storeErr := errors.New("store offline")
	failing := NewGitDB(repo.(*gitRepo).storage, nil, failingUsageStore{Storage: xs, err: storeErr}, 0)
	if usage, err := failing.Usage(ctx); !errors.Is(err, storeErr) || usage != nil {
		t.Fatalf("Usage() with failing store = %+v, %v; want %v", usage, err, storeErr)
	}

	// A repository whose objects directory is a plain file cannot be measured.
	objects := filepath.Join(rootDir, "repositories", "p", "broken.git", "objects")
	if err := os.RemoveAll(objects); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objects, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if usage, err := repo.Usage(ctx); err == nil || usage != nil {
		t.Fatalf("Usage() with unreadable repository = %+v, %v; want an error", usage, err)
	}
}

func TestPruneRepos(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, nil, 0)

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
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, nil, 0)

	if orphaned, err := repo.PruneRepos(ctx, nil, nil, false); err != nil || len(orphaned) != 0 {
		t.Fatalf("PruneRepos() = %+v, %v; want none", orphaned, err)
	}
}

func TestPruneReposConfinedToRepositoriesRoot(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	repo := NewGitDB(newRepoTestStorage(t, rootDir), nil, nil, 0)
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
func lfsStored(t *testing.T, xs *xetlocal.Storage, oid string) bool {
	t.Helper()
	raw, err := hex.DecodeString(oid)
	if err != nil {
		t.Fatalf("decode oid: %v", err)
	}
	_, err = xs.GetFileHashBySHA256(context.Background(), "default", [32]byte(raw))
	return err == nil
}

// lfsPayloadBytes sums the shard and xorb bytes a sweep reclaims; index entries stay out of it.
func lfsPayloadBytes(t *testing.T, repo git.IGitRepo) int64 {
	t.Helper()
	usage, err := repo.Usage(context.Background())
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}
	return usage.Xet.Shards.Bytes + usage.Xet.Xorbs.Bytes
}

// newLFSCollectFixture seeds a xet store with a live object referenced by a repo pointer and an unreferenced dead one.
func newLFSCollectFixture(t *testing.T) (git.IGitRepo, *xetlocal.Storage, string, string) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	xs, err := xetlocal.NewStorage(xetlocal.WithBasePath(filepath.Join(dir, "storage")))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
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

	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), m, xs, -1)
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

// checkPrune asserts the fixture's prune header (one repository, one live object, nothing in grace), the exact Unlinked list and a finished sweep.
func checkPrune(t *testing.T, res *git.GCResult, dryRun bool, unlinked ...string) {
	t.Helper()
	if res.DryRun != dryRun || res.Repositories != 1 || res.LiveObjects != 1 || res.PruneSkippedInGrace != 0 {
		t.Fatalf("Prune() = %+v, want dryRun=%t repositories=1 live=1 skipped=0", res, dryRun)
	}
	if !slices.Equal(res.Unlinked, unlinked) {
		t.Fatalf("Prune() unlinked = %v, want %v", res.Unlinked, unlinked)
	}
	if res.SweepDone == nil || !*res.SweepDone {
		t.Fatalf("Prune() sweepDone = %v, want true", res.SweepDone)
	}
}

func TestPruneDryRunListsWithoutDeleting(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)
	before := lfsPayloadBytes(t, repo)
	if before <= 0 {
		t.Fatalf("LFS payload before = %d, want > 0", before)
	}

	res, err := repo.Prune(ctx, git.PruneOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	checkPrune(t, res, true, dead)
	// The candidate is still linked, so this run's sweep estimate cannot include it.
	if res.XetReclaimedBytes != 0 {
		t.Fatalf("dry run sweep = %+v, want 0 bytes", res)
	}
	if !lfsStored(t, xs, dead) {
		t.Fatal("listing unlinked the dead object")
	}
	if !lfsStored(t, xs, live) {
		t.Fatal("listing unlinked the live object")
	}
	if after := lfsPayloadBytes(t, repo); after != before {
		t.Fatalf("LFS payload after listing = %d, want %d", after, before)
	}
}

func TestPruneReclaimsUnreferencedObjects(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)

	before := lfsPayloadBytes(t, repo)
	if before <= 0 {
		t.Fatalf("LFS payload before = %d, want > 0", before)
	}

	res, err := repo.Prune(ctx, git.PruneOptions{})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	checkPrune(t, res, false, dead)
	if lfsStored(t, xs, dead) || !lfsStored(t, xs, live) {
		t.Fatalf("dead stored = %v, live stored = %v", lfsStored(t, xs, dead), lfsStored(t, xs, live))
	}
	after := lfsPayloadBytes(t, repo)
	if after <= 0 || after >= before {
		t.Fatalf("LFS payload after = %d, want in (0, %d)", after, before)
	}
	if res.XetReclaimedBytes != before-after || res.SweptShards < 1 {
		t.Fatalf("sweep = %+v, want %d bytes, >= 1 shard", res, before-after)
	}
	if res, err := repo.Prune(ctx, git.PruneOptions{DryRun: true}); err != nil || len(res.Unlinked) != 0 || res.XetReclaimedBytes != 0 {
		t.Fatalf("Prune() dry run after delete = %+v, %v; want none", res, err)
	}
	if res, err := repo.Prune(ctx, git.PruneOptions{}); err != nil || len(res.Unlinked) != 0 || res.XetReclaimedBytes != 0 {
		t.Fatalf("Prune() after prune = %+v, %v; want none", res, err)
	}
}

func TestPruneGitGC(t *testing.T) {
	ctx := context.Background()
	repo, store, live, dead := newLFSCollectFixture(t)
	backend := repo.(*gitRepo)
	repoPath := backend.gitPath("model", "p", "live")
	gitRepository, err := backend.openRepo(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	filler := make([]byte, lfsTestObjectSize)
	if _, err := rand.Read(filler); err != nil {
		t.Fatal(err)
	}
	pointer := fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", dead, lfsTestObjectSize)
	orphan, err := repo.CreateCommit(ctx, "model", "p", "live", "orphan", &git.Commit{
		Message: "orphan", AuthorName: "test", AuthorEmail: "test@example.com",
	}, []git.CommitOperation{
		{Type: git.CommitOperationAdd, Path: "dead.bin", Content: []byte(pointer)},
		{Type: git.CommitOperationAdd, Path: "filler.bin", Content: filler},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := gitRepository.DeleteBranch("orphan"); err != nil {
		t.Fatal(err)
	}
	before, err := gitRepository.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	beforeLFS := lfsPayloadBytes(t, repo)
	hour := time.Hour
	graced, err := repo.Prune(ctx, git.PruneOptions{DryRun: true, Grace: &hour})
	if err != nil {
		t.Fatal(err)
	}
	if graced.DeletedGitObjects != 0 || graced.DeletedGitBytes != 0 || graced.GitReclaimedBytes != 0 || len(graced.Unlinked) != 0 || graced.LiveObjects != 2 {
		t.Fatalf("Prune(grace=1h) = %+v, want fresh Git objects and both LFS pointers kept", graced)
	}

	preview, err := repo.Prune(ctx, git.PruneOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	checkPrune(t, preview, true, dead)
	if preview.DeletedGitObjects != 4 || preview.DeletedGitBytes <= lfsTestObjectSize || preview.GitReclaimedBytes != 0 || len(preview.Failed) != 0 {
		t.Fatalf("Prune(dry-run) = %+v, want four Git objects previewed without reclaiming bytes", preview)
	}
	usage, err := repo.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Git != (git.GitUsage{Objects: git.ObjectUsage(before.Objects), Other: git.ObjectUsage(before.Other)}) || usage.Xet.Shards.Bytes+usage.Xet.Xorbs.Bytes != beforeLFS || !lfsStored(t, store, dead) {
		t.Fatal("dry run changed Git or LFS storage")
	}
	if _, err := repo.GetCommit(ctx, "model", "p", "live", orphan); err != nil {
		t.Fatalf("GetCommit(orphan) after dry run: %v", err)
	}

	result, err := repo.Prune(ctx, git.PruneOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkPrune(t, result, false, dead)
	gitRepository, err = backend.openRepo(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	after, err := gitRepository.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.DeletedGitObjects != preview.DeletedGitObjects || result.DeletedGitBytes != preview.DeletedGitBytes || len(result.Failed) != 0 {
		t.Fatalf("Prune() Git deletions = %+v, want preview %+v", result, preview)
	}
	if result.GitReclaimedBytes <= 0 || result.GitReclaimedBytes != before.Objects.Bytes-after.Objects.Bytes {
		t.Fatalf("Prune() reclaimed %d Git bytes, objects shrank by %d", result.GitReclaimedBytes, before.Objects.Bytes-after.Objects.Bytes)
	}
	if result.XetReclaimedBytes != beforeLFS-lfsPayloadBytes(t, repo) || lfsStored(t, store, dead) || !lfsStored(t, store, live) {
		t.Fatalf("Prune() LFS sweep = %+v, want only dead data reclaimed", result)
	}
	if _, err := repo.GetCommit(ctx, "model", "p", "live", orphan); err == nil {
		t.Fatal("orphan commit survived Git GC")
	}
}

func TestPruneDryRunSkipsUnlinkedObjects(t *testing.T) {
	ctx := context.Background()
	repo, store, live, dead := newLFSCollectFixture(t)
	removed, err := store.DeleteSHA256IndexEntry(ctx, dead)
	if err != nil || !removed {
		t.Fatalf("unlink dead object = %v, %v", removed, err)
	}
	before := lfsPayloadBytes(t, repo)
	res, err := repo.Prune(ctx, git.PruneOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	checkPrune(t, res, true)
	if res.XetReclaimedBytes <= 0 {
		t.Fatalf("dry run sweep = %+v, want > 0 bytes for already-unlinked data", res)
	}
	if lfsPayloadBytes(t, repo) != before || !lfsStored(t, store, live) {
		t.Fatal("listing modified storage")
	}

	swept, err := repo.Prune(ctx, git.PruneOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkPrune(t, swept, false)
	if after := lfsPayloadBytes(t, repo); swept.XetReclaimedBytes != before-after || swept.XetReclaimedBytes != res.XetReclaimedBytes {
		t.Fatalf("sweep reclaimed %d bytes, dry run predicted %d, disk shrank by %d", swept.XetReclaimedBytes, res.XetReclaimedBytes, before-after)
	}
}

func TestPruneGraceOverride(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)

	// The fixture disables grace (-1); a 1h override shields the freshly written dead object.
	hour := time.Hour
	res, err := repo.Prune(ctx, git.PruneOptions{Grace: &hour})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if len(res.Unlinked) != 0 || res.PruneSkippedInGrace != 1 || res.SweepDone == nil || res.XetReclaimedBytes != 0 {
		t.Fatalf("Prune(grace=1h) = %+v, want the dead object skipped in grace", res)
	}
	if !lfsStored(t, xs, dead) || !lfsStored(t, xs, live) {
		t.Fatal("grace override did not preserve the objects")
	}

	// A configured 1h grace over the same store with no repositories shields both objects until a negative override disables it.
	graced := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, xs, time.Hour)
	res, err = graced.Prune(ctx, git.PruneOptions{DryRun: true})
	if err != nil || len(res.Unlinked) != 0 || res.PruneSkippedInGrace != 2 {
		t.Fatalf("Prune(configured grace) = %+v, %v; want both objects skipped", res, err)
	}
	disabled := -time.Second
	want := []string{live, dead}
	slices.Sort(want)
	res, err = graced.Prune(ctx, git.PruneOptions{DryRun: true, Grace: &disabled})
	if err != nil || !slices.Equal(res.Unlinked, want) || res.PruneSkippedInGrace != 0 {
		t.Fatalf("Prune(grace=-1s) = %+v, %v; want %v", res, err, want)
	}
}

func TestPruneBoundsSweep(t *testing.T) {
	ctx := context.Background()
	_, xs, _, _ := newLFSCollectFixture(t)
	// A collector over an empty repositories root treats both stored objects as garbage: two dead shards to bound.
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, xs, -1)
	before := lfsPayloadBytes(t, repo)

	dry, err := repo.Prune(ctx, git.PruneOptions{DryRun: true, MaxDeletes: 1, Budget: time.Nanosecond})
	if err != nil {
		t.Fatalf("Prune(dry) error = %v", err)
	}
	if len(dry.Unlinked) != 2 || dry.SweepDone == nil || !*dry.SweepDone || dry.RemainingShards != 0 || lfsPayloadBytes(t, repo) != before {
		t.Fatalf("Prune(dry, bounded) = %+v, want bounds ignored and nothing deleted", dry)
	}

	first, err := repo.Prune(ctx, git.PruneOptions{MaxDeletes: 1})
	if err != nil {
		t.Fatalf("Prune(maxDeletes=1) error = %v", err)
	}
	if len(first.Unlinked) != 2 || first.SweepDone == nil || *first.SweepDone || first.SweptShards != 1 || first.RemainingShards != 1 {
		t.Fatalf("Prune(maxDeletes=1) = %+v, want one swept shard, one remaining and sweepDone=false", first)
	}
	// The budget is checked once anything was swept, so a 1ns budget stops after the first deletion.
	second, err := repo.Prune(ctx, git.PruneOptions{Budget: time.Nanosecond})
	if err != nil {
		t.Fatalf("Prune(budget) error = %v", err)
	}
	if len(second.Unlinked) != 0 || second.SweepDone == nil || *second.SweepDone || second.SweptShards != 1 {
		t.Fatalf("Prune(budget=1ns) = %+v, want one swept shard and sweepDone=false", second)
	}
	third, err := repo.Prune(ctx, git.PruneOptions{})
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if len(third.Unlinked) != 0 || third.SweepDone == nil || !*third.SweepDone || third.SweptXorbs == 0 {
		t.Fatalf("Prune() = %+v, want the xorbs swept and sweepDone=true", third)
	}
	after := lfsPayloadBytes(t, repo)
	reclaimed := first.XetReclaimedBytes + second.XetReclaimedBytes + third.XetReclaimedBytes
	if after >= before || reclaimed != before-after {
		t.Fatalf("reclaimed %d bytes over three steps, disk shrank from %d to %d", reclaimed, before, after)
	}
}

// A Git GC failure returns the partial prune result with its error; the sweep does not run, so SweepDone stays nil and unlinked data is kept.
func TestPruneFailedRepositoryHasNoSweepResult(t *testing.T) {
	ctx := context.Background()
	repo, xs, live, dead := newLFSCollectFixture(t)
	backend := repo.(*gitRepo)
	repoPath := backend.gitPath("model", "p", "live")
	// A ref to a missing object fails Git GC before it deletes anything.
	dangling, err := backend.storage.RepositoriesFS().Create(repoPath + "/refs/heads/dangling")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dangling.Write([]byte(strings.Repeat("1", 40) + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := dangling.Close(); err != nil {
		t.Fatal(err)
	}
	before := lfsPayloadBytes(t, repo)

	res, err := repo.Prune(ctx, git.PruneOptions{})
	if err == nil || res == nil {
		t.Fatalf("Prune() = %+v, %v; want the partial result and the GC failure", res, err)
	}
	if res.Repositories != 1 || res.Failed[repoPath] == "" || res.LiveObjects != 1 || !slices.Equal(res.Unlinked, []string{dead}) {
		t.Fatalf("Prune() = %+v, want the failed repository reported with its pointer live and the dead object unlinked", res)
	}
	if res.SweepDone != nil || res.SweptShards != 0 || res.XetReclaimedBytes != 0 || lfsPayloadBytes(t, repo) != before {
		t.Fatalf("Prune() = %+v, want no sweep result and the unlinked data still stored", res)
	}
	if lfsStored(t, xs, dead) || !lfsStored(t, xs, live) {
		t.Fatalf("dead stored = %v, live stored = %v", lfsStored(t, xs, dead), lfsStored(t, xs, live))
	}
}

func TestLFSCleanupWithoutXetStore(t *testing.T) {
	ctx := context.Background()
	repo := NewGitDB(newRepoTestStorage(t, t.TempDir()), nil, nil, 0)

	if _, err := repo.Prune(ctx, git.PruneOptions{DryRun: true}); err == nil {
		t.Fatal("expected Prune dry-run to fail without a xet store")
	}
	if _, err := repo.Prune(ctx, git.PruneOptions{}); err == nil {
		t.Fatal("expected Prune to fail without a xet store")
	}
	if usage, err := repo.Usage(ctx); err != nil || *usage != (git.StorageUsage{}) {
		t.Fatalf("Usage() = %+v, %v; want all zero without repositories or a xet store", usage, err)
	}
}
