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
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

const (
	labelInsertSQL = "INSERT INTO `labels` (`name`,`category`,`scope`,`created_at`,`updated_at`) VALUES (?,?,?,?,?) " +
		"ON DUPLICATE KEY UPDATE `id`=`id`"
	labelLookupSQL = "SELECT * FROM `labels` WHERE name = ? AND category = ? AND scope = ? ORDER BY `labels`.`id` LIMIT ?"
)

func newLabelDBForTest(t *testing.T) (*labelDB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := newMockGormDB(t)
	return NewLabelDB(db).(*labelDB), mock
}

func labelRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "name", "category", "scope", "created_at", "updated_at"})
}

func TestLabelDB_ListByCategoryAndScope(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    []*model.Label
		wantErr error
	}{
		{
			name: "maps matching labels",
			rows: labelRows().
				AddRow(11, "text-generation", "task", "model", repoTestCreatedAt, repoTestUpdatedAt).
				AddRow(12, "image-classification", "task", "model", repoTestCreatedAt, repoTestUpdatedAt),
			want: []*model.Label{
				{ID: 11, Name: "text-generation", Category: "task", Scope: "model", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt},
				{ID: 12, Name: "image-classification", Category: "task", Scope: "model", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt},
			},
		},
		{
			name:    "query error",
			err:     dbErr,
			wantErr: dbErr,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newLabelDBForTest(t)
			query := mock.ExpectQuery(exactSQL("SELECT * FROM `labels` WHERE category = ? AND scope = ?")).
				WithArgs("task", "model")
			if test.err != nil {
				query.WillReturnError(test.err)
			} else {
				query.WillReturnRows(test.rows)
			}

			labels, err := repo.ListByCategoryAndScope(ctx, "task", "model")

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.Empty(t, labels)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, labels)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLabelDB_GetByModelID(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    []*model.Label
		wantErr error
	}{
		{
			name: "maps the labels associated with the model",
			rows: labelRows().AddRow(11, "transformers", "library", "model", repoTestCreatedAt, repoTestUpdatedAt),
			want: []*model.Label{
				{ID: 11, Name: "transformers", Category: "library", Scope: "model", CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt},
			},
		},
		{
			name:    "query error",
			err:     dbErr,
			wantErr: dbErr,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newLabelDBForTest(t)
			query := mock.ExpectQuery(exactSQL("SELECT `l`.`id`,`l`.`name`,`l`.`category`,`l`.`scope`,`l`.`created_at`,`l`.`updated_at` " +
				"FROM labels l INNER JOIN models_labels ml ON l.id = ml.label_id WHERE ml.model_id = ?")).
				WithArgs(int64(7))
			if test.err != nil {
				query.WillReturnError(test.err)
			} else {
				query.WillReturnRows(test.rows)
			}

			labels, err := repo.GetByModelID(ctx, 7)

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.Empty(t, labels)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, labels)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLabelDB_GetOrCreateByName(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")
	expectInsert := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedExec {
		return mock.ExpectExec(exactSQL(labelInsertSQL)).
			WithArgs("transformers", "library", "model", anyTime{}, anyTime{})
	}
	expectLookup := func(mock sqlmock.Sqlmock) *sqlmock.ExpectedQuery {
		return mock.ExpectQuery(exactSQL(labelLookupSQL)).
			WithArgs("transformers", "library", "model", 1)
	}

	for _, test := range []struct {
		name    string
		expect  func(sqlmock.Sqlmock)
		want    *model.Label
		wantErr error
		wantMsg string
	}{
		{
			name: "creates a new label",
			expect: func(mock sqlmock.Sqlmock) {
				expectInsert(mock).WillReturnResult(sqlmock.NewResult(11, 1))
			},
			want: &model.Label{ID: 11, Name: "transformers", Category: "library", Scope: "model"},
		},
		{
			name: "returns the existing label on conflict",
			expect: func(mock sqlmock.Sqlmock) {
				expectInsert(mock).WillReturnResult(sqlmock.NewResult(0, 0))
				expectLookup(mock).WillReturnRows(labelRows().
					AddRow(5, "transformers", "library", "model", repoTestCreatedAt, repoTestUpdatedAt))
			},
			want: &model.Label{
				ID: 5, Name: "transformers", Category: "library", Scope: "model",
				CreatedAt: repoTestCreatedAt, UpdatedAt: repoTestUpdatedAt,
			},
		},
		{
			name: "conflicting label disappeared",
			expect: func(mock sqlmock.Sqlmock) {
				expectInsert(mock).WillReturnResult(sqlmock.NewResult(0, 0))
				expectLookup(mock).WillReturnRows(labelRows())
			},
			wantMsg: "label disappeared after conflict for name=transformers category=library scope=model",
		},
		{
			name: "existing label lookup error",
			expect: func(mock sqlmock.Sqlmock) {
				expectInsert(mock).WillReturnResult(sqlmock.NewResult(0, 0))
				expectLookup(mock).WillReturnError(dbErr)
			},
			wantErr: dbErr,
			wantMsg: "failed to load existing label",
		},
		{
			name: "insert error",
			expect: func(mock sqlmock.Sqlmock) {
				expectInsert(mock).WillReturnError(dbErr)
			},
			wantErr: dbErr,
			wantMsg: "failed to create label",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, mock := newLabelDBForTest(t)
			test.expect(mock)

			label, err := repo.GetOrCreateByName(ctx, "transformers", "library", "model")

			if test.wantMsg != "" {
				require.ErrorContains(t, err, test.wantMsg)
				if test.wantErr != nil {
					require.ErrorIs(t, err, test.wantErr)
				}
				require.Nil(t, label)
			} else {
				require.NoError(t, err)
				if test.want.CreatedAt.IsZero() {
					// gorm stamps the timestamps of a newly inserted label.
					require.False(t, label.CreatedAt.IsZero())
					require.False(t, label.UpdatedAt.IsZero())
					label.CreatedAt, label.UpdatedAt = test.want.CreatedAt, test.want.UpdatedAt
				}
				require.Equal(t, test.want, label)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLabelDB_ReplaceLabelAssociations(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, target := range []struct {
		name   string
		table  string
		column string
		update func(*labelDB, context.Context, int64, []int) error
	}{
		{name: "model", table: "models_labels", column: "model_id", update: (*labelDB).UpdateModelLabels},
		{name: "dataset", table: "datasets_labels", column: "dataset_id", update: (*labelDB).UpdateDatasetLabels},
	} {
		deleteSQL := "DELETE FROM " + target.table + " WHERE " + target.column + " = ?"
		insertSQL := "INSERT INTO `" + target.table + "` (`" + target.column + "`,`label_id`) VALUES (?,?),(?,?)"

		for _, test := range []struct {
			name     string
			labelIDs []int
			expect   func(sqlmock.Sqlmock)
			wantMsg  string
		}{
			{
				name:     "replaces existing associations",
				labelIDs: []int{11, 12},
				expect: func(mock sqlmock.Sqlmock) {
					mock.ExpectBegin()
					mock.ExpectExec(exactSQL(deleteSQL)).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 3))
					mock.ExpectExec(exactSQL(insertSQL)).
						WithArgs(int64(7), 11, int64(7), 12).
						WillReturnResult(sqlmock.NewResult(0, 2))
					mock.ExpectCommit()
				},
			},
			{
				name: "clears associations without inserting",
				expect: func(mock sqlmock.Sqlmock) {
					mock.ExpectBegin()
					mock.ExpectExec(exactSQL(deleteSQL)).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 3))
					mock.ExpectCommit()
				},
			},
			{
				name:     "rolls back when deleting fails",
				labelIDs: []int{11, 12},
				expect: func(mock sqlmock.Sqlmock) {
					mock.ExpectBegin()
					mock.ExpectExec(exactSQL(deleteSQL)).WithArgs(int64(7)).WillReturnError(dbErr)
					mock.ExpectRollback()
				},
				wantMsg: "failed to delete old labels",
			},
			{
				name:     "rolls back when inserting fails",
				labelIDs: []int{11, 12},
				expect: func(mock sqlmock.Sqlmock) {
					mock.ExpectBegin()
					mock.ExpectExec(exactSQL(deleteSQL)).WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 3))
					mock.ExpectExec(exactSQL(insertSQL)).
						WithArgs(int64(7), 11, int64(7), 12).
						WillReturnError(dbErr)
					mock.ExpectRollback()
				},
				wantMsg: "failed to insert new labels",
			},
		} {
			t.Run(target.name+"/"+test.name, func(t *testing.T) {
				repo, mock := newLabelDBForTest(t)
				test.expect(mock)

				err := target.update(repo, ctx, 7, test.labelIDs)

				if test.wantMsg != "" {
					require.ErrorIs(t, err, dbErr)
					require.ErrorContains(t, err, test.wantMsg)
				} else {
					require.NoError(t, err)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}
