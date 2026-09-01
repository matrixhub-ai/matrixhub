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

package tools

import (
	"context"
	"errors"
	"fmt"

	v1alpha1current_user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/current_user"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
)

const protocolTestPassword = "Test@123456"

// ProjectUserFixture is an isolated non-admin user with an explicit role in a
// disposable project. Protocol tests use the user's credentials while setup
// and cleanup continue to use the generated API clients.
type ProjectUserFixture struct {
	UserID      int32
	Username    string
	Cookie      string
	Project     ProjectFixture
	CurrentUser *v1alpha1current_user.CurrentUserApiService
}

// CreateProjectUserFixture creates a private project and grants the new user
// the requested project role.
func CreateProjectUserFixture(ctx context.Context, prefix string, projectRole v1alpha1project.V1alpha1ProjectRoleType) (*ProjectUserFixture, error) {
	username := GenerateTestUsername(prefix)
	userID, cookie, err := CreateUserAndLoginWithID(username, protocolTestPassword, false)
	if err != nil {
		return nil, err
	}

	project, err := CreateProjectFixture(ctx, prefix)
	if err != nil {
		_ = DeleteUser(int64(userID))
		return nil, err
	}

	memberType := v1alpha1project.USER_V1alpha1MemberType
	_, _, err = GetV1alpha1ProjectsApi().ProjectsAddProjectMemberWithRole(ctx, project.Name, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
		MemberType: &memberType,
		MemberId:   userID,
		Role:       &projectRole,
	})
	if err != nil {
		project.Cleanup(ctx)
		_ = DeleteUser(int64(userID))
		return nil, fmt.Errorf("grant project role: %w", err)
	}

	return &ProjectUserFixture{
		UserID:      userID,
		Username:    username,
		Cookie:      cookie,
		Project:     project,
		CurrentUser: CreateCurrentUserClientWithCookie(cookie),
	}, nil
}

// Cleanup removes the project before the user so project membership does not
// outlive either resource.
func (f *ProjectUserFixture) Cleanup(ctx context.Context) error {
	if f == nil {
		return nil
	}
	f.Project.Cleanup(ctx)
	return DeleteUser(int64(f.UserID))
}

// CreateAccessToken creates a token owned by the fixture user.
func (f *ProjectUserFixture) CreateAccessToken(ctx context.Context) (string, error) {
	resp, _, err := f.CurrentUser.CurrentUserCreateAccessToken(ctx, v1alpha1current_user.V1alpha1CreateAccessTokenRequest{
		Name: "protocol-e2e-token",
	})
	if err != nil {
		return "", err
	}
	if resp.Token == "" {
		return "", errors.New("create access token: empty token")
	}
	return resp.Token, nil
}

// RegisterSSHKey registers an OpenSSH public key for the fixture user.
func (f *ProjectUserFixture) RegisterSSHKey(ctx context.Context, publicKey string) error {
	_, _, err := f.CurrentUser.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
		Name:      "protocol-e2e-key",
		PublicKey: publicKey,
	})
	return err
}
