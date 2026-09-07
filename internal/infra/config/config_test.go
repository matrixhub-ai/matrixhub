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
debug: true
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
	for _, test := range []struct {
		name    string
		debug   bool
		secret  string
		env     string
		want    string
		wantErr bool
	}{
		{"generated production key", false, "", "", "", false},
		{"default production key", false, defaultTokenSigningSecret, "", "", true},
		{"short production key", false, "short", "", "", true},
		{"configured key", false, "configured-signing-secret-32-bytes", "", "configured-signing-secret-32-bytes", false},
		{"environment override", false, "config-value", "environment-signing-secret-32-bytes", "environment-signing-secret-32-bytes", false},
		{"environment without config key", false, "", "environment-signing-secret-32-bytes", "environment-signing-secret-32-bytes", false},
		{"generated development key", true, "", "", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("MATRIXHUB_TOKEN_SIGNING_SECRET", test.env)
			t.Setenv(db.MATRIXHUB_DSN_ENV, "")
			configPath := filepath.Join(t.TempDir(), "config.yaml")
			contents := fmt.Sprintf("debug: %t\nmigrationPath: %q\ndataDir: %q\ndatabase:\n  driver: sqlite\napiServer:\n  port: 3001\n", test.debug, t.TempDir(), t.TempDir())
			if test.secret != "" {
				contents += fmt.Sprintf("  tokenSigningSecret: %q\n", test.secret)
			}
			require.NoError(t, os.WriteFile(configPath, []byte(contents), 0o600))
			cfg, err := Init(configPath, "")
			if test.wantErr {
				require.ErrorContains(t, err, "tokenSigningSecret")
				return
			}
			require.NoError(t, err)
			if test.want != "" {
				require.Equal(t, test.want, cfg.APIServer.TokenSigningSecret)
			} else {
				require.GreaterOrEqual(t, len(cfg.APIServer.TokenSigningSecret), 32)
				require.NotEqual(t, defaultTokenSigningSecret, cfg.APIServer.TokenSigningSecret)
			}
		})
	}
}

func TestInitPersistsRandomSigningSecret(t *testing.T) {
	t.Setenv("MATRIXHUB_TOKEN_SIGNING_SECRET", "")
	t.Setenv(db.MATRIXHUB_DSN_ENV, "")
	var previous string
	for range 2 {
		dataDir := filepath.Join(t.TempDir(), "data")
		configPath := filepath.Join(t.TempDir(), "config.yaml")
		contents := fmt.Sprintf("migrationPath: %q\ndataDir: %q\ndatabase:\n  driver: sqlite\napiServer:\n  port: 3001\n", t.TempDir(), dataDir)
		require.NoError(t, os.WriteFile(configPath, []byte(contents), 0o600))
		first, err := Init(configPath, "")
		require.NoError(t, err)
		second, err := Init(configPath, "")
		require.NoError(t, err)
		require.Equal(t, first.APIServer.TokenSigningSecret, second.APIServer.TokenSigningSecret)
		require.NotEqual(t, previous, first.APIServer.TokenSigningSecret)
		previous = first.APIServer.TokenSigningSecret
		keyFile := filepath.Join(dataDir, "token-signing-secret")
		stored, err := os.ReadFile(keyFile)
		require.NoError(t, err)
		require.Equal(t, previous, string(stored))
		info, err := os.Stat(keyFile)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestInitRejectsUnreadableSigningSecret(t *testing.T) {
	t.Setenv("MATRIXHUB_TOKEN_SIGNING_SECRET", "")
	t.Setenv(db.MATRIXHUB_DSN_ENV, "")
	dataDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dataDir, "token-signing-secret"), 0o700))
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	contents := fmt.Sprintf("migrationPath: %q\ndataDir: %q\ndatabase:\n  driver: sqlite\napiServer:\n  port: 3001\n", t.TempDir(), dataDir)
	require.NoError(t, os.WriteFile(configPath, []byte(contents), 0o600))
	_, err := Init(configPath, "")
	require.ErrorContains(t, err, "signing secret")
}
