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
	"time"

	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type CurrentUserHandler struct {
	userRepo        user.IUserRepo
	accessTokenRepo user.IAccessTokenRepo
	sshKeyRepo      user.ISSHKeyRepo
}

func (cu *CurrentUserHandler) CreateSSHKey(ctx context.Context, request *v1alpha1.CreateSSHKeyRequest) (*v1alpha1.CreateSSHKeyResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	expireAt, err := cu.parseExpireAt(request.GetExpireAt())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	fp, err := utils.ParseFingerprint(request.GetPublicKey())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	sk := user.SSHKey{
		Name:        request.GetName(),
		UserId:      user.GetCurrentUserId(ctx),
		PublicKey:   request.GetPublicKey(),
		Fingerprint: fp,
		ExpireAt:    expireAt,
	}
	err = cu.sshKeyRepo.CreateSSHKey(ctx, sk)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, status.Error(codes.AlreadyExists, "ssh key already exists")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.CreateSSHKeyResponse{}, nil
}

func (cu *CurrentUserHandler) DeleteSSHKey(ctx context.Context, request *v1alpha1.DeleteSSHKeyRequest) (*v1alpha1.DeleteSSHKeyResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := cu.sshKeyRepo.DeleteSSHKey(ctx, user.GetCurrentUserId(ctx), int(request.GetId())); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.DeleteSSHKeyResponse{}, nil
}

func (cu *CurrentUserHandler) ListSSHKeys(ctx context.Context, request *v1alpha1.ListSSHKeysRequest) (*v1alpha1.ListSSHKeysResponse, error) {
	sks, err := cu.sshKeyRepo.ListSSHKeys(ctx, user.GetCurrentUserId(ctx))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.ListSSHKeysResponse{
		Items: lo.Map(sks, func(item *user.SSHKey, index int) *v1alpha1.SSHKey {
			expireAt := ""
			if item.ExpireAt != nil {
				expireAt = strconv.Itoa(int(item.ExpireAt.Unix()))
			}
			status := v1alpha1.SSHKeyStatus_SSH_KEY_STATUS_UNKNOWN
			if item.ExpireAt == nil || item.ExpireAt.After(time.Now()) {
				status = v1alpha1.SSHKeyStatus_SSH_KEY_STATUS_VALID
			} else if time.Now().After(*item.ExpireAt) {
				status = v1alpha1.SSHKeyStatus_SSH_KEY_STATUS_EXPIRED
			}
			return &v1alpha1.SSHKey{
				Id:        uint32(item.Id),
				Name:      item.Name,
				Status:    status,
				CreatedAt: strconv.Itoa(int(item.CreatedAt.Unix())),
				ExpireAt:  expireAt,
			}
		}),
	}, nil
}

func (cu *CurrentUserHandler) GetCurrentUser(ctx context.Context, request *v1alpha1.GetCurrentUserRequest) (*v1alpha1.GetCurrentUserResponse, error) {
	user, err := cu.userRepo.GetUser(ctx, user.GetCurrentUserId(ctx))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user not found")
	}
	isAdmin, err := cu.userRepo.IsUserSysAdmin(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot check user admin")
	}
	return &v1alpha1.GetCurrentUserResponse{
		Id:       uint32(user.ID),
		Username: user.Username,
		IsAdmin:  isAdmin,
	}, nil
}

func (cu *CurrentUserHandler) ResetPassword(ctx context.Context, request *v1alpha1.ResetPasswordRequest) (*v1alpha1.ResetPasswordResponse, error) {
	user, err := cu.userRepo.GetUser(ctx, user.GetCurrentUserId(ctx))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "user not found")
	}
	if !user.CheckPassword(request.OldPassword) {
		return nil, status.Error(codes.InvalidArgument, "old password is incorrect")
	}
	if user.CheckPassword(request.NewPassword) {
		return nil, status.Error(codes.InvalidArgument, "new password is same as old password")
	}

	if err = cu.userRepo.UpdateUserPassword(ctx, user.ID, request.NewPassword); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1alpha1.ResetPasswordResponse{}, nil
}

func (cu *CurrentUserHandler) ListAccessTokens(ctx context.Context, request *v1alpha1.ListAccessTokensRequest) (*v1alpha1.ListAccessTokensResponse, error) {
	aks, err := cu.accessTokenRepo.ListUserAccessTokens(ctx, user.GetCurrentUserId(ctx))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.ListAccessTokensResponse{
		Items: lo.Map(aks, func(item *user.AccessToken, _ int) *v1alpha1.AccessToken {
			expiredAt := ""
			if item.ExpireAt != nil {
				expiredAt = strconv.Itoa(int(item.ExpireAt.Unix()))
			}
			status := v1alpha1.AccessTokenStatus_ACCESS_TOKEN_STATUS_UNKNOWN
			if item.ExpireAt == nil || item.ExpireAt.After(time.Now()) {
				status = v1alpha1.AccessTokenStatus_ACCESS_TOKEN_STATUS_VALID
			} else if time.Now().After(*item.ExpireAt) {
				status = v1alpha1.AccessTokenStatus_ACCESS_TOKEN_STATUS_EXPIRED
			}
			return &v1alpha1.AccessToken{
				Id:        uint32(item.Id),
				Name:      item.Name,
				Status:    status,
				CreatedAt: strconv.Itoa(int(item.CreatedAt.Unix())),
				ExpiredAt: expiredAt,
			}
		}),
	}, nil
}

func (cu *CurrentUserHandler) CreateAccessToken(ctx context.Context, request *v1alpha1.CreateAccessTokenRequest) (*v1alpha1.CreateAccessTokenResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	expireAt, err := cu.parseExpireAt(request.GetExpireAt())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	for attempt := 1; attempt <= utils.MaxTokenRetries; attempt++ {
		raw, hash, err := utils.GenerateToken()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		ak := user.AccessToken{
			Name:      request.GetName(),
			UserId:    user.GetCurrentUserId(ctx),
			TokenHash: hash,
			Enabled:   true,
			ExpireAt:  expireAt,
		}
		err = cu.accessTokenRepo.CreateAccessToken(ctx, ak)
		if err == nil {
			return &v1alpha1.CreateAccessTokenResponse{
				Token: raw,
			}, nil
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) && attempt < utils.MaxTokenRetries {
			continue
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("token: persist (attempt %d): %s", attempt, err))
	}

	return nil, status.Error(codes.Internal, "token: exceeded max retries — this should never happen")
}

func (cu *CurrentUserHandler) DeleteAccessToken(ctx context.Context, request *v1alpha1.DeleteAccessTokenRequest) (*v1alpha1.DeleteAccessTokenResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := cu.accessTokenRepo.DeleteAccessToken(ctx, user.GetCurrentUserId(ctx), int(request.GetId())); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &v1alpha1.DeleteAccessTokenResponse{}, nil
}

func (cu *CurrentUserHandler) GetProjectRoles(ctx context.Context, request *v1alpha1.GetProjectRolesRequest) (*v1alpha1.GetProjectRolesResponse, error) {
	userID := user.GetCurrentUserId(ctx)
	projectRoles, err := cu.userRepo.GetUserAllProjectRoles(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	result := make(map[string]v1alpha1.ProjectRoleType)
	for projectName, roleID := range projectRoles {
		result[projectName] = convertDomainRoleToProto(role.RoleType(roleID))
	}

	return &v1alpha1.GetProjectRolesResponse{
		ProjectRoles: result,
	}, nil
}

func (cu *CurrentUserHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterCurrentUserServer(options.GRPCServer, cu)
	if err := v1alpha1.RegisterCurrentUserHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
		log.Errorf("register handler error: %s", err.Error())
	}
}

func NewCurrentUserHandler(userRepo user.IUserRepo, akRepo user.IAccessTokenRepo, skRepo user.ISSHKeyRepo) IHandler {
	handler := &CurrentUserHandler{
		userRepo:        userRepo,
		accessTokenRepo: akRepo,
		sshKeyRepo:      skRepo,
	}
	return handler
}

func (cu *CurrentUserHandler) parseExpireAt(expireAt string) (*time.Time, error) {
	if expireAt == "" {
		return nil, nil
	}
	expire, err := strconv.Atoi(expireAt)
	if err != nil {
		return nil, err
	}
	expTime := time.Unix(int64(expire), 0)
	return &expTime, nil
}
