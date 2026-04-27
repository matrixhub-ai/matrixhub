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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type RobotHandler struct {
	robotRepo   robot.IRobotRepo
	projectRepo project.IProjectRepo
}

func (r *RobotHandler) CreateRobotAccount(ctx context.Context, request *v1alpha1.CreateRobotAccountRequest) (*v1alpha1.CreateRobotAccountResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	robotName := robot.RobotPrefix + request.Name
	_, err := r.robotRepo.GetRobotByName(ctx, robotName)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "robot name already exists")
	}
	platformPermissions := role.StringsToPermissions(request.PlatformPermissions)
	projectPermissions := role.StringsToPermissions(request.ProjectPermissions)
	if err = role.PlatformPermissions.CheckPermissions(platformPermissions); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err = role.ProjectPermissions.CheckPermissions(projectPermissions); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	token, hash, err := utils.GenerateRobotToken()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	scope := robot.ProjectScopeSelected
	if request.ProjectScope == v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_ALL {
		scope = robot.ProjectScopeAll
	}
	projects, err := r.checkProjects(ctx, request.ProjectScope, request.Projects)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rb := &robot.Robot{
		Name:                robotName,
		Description:         request.Description,
		Duration:            int(request.ExpireDays),
		PlatformPermissions: platformPermissions,
		ProjectPermissions:  projectPermissions,
		Enabled:             true,
		TokenHash:           hash,
		Projects:            projects,
		ProjectScope:        scope,
		CreateBy:            user.GetCurrentUserId(ctx),
	}
	if request.ExpireDays > 0 {
		expireAt := time.Now().AddDate(0, 0, int(request.ExpireDays))
		rb.ExpireAt = &expireAt
	}
	if err = r.robotRepo.CreateRobot(ctx, rb); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("robot %s already exists", rb.Name))
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.CreateRobotAccountResponse{
		Token: token,
	}, nil
}

func (r *RobotHandler) ListRobotAccounts(ctx context.Context, request *v1alpha1.ListRobotAccountsRequest) (*v1alpha1.ListRobotAccountsResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	page := utils.NewPage(request.Page, request.PageSize)
	robots, total, err := r.robotRepo.ListSystemRobots(ctx, int(page.Page), int(page.PageSize), request.Search)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1alpha1.ListRobotAccountsResponse{
		Items: lo.Map(robots, func(item *robot.Robot, index int) *v1alpha1.GetRobotAccountResponse {
			return r.transferRobot(item)
		}),
		Pagination: &v1alpha1.Pagination{
			Total:    int32(total),
			Page:     page.Page,
			PageSize: page.PageSize,
			Pages:    utils.CalculatePages(total, request.PageSize),
		},
	}, nil
}

func (r *RobotHandler) transferRobot(item *robot.Robot) *v1alpha1.GetRobotAccountResponse {
	status := v1alpha1.RobotAccountStatus_ROBOT_ACCOUNT_STATUS_DISABLED
	if item.Enabled {
		status = v1alpha1.RobotAccountStatus_ROBOT_ACCOUNT_STATUS_ENABLED
	}
	expireStatus := v1alpha1.RobotAccountExpireStatus_ROBOT_ACCOUNT_EXPIRE_STATUS_EXPIRED
	remainPeriod := ""
	if item.Duration == 0 {
		expireStatus = v1alpha1.RobotAccountExpireStatus_ROBOT_ACCOUNT_EXPIRE_STATUS_NEVER
	} else if item.ExpireAt != nil && item.ExpireAt.After(time.Now()) {
		expireStatus = v1alpha1.RobotAccountExpireStatus_ROBOT_ACCOUNT_EXPIRE_STATUS_VALID
		remainPeriod = utils.FormatDuration(time.Until(*item.ExpireAt))
	}
	scope := v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_SELECTED
	if item.ProjectScope == robot.ProjectScopeAll {
		scope = v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_ALL
	}

	return &v1alpha1.GetRobotAccountResponse{
		Id:                  uint32(item.ID),
		Name:                item.Name,
		Description:         item.Description,
		Status:              status,
		PlatformPermissions: role.PermissionsToStrings(item.PlatformPermissions),
		ProjectPermissions:  role.PermissionsToStrings(item.ProjectPermissions),
		//Projects:            ,
		CreatedAt:    strconv.Itoa(int(item.CreatedAt.Unix())),
		ExpireStatus: expireStatus,
		RemainPeriod: remainPeriod,
		ExpireDays:   int32(item.Duration),
		ProjectScope: scope,
	}
}

func (r *RobotHandler) GetRobotAccount(ctx context.Context, request *v1alpha1.GetRobotAccountRequest) (*v1alpha1.GetRobotAccountResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	robot, err := r.robotRepo.GetRobot(ctx, int(request.Id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("robot %d not found", request.Id))
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return r.transferRobot(robot), nil
}

func (r *RobotHandler) DeleteRobotAccount(ctx context.Context, request *v1alpha1.DeleteRobotAccountRequest) (*v1alpha1.DeleteRobotAccountResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := r.robotRepo.DeleteRobot(ctx, int(request.Id)); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.DeleteRobotAccountResponse{}, nil
}

func (r *RobotHandler) UpdateRobotAccount(ctx context.Context, request *v1alpha1.UpdateRobotAccountRequest) (*v1alpha1.UpdateRobotAccountResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	platformPermissions := role.StringsToPermissions(request.PlatformPermissions)
	projectPermissions := role.StringsToPermissions(request.ProjectPermissions)
	if err := role.PlatformPermissions.CheckPermissions(platformPermissions); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := role.ProjectPermissions.CheckPermissions(projectPermissions); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	rb, err := r.robotRepo.GetRobot(ctx, int(request.Id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("robot %d not found", request.Id))
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	scope := robot.ProjectScopeSelected
	if request.ProjectScope == v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_ALL {
		scope = robot.ProjectScopeAll
	}
	rb.Description = request.Description
	rb.Duration = int(request.ExpireDays)
	rb.PlatformPermissions = platformPermissions
	rb.ProjectPermissions = projectPermissions
	rb.ProjectScope = scope

	if request.ExpireDays > 0 {
		expireAt := time.Now().AddDate(0, 0, int(request.ExpireDays))
		rb.ExpireAt = &expireAt
	}

	rb.Projects, err = r.checkProjects(ctx, request.ProjectScope, request.Projects)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err = r.robotRepo.UpdateRobot(ctx, rb); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.UpdateRobotAccountResponse{}, nil
}

func (r *RobotHandler) checkProjects(ctx context.Context, scope v1alpha1.RobotAccountProjectScope, projects []string) ([]*project.Project, error) {
	if len(projects) == 0 || scope == v1alpha1.RobotAccountProjectScope_ROBOT_ACCOUNT_PROJECT_SCOPE_ALL {
		return nil, nil
	}

	projects = lo.Uniq(projects)
	projectMap := lo.SliceToMap(projects, func(item string) (string, bool) {
		return item, true
	})
	ps, err := r.projectRepo.ListProjectInfoByNames(ctx, projects)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		delete(projectMap, p.Name)
	}

	if len(projectMap) > 0 {
		remain := lo.MapToSlice(projectMap, func(key string, value bool) string {
			return key
		})
		return nil, fmt.Errorf("projects %s not exist", strings.Join(remain, ","))
	}

	return ps, nil
}

func (r *RobotHandler) RefreshRobotAccountToken(ctx context.Context, request *v1alpha1.RefreshRobotAccountTokenRequest) (*v1alpha1.RefreshRobotAccountTokenResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	rb, err := r.robotRepo.GetRobot(ctx, int(request.Id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("robot %d not found", request.Id))
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	var (
		token string
		hash  string
	)
	if request.AutoGenerate {
		token, hash, err = utils.GenerateRobotToken()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		token = request.Token
		hash = utils.Sha256Hex(token)
	}

	rb.TokenHash = hash
	if err = r.robotRepo.UpdateRobot(ctx, rb); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.RefreshRobotAccountTokenResponse{
		Token: token,
	}, nil
}

func (r *RobotHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterRobotsServer(options.GRPCServer, r)
	if err := v1alpha1.RegisterRobotsHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewRobotHandler(robotRepo robot.IRobotRepo, projectRepo project.IProjectRepo) IHandler {
	return &RobotHandler{
		robotRepo:   robotRepo,
		projectRepo: projectRepo,
	}
}
