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
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "mysql unique violation",
			err:  &mysql.MySQLError{Number: 1062},
			want: true,
		},
		{
			name: "postgres unique violation",
			err:  &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			want: true,
		},
		{
			name: "other database error",
			err:  &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("database error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsUniqueViolationError(tt.err))
		})
	}
}
