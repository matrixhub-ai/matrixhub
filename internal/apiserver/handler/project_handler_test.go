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

package handler

import (
	"testing"

	projectv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestProjectHandlerCreateProjectRejectsOrganizationWithoutRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	h := &ProjectHandler{projectRepo: projectRepo}
	projectRepo.EXPECT().
		GetProjectByName(gomock.Any(), "proxy-project").
		Return(nil, gorm.ErrRecordNotFound).
		AnyTimes()
	projectRepo.EXPECT().CreateProject(gomock.Any(), gomock.Any()).Times(0)

	_, err := h.CreateProject(t.Context(), &projectv1alpha1.CreateProjectRequest{
		Name:         "proxy-project",
		Type:         projectv1alpha1.ProjectType_PROJECT_TYPE_PUBLIC,
		Organization: "Qwen",
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s (err: %v)", status.Code(err), codes.InvalidArgument, err)
	}
}
