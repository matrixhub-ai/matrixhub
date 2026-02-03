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

package integration

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
)

func TestProject(t *testing.T) {
	t.Run("Create Project", func(t *testing.T) {
		output, err := projectClient.Create(t.Context(), &v1alpha1.CreateProjectRequest{
			Name:        "test-project",
			DisplayName: "Test Project",
			Description: "This is a test project",
		})
		assert.NoError(t, err)

		assert.Equal(t, "test-project", output.Name)
		assert.Equal(t, "Test Project", output.DisplayName)
		assert.Equal(t, "This is a test project", output.Description)
	})
	t.Run("Get Project", func(t *testing.T) {
		output, err := projectClient.Get(t.Context(), &v1alpha1.GetProjectRequest{
			Name: "test-project",
		})
		assert.NoError(t, err)

		assert.Equal(t, "test-project", output.Name)
		assert.Equal(t, "Test Project", output.DisplayName)
		assert.Equal(t, "This is a test project", output.Description)
	})
	t.Run("List Projects", func(t *testing.T) {
		output, err := projectClient.List(t.Context(), &v1alpha1.ListProjectsRequest{
			Limit: 10,
		})
		assert.NoError(t, err)

		assert.Equal(t, len(output.Items), 1)
		found := false
		for _, project := range output.Items {
			if project.Name == "test-project" {
				found = true
				assert.Equal(t, "Test Project", project.DisplayName)
				assert.Equal(t, "This is a test project", project.Description)
			}
		}

		assert.Equal(t, output.Count, int64(1))
		assert.True(t, found, "Created project not found in list")
	})

	t.Run("Create Duplicate Project", func(t *testing.T) {
		_, err := projectClient.Create(t.Context(), &v1alpha1.CreateProjectRequest{
			Name:        "test-project",
			DisplayName: "Test Project Duplicate",
			Description: "This is a duplicate test project",
		})
		assert.Error(t, err)
	})

	t.Run("Get Non-Existent Project", func(t *testing.T) {
		_, err := projectClient.Get(t.Context(), &v1alpha1.GetProjectRequest{
			Name: "non-existent-project",
		})
		assert.Error(t, err)
	})

	t.Run("List Projects with pageination", func(t *testing.T) {
		// Create additional projects
		for i := 1; i <= 15; i++ {
			_, err := projectClient.Create(t.Context(), &v1alpha1.CreateProjectRequest{
				Name:        "test-project-" + strconv.Itoa(i),
				DisplayName: "Test Project " + strconv.Itoa(i),
				Description: "This is test project number " + strconv.Itoa(i),
			})
			assert.NoError(t, err)
		}

		var allProjects []*v1alpha1.ProjectItem
		var cursor string
		for {
			output, err := projectClient.List(t.Context(), &v1alpha1.ListProjectsRequest{
				Limit:  5,
				Cursor: cursor,
			})
			assert.NoError(t, err)

			allProjects = append(allProjects, output.Items...)
			if output.NextCursor == "" {
				break
			}
			cursor = output.NextCursor
		}

		// check content
		for i := range 16 {
			found := false
			expectedName := "test-project"
			if i > 0 {
				expectedName += "-" + strconv.Itoa(i)
			}
			for _, project := range allProjects {
				if project.Name == expectedName {
					found = true
					break
				}
			}
			assert.True(t, found, "Project %s not found in paginated list", expectedName)
		}

		assert.Equal(t, 16, len(allProjects)) // 15 created + 1 original
	})

	t.Run("List Projects with desc ordering", func(t *testing.T) {
		output, err := projectClient.List(t.Context(), &v1alpha1.ListProjectsRequest{
			Limit:   10,
			OrderBy: []string{"-name"},
		})
		assert.NoError(t, err)

		assert.Equal(t, 10, len(output.Items))
		assert.Equal(t, "test-project-9", output.Items[0].Name)
		assert.Equal(t, "test-project-8", output.Items[1].Name)
	})

	t.Run("List Projects with asc ordering", func(t *testing.T) {
		output, err := projectClient.List(t.Context(), &v1alpha1.ListProjectsRequest{
			Limit:   10,
			OrderBy: []string{"+name"},
		})
		assert.NoError(t, err)

		assert.Equal(t, 10, len(output.Items))
		assert.Equal(t, "test-project", output.Items[0].Name)
		assert.Equal(t, "test-project-1", output.Items[1].Name)
	})
}
