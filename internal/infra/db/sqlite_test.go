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

package db

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewSQLiteMigratesDatabase(t *testing.T) {
	dataDir := t.TempDir()
	dsn, err := DefaultSQLiteDSN(dataDir)
	require.NoError(t, err)

	database, err := New(Config{
		Driver:  DriverSQLite,
		DSN:     dsn,
		SQLPath: migrationSQLPath(t),
		Migrate: true,
	})
	require.NoError(t, err)

	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	require.Equal(t, 1, sqlDB.Stats().MaxOpenConnections)
	requirePrivateSQLiteFile(t, filepath.Join(dataDir, "matrixhub.db"))

	var foreignKeys int
	require.NoError(t, database.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error)
	require.Equal(t, 1, foreignKeys)

	var journalMode string
	require.NoError(t, database.Raw("PRAGMA journal_mode").Scan(&journalMode).Error)
	require.Equal(t, "wal", journalMode)

	var synchronous int
	require.NoError(t, database.Raw("PRAGMA synchronous").Scan(&synchronous).Error)
	require.Equal(t, 2, synchronous)

	var sqliteVersion string
	require.NoError(t, database.Raw("SELECT sqlite_version()").Scan(&sqliteVersion).Error)
	requireSQLiteVersionAtLeast(t, sqliteVersion, 3, 51, 3)

	var migrationVersion uint
	var dirty bool
	require.NoError(t, database.Table("schema_migrations").
		Select("version, dirty").
		Row().Scan(&migrationVersion, &dirty))
	require.Equal(t, uint(1), migrationVersion)
	require.False(t, dirty)

	var userCount int64
	require.NoError(t, database.Table("users").Where("username = ?", "admin").Count(&userCount).Error)
	require.Equal(t, int64(1), userCount)

	var roleCount int64
	require.NoError(t, database.Table("roles").Count(&roleCount).Error)
	require.Equal(t, int64(4), roleCount)

	var syncedAtColumnCount int64
	require.NoError(t, database.Raw(
		"SELECT COUNT(*) FROM pragma_table_info('models') WHERE name = 'synced_at'",
	).Scan(&syncedAtColumnCount).Error)
	require.Equal(t, int64(1), syncedAtColumnCount)

	var foreignKeyViolations []struct{}
	require.NoError(t, database.Raw("PRAGMA foreign_key_check").Scan(&foreignKeyViolations).Error)
	require.Empty(t, foreignKeyViolations)

	require.NoError(t, shouldMigrate(database, migrationSQLPath(t), ""))

	duplicateUser := database.Exec("INSERT INTO users (username) VALUES (?)", "ADMIN").Error
	require.Error(t, duplicateUser)
	require.True(t, errors.Is(duplicateUser, gorm.ErrDuplicatedKey))

	assertSQLiteCaseInsensitiveUniqueNames(t, database)
}

func TestNewSQLiteRestrictsExistingCustomDatabaseFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}

	databasePath := filepath.Join(t.TempDir(), "custom database.db")
	require.NoError(t, os.WriteFile(databasePath, nil, 0o644))
	require.NoError(t, os.Chmod(databasePath, 0o644))
	dsn := (&url.URL{Scheme: "file", Path: databasePath}).String()

	database, err := New(Config{Driver: DriverSQLite, DSN: dsn})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	requirePrivateSQLiteFile(t, databasePath)
}

func TestNewSQLiteRejectsSymbolicLinkDatabaseFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link behavior differs on Windows")
	}

	targetPath := filepath.Join(t.TempDir(), "target.db")
	require.NoError(t, os.WriteFile(targetPath, nil, 0o644))
	require.NoError(t, os.Chmod(targetPath, 0o644))
	linkPath := filepath.Join(t.TempDir(), "database.db")
	require.NoError(t, os.Symlink(targetPath, linkPath))
	dsn := (&url.URL{Scheme: "file", Path: linkPath}).String()

	_, err := New(Config{Driver: DriverSQLite, DSN: dsn})
	require.ErrorContains(t, err, "must not be a symbolic link")

	info, err := os.Stat(targetPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestNewSQLiteForcesForeignKeysForCustomDSN(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "custom.db")
	dsn := (&url.URL{
		Scheme:   "file",
		Path:     databasePath,
		RawQuery: "_fk=0&_foreign_keys=off",
	}).String()

	database, err := New(Config{
		Driver:  DriverSQLite,
		DSN:     dsn,
		SQLPath: migrationSQLPath(t),
		Migrate: true,
	})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	var foreignKeys int
	require.NoError(t, database.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error)
	require.Equal(t, 1, foreignKeys)
	require.Error(t, database.Exec(`
		INSERT INTO members_roles_projects (member_id, member_type, project_id)
		VALUES (1, 'user', 999)
	`).Error)
}

func TestNewSQLiteKeepsInMemoryDatabaseAcrossConfiguredConnectionRecycling(t *testing.T) {
	database, err := New(Config{
		Driver:                 DriverSQLite,
		DSN:                    ":memory:",
		ConnMaxLifetimeSeconds: 1,
		ConnMaxIdleSeconds:     1,
		SQLPath:                migrationSQLPath(t),
		Migrate:                true,
	})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	time.Sleep(1100 * time.Millisecond)
	var userCount int64
	require.NoError(t, database.Table("users").Count(&userCount).Error)
	require.Equal(t, int64(1), userCount)
}

func TestSQLiteFilePathPreservesDSNModes(t *testing.T) {
	for _, dsn := range []string{
		":memory:",
		":memory:?cache=shared",
		"file::memory:?cache=shared",
		"file:matrixhub-memory?mode=memory&cache=shared",
	} {
		path, fileBacked, createAllowed, err := sqliteFilePath(dsn)
		require.NoError(t, err)
		require.Empty(t, path)
		require.False(t, fileBacked)
		require.False(t, createAllowed)
	}

	databasePath := filepath.Join(t.TempDir(), "missing.db")
	for _, query := range []string{"mode=ro", "mode=rw", "immutable=1"} {
		dsn := (&url.URL{Scheme: "file", Path: databasePath, RawQuery: query}).String()
		path, fileBacked, createAllowed, err := sqliteFilePath(dsn)
		require.NoError(t, err)
		require.Equal(t, databasePath, path)
		require.True(t, fileBacked)
		require.False(t, createAllowed)

		_, created, err := prepareSQLiteFile(dsn)
		require.NoError(t, err)
		require.False(t, created)
		require.NoFileExists(t, databasePath)
	}

	dsn := (&url.URL{Scheme: "file", Path: databasePath, RawQuery: "mode=rwc"}).String()
	path, fileBacked, createAllowed, err := sqliteFilePath(dsn)
	require.NoError(t, err)
	require.Equal(t, databasePath, path)
	require.True(t, fileBacked)
	require.True(t, createAllowed)
	_, created, err := prepareSQLiteFile(dsn)
	require.NoError(t, err)
	require.True(t, created)
	requirePrivateSQLiteFile(t, databasePath)
}

func TestSQLiteURIPathToNativeWindows(t *testing.T) {
	require.Equal(t, `C:\data\matrixhub.db`, sqliteURIPathToNative("/C:/data/matrixhub.db", "windows"))
	require.Equal(t, `C:\data\matrixhub.db`, sqliteURIPathToNative("C:/data/matrixhub.db", "windows"))
}

func TestSQLiteMigrationsDown(t *testing.T) {
	dsn, err := DefaultSQLiteDSN(t.TempDir())
	require.NoError(t, err)

	database, err := New(Config{
		Driver:  DriverSQLite,
		DSN:     dsn,
		SQLPath: migrationSQLPath(t),
		Migrate: true,
	})
	require.NoError(t, err)

	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	sourceURL := fmt.Sprintf("file://%s", filepath.Join(migrationSQLPath(t), DriverSQLite))
	session, err := newSQLiteMigrateSession(sqlDB, sourceURL, "")
	require.NoError(t, err)
	require.NoError(t, session.Down())

	var tableCount int64
	require.NoError(t, database.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'users'",
	).Scan(&tableCount).Error)
	require.Zero(t, tableCount)
}

func TestMigrationVersionsMatchMySQL(t *testing.T) {
	mysqlMigrations, err := filepath.Glob(filepath.Join(migrationSQLPath(t), DriverMySQL, "*.sql"))
	require.NoError(t, err)
	sqliteMigrations, err := filepath.Glob(filepath.Join(migrationSQLPath(t), DriverSQLite, "*.sql"))
	require.NoError(t, err)

	require.Equal(t, migrationFileNames(mysqlMigrations), migrationFileNames(sqliteMigrations))
}

func migrationSQLPath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "db", "migrations", "sql"))
}

func migrationFileNames(paths []string) []string {
	names := make([]string, 0, len(paths))
	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}
	return names
}

func requireSQLiteVersionAtLeast(t *testing.T, version string, wantMajor, wantMinor, wantPatch int) {
	t.Helper()
	var major, minor, patch int
	_, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	require.NoError(t, err)

	actual := major*1_000_000 + minor*1_000 + patch
	want := wantMajor*1_000_000 + wantMinor*1_000 + wantPatch
	require.GreaterOrEqual(t, actual, want)
}

func requirePrivateSQLiteFile(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		info, err = os.Stat(path + suffix)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func assertSQLiteCaseInsensitiveUniqueNames(t *testing.T, database *gorm.DB) {
	t.Helper()

	require.NoError(t, database.Exec(
		"INSERT INTO projects (name, type) VALUES (?, ?)", "case-sensitive-project", 1,
	).Error)
	var projectID int
	require.NoError(t, database.Raw(
		"SELECT id FROM projects WHERE name = ?", "case-sensitive-project",
	).Scan(&projectID).Error)

	require.NoError(t, database.Exec(`INSERT INTO models
		(name, project_id, size, default_branch, parameter_count, readme_content)
		VALUES (?, ?, 0, 'main', 0, '')`, "Tiny-Model", projectID).Error)
	require.True(t, errors.Is(database.Exec(`INSERT INTO models
		(name, project_id, size, default_branch, parameter_count, readme_content)
		VALUES (?, ?, 0, 'main', 0, '')`, "tiny-model", projectID).Error, gorm.ErrDuplicatedKey))

	require.NoError(t, database.Exec(`INSERT INTO datasets
		(name, project_id, default_branch, readme_content) VALUES (?, ?, 'main', '')`,
		"Tiny-Dataset", projectID).Error)
	require.True(t, errors.Is(database.Exec(`INSERT INTO datasets
		(name, project_id, default_branch, readme_content) VALUES (?, ?, 'main', '')`,
		"tiny-dataset", projectID).Error, gorm.ErrDuplicatedKey))

	require.NoError(t, database.Exec(
		"INSERT INTO labels (name, category, scope) VALUES (?, ?, ?)", "Text", "Task", "Model",
	).Error)
	require.True(t, errors.Is(database.Exec(
		"INSERT INTO labels (name, category, scope) VALUES (?, ?, ?)", "text", "task", "model",
	).Error, gorm.ErrDuplicatedKey))
}
