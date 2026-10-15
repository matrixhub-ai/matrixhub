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

package repo

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
	infraDB "github.com/matrixhub-ai/matrixhub/internal/infra/db"
)

// MH_MYSQL_TEST_DSN must permit creating a disposable database. No tables in
// the supplied database are modified; each run creates and drops its own schema.
func TestArtifactStoreMySQLMigrationAndPersistence(t *testing.T) {
	dsn := os.Getenv("MH_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set MH_MYSQL_TEST_DSN to run the MySQL integration regression")
	}
	config, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	config.DBName = ""
	admin, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := fmt.Sprintf("mh_security_test_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE `" + name + "`")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, dropErr := admin.Exec("DROP DATABASE `" + name + "`")
		require.NoError(t, dropErr)
	})
	config.DBName = name
	config.MultiStatements, config.ParseTime = true, true
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	databaseConfig := infraDB.Config{
		Driver: infraDB.DriverMySQL, DSN: config.FormatDSN(), Migrate: true,
		SQLPath: filepath.Join(filepath.Dir(source), "../../db/migrations/sql"),
	}
	database, err := infraDB.New(databaseConfig)
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	var version uint
	var dirty bool
	require.NoError(t, database.Table("schema_migrations").Select("version, dirty").Row().Scan(&version, &dirty))
	require.Equal(t, uint(2), version)
	require.False(t, dirty)
	store, err := NewArtifactScanStore(database)
	require.NoError(t, err)
	ctx := context.Background()
	repo, revision := "models/mysql/fixture", strings.Repeat("a", 40)
	require.NoError(t, store.Enqueue(ctx, repo, revision))
	report, found, err := store.Claim(ctx)
	require.NoError(t, err)
	require.True(t, found)
	report.Status, report.UpdatedAt = artifactscan.Passed, time.Now().UTC()
	require.NoError(t, store.Save(ctx, report))
	saved, err := store.Get(ctx, repo, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Passed, saved.Status)
	require.NoError(t, store.Rescan(ctx, repo, revision, "mysql-test", true))
	forced, found, err := store.Claim(ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, forced.Force)
	require.Equal(t, int64(2), forced.Attempt)
	file := artifactscan.FileResult{
		Path: "weights.bin", SHA256: strings.Repeat("b", 64), Ruleset: strings.Repeat("c", 64),
		Status: artifactscan.Passed,
	}
	require.NoError(t, store.Cache(ctx, repo, file.Ruleset, file))
	cached, found, err := store.Cached(ctx, repo, file.Path, file.SHA256, file.Ruleset)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, file.Path, cached.Path)
	policy := artifactscan.Policy{BlockSeverity: "high", OnPending: "block", OnFailure: "block"}
	require.NoError(t, store.PutPolicy(ctx, repo, policy, "mysql-test"))
	savedPolicy, found, err := store.GetPolicy(ctx, repo)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, policy, savedPolicy)
	require.NoError(t, store.Cancel(ctx, repo, revision, "mysql-test"))
	events, err := store.Events(ctx, repo, 20)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	reopened, err := infraDB.New(databaseConfig)
	require.NoError(t, err)
	reopenedSQL, err := reopened.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, reopenedSQL.Close()) })
	reopenedStore, err := NewArtifactScanStore(reopened)
	require.NoError(t, err)
	restored, err := reopenedStore.Get(ctx, repo, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Cancelled, restored.Status)
	require.Equal(t, int64(2), restored.Attempt)
}
