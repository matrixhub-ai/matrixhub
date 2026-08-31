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
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

func newModelDBForTest(t *testing.T) (*modelDB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, sqlDB.Close())
	})

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)

	return NewModelDB(db).(*modelDB), mock
}

func expectModelRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "name", "project_id", "size", "parameter_count", "readme_content",
		"is_popular", "default_branch", "synced_at", "created_at", "updated_at", "project_name",
	}).AddRow(7, "model", 3, 1024, 2048, "# README", true, "main", nil, time.Time{}, time.Time{}, "proj")
}

func TestModelDB_ListAllPathsReturnsJoinedProjectPaths(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT p.name AS project_name, m.name AS name FROM models m LEFT JOIN projects p ON m.project_id = p.id WHERE p.name IS NOT NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"project_name", "name"}).AddRow("proj", "model"))

	paths, err := repo.ListAllPaths(ctx)

	require.NoError(t, err)
	require.Equal(t, []string{"proj/model"}, paths)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelDB_ListReturnsModelsAndLabels(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)
	filter := &model.Filter{Page: 1, PageSize: 20}
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM models m INNER JOIN projects p ON m.project_id = p.id").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT m.id, m.name, m.project_id").
		WillReturnRows(expectModelRows())
	mock.ExpectQuery("SELECT ml.model_id, l.\\* FROM models_labels ml INNER JOIN labels l ON ml.label_id = l.id WHERE ml.model_id IN").
		WillReturnRows(sqlmock.NewRows([]string{"model_id", "id", "name", "category", "scope", "created_at", "updated_at"}).
			AddRow(7, 11, "text-generation", "task", "model", time.Time{}, time.Time{}))

	models, total, err := repo.List(ctx, filter)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, models, 1)
	require.Equal(t, "proj", models[0].ProjectName)
	require.Equal(t, "text-generation", models[0].Labels[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelDB_ListReturnsEmptyResultsWithoutLabelQuery(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)
	filter := &model.Filter{Page: 1, PageSize: 20}
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM models m INNER JOIN projects p ON m.project_id = p.id").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT m.id, m.name, m.project_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	models, total, err := repo.List(ctx, filter)

	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Empty(t, models)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelDB_CreateValidatesProjectAndPersistsDefaultBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects empty project name", func(t *testing.T) {
		repo, mock := newModelDBForTest(t)

		created, err := repo.Create(ctx, &model.Model{Name: "model"})

		require.Error(t, err)
		require.Nil(t, created)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates with resolved project ID and main branch", func(t *testing.T) {
		repo, mock := newModelDBForTest(t)
		mock.ExpectQuery("SELECT `id` FROM `projects` WHERE name = \\?").
			WithArgs("proj", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
		mock.ExpectExec("INSERT INTO `models`").
			WithArgs("model", 3, int64(0), "main", int64(0), "", false, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(7, 1))

		created, err := repo.Create(ctx, &model.Model{ProjectName: "proj", Name: "model"})

		require.NoError(t, err)
		require.Equal(t, int64(7), created.ID)
		require.Equal(t, 3, created.ProjectID)
		require.Equal(t, "main", created.DefaultBranch)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestModelDB_GetByProjectAndNameIncludesLabels(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)
	mock.ExpectQuery("SELECT m.id, m.name, m.project_id").
		WithArgs("proj", "model", 1).
		WillReturnRows(expectModelRows())
	mock.ExpectQuery("SELECT l.\\* FROM models_labels ml INNER JOIN labels l ON ml.label_id = l.id WHERE ml.model_id = \\?").
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "category", "scope", "created_at", "updated_at"}).
			AddRow(11, "text-generation", "task", "model", time.Time{}, time.Time{}))

	got, err := repo.GetByProjectAndName(ctx, "proj", "model")

	require.NoError(t, err)
	require.Equal(t, int64(7), got.ID)
	require.Equal(t, "text-generation", got.Labels[0].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelDB_DeleteRemovesModelAfterLookup(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)
	mock.ExpectQuery("SELECT m.id, m.name, m.project_id").
		WithArgs("proj", "model", 1).
		WillReturnRows(expectModelRows())
	mock.ExpectQuery("SELECT l.\\* FROM models_labels ml INNER JOIN labels l ON ml.label_id = l.id WHERE ml.model_id = \\?").
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "category", "scope", "created_at", "updated_at"}))
	mock.ExpectExec("DELETE FROM `models` WHERE `models`.`id` = \\?").
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(ctx, "proj", "model")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelDB_UpdatesOnlyRequestedFields(t *testing.T) {
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		repo, mock := newModelDBForTest(t)
		readme := "# Updated"
		size := int64(1024)
		parameterCount := int64(2048)
		mock.ExpectExec("UPDATE `models` SET `parameter_count`=\\?,`readme_content`=\\?,`size`=\\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs(parameterCount, readme, size, sqlmock.AnyArg(), 7).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateMetadata(ctx, 7, &model.MetadataUpdate{
			ReadmeContent:  &readme,
			Size:           &size,
			ParameterCount: &parameterCount,
		})

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("setting", func(t *testing.T) {
		repo, mock := newModelDBForTest(t)
		popular := true
		mock.ExpectExec("UPDATE `models` SET `is_popular`=\\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs(true, sqlmock.AnyArg(), 7).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateSetting(ctx, 7, &model.SettingUpdate{IsPopular: &popular})

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("synced at", func(t *testing.T) {
		repo, mock := newModelDBForTest(t)
		syncedAt := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
		mock.ExpectExec("UPDATE `models` SET `synced_at`=\\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs(syncedAt, sqlmock.AnyArg(), 7).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateSyncedAt(ctx, 7, &syncedAt)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestModelDB_UpdateMethodsSkipEmptyUpdates(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)

	require.NoError(t, repo.UpdateMetadata(ctx, 7, &model.MetadataUpdate{}))
	require.NoError(t, repo.UpdateSetting(ctx, 7, &model.SettingUpdate{}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModelDB_CreateReturnsProjectLookupErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newModelDBForTest(t)
	wantErr := errors.New("database unavailable")
	mock.ExpectQuery("SELECT `id` FROM `projects` WHERE name = \\?").
		WithArgs("proj", 1).
		WillReturnError(wantErr)

	created, err := repo.Create(ctx, &model.Model{ProjectName: "proj", Name: "model"})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, created)
	require.NoError(t, mock.ExpectationsWereMet())
}
