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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

func TestBuildUserListQueryExcludesProjectMembers(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	require.NoError(t, err)

	query := buildUserListQuery(db, "", "demo")
	var total int64
	require.NoError(t, query.Count(&total).Error)
	countSQL := strings.ToLower(query.Statement.SQL.String())

	query = buildUserListQuery(db, "", "demo")
	var users []*user.User
	require.NoError(t, query.Find(&users).Error)
	listSQL := strings.ToLower(query.Statement.SQL.String())

	for _, sql := range []string{countSQL, listSQL} {
		require.Contains(t, sql, "not exists")
		require.Contains(t, sql, "members_roles_projects")
		require.Contains(t, sql, "projects")
		require.Contains(t, sql, "member_type")
	}
	require.Contains(t, query.Statement.Vars, "demo")
}
