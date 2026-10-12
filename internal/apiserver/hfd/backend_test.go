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

package hfd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"go.uber.org/mock/gomock"

	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
)

func testConfig(dataDir string) *config.Config {
	return &config.Config{DataDir: dataDir, APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}}
}

// legacyLFSPath is where the pre-xet hfd local LFS store kept an object:
// DataDir/lfs/<oid[0:2]>/<oid[2:4]>/<oid[4:]>.
func legacyLFSPath(dataDir, oid string) string {
	return filepath.Join(dataDir, "lfs", oid[0:2], oid[2:4], oid[4:])
}

// legacyLFSBackupPath is where a successful migration moves that object (DataDir/lfs.bak/...).
func legacyLFSBackupPath(dataDir, oid string) string {
	return filepath.Join(dataDir, "lfs.bak", oid[0:2], oid[2:4], oid[4:])
}

func writeLegacyLFSObject(t *testing.T, dataDir string, content []byte) string {
	t.Helper()
	sum := sha256.Sum256(content)
	oid := hex.EncodeToString(sum[:])
	path := legacyLFSPath(dataDir, oid)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	return oid
}

func readObject(t *testing.T, b *Backend, oid string) []byte {
	t.Helper()
	rc, size, err := b.Mirror().OpenObject(context.Background(), oid)
	if err != nil {
		t.Fatalf("OpenObject(%s): %v", oid, err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(got)) != size {
		t.Fatalf("OpenObject(%s) size %d, read %d bytes", oid, size, len(got))
	}
	return got
}

func TestNewMigratesLegacyLFS(t *testing.T) {
	dataDir := t.TempDir()
	content := bytes.Repeat([]byte("legacy lfs payload "), 4096)
	oid := writeLegacyLFSObject(t, dataDir, content)

	b, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("migrated object differs from legacy content (%d vs %d bytes)", len(got), len(content))
	}
	if got, err := os.ReadFile(legacyLFSBackupPath(dataDir, oid)); err != nil || !bytes.Equal(got, content) {
		t.Fatalf("legacy object must be retained in the backup after migration: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "lfs")); !os.IsNotExist(err) {
		t.Fatalf("legacy root must be moved aside after migration, stat: %v", err)
	}
}

func TestNewSkipsEmptyLegacyLFSObject(t *testing.T) {
	dataDir := t.TempDir()
	oid := writeLegacyLFSObject(t, dataDir, nil)
	blockXetObjectWrites(t, dataDir)

	backend, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if backend.Mirror().HasObject(context.Background(), oid) {
		t.Fatal("empty legacy object must not be imported")
	}
	if got, err := os.ReadFile(legacyLFSBackupPath(dataDir, oid)); err != nil || len(got) != 0 {
		t.Fatalf("empty legacy object must be retained: %q, %v", got, err)
	}
}

func TestNewMigratesLegacyLFSSkipsImportedObjects(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("imported once")
	oid := writeLegacyLFSObject(t, dataDir, content)
	if _, err := New(t.Context(), testConfig(dataDir)); err != nil {
		t.Fatal(err)
	}

	// Restoring the moved-aside store simulates a migration interrupted before the rename.
	if err := os.Rename(filepath.Join(dataDir, "lfs.bak"), filepath.Join(dataDir, "lfs")); err != nil {
		t.Fatal(err)
	}
	// A second import would now fail the OID check, so a clean restart proves the skip.
	if err := os.WriteFile(legacyLFSPath(dataDir, oid), []byte("changed after import"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("object served %q, want original %q", got, content)
	}
}

func TestNewMigratesLegacyLFSIgnoresNonCanonicalEntries(t *testing.T) {
	dataDir := t.TempDir()
	root := filepath.Join(dataDir, "lfs")
	content := []byte("canonical")
	oid := writeLegacyLFSObject(t, dataDir, content)
	stray := []byte("stray")
	straySum := sha256.Sum256(stray)
	strayOID := hex.EncodeToString(straySum[:])
	strayShard := filepath.Join(root, strayOID[0:2], strayOID[2:4])
	if err := os.MkdirAll(strayShard, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "00", "00"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		filepath.Join(strayShard, "lfsd_tmp_123456"):       stray,
		filepath.Join(root, strayOID):                      stray,
		filepath.Join(root, strayOID[0:2], strayOID[2:]):   stray,
		filepath.Join(strayShard, strings.Repeat("z", 60)): stray,
		filepath.Join(strayShard, strayOID[4:]+"0"):        stray,
	}
	for path, data := range files {
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// A symlink at the canonical path and a symlinked shard dir must both be ignored:
	// following either would import stray or fail the OID check.
	if err := os.Symlink(filepath.Join(strayShard, "lfsd_tmp_123456"), filepath.Join(strayShard, strayOID[4:])); err != nil {
		t.Fatal(err)
	}
	linkedShard := "ff"
	if oid[0:2] == linkedShard {
		linkedShard = "00"
	}
	if err := os.Symlink(filepath.Join(root, oid[0:2]), filepath.Join(root, linkedShard)); err != nil {
		t.Fatal(err)
	}

	b, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("object served %q, want %q", got, content)
	}
	if b.Mirror().HasObject(context.Background(), strayOID) {
		t.Fatal("non-canonical entries must not be imported")
	}
	for path, data := range files {
		backup := filepath.Join(root+".bak", strings.TrimPrefix(path, root))
		got, err := os.ReadFile(backup)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("legacy entry %s changed: %q, %v", backup, got, err)
		}
	}
}

func TestNewFailsOnLegacyLFSHashMismatchUntilRepaired(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("expected content")
	sum := sha256.Sum256(content)
	oid := hex.EncodeToString(sum[:])
	path := legacyLFSPath(dataDir, oid)
	corrupt := []byte("corrupt content")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, corrupt, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(t.Context(), testConfig(dataDir))
	if err == nil || !strings.Contains(err.Error(), oid) || !strings.Contains(err.Error(), path) {
		t.Fatalf("New() error = %v, want hash mismatch naming %s and %s", err, oid, path)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, corrupt) {
		t.Fatalf("legacy object must be retained: %q, %v", got, err)
	}
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(t.Context(), testConfig(dataDir)); err == nil || !strings.Contains(err.Error(), "content hash does not match OID") {
		t.Fatalf("New() error = %v, want truncated legacy object to fail the OID check", err)
	}

	// Nothing may have been published under the OID: without the source it stays absent.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	b, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.Mirror().OpenObject(context.Background(), oid); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("OpenObject after failed import: %v, want %v", err, os.ErrNotExist)
	}

	// The successful pass moved the emptied store aside; clear that backup so the repaired pass can move too.
	if err := os.RemoveAll(filepath.Join(dataDir, "lfs.bak")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	b, err = New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("repaired object served %q, want %q", got, content)
	}
}

// blockXetObjectWrites puts regular files where the xet store nests xorbs by hash prefix, so any xet write fails while construction succeeds.
func blockXetObjectWrites(t *testing.T, dataDir string) []string {
	t.Helper()
	xorbs := filepath.Join(dataDir, "xet", "xorbs")
	if err := os.MkdirAll(xorbs, 0755); err != nil {
		t.Fatal(err)
	}
	blockers := make([]string, 0, 256)
	for i := range 256 {
		blocker := filepath.Join(xorbs, fmt.Sprintf("%02x", i))
		if err := os.WriteFile(blocker, nil, 0644); err != nil {
			t.Fatal(err)
		}
		blockers = append(blockers, blocker)
	}
	return blockers
}

func TestNewFailsOnLegacyLFSImportErrorAndRetries(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("blocked import")
	oid := writeLegacyLFSObject(t, dataDir, content)
	blockers := blockXetObjectWrites(t, dataDir)

	_, err := New(t.Context(), testConfig(dataDir))
	if err == nil || !strings.Contains(err.Error(), oid) || !strings.Contains(err.Error(), legacyLFSPath(dataDir, oid)) {
		t.Fatalf("New() error = %v, want import failure naming %s", err, oid)
	}
	if got, err := os.ReadFile(legacyLFSPath(dataDir, oid)); err != nil || !bytes.Equal(got, content) {
		t.Fatalf("legacy object must be retained: %q, %v", got, err)
	}

	for _, blocker := range blockers {
		if err := os.Remove(blocker); err != nil {
			t.Fatal(err)
		}
	}
	b, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("retried object served %q, want %q", got, content)
	}
}

func TestNewToleratesMissingOrEmptyLegacyLFS(t *testing.T) {
	if _, err := New(t.Context(), testConfig(t.TempDir())); err != nil {
		t.Fatalf("missing legacy root: %v", err)
	}
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "lfs", "ab", "cd"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := New(t.Context(), testConfig(dataDir)); err != nil {
		t.Fatalf("empty legacy root: %v", err)
	}
}

// mirror/download and mirror/upload are not checked: the xet client cache creates them on first use.
func TestNewLaysOutXetDirLikeXetd(t *testing.T) {
	dataDir := t.TempDir()
	if _, err := New(t.Context(), testConfig(dataDir)); err != nil {
		t.Fatal(err)
	}
	xetDir := filepath.Join(dataDir, "xet")
	for _, dir := range []string{"xorbs", "shards", "index", "spool", filepath.Join("mirror", "index")} {
		fi, err := os.Stat(filepath.Join(xetDir, dir))
		if err != nil || !fi.IsDir() {
			t.Errorf("stat xet/%s: %v, want directory", dir, err)
		}
	}
	for _, dir := range []string{"storage", "chunks", filepath.Join("mirror", "spool")} {
		if _, err := os.Stat(filepath.Join(xetDir, dir)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("stat xet/%s: %v, want %v", dir, err, fs.ErrNotExist)
		}
	}
}

// A symlinked legacy store is imported through its target; only the link moves aside.
func TestNewMigratesSymlinkedLegacyLFS(t *testing.T) {
	dataDir, legacy := t.TempDir(), t.TempDir()
	content := []byte("linked legacy payload")
	oid := writeLegacyLFSObject(t, legacy, content)
	root := filepath.Join(dataDir, "lfs")
	if err := os.Symlink(filepath.Join(legacy, "lfs"), root); err != nil {
		t.Fatal(err)
	}

	b, err := New(t.Context(), testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("object served %q, want %q", got, content)
	}
	if info, err := os.Lstat(root + ".bak"); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("lfs.bak must be the moved symlink: %v, %v", info, err)
	}
	if got, err := os.ReadFile(legacyLFSPath(legacy, oid)); err != nil || !bytes.Equal(got, content) {
		t.Fatalf("symlink target must keep the legacy object: %q, %v", got, err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("legacy root must be moved aside, lstat: %v", err)
	}
}

func TestNewStopsLegacyLFSImportOnCancel(t *testing.T) {
	dataDir := t.TempDir()
	oid := writeLegacyLFSObject(t, dataDir, []byte("not yet imported"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New(ctx, testConfig(dataDir))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("New() error = %v, want %v", err, context.Canceled)
	}
	if _, err := os.Stat(legacyLFSPath(dataDir, oid)); err != nil {
		t.Fatalf("legacy store must stay in place after a cancelled import: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dataDir, "lfs.bak")); !os.IsNotExist(err) {
		t.Fatalf("cancelled import must not move the store aside, lstat: %v", err)
	}
}

// The import passes its file straight to hfd, so hfd itself must stop a copy once the context is cancelled.
func TestLegacyLFSImportStopsCopyOnCancel(t *testing.T) {
	b, err := New(t.Context(), testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	// A source far larger than any read window: only the cancellation can end the copy early.
	source := &cancelAfterFirstRead{Reader: &repeatReader{chunk: []byte("legacy bytes "), limit: 8 << 20}, cancel: cancel}
	oid := strings.Repeat("ab", 32)
	err = b.Mirror().PutObject(ctx, oid, source, -1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PutObject after cancel: err = %v, want %v", err, context.Canceled)
	}
	if source.reads == 0 {
		t.Fatal("the copy never read the source")
	}
	if b.Mirror().HasObject(context.Background(), oid) {
		t.Fatal("a cancelled copy must not store the object")
	}
}

// cancelAfterFirstRead cancels its context as soon as the first Read returns.
type cancelAfterFirstRead struct {
	io.Reader
	cancel context.CancelFunc
	reads  int
}

func (r *cancelAfterFirstRead) Read(p []byte) (int, error) {
	r.reads++
	n, err := r.Reader.Read(p[:min(len(p), 512)])
	r.cancel()
	return n, err
}

// repeatReader yields chunk until limit bytes were read.
type repeatReader struct {
	chunk []byte
	limit int
	read  int
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.read >= r.limit {
		return 0, io.EOF
	}
	n := 0
	for n+len(r.chunk) <= len(p) && r.read+n < r.limit {
		n += copy(p[n:], r.chunk)
	}
	if n == 0 {
		n = copy(p, r.chunk)
	}
	r.read += n
	return n, nil
}

func TestNewDoesNotRegisterMirrorReceiveHooks(t *testing.T) {
	b, _ := New(t.Context(), &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})

	m := reflect.ValueOf(b.storage.sharedMirror).Elem()
	for _, field := range []string{"preReceiveHookFunc", "postReceiveHookFunc"} {
		if !m.FieldByName(field).IsNil() {
			t.Fatalf("mirror must not register %s: the pull callers refresh metadata themselves", field)
		}
	}
}

func TestPreOpenReadDoesNotCreateModel(t *testing.T) {
	ctx := context.Background()
	modelService := modelmocks.NewMockIModelService(gomock.NewController(t))
	modelService.EXPECT().CheckOrSyncFromRemote(ctx, "public", "missing").Return(nil)
	backend, _ := New(ctx, &config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.modelService = modelService

	if err := backend.preOpenHook(ctx, "public/missing", false); err != nil {
		t.Fatal(err)
	}
}

// Push sync jobs name their destination per call; hfd skips the push entirely
// when the mirror has no destination callback, reporting success to the job.
func TestMirrorPushHonoursExplicitDestination(t *testing.T) {
	ctx := context.Background()
	b, err := New(ctx, testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Mirror().Wait)
	gitPath := repository.ResolvePath("proj/model")
	repo, err := repository.Init(ctx, b.Storage().RepositoriesFS(), gitPath, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateCommit(ctx, "main", "init", "Test", "test@test.com",
		[]repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "README.md", Content: []byte("# model\n")}}, ""); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(remote.Close)

	err = b.Mirror().PushToRemote(ctx, gitPath, "proj/model", &mirror.PushOptions{DestinationURL: remote.URL + "/org/model", Output: io.Discard})
	if requests.Load() == 0 {
		t.Fatalf("PushToRemote() = %v without contacting the destination", err)
	}
	if err == nil {
		t.Fatal("PushToRemote() = nil, want the destination's 503 reported")
	}
}
