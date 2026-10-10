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

package apiserver

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/preset"
	presetmocks "github.com/matrixhub-ai/matrixhub/internal/domain/preset/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
)

func TestInitGitStorageDoesNotRegisterMetadataPostReceiveHook(t *testing.T) {
	server := &APIServer{
		config: &config.Config{DataDir: t.TempDir()},
	}
	server.initMirrorHooks()
	server.initGitStorage()

	postReceiveHook := reflect.ValueOf(server.gitStorage.sharedMirror).
		Elem().FieldByName("postReceiveHookFunc")
	if !postReceiveHook.IsNil() {
		t.Fatal("mirror must not register the metadata post-receive hook")
	}
}

func TestInitPresetsWaitsForRepositoryReconciliation(t *testing.T) {
	ctrl := gomock.NewController(t)
	presetRepo := presetmocks.NewMockIPresetRepo(ctrl)
	modelRepo := modelmocks.NewMockIModelRepo(ctrl)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	gitRepo := gitmocks.NewMockIGitRepo(ctrl)
	manifest := preset.DefaultManifest()
	modelCount := int32(len(manifest.Models))
	registryID := 3

	seed := presetRepo.EXPECT().SeedIfEmpty(gomock.Any(), manifest).Return(true, nil)
	modelRepo.EXPECT().GetByProjectAndName(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&model.Model{}, nil).Times(len(manifest.Models)).After(seed)
	projectRepo.EXPECT().GetProjectByName(gomock.Any(), gomock.Any()).
		Return(&project.Project{RegistryID: &registryID}, nil).Times(len(manifest.Models))
	gitRepo.EXPECT().RepositoryExists(gomock.Any(), "models", gomock.Any(), gomock.Any()).
		Return(false, nil).Times(len(manifest.Models))

	started := make(chan struct{}, len(manifest.Models))
	release := make(chan struct{})
	reconciled := make(chan struct{})
	var created atomic.Int32
	gitRepo.EXPECT().CreateRepository(gomock.Any(), "models", gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string, string, string) error {
			started <- struct{}{}
			<-release
			if created.Add(1) == modelCount {
				close(reconciled)
			}
			return nil
		}).Times(len(manifest.Models))

	server := &APIServer{
		config: &config.Config{},
		repos: &repo.Repos{
			Preset: presetRepo, Project: projectRepo, Model: modelRepo, Git: gitRepo,
		},
	}
	initialized := make(chan struct{})
	go func() {
		server.initPresets()
		close(initialized)
	}()

	select {
	case <-started:
		select {
		case <-initialized:
			t.Error("preset initialization returned before repository reconciliation completed")
		case <-time.After(100 * time.Millisecond):
		}
	case <-time.After(5 * time.Second):
		t.Error("repository reconciliation did not start")
	}
	close(release)

	select {
	case <-initialized:
		if count := created.Load(); count != modelCount {
			t.Errorf("preset initialization returned after creating %d of %d repositories", count, modelCount)
		}
	case <-time.After(5 * time.Second):
		t.Error("preset initialization did not return after repository reconciliation")
	}
	select {
	case <-reconciled:
	case <-time.After(5 * time.Second):
		t.Fatal("repository reconciliation did not complete")
	}
}
