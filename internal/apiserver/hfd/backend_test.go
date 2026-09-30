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
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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

	b, err := New(testConfig(dataDir))
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
	spool := filepath.Join(dataDir, "xet", "spool")
	if err := os.MkdirAll(filepath.Dir(spool), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spool, nil, 0644); err != nil {
		t.Fatal(err)
	}

	backend, err := New(testConfig(dataDir))
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
	if _, err := New(testConfig(dataDir)); err != nil {
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
	b, err := New(testConfig(dataDir))
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

	b, err := New(testConfig(dataDir))
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

	_, err := New(testConfig(dataDir))
	if err == nil || !strings.Contains(err.Error(), oid) || !strings.Contains(err.Error(), path) {
		t.Fatalf("New() error = %v, want hash mismatch naming %s and %s", err, oid, path)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, corrupt) {
		t.Fatalf("legacy object must be retained: %q, %v", got, err)
	}
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(testConfig(dataDir)); err == nil || !strings.Contains(err.Error(), "content hash does not match OID") {
		t.Fatalf("New() error = %v, want truncated legacy object to fail the OID check", err)
	}

	// Nothing may have been published under the OID: without the source it stays absent.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	b, err := New(testConfig(dataDir))
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
	b, err = New(testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("repaired object served %q, want %q", got, content)
	}
}

func TestNewFailsOnLegacyLFSImportErrorAndRetries(t *testing.T) {
	dataDir := t.TempDir()
	content := []byte("blocked import")
	oid := writeLegacyLFSObject(t, dataDir, content)
	// A regular file where the mirror spools ingests makes the xet write fail.
	spool := filepath.Join(dataDir, "xet", "spool")
	if err := os.MkdirAll(filepath.Dir(spool), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spool, nil, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := New(testConfig(dataDir))
	if err == nil || !strings.Contains(err.Error(), oid) || !strings.Contains(err.Error(), legacyLFSPath(dataDir, oid)) {
		t.Fatalf("New() error = %v, want import failure naming %s", err, oid)
	}
	if got, err := os.ReadFile(legacyLFSPath(dataDir, oid)); err != nil || !bytes.Equal(got, content) {
		t.Fatalf("legacy object must be retained: %q, %v", got, err)
	}

	if err := os.Remove(spool); err != nil {
		t.Fatal(err)
	}
	b, err := New(testConfig(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got := readObject(t, b, oid); !bytes.Equal(got, content) {
		t.Fatalf("retried object served %q, want %q", got, content)
	}
}

func TestNewToleratesMissingOrEmptyLegacyLFS(t *testing.T) {
	if _, err := New(testConfig(t.TempDir())); err != nil {
		t.Fatalf("missing legacy root: %v", err)
	}
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "lfs", "ab", "cd"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := New(testConfig(dataDir)); err != nil {
		t.Fatalf("empty legacy root: %v", err)
	}
}

func TestNewDoesNotRegisterMirrorReceiveHooks(t *testing.T) {
	b, _ := New(&config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})

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
	backend, _ := New(&config.Config{DataDir: t.TempDir(), APIServer: &config.APIServerConfig{TokenSigningSecret: "test-secret"}})
	backend.modelService = modelService

	if err := backend.preOpenHook(ctx, "public/missing", false); err != nil {
		t.Fatal(err)
	}
}
