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

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
)

func TestAuthzDBRepo_GetUserAccessibleProjectIDs(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")

	for _, test := range []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    []int
		wantErr error
	}{
		{
			name: "public projects and projects with user membership",
			rows: sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(3).AddRow(4),
			want: []int{1, 3, 4},
		},
		{
			name: "no accessible projects",
			rows: sqlmock.NewRows([]string{"id"}),
			want: []int{},
		},
		{
			name:    "query error",
			err:     dbErr,
			wantErr: dbErr,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, mock := newMockGormDB(t)
			query := mock.ExpectQuery(exactSQL("SELECT DISTINCT id FROM `projects` WHERE type = ? OR id IN "+
				"(SELECT project_id FROM `members_roles_projects` WHERE member_id = ? AND member_type = ? AND project_id IS NOT NULL)")).
				WithArgs(project.ProjectTypePublic, 5, project.MemberTypeUser)
			if test.err != nil {
				query.WillReturnError(test.err)
			} else {
				query.WillReturnRows(test.rows)
			}

			projectIDs, err := NewAuthzDBRepo(db).GetUserAccessibleProjectIDs(ctx, 5)

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				require.Nil(t, projectIDs)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, projectIDs)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
