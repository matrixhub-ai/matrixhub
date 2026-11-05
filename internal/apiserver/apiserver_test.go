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
	"net/http"
	"net/http/httptest"
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

func TestAPIFirstDispatchesOnlyAPIServerPathsToGin(t *testing.T) {
	tests := []struct {
		path     string
		staticUI bool
		want     string
	}{
		{"/healthz", true, "api"},
		{"/healthzx", true, "protocol"},
		{"/api/v1alpha1/models", true, "api"},
		{"/apis/v1alpha1/x", true, "api"},
		{"/api/models/a/b", true, "protocol"},
		{"/api/datasets/a/b/revision/main", true, "protocol"},
		{"/api/whoami-v2", true, "protocol"},
		{"/api/repos/create", true, "protocol"},
		{"/api/settings/x", true, "protocol"},
		{"/assets/app.js", true, "api"},
		{"/assets/app.js", false, "protocol"},
		{"/favicon.ico", true, "api"},
		{"/favicon.ico", false, "protocol"},
		{"/proj/repo.git/info/refs", true, "protocol"},
		{"/", true, "protocol"},
		// Projects named api, apis or assets are shadowed under these prefixes, like v1/v2 are by the CAS routes.
		{"/api/v1alpha1/resolve/main/README.md", true, "api"},
		{"/apis/v1alpha1/info/refs", true, "api"},
		{"/assets/repo/resolve/main/README.md", true, "api"},
		{"/api/v1alpha1.git/info/refs", true, "protocol"},
	}
	for _, test := range tests {
		name := test.path
		if !test.staticUI {
			name += " without static dir"
		}
		t.Run(name, func(t *testing.T) {
			var served string
			record := func(name string) http.Handler {
				return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { served = name })
			}
			handler := apiFirst(record("api"), record("protocol"), apiServerPath(test.staticUI))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, test.path, nil))

			if served != test.want {
				t.Fatalf("%s served by %q, want %q", test.path, served, test.want)
			}
		})
	}
}

func TestInitPresetsSkipsDisabledPresets(t *testing.T) {
	ctrl := gomock.NewController(t)
	disabled := false
	server := &APIServer{
		config: &config.Config{Presets: config.PresetsConfig{Enabled: &disabled}},
		repos: &repo.Repos{
			Preset:  presetmocks.NewMockIPresetRepo(ctrl),
			Project: projectmocks.NewMockIProjectRepo(ctrl),
			Model:   modelmocks.NewMockIModelRepo(ctrl),
			Git:     gitmocks.NewMockIGitRepo(ctrl),
		},
	}

	// No expectations are registered, so any repository access fails the test.
	server.initPresets()
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
