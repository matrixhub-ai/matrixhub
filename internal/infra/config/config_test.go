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
	"time"

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
  gcGrace: -1s
`, migrationDir, dataDir)
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0o600))

	config, err := Init(configPath, "")
	require.NoError(t, err)
	require.Equal(t, db.DriverSQLite, config.Database.Driver)
	require.Equal(t, migrationDir, config.Database.SQLPath)
	require.Equal(t, -time.Second, config.APIServer.GCGrace)

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

func TestInitTokenSigningSecret(t *testing.T) {
	t.Setenv(db.MATRIXHUB_DSN_ENV, "")
	for _, secret := range []string{"configured-secret", ""} {
		t.Run("secret="+secret, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			contents := fmt.Sprintf("migrationPath: %q\ndataDir: %q\ndatabase:\n  driver: sqlite\napiServer:\n  port: 3001\n", t.TempDir(), t.TempDir())
			if secret != "" {
				contents += fmt.Sprintf("  tokenSigningSecret: %q\n", secret)
			}
			require.NoError(t, os.WriteFile(configPath, []byte(contents), 0o600))
			cfg, err := Init(configPath, "")
			require.NoError(t, err)
			if secret != "" {
				require.Equal(t, secret, cfg.APIServer.TokenSigningSecret)
			} else {
				require.NotEmpty(t, cfg.APIServer.TokenSigningSecret)
				second, err := Init(configPath, "")
				require.NoError(t, err)
				require.NotEmpty(t, second.APIServer.TokenSigningSecret)
				require.NotEqual(t, cfg.APIServer.TokenSigningSecret, second.APIServer.TokenSigningSecret)
			}
		})
	}
}
