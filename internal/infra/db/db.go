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
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gmysql "gorm.io/driver/mysql"
	gpostgres "gorm.io/driver/postgres"
	gsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const databasePingTimeout = 10 * time.Second

func New(config Config) (*gorm.DB, error) {
	config.applyDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}

	var sqlitePath string
	var sqliteMemory bool
	var sqliteDSN string
	if config.Driver == DriverSQLite {
		var err error
		_, fileBacked, _, err := sqliteFilePath(config.DSN)
		if err != nil {
			return nil, err
		}
		sqliteMemory = !fileBacked
		sqlitePath, _, err = prepareSQLiteFile(config.DSN)
		if err != nil {
			return nil, err
		}
		sqliteDSN, err = sqliteDSNWithForeignKeys(config.DSN)
		if err != nil {
			return nil, err
		}
	}

	var dialector gorm.Dialector
	switch config.Driver {
	case DriverMySQL:
		dialector = gmysql.Open(config.DSN)
	case DriverPostgres:
		dialector = gpostgres.Open(config.DSN)
	case DriverSQLite:
		dialector = gsqlite.Open(sqliteDSN)
	}

	database, err := gorm.Open(dialector, &gorm.Config{
		DisableAutomaticPing: true,
		TranslateError:       true,
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", config.Driver, err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("get database connection pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	connMaxLifetime := time.Duration(config.ConnMaxLifetimeSeconds) * time.Second
	connMaxIdleTime := time.Duration(config.ConnMaxIdleSeconds) * time.Second
	if sqliteMemory {
		// Recycling the sole connection destroys an in-memory SQLite database.
		connMaxLifetime = 0
		connMaxIdleTime = 0
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	pingContext, cancel := context.WithTimeout(context.Background(), databasePingTimeout)
	defer cancel()
	if err = sqlDB.PingContext(pingContext); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping %s database: %w", config.Driver, err)
	}
	if config.Driver == DriverSQLite {
		var foreignKeys int
		if err = database.Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("verify sqlite foreign key enforcement: %w", err)
		}
		if foreignKeys != 1 {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("sqlite foreign key enforcement is disabled")
		}
	}
	if sqlitePath != "" {
		if err = restrictSQLiteFilePermissions(sqlitePath); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	if config.Migrate {
		if err = shouldMigrate(database, config.SQLPath, ""); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	if config.Debug {
		database = database.Debug()
	}

	return database, nil
}

// prepareSQLiteFile creates a file-backed SQLite database with private
// permissions before SQLite opens it. This prevents the process umask from
// briefly exposing credentials and session data through a world-readable file.
func prepareSQLiteFile(dsn string) (path string, created bool, err error) {
	path, fileBacked, createAllowed, err := sqliteFilePath(dsn)
	if err != nil || !fileBacked {
		return "", false, err
	}

	exists, err := restrictSQLiteFilePermission(path)
	switch {
	case err != nil:
		return "", false, err
	case exists:
		return path, false, nil
	case !createAllowed:
		// Preserve mode=ro, mode=rw and immutable fail-if-missing semantics.
		// The SQLite driver will return the authoritative open error.
		return path, false, nil
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return "", false, fmt.Errorf("create sqlite database securely: %w", err)
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(path)
		return "", false, fmt.Errorf("close sqlite database after secure creation: %w", err)
	}

	return path, true, nil
}

func sqliteFilePath(dsn string) (path string, fileBacked bool, createAllowed bool, err error) {
	if strings.HasPrefix(dsn, "file:") {
		parsed, parseErr := url.Parse(dsn)
		if parseErr != nil {
			return "", false, false, fmt.Errorf("parse sqlite DSN: %w", parseErr)
		}
		if parsed.Host != "" && parsed.Host != "localhost" {
			return "", false, false, fmt.Errorf("unsupported sqlite file URI host %q", parsed.Host)
		}
		mode := parsed.Query().Get("mode")
		if mode == "memory" {
			return "", false, false, nil
		}
		path = parsed.Path
		if path == "" {
			path, err = url.PathUnescape(parsed.Opaque)
			if err != nil {
				return "", false, false, fmt.Errorf("decode sqlite database path: %w", err)
			}
		}
		if path == "" || path == ":memory:" {
			return "", false, false, nil
		}
		path = sqliteURIPathToNative(path, runtime.GOOS)
		createAllowed = (mode == "" || mode == "rwc") && !sqliteTruthy(parsed.Query().Get("immutable"))
	} else {
		path, _, _ = strings.Cut(dsn, "?")
		if path == ":memory:" {
			return "", false, false, nil
		}
		createAllowed = true
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", false, false, fmt.Errorf("resolve sqlite database path: %w", err)
	}
	return path, true, createAllowed, nil
}

func sqliteURIPathToNative(path, goos string) string {
	if goos == "windows" {
		if len(path) >= 3 && path[0] == '/' && path[2] == ':' &&
			((path[1] >= 'A' && path[1] <= 'Z') || (path[1] >= 'a' && path[1] <= 'z')) {
			path = path[1:]
		}
		return strings.ReplaceAll(path, "/", `\`)
	}
	return filepath.FromSlash(path)
}

func sqliteTruthy(value string) bool {
	switch strings.ToLower(value) {
	case "1", "on", "true", "yes":
		return true
	default:
		return false
	}
}

func sqliteDSNWithForeignKeys(dsn string) (string, error) {
	base, rawQuery, _ := strings.Cut(dsn, "?")
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", fmt.Errorf("parse sqlite DSN query: %w", err)
	}
	query.Del("_fk")
	query.Set("_foreign_keys", "on")
	return base + "?" + query.Encode(), nil
}

func restrictSQLiteFilePermissions(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm", path + "-journal"} {
		if _, err := restrictSQLiteFilePermission(candidate); err != nil {
			return err
		}
	}
	return nil
}

// restrictSQLiteFilePermission rejects symlinks and changes the permissions
// through an open descriptor after verifying that it still refers to the file
// inspected with Lstat. This avoids following a swapped link during startup.
func restrictSQLiteFilePermission(path string) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect sqlite file %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("sqlite file %q must not be a symbolic link", path)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("sqlite file %q must be a regular file", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open sqlite file %q to restrict permissions: %w", path, err)
	}
	currentInfo, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return false, fmt.Errorf("verify sqlite file %q: %w", path, statErr)
	}
	if !os.SameFile(info, currentInfo) {
		_ = file.Close()
		return false, fmt.Errorf("sqlite file %q changed while restricting permissions", path)
	}
	if err = file.Chmod(0o600); err != nil {
		_ = file.Close()
		return false, fmt.Errorf("restrict sqlite file permissions for %q: %w", path, err)
	}
	if err = file.Close(); err != nil {
		return false, fmt.Errorf("close sqlite file %q after restricting permissions: %w", path, err)
	}
	return true, nil
}
