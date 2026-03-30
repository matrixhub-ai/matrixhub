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

package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

var publicAuthzMethods = map[string]bool{
	// Login/Logout
	"/matrixhub.v1alpha1.Login/Login":  true,
	"/matrixhub.v1alpha1.Login/Logout": true,

	// Current user
	"/matrixhub.v1alpha1.CurrentUser/GetCurrentUser":    true,
	"/matrixhub.v1alpha1.CurrentUser/ResetPassword":     true,
	"/matrixhub.v1alpha1.CurrentUser/ListAccessTokens":  true,
	"/matrixhub.v1alpha1.CurrentUser/CreateAccessToken": true,
	"/matrixhub.v1alpha1.CurrentUser/DeleteAccessToken": true,
	"/matrixhub.v1alpha1.CurrentUser/GetProjectRoles":   true,

	// Projects
	"/matrixhub.v1alpha1.Projects/CreateProject":            true,
	"/matrixhub.v1alpha1.Projects/ListProjects":             true,
	"/matrixhub.v1alpha1.Projects/GetProject":               true,
	"/matrixhub.v1alpha1.Projects/UpdateProject":            true,
	"/matrixhub.v1alpha1.Projects/DeleteProject":            true,
	"/matrixhub.v1alpha1.Projects/ListProjectMembers":       true,
	"/matrixhub.v1alpha1.Projects/AddProjectMemberWithRole": true,
	"/matrixhub.v1alpha1.Projects/RemoveProjectMembers":     true,
	"/matrixhub.v1alpha1.Projects/UpdateProjectMemberRole":  true,

	// Models
	"/matrixhub.v1alpha1.Models/ListModels":           true,
	"/matrixhub.v1alpha1.Models/GetModel":             true,
	"/matrixhub.v1alpha1.Models/CreateModel":          true,
	"/matrixhub.v1alpha1.Models/DeleteModel":          true,
	"/matrixhub.v1alpha1.Models/ListModelRevisions":   true,
	"/matrixhub.v1alpha1.Models/GetModelTree":         true,
	"/matrixhub.v1alpha1.Models/GetModelBlob":         true,
	"/matrixhub.v1alpha1.Models/ListModelCommits":     true,
	"/matrixhub.v1alpha1.Models/GetModelCommit":       true,
	"/matrixhub.v1alpha1.Models/GetModelCommitByHash": true,
	"/matrixhub.v1alpha1.Models/ListModelFrameLabels": true,
	"/matrixhub.v1alpha1.Models/ListModelTaskLabels":  true,
	"/matrixhub.v1alpha1.Models/UpdateModelSetting":   true,

	// Datasets
	"/matrixhub.v1alpha1.Datasets/ListDatasets":          true,
	"/matrixhub.v1alpha1.Datasets/GetDataset":            true,
	"/matrixhub.v1alpha1.Datasets/CreateDataset":         true,
	"/matrixhub.v1alpha1.Datasets/DeleteDataset":         true,
	"/matrixhub.v1alpha1.Datasets/ListDatasetRevisions":  true,
	"/matrixhub.v1alpha1.Datasets/GetDatasetTree":        true,
	"/matrixhub.v1alpha1.Datasets/GetDatasetBlob":        true,
	"/matrixhub.v1alpha1.Datasets/ListDatasetCommits":    true,
	"/matrixhub.v1alpha1.Datasets/GetDatasetCommit":      true,
	"/matrixhub.v1alpha1.Datasets/ListDatasetTaskLabels": true,
}

// methodPermissions maps GRPC methods to required permissions
var methodPermissions = map[string]authz.Permission{
	// User management
	"/matrixhub.v1alpha1.Users/ListUsers":         authz.UserGet,
	"/matrixhub.v1alpha1.Users/GetUser":           authz.UserGet,
	"/matrixhub.v1alpha1.Users/CreateUser":        authz.UserCreate,
	"/matrixhub.v1alpha1.Users/SetUserSysAdmin":   authz.UserAuthorize,
	"/matrixhub.v1alpha1.Users/DeleteUser":        authz.UserDelete,
	"/matrixhub.v1alpha1.Users/ResetUserPassword": authz.UserResetPassword,

	// Registry management
	"/matrixhub.v1alpha1.Registries/ListRegistries": authz.RegistryGet,
	"/matrixhub.v1alpha1.Registries/GetRegistry":    authz.RegistryGet,
	"/matrixhub.v1alpha1.Registries/PingRegistry":   authz.RegistryGet,
	"/matrixhub.v1alpha1.Registries/CreateRegistry": authz.RegistryCreate,
	"/matrixhub.v1alpha1.Registries/UpdateRegistry": authz.RegistryUpdate,
	"/matrixhub.v1alpha1.Registries/DeleteRegistry": authz.RegistryDelete,

	// Sync policy management
	"/matrixhub.v1alpha1.SyncPolicy/ListSyncPolicies": authz.SyncGet,
	"/matrixhub.v1alpha1.SyncPolicy/GetSyncPolicy":    authz.SyncGet,
	"/matrixhub.v1alpha1.SyncPolicy/ListSyncTasks":    authz.SyncGet,
	"/matrixhub.v1alpha1.SyncPolicy/CreateSyncPolicy": authz.SyncCreate,
	"/matrixhub.v1alpha1.SyncPolicy/CreateSyncTask":   authz.SyncCreate,
	"/matrixhub.v1alpha1.SyncPolicy/UpdateSyncPolicy": authz.SyncUpdate,
	"/matrixhub.v1alpha1.SyncPolicy/StopSyncTask":     authz.SyncUpdate,
	"/matrixhub.v1alpha1.SyncPolicy/DeleteSyncPolicy": authz.SyncDelete,
}

// AuthzInterceptor returns a GRPC interceptor that checks platform-level permissions
func AuthzInterceptor(verifyFunc func(ctx context.Context, perm authz.Permission) (bool, error)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		// Public methods don't need authz check
		if publicAuthzMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Check if user is authenticated
		userID := ctx.Value(user.UserIdCtxKey)
		if userID == nil {
			return nil, status.Error(codes.Unauthenticated, codes.Unauthenticated.String())
		}

		// Get required permission for the method
		requiredPerm, ok := methodPermissions[info.FullMethod]
		if !ok {
			// No permission configured, allow by default (may be unregistered new API)
			return handler(ctx, req)
		}

		// Verify permission
		allowed, err := verifyFunc(ctx, requiredPerm)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		}

		return handler(ctx, req)
	}
}
