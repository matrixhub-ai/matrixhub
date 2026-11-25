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
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/preset"
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
			cfg, err := initWithSection(t, test.section)
			require.NoError(t, err)
			require.Equal(t, test.enabled, cfg.Presets.IsEnabled())
		})
	}
	require.True(t, PresetsConfig{}.IsEnabled())
}

func TestPresetsManifestConfiguration(t *testing.T) {
	builtIn, err := os.ReadFile(filepath.Join("..", "..", "domain", "preset", "default_manifest.yaml"))
	require.NoError(t, err)
	notRecommended := false

	for _, test := range []struct {
		name    string
		section string
		want    *preset.Manifest
		wantErr []string
	}{
		{name: "omitted"},
		{name: "empty manifest", section: "presets:\n  manifest: {}"},
		{name: "null manifest", section: "presets:\n  manifest:"},
		{
			name:    "built-in manifest pasted unchanged",
			section: "presets:\n  manifest:\n" + indent(string(builtIn), "    "),
			want:    preset.DefaultManifest(),
		},
		{
			name: "custom manifest",
			section: `presets:
  manifest:
    registries:
      - name: mirror
        url: https://mirror.example.com
    projects:
      - name: team
        registry: mirror
    models:
      - project: team
        name: model
        recommended: false
        size: 4892388741252
        parameterCount: 2446182725504
        labels:
          - {category: task, name: text-generation}`,
			want: &preset.Manifest{
				Registries: []preset.RegistrySpec{{Name: "mirror", URL: "https://mirror.example.com"}},
				Projects:   []preset.ProjectSpec{{Name: "team", Registry: "mirror"}},
				Models: []preset.ModelSpec{{
					Project: "team", Name: "model", Recommended: &notRecommended,
					Size: 4892388741252, ParameterCount: 2446182725504,
					Labels: []preset.LabelSpec{{Category: "task", Name: "text-generation"}},
				}},
			},
		},
		{
			name: "disabled presets ignore the manifest",
			section: `presets:
  enabled: false
  manifest:
    projects: [{name: team, registry: missing, unknown: true}]`,
		},
		{
			name: "unknown manifest field",
			section: `presets:
  manifest:
    models: [{project: team, name: model, sise: 1}]`,
			wantErr: []string{"invalid presets config", "'Manifest.Models[0]' has invalid keys: sise"},
		},
		{
			name: "missing manifest level",
			section: `presets:
  models: [{project: team, name: model}]`,
			wantErr: []string{"invalid presets config", "invalid keys: models"},
		},
		{
			name:    "misspelled enabled",
			section: "presets:\n  enable: false",
			wantErr: []string{"invalid presets config", "invalid keys: enable"},
		},
		{
			name:    "scalar manifest",
			section: "presets:\n  manifest: models",
			wantErr: []string{"invalid presets config", "'Manifest' expected a map or struct"},
		},
		{
			name: "invalid manifest",
			section: `presets:
  manifest:
    projects: [{name: team, registry: missing}]
    models: [{project: other, name: model}]`,
			wantErr: []string{
				"invalid presets.manifest:\n",
				`projects[0].registry: "missing" is not defined in the manifest`,
				`models[0].project: "other" is not defined in the manifest`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := initWithSection(t, test.section)
			for _, want := range test.wantErr {
				require.ErrorContains(t, err, want)
			}
			if len(test.wantErr) > 0 {
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, cfg.Presets.Manifest)
		})
	}
}

func TestHelmValuesPresetManifestMatchesBuiltIn(t *testing.T) {
	values := viper.New()
	values.SetConfigFile(filepath.Join("..", "..", "..", "deploy", "charts", "matrixhub", "values.yaml"))
	require.NoError(t, values.ReadInConfig())

	manifest, err := loadPresetManifest(values.Sub("presets"))
	require.NoError(t, err)
	require.Equal(t, preset.DefaultManifest(), manifest,
		"presets.manifest in the chart values must match internal/domain/preset/default_manifest.yaml")
}

func initWithSection(t *testing.T, section string) (*Config, error) {
	t.Helper()
	t.Setenv(db.MATRIXHUB_DSN_ENV, "")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := fmt.Sprintf("migrationPath: %q\ndataDir: %q\napiServer:\n  port: 3101\n%s\n", dir, dir, section)
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
	return Init(configPath, "")
}

func indent(text, prefix string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
