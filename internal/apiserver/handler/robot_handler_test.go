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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	projectmocks "github.com/matrixhub-ai/matrixhub/internal/domain/project/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	robotmocks "github.com/matrixhub-ai/matrixhub/internal/domain/robot/mocks"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

func robotUserContext() context.Context {
	return auth.WithIdentity(context.Background(), user.NewUserIdentity(42, "alice"))
}

func TestRobotHandlerCreateRobotAccount(t *testing.T) {
	ctx := robotUserContext()
	request := &v1alpha1.CreateRobotAccountRequest{
		Name:                "ci",
		Description:         "build robot",
		ExpireDays:          7,
		PlatformPermissions: []string{role.UserCreate.String()},
		ProjectPermissions:  []string{role.ModelPull.String()},
		Projects:            []string{"demo", "demo"},
		ProjectScope:        v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED,
	}

	t.Run("creates a selected-project robot", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		robotRepo := robotmocks.NewMockIRobotRepo(ctrl)
		projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
		robotRepo.EXPECT().GetRobotByName(ctx, "robot$ci").Return(nil, gorm.ErrRecordNotFound)
		projectRepo.EXPECT().ListProjectInfoByNames(ctx, []string{"demo"}).Return([]*project.Project{{ID: 1, Name: "demo"}}, nil)
		robotRepo.EXPECT().CreateRobot(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, rb *robot.Robot) error {
			require.Equal(t, "robot$ci", rb.Name)
			require.Equal(t, 42, rb.CreateBy)
			require.Equal(t, robot.ProjectScopeSelected, rb.ProjectScope)
			require.Equal(t, datatypes.JSONSlice[role.Permission]{role.UserCreate}, rb.PlatformPermissions)
			require.Equal(t, datatypes.JSONSlice[role.Permission]{role.ModelPull}, rb.ProjectPermissions)
			require.Len(t, rb.Projects, 1)
			require.NotEmpty(t, rb.TokenHash)
			require.NotNil(t, rb.ExpireAt)
			return nil
		})

		response, err := (&RobotHandler{robotRepo: robotRepo, projectRepo: projectRepo}).CreateRobotAccount(ctx, request)
		require.NoError(t, err)
		require.NotEmpty(t, response.Token)
	})

	t.Run("rejects a duplicate name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		robotRepo := robotmocks.NewMockIRobotRepo(ctrl)
		projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
		robotRepo.EXPECT().GetRobotByName(ctx, "robot$ci").Return(&robot.Robot{ID: 1}, nil)

		response, err := (&RobotHandler{robotRepo: robotRepo, projectRepo: projectRepo}).CreateRobotAccount(ctx, request)
		require.Nil(t, response)
		require.Equal(t, codes.AlreadyExists, status.Code(err))
	})

	t.Run("rejects an unknown project", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		robotRepo := robotmocks.NewMockIRobotRepo(ctrl)
		projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
		robotRepo.EXPECT().GetRobotByName(ctx, "robot$ci").Return(nil, gorm.ErrRecordNotFound)
		projectRepo.EXPECT().ListProjectInfoByNames(ctx, []string{"demo"}).Return(nil, nil)

		response, err := (&RobotHandler{robotRepo: robotRepo, projectRepo: projectRepo}).CreateRobotAccount(ctx, request)
		require.Nil(t, response)
		require.Equal(t, codes.Internal, status.Code(err))
	})
}

func TestRobotHandlerUpdateRobotAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := robotmocks.NewMockIRobotRepo(ctrl)
	projectRepo := projectmocks.NewMockIProjectRepo(ctrl)
	ctx := context.Background()
	createdAt := time.Now().Add(-24 * time.Hour)
	existing := &robot.Robot{ID: 7, Enabled: true, Duration: 1, CreatedAt: createdAt}
	repo.EXPECT().GetRobot(ctx, 7).Return(existing, nil)
	repo.EXPECT().UpdateRobot(ctx, existing).DoAndReturn(func(_ context.Context, rb *robot.Robot) error {
		require.True(t, rb.Enabled, "unspecified status must preserve the current value")
		require.Equal(t, 3, rb.Duration)
		require.Equal(t, createdAt.AddDate(0, 0, 3), *rb.ExpireAt)
		require.Equal(t, robot.ProjectScopeAll, rb.ProjectScope)
		require.Nil(t, rb.Projects)
		return nil
	})

	response, err := (&RobotHandler{robotRepo: repo, projectRepo: projectRepo}).UpdateRobotAccount(ctx, &v1alpha1.UpdateRobotAccountRequest{
		Id:           7,
		ExpireDays:   3,
		ProjectScope: v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_ALL,
	})
	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestRobotHandlerRefreshRobotAccountToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := robotmocks.NewMockIRobotRepo(ctrl)
	ctx := context.Background()
	existing := &robot.Robot{ID: 7, TokenHash: "old"}
	repo.EXPECT().GetRobot(ctx, 7).Return(existing, nil)
	repo.EXPECT().UpdateRobot(ctx, existing).DoAndReturn(func(_ context.Context, rb *robot.Robot) error {
		require.Equal(t, utils.Sha256Hex("Abcdefg1"), rb.TokenHash)
		return nil
	})

	response, err := (&RobotHandler{robotRepo: repo}).RefreshRobotAccountToken(ctx, &v1alpha1.RefreshRobotAccountTokenRequest{
		Id: 7, Token: "Abcdefg1",
	})
	require.NoError(t, err)
	require.Equal(t, "Abcdefg1", response.Token)
}

func TestRobotHandlerListAndGetRobotAccounts(t *testing.T) {
	ctx := context.Background()
	createdAt := time.Unix(1_700_000_000, 0)
	expiresAt := time.Now().Add(time.Hour)
	rb := &robot.Robot{
		ID:           7,
		Name:         "robot$ci",
		Enabled:      true,
		Duration:     1,
		ExpireAt:     &expiresAt,
		ProjectScope: robot.ProjectScopeSelected,
		Projects:     []*project.Project{{Name: "demo"}},
		CreatedAt:    createdAt,
	}

	t.Run("lists robots", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := robotmocks.NewMockIRobotRepo(ctrl)
		repo.EXPECT().ListSystemRobots(ctx, 1, 10, "ci").Return([]*robot.Robot{rb}, int64(1), nil)

		response, err := (&RobotHandler{robotRepo: repo}).ListRobotAccounts(ctx, &v1alpha1.ListRobotAccountsRequest{Page: 1, PageSize: 10, Search: "ci"})
		require.NoError(t, err)
		require.Len(t, response.Items, 1)
		require.Equal(t, "robot$ci", response.Items[0].Name)
		require.Equal(t, v1alpha1.RobotAccountStatus_ROBOT_ACCOUNT_STATUS_ENABLED, response.Items[0].Status)
		require.Equal(t, v1alpha1.RobotAccountExpireStatus_ROBOT_ACCOUNT_EXPIRE_STATUS_VALID, response.Items[0].ExpireStatus)
		require.Equal(t, []string{"demo"}, response.Items[0].Projects)
	})

	t.Run("maps not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := robotmocks.NewMockIRobotRepo(ctrl)
		repo.EXPECT().GetRobot(ctx, 99).Return(nil, gorm.ErrRecordNotFound)

		response, err := (&RobotHandler{robotRepo: repo}).GetRobotAccount(ctx, &v1alpha1.GetRobotAccountRequest{Id: 99})
		require.Nil(t, response)
		require.Equal(t, codes.NotFound, status.Code(err))
	})
}

func TestRobotHandlerDeleteRobotAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := robotmocks.NewMockIRobotRepo(ctrl)
	ctx := context.Background()
	wantErr := errors.New("database unavailable")
	repo.EXPECT().DeleteRobot(ctx, 7).Return(wantErr)

	response, err := (&RobotHandler{robotRepo: repo}).DeleteRobotAccount(ctx, &v1alpha1.DeleteRobotAccountRequest{Id: 7})
	require.Nil(t, response)
	require.Equal(t, codes.Internal, status.Code(err))
}
