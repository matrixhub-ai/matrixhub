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
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

const (
	datasetFromSQL = "FROM datasets d INNER JOIN projects p ON d.project_id = p.id"
	datasetListSQL = "SELECT d.id, d.name, d.project_id, d.default_branch, d.num_rows, d.size, d.readme_content, " +
		"d.is_popular, d.created_at, d.updated_at, p.name as project_name " + datasetFromSQL
	datasetCountSQL       = "SELECT count(*) " + datasetFromSQL
	datasetLabelFilterSQL = "EXISTS ( SELECT 1 FROM datasets_labels dl INNER JOIN labels l ON dl.label_id = l.id " +
		"WHERE dl.dataset_id = d.id AND l.name = ? )"
	datasetGetSQL = "SELECT d.id, d.name, d.project_id, d.default_branch, d.num_rows, d.size, d.readme_content, " +
		"d.created_at, d.updated_at, p.name as project_name " + datasetFromSQL +
		" WHERE p.name = ? AND d.name = ? ORDER BY `d`.`id` LIMIT ?"
)

func newDatasetDBForTest(t *testing.T) (*datasetDB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := newMockGormDB(t)
	return NewDatasetDB(db).(*datasetDB), mock
}

func datasetListRows(names ...string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "name", "project_id", "default_branch", "num_rows", "size", "readme_content",
		"is_popular", "created_at", "updated_at", "project_name",
	})
	for i, name := range names {
		rows.AddRow(int64(7+i), name, 3, "main", "1000", int64(2048), "# "+name, i == 0, repoTestCreatedAt, repoTestUpdatedAt, "proj")
	}
	return rows
}

func TestDatasetDB_ListBuildsFilteredPagedQuery(t *testing.T) {
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
			name:     "defaults to the first page of 20 most recently updated",
			filter:   &model.Filter{},
			order:    " ORDER BY d.updated_at DESC LIMIT ?",
			pageArgs: []driver.Value{20},
		},
		{
			name:      "project and popular filters with offset",
			filter:    &model.Filter{Project: "proj", Popular: &popular, Page: 2, PageSize: 10},
			where:     " WHERE p.name = ? AND d.is_popular = ?",
			whereArgs: []driver.Value{"proj", 1},
			order:     " ORDER BY d.updated_at DESC LIMIT ? OFFSET ?",
			pageArgs:  []driver.Value{10, 10},
		},
		{
			name:      "search matches project or dataset name",
			filter:    &model.Filter{Search: "llm", Page: 1, PageSize: 20},
			where:     " WHERE p.name LIKE ? OR d.name LIKE ?",
			whereArgs: []driver.Value{"%llm%", "%llm%"},
			order:     " ORDER BY d.updated_at DESC LIMIT ?",
			pageArgs:  []driver.Value{20},
		},
		{
			name:      "every label must match with ascending sort",
			filter:    &model.Filter{Label: []string{"text-generation", "transformers"}, Sort: "asc", Page: 1, PageSize: 20},
			where:     " WHERE (" + datasetLabelFilterSQL + ") AND (" + datasetLabelFilterSQL + ")",
			whereArgs: []driver.Value{"text-generation", "transformers"},
			order:     " ORDER BY d.updated_at ASC LIMIT ?",
			pageArgs:  []driver.Value{20},
		},
		{
			name: "all filters combined",
			filter: &model.Filter{
				Project: "proj", Search: "llm", Label: []string{"text-generation"},
				Popular: &popular, Sort: "updated_at_asc", Page: 3, PageSize: 5,
			},
			where:     " WHERE p.name = ? AND (p.name LIKE ? OR d.name LIKE ?) AND (" + datasetLabelFilterSQL + ") AND d.is_popular = ?",
			whereArgs: []driver.Value{"proj", "%llm%", "%llm%", "text-generation", 1},
			order:     " ORDER BY d.updated_at ASC LIMIT ? OFFSET ?",
			pageArgs:  []driver.Value{5, 10},
		},
		{
			name:     "popular false does not filter",
			filter:   &model.Filter{Popular: &notPopular, Sort: "name", Page: 1, PageSize: 20},
			order:    " ORDER BY d.updated_at DESC LIMIT ?",
			pageArgs: []driver.Value{20},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newDatasetDBForTest(t)
			mock.ExpectQuery(exactSQL(datasetCountSQL + test.where)).
				WithArgs(test.whereArgs...).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))
			mock.ExpectQuery(exactSQL(datasetListSQL + test.where + test.order)).
				WithArgs(append(append([]driver.Value{}, test.whereArgs...), test.pageArgs...)...).
				WillReturnRows(datasetListRows())

			datasets, total, err := repo.List(ctx, test.filter)

			require.NoError(t, err)
			require.Equal(t, int64(42), total)
			require.Empty(t, datasets)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDatasetDB_ListAttachesLabelsToDatasets(t *testing.T) {
	ctx := context.Background()
	repo, mock := newDatasetDBForTest(t)
	mock.ExpectQuery(exactSQL(datasetCountSQL)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(exactSQL(datasetListSQL + " ORDER BY d.updated_at DESC LIMIT ?")).
		WithArgs(20).
		WillReturnRows(datasetListRows("corpus", "images"))
	mock.ExpectQuery(exactSQL("SELECT dl.dataset_id, l.* FROM datasets_labels dl INNER JOIN labels l ON dl.label_id = l.id WHERE dl.dataset_id IN (?,?)")).
		WithArgs(int64(7), int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"dataset_id", "id", "name", "category", "scope", "created_at", "updated_at"}).
			AddRow(int64(7), 11, "text-generation", "task", "dataset", repoTestCreatedAt, repoTestUpdatedAt).
			AddRow(int64(7), 12, "transformers", "library", "dataset", repoTestCreatedAt, repoTestUpdatedAt))

	datasets, total, err := repo.List(ctx, &model.Filter{Page: 1, PageSize: 20})

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Equal(t, []*dataset.Dataset{
		{
			ID: 7, Name: "corpus", ProjectID: 3, ProjectName: "proj", DefaultBranch: "main", NumRows: "1000",
			Size: 2048, ReadmeContent: "# corpus", IsPopular: true, CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt,
			Labels: []model.Label{
				{ID: 11, Name: "text-generation", Category: "task", Scope: "dataset", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt},
				{ID: 12, Name: "transformers", Category: "library", Scope: "dataset", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt},
			},
		},
		{
			ID: 8, Name: "images", ProjectID: 3, ProjectName: "proj", DefaultBranch: "main", NumRows: "1000",
			Size: 2048, ReadmeContent: "# images", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt,
		},
	}, datasets)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDatasetDB_ListWrapsQueryErrors(t *testing.T) {
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
				mock.ExpectQuery(exactSQL(datasetCountSQL)).WillReturnError(dbErr)
			},
			wantErr: "failed to count datasets",
		},
		{
			name: "list",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(datasetCountSQL)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(exactSQL(datasetListSQL + " ORDER BY d.updated_at DESC LIMIT ?")).WithArgs(20).WillReturnError(dbErr)
			},
			wantErr: "failed to list datasets",
		},
		{
			name: "labels",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(exactSQL(datasetCountSQL)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(exactSQL(datasetListSQL + " ORDER BY d.updated_at DESC LIMIT ?")).WithArgs(20).WillReturnRows(datasetListRows("corpus"))
				mock.ExpectQuery(exactSQL("SELECT dl.dataset_id, l.* FROM datasets_labels dl INNER JOIN labels l ON dl.label_id = l.id WHERE dl.dataset_id IN (?)")).
					WithArgs(int64(7)).
					WillReturnError(dbErr)
			},
			wantErr: "failed to fetch labels",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newDatasetDBForTest(t)
			test.expect(mock)

			datasets, total, err := repo.List(ctx, &model.Filter{Page: 1, PageSize: 20})

			require.ErrorIs(t, err, dbErr)
			require.ErrorContains(t, err, test.wantErr)
			require.Nil(t, datasets)
			require.Zero(t, total)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDatasetDB_GetByProjectAndName(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    *dataset.Dataset
		wantErr error
	}{
		{
			name: "maps the joined row",
			rows: sqlmock.NewRows([]string{
				"id", "name", "project_id", "default_branch", "num_rows", "size", "readme_content",
				"created_at", "updated_at", "project_name",
			}).AddRow(int64(7), "corpus", 3, "main", "1000", int64(2048), "# corpus", repoTestCreatedAt, repoTestUpdatedAt, "proj"),
			want: &dataset.Dataset{
				ID: 7, Name: "corpus", ProjectID: 3, ProjectName: "proj", DefaultBranch: "main", NumRows: "1000",
				Size: 2048, ReadmeContent: "# corpus", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt,
			},
		},
		{
			name:    "not found",
			rows:    sqlmock.NewRows([]string{"id"}),
			wantErr: gorm.ErrRecordNotFound,
		},
		{
			name:    "query error",
			err:     dbErr,
			wantErr: dbErr,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newDatasetDBForTest(t)
			query := mock.ExpectQuery(exactSQL(datasetGetSQL)).WithArgs("proj", "corpus", 1)
			if test.err != nil {
				query.WillReturnError(test.err)
			} else {
				query.WillReturnRows(test.rows)
			}

			got, err := repo.GetByProjectAndName(ctx, "proj", "corpus")

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.ErrorContains(t, err, "failed to get dataset")
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, got)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDatasetDB_Delete(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")
	expectLookup := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(exactSQL(datasetGetSQL)).WithArgs("proj", "corpus", 1)
	}
	found := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "name", "project_name"}).AddRow(int64(7), "corpus", "proj")
	}

	for _, test := range []struct {
		name       string
		expect     func(sqlmock.Sqlmock)
		wantErr    error
		wantErrMsg string
	}{
		{
			name: "deletes by the looked up ID",
			expect: func(mock sqlmock.Sqlmock) {
				expectLookup(mock).WillReturnRows(found())
				mock.ExpectExec(exactSQL("DELETE FROM `datasets` WHERE `datasets`.`id` = ?")).
					WithArgs(int64(7)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "lookup error",
			expect: func(mock sqlmock.Sqlmock) {
				expectLookup(mock).WillReturnRows(sqlmock.NewRows([]string{"id"}))
			},
			wantErr:    gorm.ErrRecordNotFound,
			wantErrMsg: "failed to get dataset",
		},
		{
			name: "delete error",
			expect: func(mock sqlmock.Sqlmock) {
				expectLookup(mock).WillReturnRows(found())
				mock.ExpectExec(exactSQL("DELETE FROM `datasets` WHERE `datasets`.`id` = ?")).
					WithArgs(int64(7)).
					WillReturnError(dbErr)
			},
			wantErr:    dbErr,
			wantErrMsg: "failed to delete dataset",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newDatasetDBForTest(t)
			test.expect(mock)

			err := repo.Delete(ctx, "proj", "corpus")

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.ErrorContains(t, err, test.wantErrMsg)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
