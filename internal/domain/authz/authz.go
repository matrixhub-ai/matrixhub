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

package authz

import (
	"context"

	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

// IAuthzProjectRepo project repository interface required for permission verification
type IAuthzProjectRepo interface {
	GetUserProjectPermissions(ctx context.Context, userID int, projectID int) ([]role.Permission, error)
	GetUserPlatformPermissions(ctx context.Context, userID int) ([]role.Permission, error)
	GetUserAccessibleProjectIDs(ctx context.Context, userID int) ([]int, error)
}
