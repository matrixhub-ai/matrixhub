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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/mirror"
)

// migrateLegacyLFS imports objects left by the pre-xet local LFS store
// (root/<oid[0:2]>/<oid[2:4]>/<oid[4:]>) into the xet storage. A rerun skips
// objects the xet storage already serves; a successful pass moves the store to root.bak.
func migrateLegacyLFS(ctx context.Context, root string, m *mirror.Mirror) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root && os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("scan legacy lfs dir %s: %w", path, err)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) != 3 || len(parts[0]) != 2 || len(parts[1]) != 2 || len(parts[2]) != 60 {
			return nil
		}
		oid := parts[0] + parts[1] + parts[2]
		if _, err := hex.DecodeString(oid); err != nil {
			return nil
		}
		if m.HasObject(ctx, oid) {
			return nil
		}
		return importLegacyLFSObject(ctx, m, path, oid)
	})
	if err != nil {
		return fmt.Errorf("migrate legacy lfs: %w", err)
	}

	// Move lfs dir to a backup location to avoid reprocessing in future runs.
	backupDir := root + ".bak"
	if err := os.Rename(root, backupDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("move legacy lfs dir to backup %s: %w", backupDir, err)
	}

	return nil
}

func importLegacyLFSObject(ctx context.Context, m *mirror.Mirror, path, oid string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open legacy lfs object %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat legacy lfs object %s: %w", path, err)
	}
	if info.Size() == 0 && oid == fmt.Sprintf("%x", sha256.Sum256(nil)) {
		return nil
	}
	if err := m.PutObject(ctx, oid, f, info.Size()); err != nil {
		return fmt.Errorf("migrate legacy lfs object %s from %s: %w", oid, path, err)
	}
	return nil
}
