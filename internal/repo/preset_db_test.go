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
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/preset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

func TestPresetDBSeedIfEmpty(t *testing.T) {
	database, _ := newSQLiteRepositoryTestDatabase(t)
	ctx := context.Background()
	manifest := preset.DefaultManifest()
	require.Len(t, manifest.Projects, 4)
	require.Len(t, manifest.Models, 13)
	repository := NewPresetDB(database)

	applied, err := repository.SeedIfEmpty(ctx, manifest)
	require.NoError(t, err)
	require.True(t, applied)

	registries, total, err := NewRegistryRepo(database).ListRegistries(ctx, 1, 20, "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	for i, spec := range manifest.Registries {
		got := registries[i]
		require.Equal(t, spec.ID, got.ID)
		require.Equal(t, spec.Name, got.Name)
		require.Equal(t, spec.URL, got.URL)
		require.Equal(t, spec.Description, got.Description)
		require.Equal(t, "REGISTRY_TYPE_HUGGINGFACE", got.Type)
		require.False(t, got.Insecure)
		require.Empty(t, got.AuthInfo)
	}

	projectRepo := NewProjectDBRepo(database)
	for _, spec := range manifest.Projects {
		prj, err := projectRepo.GetProjectByName(ctx, spec.Name)
		require.NoError(t, err)
		require.Equal(t, project.ProjectTypePublic, prj.Type)
		require.Equal(t, spec.Organization, prj.Organization)
		require.NotNil(t, prj.RegistryID)
		require.Equal(t, 3, *prj.RegistryID)
		require.Equal(t, "https://hf.m.daocloud.io", prj.RegistryURL)
		members, count, err := projectRepo.ListProjectMembers(ctx, prj.ID, "", 1, 20)
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
		require.Equal(t, "admin", members[0].MemberName)
		require.Equal(t, role.ProjectRoleAdmin, members[0].RoleID)
		require.Equal(t, project.MemberTypeUser, members[0].MemberType)
	}

	modelRepo := NewModelDB(database)
	popular := true
	models, total, err := modelRepo.List(ctx, &model.Filter{Page: 1, PageSize: 20, Popular: &popular})
	require.NoError(t, err)
	require.Equal(t, int64(13), total)
	require.Len(t, models, 13)
	for _, spec := range manifest.Models {
		mod, err := modelRepo.GetByProjectAndName(ctx, spec.Project, spec.Name)
		require.NoError(t, err)
		require.True(t, mod.IsPopular)
		require.Nil(t, mod.SyncedAt)
		require.Equal(t, "main", mod.DefaultBranch)
		require.Empty(t, mod.ReadmeContent)
		require.Positive(t, spec.Size)
		require.Equal(t, spec.Size, mod.Size)
		require.Positive(t, spec.ParameterCount)
		require.Equal(t, spec.ParameterCount, mod.ParameterCount)
		require.NotEmpty(t, spec.Labels)
		labels := make([]preset.LabelSpec, 0, len(mod.Labels))
		for _, label := range mod.Labels {
			require.Equal(t, "model", label.Scope)
			labels = append(labels, preset.LabelSpec{Category: label.Category, Name: label.Name})
		}
		require.ElementsMatch(t, spec.Labels, labels)
	}
	var labelsBefore []model.Label
	require.NoError(t, database.Find(&labelsBefore).Error)
	require.Len(t, labelsBefore, 50)
	var associationsBefore int64
	require.NoError(t, database.Table("models_labels").Count(&associationsBefore).Error)
	require.EqualValues(t, 96, associationsBefore)

	applied, err = repository.SeedIfEmpty(ctx, manifest)
	require.NoError(t, err)
	require.False(t, applied)
	requirePresetCounts(t, database, 3, 4, 4, 13)
	var labelsAfter []model.Label
	require.NoError(t, database.Find(&labelsAfter).Error)
	require.Equal(t, labelsBefore, labelsAfter)
	var associationsAfter int64
	require.NoError(t, database.Table("models_labels").Count(&associationsAfter).Error)
	require.Equal(t, associationsBefore, associationsAfter)
}

func TestPresetDBReusesExistingLabels(t *testing.T) {
	for _, test := range []struct {
		name     string
		category string
		scope    string
	}{
		{name: "transformers", category: "library", scope: "model"},
		{name: "TRANSFORMERS", category: "LIBRARY", scope: "MODEL"},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, _ := newSQLiteRepositoryTestDatabase(t)
			ctx := context.Background()
			existing := &model.Label{Name: test.name, Category: test.category, Scope: test.scope}
			require.NoError(t, database.Create(existing).Error)
			require.NoError(t, database.First(existing, existing.ID).Error)
			for _, label := range []model.Label{
				{Name: "transformers", Category: "library", Scope: "dataset"},
				{Name: "transformers", Category: "other", Scope: "model"},
			} {
				require.NoError(t, database.Create(&label).Error)
			}

			manifest := preset.DefaultManifest()
			applied, err := NewPresetDB(database).SeedIfEmpty(ctx, manifest)
			require.NoError(t, err)
			require.True(t, applied)
			uniqueLabels := make(map[preset.LabelSpec]struct{})
			for _, spec := range manifest.Models {
				mod, err := NewModelDB(database).GetByProjectAndName(ctx, spec.Project, spec.Name)
				require.NoError(t, err)
				for _, label := range spec.Labels {
					uniqueLabels[label] = struct{}{}
					if label.Category == "library" && label.Name == "transformers" {
						require.Contains(t, mod.Labels, *existing)
					}
				}
			}
			var labels int64
			require.NoError(t, database.Model(&model.Label{}).Count(&labels).Error)
			require.EqualValues(t, len(uniqueLabels)+2, labels)
			var reused model.Label
			require.NoError(t, database.First(&reused, existing.ID).Error)
			require.Equal(t, *existing, reused)
		})
	}
}

func TestPresetDBSkipsNonEmptyInstance(t *testing.T) {
	for _, table := range []string{"registries", "projects"} {
		t.Run(table, func(t *testing.T) {
			database, _ := newSQLiteRepositoryTestDatabase(t)
			var registries, projects int64
			if table == "registries" {
				require.NoError(t, database.Create(&registry.Registry{Name: "existing"}).Error)
				registries = 1
			} else {
				require.NoError(t, database.Create(&project.Project{Name: "existing"}).Error)
				projects = 1
			}
			applied, err := NewPresetDB(database).SeedIfEmpty(context.Background(), preset.DefaultManifest())
			require.NoError(t, err)
			require.False(t, applied)
			requirePresetCounts(t, database, registries, projects, 0, 0)
		})
	}
}

func TestPresetDBRollsBackPrimaryKeyConflict(t *testing.T) {
	database, _ := newSQLiteRepositoryTestDatabase(t)
	manifest := preset.DefaultManifest()
	manifest.Registries[1].ID = manifest.Registries[0].ID
	applied, err := NewPresetDB(database).SeedIfEmpty(context.Background(), manifest)
	require.NoError(t, err)
	require.False(t, applied)
	requirePresetCounts(t, database, 0, 0, 0, 0)
}

func TestPresetDBRollsBackOtherInsertErrors(t *testing.T) {
	for _, table := range []string{"registries", "projects", "members_roles_projects", "models", "labels", "models_labels"} {
		t.Run(table, func(t *testing.T) {
			database, _ := newSQLiteRepositoryTestDatabase(t)
			require.NoError(t, database.Exec("CREATE TRIGGER fail_preset_insert BEFORE INSERT ON "+table+
				" BEGIN SELECT RAISE(ABORT, 'injected insert failure'); END").Error)
			applied, err := NewPresetDB(database).SeedIfEmpty(context.Background(), preset.DefaultManifest())
			require.ErrorContains(t, err, "injected insert failure")
			require.False(t, applied)
			requirePresetCounts(t, database, 0, 0, 0, 0)
			for _, labelTable := range []string{"labels", "models_labels"} {
				var count int64
				require.NoError(t, database.Table(labelTable).Count(&count).Error)
				require.Zero(t, count, labelTable)
			}
		})
	}
}

func TestPresetDBSeedsWithoutAdmin(t *testing.T) {
	database, _ := newSQLiteRepositoryTestDatabase(t)
	require.NoError(t, database.Exec("DELETE FROM members_roles_projects").Error)
	require.NoError(t, database.Exec("DELETE FROM users WHERE username = ?", "admin").Error)
	applied, err := NewPresetDB(database).SeedIfEmpty(context.Background(), preset.DefaultManifest())
	require.NoError(t, err)
	require.True(t, applied)
	requirePresetCounts(t, database, 3, 4, 0, 13)
}

func TestPresetDBPreservesRegistrySequence(t *testing.T) {
	for _, initialID := range []int{0, 100} {
		t.Run(strconv.Itoa(initialID), func(t *testing.T) {
			database, _ := newSQLiteRepositoryTestDatabase(t)
			ctx := context.Background()
			if initialID > 0 {
				reg := &registry.Registry{ID: initialID, Name: "deleted"}
				require.NoError(t, database.Create(reg).Error)
				require.NoError(t, database.Delete(reg).Error)
			}
			applied, err := NewPresetDB(database).SeedIfEmpty(ctx, preset.DefaultManifest())
			require.NoError(t, err)
			require.True(t, applied)
			next, err := NewRegistryRepo(database).CreateRegistry(ctx, registry.Registry{Name: "next"})
			require.NoError(t, err)
			require.Equal(t, max(3, initialID)+1, next.ID)
		})
	}
}

func requirePresetCounts(t *testing.T, database *gorm.DB, registries, projects, memberships, models int64) {
	t.Helper()
	for table, want := range map[string]int64{
		"registries": registries, "projects": projects, "members_roles_projects": memberships, "models": models,
	} {
		var count int64
		query := database.Table(table)
		if table == "members_roles_projects" {
			query = query.Where("project_id IS NOT NULL")
		}
		require.NoError(t, query.Count(&count).Error)
		require.Equal(t, want, count, table)
	}
}
