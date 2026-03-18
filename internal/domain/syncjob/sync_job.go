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

package syncjob

import (
	"context"
)

type SyncJob struct {
	ID                 int
	RemoteRegistryID   int
	RemoteProjectName  string
	RemoteResourceName string
	ProjectName        string
	ResourceName       string
	ResourceType       string
	SyncType           string
	SyncTaskID         int
	CompletePercents   int
}

func (p *SyncJob) HasSyncTask() bool {
	return p.SyncTaskID > 0
}

type ISyncJobRepo interface {
	CreateSyncJob(ctx context.Context, syncJob *SyncJob) error
	GetSyncJob(ctx context.Context, syncJob *SyncJob) (*SyncJob, error)
	UpdateSyncJob(ctx context.Context, syncJob *SyncJob) error
	DeleteSyncJob(ctx context.Context, syncJob *SyncJob) error
	ListSyncJobsByTaskID()
}
