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

package project_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

func TestProjectDomainRules(t *testing.T) {
	registryID := 7
	proxied := &project.Project{Type: project.ProjectTypePublic, RegistryID: &registryID}
	require.Equal(t, "projects", proxied.TableName())
	require.True(t, proxied.HasProxy())
	require.True(t, proxied.IsPublic())

	private := &project.Project{Type: project.ProjectTypePrivate}
	require.False(t, private.HasProxy())
	require.False(t, private.IsPublic())
	require.Equal(t, "members_roles_projects", (project.ProjectMember{}).TableName())
}

func TestPermissionsForFilter(t *testing.T) {
	require.Equal(t, []role.Permission{role.ModelPush, role.DatasetPush}, project.PermissionsForFilter(project.PermissionFilterCanWrite))
	for _, filter := range []project.PermissionFilter{
		project.PermissionFilterUnspecified,
		project.PermissionFilterManagedOnly,
		project.PermissionFilterCanRead,
	} {
		require.Equal(t, []role.Permission{role.ProjectGet}, project.PermissionsForFilter(filter))
	}
}
