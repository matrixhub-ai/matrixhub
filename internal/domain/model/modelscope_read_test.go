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

package model_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	registrymocks "github.com/matrixhub-ai/matrixhub/internal/domain/registry/mocks"
)

func TestPrepareModelScopeReadSkipsOtherProviders(t *testing.T) {
	id := 1
	for _, tc := range []struct {
		name string
		prj  *project.Project
		err  error
	}{
		{name: "missing project", err: errors.New("project not found")},
		{name: "local project", prj: &project.Project{Name: "local"}},
		{name: "HF proxy", prj: &project.Project{RegistryID: &id, Organization: "team", RegistryType: "REGISTRY_TYPE_HUGGINGFACE"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			projects := projectmocks.NewMockIProjectRepo(ctrl)
			projects.EXPECT().GetProjectByName(t.Context(), "proj").Return(tc.prj, tc.err)
			// Nil dependencies ensure other providers never initiate an MS fetch.
			service := model.NewModelService(nil, nil, nil, projects, nil)
			remote, err := service.PrepareModelScopeRead(t.Context(), "proj", "demo")
			require.ErrorIs(t, err, tc.err)
			require.Nil(t, remote)
		})
	}
}

func TestPrepareModelScopeReadPropagatesUpstreamFailures(t *testing.T) {
	for _, stage := range []string{"registry lookup", "Git refresh"} {
		t.Run(stage, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			projects := projectmocks.NewMockIProjectRepo(ctrl)
			registries := registrymocks.NewMockIRegistryRepo(ctrl)
			gitRepo := gitmocks.NewMockIGitRepo(ctrl)
			id := 1
			projects.EXPECT().GetProjectByName(t.Context(), "proj").Return(&project.Project{
				Name: "proj", RegistryID: &id, Organization: "team", RegistryType: registry.TypeModelScope,
			}, nil)
			wantErr := errors.New("upstream unavailable")
			if stage == "registry lookup" {
				registries.EXPECT().GetRegistry(t.Context(), id).Return(nil, wantErr)
			} else {
				registries.EXPECT().GetRegistry(t.Context(), id).Return(&registry.Registry{Type: registry.TypeModelScope}, nil)
				gitRepo.EXPECT().RefreshModelScopeRefs(t.Context(), gomock.Any()).Return(false, wantErr)
			}
			service := model.NewModelService(nil, nil, gitRepo, projects, registries)
			remote, err := service.PrepareModelScopeRead(t.Context(), "proj", "demo")
			require.ErrorIs(t, err, wantErr)
			require.Nil(t, remote)
		})
	}
}

func TestPrepareModelScopeReadCreatesRecordsWithoutPullingWeights(t *testing.T) {
	for _, mode := range []string{"existing", "first reader", "concurrent insert", "insert failure"} {
		t.Run(mode, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			projects := projectmocks.NewMockIProjectRepo(ctrl)
			registries := registrymocks.NewMockIRegistryRepo(ctrl)
			gitRepo := gitmocks.NewMockIGitRepo(ctrl)
			models := modelmocks.NewMockIModelRepo(ctrl)
			id := 1
			projects.EXPECT().GetProjectByName(t.Context(), "proj").Return(&project.Project{
				Name: "proj", RegistryID: &id, Organization: "team", RegistryType: registry.TypeModelScope,
			}, nil)
			reg := &registry.Registry{Type: registry.TypeModelScope, URL: "https://modelscope.example", Insecure: true}
			reg.SetCredential(registry.NewBasicCredential("alice", "sdk-token"))
			registries.EXPECT().GetRegistry(t.Context(), id).Return(reg, nil)
			wantRemote := &git.GitRepository{
				RemoteRegistryURL: reg.URL, RemoteRegistryType: registry.TypeModelScope, Insecure: true,
				RemoteProjectName: "team", RemoteResourceName: "demo", ProjectName: "proj", ResourceName: "demo", ResourceType: "model",
				Credential: &git.BasicCredential{Username: "alice", Password: "sdk-token"},
			}
			gitRepo.EXPECT().RefreshModelScopeRefs(t.Context(), wantRemote).Return(false, nil)
			mod := &model.Model{ID: 42, ProjectName: "proj", Name: "demo"}
			insertErr := errors.New("insert failed")
			if mode == "existing" {
				models.EXPECT().GetByProjectAndName(t.Context(), "proj", "demo").Return(mod, nil)
			} else {
				lookup := models.EXPECT().GetByProjectAndName(t.Context(), "proj", "demo").Return(nil, errors.New("model not found"))
				gitRepo.EXPECT().RepositoryExists(t.Context(), "models", "proj", "demo").Return(true, nil)
				create := models.EXPECT().Create(t.Context(), &model.Model{ProjectName: "proj", Name: "demo"})
				if mode == "first reader" {
					create.Return(mod, nil)
				} else {
					create.Return(nil, insertErr)
					recheck := models.EXPECT().GetByProjectAndName(t.Context(), "proj", "demo").After(lookup).After(create)
					if mode == "concurrent insert" {
						recheck.Return(mod, nil)
					} else {
						recheck.Return(nil, errors.New("model not found"))
					}
				}
			}
			service := model.NewModelService(models, nil, gitRepo, projects, registries)
			remote, err := service.PrepareModelScopeRead(t.Context(), "proj", "demo")
			if mode == "insert failure" {
				require.ErrorIs(t, err, insertErr)
				require.Nil(t, remote)
			} else {
				require.NoError(t, err)
				require.Equal(t, wantRemote, remote)
			}
			// No PullFromRemote or UpdateSyncedAt expectation: serving metadata
			// must neither download weights nor mark a complete weight sync.
		})
	}
}

func TestPrepareModelScopeReadServesTreeWhenModelCardRefreshFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	projects := projectmocks.NewMockIProjectRepo(ctrl)
	registries := registrymocks.NewMockIRegistryRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	models := modelmocks.NewMockIModelRepo(ctrl)
	id := 1
	projects.EXPECT().GetProjectByName(t.Context(), "proj").Return(&project.Project{
		Name: "proj", RegistryID: &id, Organization: "team", RegistryType: registry.TypeModelScope,
	}, nil)
	registries.EXPECT().GetRegistry(t.Context(), id).Return(&registry.Registry{Type: registry.TypeModelScope}, nil)
	gitRepo.EXPECT().RefreshModelScopeRefs(t.Context(), gomock.Any()).Return(true, nil)
	models.EXPECT().GetByProjectAndName(t.Context(), "proj", "demo").Return(&model.Model{ID: 42}, nil).Times(2)
	gitRepo.EXPECT().ExtractMetadata(t.Context(), "models", "proj", "demo").Return(nil, errors.New("invalid model card"))
	service := model.NewModelService(models, nil, gitRepo, projects, registries)
	remote, err := service.PrepareModelScopeRead(t.Context(), "proj", "demo")
	require.NoError(t, err)
	require.NotNil(t, remote)
}
