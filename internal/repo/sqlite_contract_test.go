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
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	appconfig "github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/db"
)

func TestSQLiteRepositoryContract(t *testing.T) {
	database, _ := newSQLiteRepositoryTestDatabase(t)
	ctx := context.Background()

	projectRepo := NewProjectDBRepo(database)
	createdProject, err := projectRepo.CreateProject(ctx, &project.Project{
		Name: "acme",
		Type: project.ProjectTypePublic,
	})
	require.NoError(t, err)

	modelRepo := NewModelDB(database)
	createdModel, err := modelRepo.Create(ctx, &model.Model{
		Name:        "tiny-model",
		ProjectName: createdProject.Name,
	})
	require.NoError(t, err)

	datasetRepo := NewDatasetDB(database)
	_, err = datasetRepo.Create(ctx, &dataset.Dataset{
		Name:        "tiny-dataset",
		ProjectName: createdProject.Name,
	})
	require.NoError(t, err)

	modelPaths, err := modelRepo.ListAllPaths(ctx)
	require.NoError(t, err)
	require.Contains(t, modelPaths, "acme/tiny-model")

	datasetPaths, err := datasetRepo.ListAllPaths(ctx)
	require.NoError(t, err)
	require.Contains(t, datasetPaths, "acme/tiny-dataset")

	models, total, err := modelRepo.List(ctx, &model.Filter{Search: "tiny", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, models, 1)

	datasets, total, err := datasetRepo.List(ctx, &model.Filter{Search: "tiny", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, datasets, 1)

	require.NoError(t, database.Create(&project.ProjectMember{
		ProjectID:  &createdProject.ID,
		MemberID:   1,
		MemberType: project.MemberTypeUser,
		RoleID:     2,
	}).Error)
	members, total, err := projectRepo.ListProjectMembers(ctx, createdProject.ID, "admin", 1, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, members, 1)
	require.Equal(t, "admin", members[0].MemberName)
	assertProjectNameSearchIsCaseSensitive(t, ctx, database)

	assertModelUpdatesTimestamp(t, database, modelRepo, createdModel.ID)
	assertCASUpdatesTimestamps(t, database)
}

func assertProjectNameSearchIsCaseSensitive(t *testing.T, ctx context.Context, database *gorm.DB) {
	t.Helper()

	projectRepo := NewProjectDBRepo(database)
	createdProject, err := projectRepo.CreateProject(ctx, &project.Project{
		Name: "CaseProject",
		Type: project.ProjectTypePublic,
	})
	require.NoError(t, err)

	_, total, err := projectRepo.ListProjects(
		ctx, "caseproject", project.ProjectTypeUnspecified,
		project.PermissionFilterUnspecified, true, 1, 20,
	)
	require.NoError(t, err)
	require.Zero(t, total)

	_, err = NewModelDB(database).Create(ctx, &model.Model{
		Name:        "unrelated-model",
		ProjectName: createdProject.Name,
	})
	require.NoError(t, err)
	models, total, err := NewModelDB(database).List(ctx, &model.Filter{
		Search: "caseproject", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, models)

	_, err = NewDatasetDB(database).Create(ctx, &dataset.Dataset{
		Name:        "unrelated-dataset",
		ProjectName: createdProject.Name,
	})
	require.NoError(t, err)
	datasets, total, err := NewDatasetDB(database).List(ctx, &model.Filter{
		Search: "caseproject", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, datasets)
}

func TestSQLiteSessionStore(t *testing.T) {
	database, config := newSQLiteRepositoryTestDatabase(t)
	session, err := NewSessionRepository(database, config)
	require.NoError(t, err)
	closer, ok := session.(interface{ Close() })
	require.True(t, ok)
	t.Cleanup(closer.Close)

	manager := session.Manager()
	ctx, err := manager.Load(context.Background(), "")
	require.NoError(t, err)
	manager.Put(ctx, "subject", "admin")
	token, _, err := manager.Commit(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	loadedCtx, err := manager.Load(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "admin", manager.GetString(loadedCtx, "subject"))
}

func assertModelUpdatesTimestamp(t *testing.T, database *gorm.DB, repository model.IModelRepo, modelID int64) {
	t.Helper()
	oldTimestamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, database.Table("models").Where("id = ?", modelID).
		UpdateColumn("updated_at", oldTimestamp).Error)
	readme := "# SQLite"
	require.NoError(t, repository.UpdateMetadata(context.Background(), modelID, &model.MetadataUpdate{
		ReadmeContent: &readme,
	}))
	requireTimestampAdvanced(t, database, "models", modelID, oldTimestamp)

	require.NoError(t, database.Table("models").Where("id = ?", modelID).
		UpdateColumn("updated_at", oldTimestamp).Error)
	syncedAt := time.Now().UTC()
	require.NoError(t, repository.UpdateSyncedAt(context.Background(), modelID, &syncedAt))
	requireTimestampAdvanced(t, database, "models", modelID, oldTimestamp)
}

func assertCASUpdatesTimestamps(t *testing.T, database *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	oldTimestamp := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	result := database.Exec(`INSERT INTO sync_policies
		(name, next_run_at, updated_at) VALUES (?, ?, ?)`, "nightly", 100, oldTimestamp)
	require.NoError(t, result.Error)

	var policy struct{ ID int }
	require.NoError(t, database.Table("sync_policies").Select("id").Where("name = ?", "nightly").Take(&policy).Error)
	claimed, err := NewSyncPolicyDB(database).AdvanceNextRunAtCAS(ctx, policy.ID, 100, 200, 150)
	require.NoError(t, err)
	require.True(t, claimed)
	requireTimestampAdvanced(t, database, "sync_policies", int64(policy.ID), oldTimestamp)

	task := &syncpolicy.SyncTask{
		SyncPolicyID: policy.ID,
		TriggerType:  syncpolicy.TriggerTypeManual,
		Status:       syncpolicy.SyncTaskStatusPending,
		UpdatedAt:    oldTimestamp,
	}
	require.NoError(t, database.Create(task).Error)
	updated, err := NewSyncTaskDB(database).UpdateTaskStatusCAS(
		ctx, task.ID, syncpolicy.SyncTaskStatusPending, syncpolicy.SyncTaskStatusRunning,
	)
	require.NoError(t, err)
	require.True(t, updated)
	requireTimestampAdvanced(t, database, "sync_tasks", int64(task.ID), oldTimestamp)

	job := &syncjob.SyncJob{
		RemoteProjectName:  "remote",
		RemoteResourceName: "resource",
		ResourceName:       "resource",
		ResourceType:       "model",
		SyncType:           "pull",
		Status:             syncjob.SyncJobStatusPending,
		UpdatedAt:          oldTimestamp,
	}
	require.NoError(t, database.Create(job).Error)
	updated, err = NewSyncJobDB(database).UpdateJobStatusCAS(
		ctx, job.ID, syncjob.SyncJobStatusPending, syncjob.SyncJobStatusRunning,
	)
	require.NoError(t, err)
	require.True(t, updated)
	requireTimestampAdvanced(t, database, "sync_jobs", int64(job.ID), oldTimestamp)
}

func requireTimestampAdvanced(t *testing.T, database *gorm.DB, table string, id int64, oldTimestamp time.Time) {
	t.Helper()
	var updatedAt time.Time
	require.NoError(t, database.Table(table).Select("updated_at").Where("id = ?", id).Scan(&updatedAt).Error)
	require.True(t, updatedAt.After(oldTimestamp), "%s.updated_at was not advanced", table)
}

func newSQLiteRepositoryTestDatabase(t *testing.T) (*gorm.DB, *appconfig.Config) {
	t.Helper()
	dataDir := t.TempDir()
	dsn, err := db.DefaultSQLiteDSN(dataDir)
	require.NoError(t, err)

	config := &appconfig.Config{
		DataDir: dataDir,
		Session: appconfig.SessionConfig{
			PersistentSessionLifetime:    24 * time.Hour,
			PersistentSessionIdleTimeout: 12 * time.Hour,
			NonPersistentIdleTimeout:     time.Hour,
		},
		Database: db.Config{
			Driver:  db.DriverSQLite,
			DSN:     dsn,
			SQLPath: repositoryMigrationSQLPath(t),
			Migrate: true,
		},
	}
	database, err := db.New(config.Database)
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	return database, config
}

func repositoryMigrationSQLPath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "db", "migrations", "sql"))
}
