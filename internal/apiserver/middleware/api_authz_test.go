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
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

// The services of platform resources check no project permission in their
// handlers, so a method that the permission table misses is open to every
// authenticated user and robot.
func TestAuthzInterceptorCoversPlatformServices(t *testing.T) {
	for _, desc := range []grpc.ServiceDesc{
		v1alpha1.Cleanup_ServiceDesc,
		v1alpha1.Registries_ServiceDesc,
		v1alpha1.Robots_ServiceDesc,
		v1alpha1.SyncPolicy_ServiceDesc,
	} {
		for _, m := range desc.Methods {
			method := "/" + desc.ServiceName + "/" + m.MethodName
			if _, ok := methodPermissions[method]; !ok {
				t.Errorf("%s has no required permission", method)
			}
		}
	}
}

func TestAuthzInterceptorSyncPermissions(t *testing.T) {
	tests := []struct {
		method     string
		permission role.Permission
	}{
		{method: "/matrixhub.v1alpha1.SyncPolicy/GetSyncTask", permission: role.SyncGet},
		{method: "/matrixhub.v1alpha1.SyncPolicy/ListSyncJobs", permission: role.SyncGet},
		{method: "/matrixhub.v1alpha1.SyncPolicy/GetSyncJobLog", permission: role.SyncGet},
		{method: "/matrixhub.v1alpha1.SyncPolicy/UpdateSyncPolicySwitch", permission: role.SyncUpdate},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			ctx := auth.WithIdentity(context.Background(), user.NewUserIdentity(2, "user"))
			var gotPerm role.Permission
			interceptor := AuthzInterceptor(func(ctx context.Context, perm role.Permission) (bool, error) {
				gotPerm = perm
				return false, nil
			})

			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tt.method}, func(context.Context, interface{}) (interface{}, error) {
				t.Fatal("handler should not be called without the permission")
				return nil, nil
			})
			if status.Code(err) != codes.PermissionDenied {
				t.Fatalf("expected PermissionDenied, got %v", err)
			}
			if gotPerm != tt.permission {
				t.Fatalf("expected permission %q, got %q", tt.permission, gotPerm)
			}
		})
	}
}

func TestAuthzInterceptorCleanupPermissions(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		permission role.Permission
	}{
		{
			name:       "preview cleanup requires cleanup get",
			method:     "/matrixhub.v1alpha1.Cleanup/PreviewCleanup",
			permission: role.CleanupGet,
		},
		{
			name:       "storage stats requires cleanup get",
			method:     "/matrixhub.v1alpha1.Cleanup/GetStorageStats",
			permission: role.CleanupGet,
		},
		{
			name:       "execute cleanup requires cleanup execute",
			method:     "/matrixhub.v1alpha1.Cleanup/ExecuteCleanup",
			permission: role.CleanupExecute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := auth.WithIdentity(context.Background(), user.NewUserIdentity(1, "admin"))
			var gotPerm role.Permission
			interceptor := AuthzInterceptor(func(ctx context.Context, perm role.Permission) (bool, error) {
				gotPerm = perm
				return true, nil
			})

			called := false
			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tt.method}, func(context.Context, interface{}) (interface{}, error) {
				called = true
				return nil, nil
			})
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !called {
				t.Fatal("expected handler to be called")
			}
			if gotPerm != tt.permission {
				t.Fatalf("expected permission %q, got %q", tt.permission, gotPerm)
			}
		})
	}
}

func TestAuthzInterceptorExecuteCleanupDenied(t *testing.T) {
	ctx := auth.WithIdentity(context.Background(), user.NewUserIdentity(1, "admin"))
	interceptor := AuthzInterceptor(func(ctx context.Context, perm role.Permission) (bool, error) {
		if perm != role.CleanupExecute {
			t.Fatalf("expected permission %q, got %q", role.CleanupExecute, perm)
		}
		return false, nil
	})

	called := false
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/matrixhub.v1alpha1.Cleanup/ExecuteCleanup"}, func(context.Context, interface{}) (interface{}, error) {
		called = true
		return nil, nil
	})

	if called {
		t.Fatal("expected handler not to be called")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}

func TestAuthzInterceptorCleanupRequiresIdentity(t *testing.T) {
	interceptor := AuthzInterceptor(func(ctx context.Context, perm role.Permission) (bool, error) {
		t.Fatal("verifyFunc should not be called without identity")
		return false, nil
	})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/matrixhub.v1alpha1.Cleanup/PreviewCleanup"}, func(context.Context, interface{}) (interface{}, error) {
		t.Fatal("handler should not be called without identity")
		return nil, nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}
