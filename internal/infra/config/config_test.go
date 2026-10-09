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

package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/infra/db"
)

func TestInitDerivesSQLiteDSNFromDataDir(t *testing.T) {
	t.Setenv(db.MATRIXHUB_DSN_ENV, "")
	migrationDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYAML := fmt.Sprintf(`
migrationPath: %q
dataDir: %q
database:
  driver: sqlite
  migrate: true
apiServer:
  port: 3001
`, migrationDir, dataDir)
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o600))

	config, err := Init(configPath, "")
	require.NoError(t, err)
	require.Equal(t, db.DriverSQLite, config.Database.Driver)
	require.Equal(t, migrationDir, config.Database.SQLPath)

	dsn, err := url.Parse(config.Database.DSN)
	require.NoError(t, err)
	require.Equal(t, "file", dsn.Scheme)
	require.Equal(t, filepath.Join(dataDir, "matrixhub.db"), dsn.Path)
	require.Equal(t, "5000", dsn.Query().Get("_busy_timeout"))
	require.Equal(t, "on", dsn.Query().Get("_foreign_keys"))
	require.Equal(t, "WAL", dsn.Query().Get("_journal_mode"))
	require.Equal(t, "FULL", dsn.Query().Get("_synchronous"))
	require.Equal(t, "immediate", dsn.Query().Get("_txlock"))
}

func TestInitParsesS3StorageConfig(t *testing.T) {
	migrationDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "data")
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	configYAML := fmt.Sprintf(`
migrationPath: %q
dataDir: %q
database:
  driver: sqlite
  migrate: true
apiServer:
  port: 3001
storage:
  lfs:
    type: s3
    endpoint: "http://minio:9000"
    signEndpoint: "http://127.0.0.1:9000"
    accessKey: "matrixhub"
    secretKey: "test-secret"
    bucket: "test"
    basePath: "lfs"
    forcePathStyle: true
`, migrationDir, dataDir)

	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o600))

	config, err := Init(configPath, "")
	require.NoError(t, err)

	require.Equal(t, "s3", config.Storage.LFS.Type)
	require.Equal(t, "http://minio:9000", config.Storage.LFS.Endpoint)
	require.Equal(t, "http://127.0.0.1:9000", config.Storage.LFS.SignEndpoint)
	require.Equal(t, "matrixhub", config.Storage.LFS.AccessKey)
	require.Equal(t, "test-secret", config.Storage.LFS.SecretKey)
	require.Equal(t, "test", config.Storage.LFS.Bucket)
	require.Equal(t, "lfs", config.Storage.LFS.BasePath)
	require.True(t, config.Storage.LFS.ForcePathStyle)
}
