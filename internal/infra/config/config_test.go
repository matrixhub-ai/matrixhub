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

func TestPresetsEnabledConfiguration(t *testing.T) {
	for _, test := range []struct {
		name    string
		section string
		enabled bool
	}{
		{name: "omitted", enabled: true},
		{name: "empty section", section: "presets: {}", enabled: true},
		{name: "enabled", section: "presets:\n  enabled: true", enabled: true},
		{name: "disabled", section: "presets:\n  enabled: false", enabled: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(db.MATRIXHUB_DSN_ENV, "")
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			content := fmt.Sprintf("migrationPath: %q\ndataDir: %q\napiServer:\n  port: 3101\n%s\n",
				dir, dir, test.section)
			require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
			cfg, err := Init(configPath, "")
			require.NoError(t, err)
			require.Equal(t, test.enabled, cfg.Presets.IsEnabled())
		})
	}
	require.True(t, PresetsConfig{}.IsEnabled())
}
