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

package syncpolicy

import (
	"context"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
)

// Todo: change it according to requirement
type SyncPolicy struct {
	ID                 int
	Name               string
	Description        string
	RemoteRegistryID   int
	RemoteProjectName  string
	RemoteResourceName string
	ProjectName        string
	ResourceName       string
	ResourceType       string
	SyncType           string
}

type ISyncPolicyRepo interface {
	CreateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	GetSyncPolicy(ctx context.Context, policy *SyncPolicy) (*SyncPolicy, error)
	UpdateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	DeleteSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	GenerateSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error)
}
