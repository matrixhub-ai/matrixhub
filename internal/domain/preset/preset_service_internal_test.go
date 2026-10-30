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

package preset

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	gitmocks "github.com/matrixhub-ai/matrixhub/internal/domain/git/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	modelmocks "github.com/matrixhub-ai/matrixhub/internal/domain/model/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
)

// Reconcile only logs reconciliation failures, so the returned errors are
// asserted on reconcileModel directly.
func TestReconcileModelReturnsFailedStep(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("dependency unavailable")
	registryID := 3
	proxyProject := &project.Project{RegistryID: &registryID}

	for _, test := range []struct {
		name    string
		setup   func(models *modelmocks.MockIModelRepo, projects *projectmocks.MockIProjectRepo, gitRepo *gitmocks.MockIGitRepo)
		wantErr string
	}{
		{
			name: "deleted model is skipped",
			setup: func(models *modelmocks.MockIModelRepo, _ *projectmocks.MockIProjectRepo, _ *gitmocks.MockIGitRepo) {
				models.EXPECT().GetByProjectAndName(ctx, "project", "model").Return(nil, errors.New("record not found"))
			},
		},
		{
			name: "model lookup",
			setup: func(models *modelmocks.MockIModelRepo, _ *projectmocks.MockIProjectRepo, _ *gitmocks.MockIGitRepo) {
				models.EXPECT().GetByProjectAndName(ctx, "project", "model").Return(nil, failure)
			},
			wantErr: "dependency unavailable",
		},
		{
			name: "project lookup",
			setup: func(models *modelmocks.MockIModelRepo, projects *projectmocks.MockIProjectRepo, _ *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					models.EXPECT().GetByProjectAndName(ctx, "project", "model").Return(&model.Model{}, nil),
					projects.EXPECT().GetProjectByName(ctx, "project").Return(nil, failure),
				)
			},
			wantErr: "dependency unavailable",
		},
		{
			name: "repository check",
			setup: func(models *modelmocks.MockIModelRepo, projects *projectmocks.MockIProjectRepo, gitRepo *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					models.EXPECT().GetByProjectAndName(ctx, "project", "model").Return(&model.Model{}, nil),
					projects.EXPECT().GetProjectByName(ctx, "project").Return(proxyProject, nil),
					gitRepo.EXPECT().RepositoryExists(ctx, "models", "project", "model").Return(false, failure),
				)
			},
			wantErr: "check preset repository: dependency unavailable",
		},
		{
			name: "repository repair",
			setup: func(models *modelmocks.MockIModelRepo, projects *projectmocks.MockIProjectRepo, gitRepo *gitmocks.MockIGitRepo) {
				gomock.InOrder(
					models.EXPECT().GetByProjectAndName(ctx, "project", "model").Return(&model.Model{}, nil),
					projects.EXPECT().GetProjectByName(ctx, "project").Return(proxyProject, nil),
					gitRepo.EXPECT().RepositoryExists(ctx, "models", "project", "model").Return(false, nil),
					gitRepo.EXPECT().CreateRepository(ctx, "models", "project", "model").Return(failure),
				)
			},
			wantErr: "repair preset repository: dependency unavailable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			models := modelmocks.NewMockIModelRepo(ctrl)
			projects := projectmocks.NewMockIProjectRepo(ctrl)
			gitRepo := gitmocks.NewMockIGitRepo(ctrl)
			test.setup(models, projects, gitRepo)
			service := NewService(true, &Manifest{}, nil, projects, models, gitRepo)

			err := service.reconcileModel(ctx, ModelSpec{Project: "project", Name: "model"})

			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, failure)
			require.EqualError(t, err, test.wantErr)
		})
	}
}
