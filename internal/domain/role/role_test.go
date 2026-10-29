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

package role_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

func TestPermissionMatching(t *testing.T) {
	require.Equal(t, "roles", (role.Role{}).TableName())
	require.Equal(t, "model.push", role.ModelPush.String())
	require.False(t, role.ModelPush.IsWildcard())
	require.Nil(t, role.ModelPush.ToRegex())
	require.True(t, role.Permission("model.*").IsWildcard())
	require.True(t, role.Permission("model.*").Match(role.ModelPull))
	require.False(t, role.Permission("model.*").Match(role.DatasetPull))
	require.True(t, role.ModelPush.Match(role.ModelPush))
	require.True(t, role.MatchPermissions([]role.Permission{role.DatasetPull, "model.*"}, role.ModelDelete))
	require.False(t, role.MatchPermissions([]role.Permission{role.DatasetPull}, role.ModelDelete))
}

func TestPermissionConversionsAndPersistence(t *testing.T) {
	permissions := []role.Permission{role.ModelPull, role.DatasetPush}
	strings := role.PermissionsToStrings(permissions)
	require.Equal(t, []string{"model.pull", "dataset.push"}, strings)
	require.Equal(t, permissions, role.StringsToPermissions(strings))

	value, err := (role.PermissionList(nil)).Value()
	require.NoError(t, err)
	require.Equal(t, "[]", value)
	value, err = role.PermissionList(permissions).Value()
	require.NoError(t, err)
	require.JSONEq(t, `["model.pull","dataset.push"]`, value.(string))

	var scanned role.PermissionList
	require.NoError(t, scanned.Scan([]byte(`["model.pull","dataset.push"]`)))
	require.Equal(t, role.PermissionList(permissions), scanned)
	require.NoError(t, scanned.Scan(nil))
	require.Nil(t, scanned)
	require.NoError(t, scanned.Scan("not bytes"))
	require.Error(t, scanned.Scan([]byte(`{`)))
}

func TestPermissionCategories(t *testing.T) {
	require.Equal(t, "model_management", role.ModelManagement.String())
	require.NoError(t, role.ProjectPermissions.CheckPermissions([]role.Permission{role.ModelPush, role.DatasetPull}))
	require.ErrorContains(t, role.ProjectPermissions.CheckPermissions([]role.Permission{role.UserDelete}), "invalid permission")
}
