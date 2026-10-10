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
	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
	infraDB "github.com/matrixhub-ai/matrixhub/internal/infra/db"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestArtifactStoreRevisionPersistenceAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.db")
	database, err := openArtifactTestDatabase(t, path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := database.DB()
	sqlDB.SetMaxOpenConns(1)
	store, err := NewArtifactScanStore(database)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	name := "models/p/r"
	oldRev := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newRev := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	missing, err := store.Get(ctx, name, oldRev)
	if err != nil || missing.Status != artifactscan.Unscanned {
		t.Fatal(missing, err)
	}
	for _, rev := range []string{oldRev, newRev, oldRev} {
		if err := store.Enqueue(ctx, name, rev); err != nil {
			t.Fatal(err)
		}
	}
	old, found, err := store.Claim(ctx)
	if err != nil || !found || old.Revision != oldRev {
		t.Fatal(old, found, err)
	}
	old.Status = artifactscan.Blocked
	old.UpdatedAt = time.Now().UTC()
	if err := store.Save(ctx, old); err != nil {
		t.Fatal(err)
	}
	newer, found, err := store.Claim(ctx)
	if err != nil || !found || newer.Revision != newRev {
		t.Fatal(newer, found, err)
	}
	_, found, err = store.Claim(ctx)
	if err != nil || found {
		t.Fatal("duplicate claim", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = openArtifactTestDatabase(t, path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, _ := database.DB()
	reopened.SetMaxOpenConns(1)
	defer reopened.Close()
	store, err = NewArtifactScanStore(database)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	historical, err := store.Get(ctx, name, oldRev)
	if err != nil || historical.Status != artifactscan.Blocked {
		t.Fatal(historical, err)
	}
	resumed, found, err := store.Claim(ctx)
	if err != nil || !found || resumed.Revision != newRev {
		t.Fatal(resumed, found, err)
	}
	resumed.Status = artifactscan.Passed
	resumed.UpdatedAt = time.Now().UTC()
	if err := store.Save(ctx, resumed); err != nil {
		t.Fatal(err)
	}
	historical, err = store.Get(ctx, name, oldRev)
	if err != nil || historical.Status != artifactscan.Blocked {
		t.Fatal(historical, err)
	}
}

func newArtifactTestStore(t *testing.T) *ArtifactScanStore {
	t.Helper()
	db, err := openArtifactTestDatabase(t, filepath.Join(t.TempDir(), "control.db"))
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	s, err := NewArtifactScanStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestArtifactOldAttemptCannotOverwriteRescanOrCancel(t *testing.T) {
	s := newArtifactTestStore(t)
	ctx := context.Background()
	rev := strings.Repeat("a", 40)
	if err := s.Enqueue(ctx, "p/r", rev); err != nil {
		t.Fatal(err)
	}
	old, found, err := s.Claim(ctx)
	if err != nil || !found {
		t.Fatal(err)
	}
	if err := s.Rescan(ctx, "p/r", rev, "operator", true); err != nil {
		t.Fatal(err)
	}
	newer, found, err := s.Claim(ctx)
	if err != nil || !found || newer.Attempt != old.Attempt+1 {
		t.Fatal(newer, err)
	}
	old.Status = artifactscan.Passed
	old.UpdatedAt = time.Now().UTC()
	if err := s.Save(ctx, old); err != nil {
		t.Fatal(err)
	}
	r, err := s.Get(ctx, "p/r", rev)
	if err != nil || r.Status != artifactscan.Scanning || r.Attempt != newer.Attempt {
		t.Fatal(r, err)
	}
	if err := s.Cancel(ctx, "p/r", rev, "operator"); err != nil {
		t.Fatal(err)
	}
	newer.Status = artifactscan.Passed
	newer.UpdatedAt = time.Now().UTC()
	if err := s.Save(ctx, newer); err != nil {
		t.Fatal(err)
	}
	r, err = s.Get(ctx, "p/r", rev)
	if err != nil || r.Status != artifactscan.Cancelled {
		t.Fatal(r, err)
	}
	events, err := s.Events(ctx, "p/r", 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range events {
		if event.Action == "passed" {
			t.Fatal("stale completion was audited as accepted")
		}
	}
}

func TestArtifactClaimAuditFailureRollsBack(t *testing.T) {
	s := newArtifactTestStore(t)
	ctx := context.Background()
	rev := strings.Repeat("a", 40)
	if err := s.Enqueue(ctx, "p/r", rev); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Migrator().DropTable(&artifactAuditRow{}); err != nil {
		t.Fatal(err)
	}
	_, found, err := s.Claim(ctx)
	if err == nil || found {
		t.Fatal("claim ignored audit failure")
	}
	r, err := s.Get(ctx, "p/r", rev)
	if err != nil || r.Status != artifactscan.Pending {
		t.Fatal(r, err)
	}
}

func TestArtifactCacheIsolationAndProjectPolicies(t *testing.T) {
	s := newArtifactTestStore(t)
	ctx := context.Background()
	rules := strings.Repeat("a", 64)
	r := artifactscan.FileResult{Path: "weights.pkl", SHA256: strings.Repeat("b", 64), Status: artifactscan.Passed, Ruleset: rules}
	if err := s.Cache(ctx, "p/r", rules, r); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		repo, path, rules string
		found             bool
	}{
		{"p/r", "weights.pkl", rules, true}, {"other/r", "weights.pkl", rules, false},
		{"p/other", "weights.pkl", rules, false}, {"p/r", "weights.bin", rules, false},
		{"p/r", "weights.pkl", strings.Repeat("c", 64), false},
	} {
		_, found, err := s.Cached(ctx, tc.repo, tc.path, r.SHA256, tc.rules)
		if err != nil || found != tc.found {
			t.Fatal(tc, found, err)
		}
	}
	r.Status = artifactscan.Failed
	if err := s.Cache(ctx, "p/r", rules, r); err == nil {
		t.Fatal("cached incomplete result")
	}
	p := artifactscan.Policy{BlockSeverity: "high", OnPending: "block", OnFailure: "block"}
	if err := s.PutPolicy(ctx, "p/r", p, "manager"); err != nil {
		t.Fatal(err)
	}
	got, found, err := s.GetPolicy(ctx, "p/other")
	if err != nil || !found || got != p {
		t.Fatal(got, found, err)
	}
	_, found, err = s.GetPolicy(ctx, "other/r")
	if err != nil || found {
		t.Fatal("policy crossed project")
	}
	events, err := s.Events(ctx, "p/other", 100)
	if err != nil || len(events) != 1 || events[0].Action != "policy-updated" {
		t.Fatal(events, err)
	}
}

func openArtifactTestDatabase(t *testing.T, path string) (*gorm.DB, error) {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return infraDB.New(infraDB.Config{Driver: infraDB.DriverSQLite, DSN: "file:" + path, Migrate: true, SQLPath: filepath.Join(filepath.Dir(current), "..", "..", "db", "migrations", "sql")})
}
func TestArtifactStoreRequiresMigration(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "unmigrated.db")), &gorm.Config{})
	require.NoError(t, err)
	raw, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, raw.Close()) })
	store, err := NewArtifactScanStore(database)
	require.Nil(t, store)
	require.ErrorContains(t, err, "migration 2")
	require.False(t, database.Migrator().HasTable(&artifactScanRow{}))
}
