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
	"database/sql/driver"
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

const projectIDByNameSQL = "SELECT `id` FROM `projects` WHERE name = ? ORDER BY `projects`.`id` LIMIT ?"

var (
	repoTestCreatedAt = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	repoTestUpdatedAt = time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
)

// newMockGormDB returns a MySQL-dialect gorm DB backed by sqlmock.
func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

	return db, mock
}

// exactSQL matches the whole statement, so missing or extra clauses fail.
func exactSQL(statement string) string {
	return "^" + regexp.QuoteMeta(statement) + "$"
}

// anyTime matches the timestamps gorm fills in on create.
type anyTime struct{}

var _ sqlmock.Argument = anyTime{}

func (anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func newModelDBForTest(t *testing.T) (*modelDB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := newMockGormDB(t)
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

const (
	modelFromSQL = "FROM models m INNER JOIN projects p ON m.project_id = p.id"
	modelListSQL = "SELECT m.id, m.name, m.project_id, m.size, m.parameter_count, m.readme_content, m.is_popular, " +
		"m.default_branch, m.synced_at, m.created_at, m.updated_at, p.name as project_name " + modelFromSQL
	modelCountSQL       = "SELECT count(*) " + modelFromSQL
	modelLabelFilterSQL = "EXISTS ( SELECT 1 FROM models_labels ml INNER JOIN labels l ON ml.label_id = l.id " +
		"WHERE ml.model_id = m.id AND l.name = ? )"
	modelGetSQL       = modelListSQL + " WHERE p.name = ? AND m.name = ? ORDER BY `m`.`id` LIMIT ?"
	modelLabelsSQL    = "SELECT l.* FROM models_labels ml INNER JOIN labels l ON ml.label_id = l.id WHERE ml.model_id = ?"
	modelDeleteSQL    = "DELETE FROM `models` WHERE `models`.`id` = ?"
	modelFirstPageSQL = " ORDER BY m.updated_at DESC LIMIT ?"
)

func TestModelDB_ListBuildsFilteredPagedQuery(t *testing.T) {
	ctx := context.Background()
	popular := true
	notPopular := false

	for _, test := range []struct {
		name      string
		filter    *model.Filter
		where     string
		whereArgs []driver.Value
		order     string
		pageArgs  []driver.Value
	}{
		{
			name:      "accessible projects",
			filter:    &model.Filter{ProjectIDs: []int{3, 4}, Page: 1, PageSize: 20},
			where:     " WHERE m.project_id IN (?,?)",
			whereArgs: []driver.Value{3, 4},
			order:     modelFirstPageSQL,
			pageArgs:  []driver.Value{20},
		},
		{
			name:      "project and popular filters with offset",
			filter:    &model.Filter{Project: "proj", Popular: &popular, Page: 2, PageSize: 10},
			where:     " WHERE p.name = ? AND m.is_popular = ?",
			whereArgs: []driver.Value{"proj", 1},
			order:     " ORDER BY m.updated_at DESC LIMIT ? OFFSET ?",
			pageArgs:  []driver.Value{10, 10},
		},
		{
			name:      "search matches project or model name",
			filter:    &model.Filter{Search: "llm", Page: 1, PageSize: 20},
			where:     " WHERE p.name LIKE ? OR m.name LIKE ?",
			whereArgs: []driver.Value{"%llm%", "%llm%"},
			order:     modelFirstPageSQL,
			pageArgs:  []driver.Value{20},
		},
		{
			name:      "every label must match with ascending sort",
			filter:    &model.Filter{Label: []string{"text-generation", "transformers"}, Sort: "asc", Page: 1, PageSize: 20},
			where:     " WHERE (" + modelLabelFilterSQL + ") AND (" + modelLabelFilterSQL + ")",
			whereArgs: []driver.Value{"text-generation", "transformers"},
			order:     " ORDER BY m.updated_at ASC LIMIT ?",
			pageArgs:  []driver.Value{20},
		},
		{
			name: "all filters combined",
			filter: &model.Filter{
				ProjectIDs: []int{3}, Project: "proj", Search: "llm", Label: []string{"text-generation"},
				Popular: &popular, Sort: "updated_at_asc", Page: 3, PageSize: 5,
			},
			where: " WHERE m.project_id IN (?) AND p.name = ? AND (p.name LIKE ? OR m.name LIKE ?) AND (" +
				modelLabelFilterSQL + ") AND m.is_popular = ?",
			whereArgs: []driver.Value{3, "proj", "%llm%", "%llm%", "text-generation", 1},
			order:     " ORDER BY m.updated_at ASC LIMIT ? OFFSET ?",
			pageArgs:  []driver.Value{5, 10},
		},
		{
			name:     "popular false does not filter",
			filter:   &model.Filter{Popular: &notPopular, Sort: "name", Page: 1, PageSize: 20},
			order:    modelFirstPageSQL,
			pageArgs: []driver.Value{20},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newModelDBForTest(t)
			mock.ExpectQuery(exactSQL(modelCountSQL + test.where)).
				WithArgs(test.whereArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))
			mock.ExpectQuery(exactSQL(modelListSQL + test.where + test.order)).
				WithArgs(append(append([]driver.Value{}, test.whereArgs...), test.pageArgs...)...).
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			models, total, err := repo.List(ctx, test.filter)

			require.NoError(t, err)
			require.Equal(t, int64(42), total)
			require.Empty(t, models)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestModelDB_ListWrapsQueryErrors(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		wantErr string
	}{
		{
			name: "count",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelCountSQL)).WillReturnError(dbErr)
			},
			wantErr: "failed to count models",
		},
		{
			name: "list",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelCountSQL)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(exactSQL(modelListSQL + modelFirstPageSQL)).WithArgs(20).WillReturnError(dbErr)
			},
			wantErr: "failed to list models",
		},
		{
			name: "labels",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelCountSQL)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(exactSQL(modelListSQL + modelFirstPageSQL)).WithArgs(20).WillReturnRows(expectModelRows())
				mock.ExpectQuery(exactSQL("SELECT ml.model_id, l.* FROM models_labels ml INNER JOIN labels l ON ml.label_id = l.id WHERE ml.model_id IN (?)")).
					WithArgs(int64(7)).
					WillReturnError(dbErr)
			},
			wantErr: "failed to fetch labels",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newModelDBForTest(t)
			test.expect(mock)

			models, total, err := repo.List(ctx, &model.Filter{Page: 1, PageSize: 20})

			require.ErrorIs(t, err, dbErr)
			require.ErrorContains(t, err, test.wantErr)
			require.Nil(t, models)
			require.Zero(t, total)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestModelDB_CreateReportsMissingProjectAndInsertErrors(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		wantErr error
		wantMsg string
	}{
		{
			name: "project not found",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(projectIDByNameSQL)).WithArgs("proj", 1).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
			},
			wantMsg: "project not found: proj",
		},
		{
			name: "insert error",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(projectIDByNameSQL)).WithArgs("proj", 1).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
				mock.ExpectExec(exactSQL("INSERT INTO `models` (`name`,`project_id`,`size`,`default_branch`,`parameter_count`,`readme_content`,`is_popular`,`synced_at`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?,?,?,?)")).
					WithArgs("model", 3, int64(0), "main", int64(0), "", false, nil, anyTime{}, anyTime{}).
					WillReturnError(dbErr)
			},
			wantErr: dbErr,
			wantMsg: "failed to create model",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newModelDBForTest(t)
			test.expect(mock)

			created, err := repo.Create(ctx, &model.Model{ProjectName: "proj", Name: "model"})

			require.ErrorContains(t, err, test.wantMsg)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
			}
			require.Nil(t, created)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestModelDB_GetByProjectAndNameWrapsErrors(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		wantErr error
		wantMsg string
	}{
		{
			name: "model not found",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelGetSQL)).WithArgs("proj", "model", 1).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
			},
			wantErr: gorm.ErrRecordNotFound,
			wantMsg: "failed to get model",
		},
		{
			name: "labels query error",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelGetSQL)).WithArgs("proj", "model", 1).
					WillReturnRows(expectModelRows())
				mock.ExpectQuery(exactSQL(modelLabelsSQL)).WithArgs(int64(7)).
					WillReturnError(dbErr)
			},
			wantErr: dbErr,
			wantMsg: "failed to get model labels",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newModelDBForTest(t)
			test.expect(mock)

			got, err := repo.GetByProjectAndName(ctx, "proj", "model")

			require.ErrorIs(t, err, test.wantErr)
			require.ErrorContains(t, err, test.wantMsg)
			require.Nil(t, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestModelDB_DeleteWrapsErrors(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		wantErr error
		wantMsg string
	}{
		{
			name: "lookup error",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelGetSQL)).WithArgs("proj", "model", 1).
					WillReturnError(dbErr)
			},
			wantErr: dbErr,
			wantMsg: "failed to get model",
		},
		{
			name: "delete error",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(modelGetSQL)).WithArgs("proj", "model", 1).
					WillReturnRows(expectModelRows())
				mock.ExpectQuery(exactSQL(modelLabelsSQL)).WithArgs(int64(7)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
				mock.ExpectExec(exactSQL(modelDeleteSQL)).WithArgs(int64(7)).
					WillReturnError(dbErr)
			},
			wantErr: dbErr,
			wantMsg: "failed to delete model",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newModelDBForTest(t)
			test.expect(mock)

			err := repo.Delete(ctx, "proj", "model")

			require.ErrorIs(t, err, test.wantErr)
			require.ErrorContains(t, err, test.wantMsg)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
