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

//go:build cgo

package repo

import (
	"context"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/mirror"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
)

// NewRepos opens the configured database, so this test relies on the cgo
// SQLite driver, like the SQLite contract tests.
func TestNewReposWiresRepositoriesToConfiguredStores(t *testing.T) {
	ctx := context.Background()
	database, conf := newSQLiteRepositoryTestDatabase(t)
	gitStorage := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	gitMirror := mirror.NewMirror(mirror.WithCacheDir(t.TempDir()))

	repos := NewRepos(conf, gitStorage, gitMirror)
	t.Cleanup(func() {
		require.NoError(t, repos.Close())
	})

	require.NotNil(t, repos.DB)
	require.Same(t, gitStorage, repos.GitStorage)
	require.Same(t, gitMirror, repos.GitMirror)
	require.Equal(t, &gitRepo{storage: gitStorage, mirror: gitMirror}, repos.Git)
	require.Same(t, repos.DB, repos.Model.(*modelDB).db)
	require.Same(t, repos.DB, repos.Label.(*labelDB).db)
	require.Same(t, repos.DB, repos.Dataset.(*datasetDB).db)
	require.Same(t, repos.DB, repos.Preset.(*presetDB).db)
	for name, repository := range map[string]any{
		"Project": repos.Project, "User": repos.User, "Session": repos.Session, "AccessToken": repos.AccessToken,
		"SSHKey": repos.SSHKey, "Registry": repos.Registry, "SyncPolicy": repos.SyncPolicy, "SyncTask": repos.SyncTask,
		"SyncJob": repos.SyncJob, "Authz": repos.Authz, "Robot": repos.Robot,
	} {
		require.NotNil(t, repository, name)
	}

	// Writes through the repositories land in the configured database.
	require.NoError(t, database.Create(&project.Project{Name: "acme", Type: project.ProjectTypePublic}).Error)
	created, err := repos.Dataset.Create(ctx, &dataset.Dataset{ProjectName: "acme", Name: "corpus"})
	require.NoError(t, err)
	stored, err := NewDatasetDB(database).GetByProjectAndName(ctx, "acme", "corpus")
	require.NoError(t, err)
	require.Equal(t, created.ID, stored.ID)
}
