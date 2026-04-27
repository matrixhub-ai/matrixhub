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
	"database/sql/driver"
	"fmt"
	"time"
)

type SyncJobStatus int

const (
	SyncJobStatusUnspecified SyncJobStatus = iota
	SyncJobStatusRunning
	SyncJobStatusSucceeded
	SyncJobStatusFailed
	SyncJobStatusStopped
)

func (s SyncJobStatus) Value() (driver.Value, error) {
	return int64(s), nil
}

func (s *SyncJobStatus) Scan(value any) error {
	if value == nil {
		*s = SyncJobStatusUnspecified
		return nil
	}
	switch v := value.(type) {
	case int64:
		*s = SyncJobStatus(v)
	case int:
		*s = SyncJobStatus(v)
	case int32:
		*s = SyncJobStatus(v)
	case uint8:
		*s = SyncJobStatus(v)
	default:
		return fmt.Errorf("cannot scan %T into SyncJobStatus", value)
	}
	return nil
}

type SyncJob struct {
	ID                 int           `gorm:"primarykey"`
	RemoteRegistryID   int           `gorm:"column:remote_registry_id"`
	RemoteProjectName  string        `gorm:"column:remote_project_name"`
	RemoteResourceName string        `gorm:"column:remote_resource_name"`
	ProjectName        string        `gorm:"column:project_name"`
	ResourceName       string        `gorm:"column:resource_name"`
	ResourceType       string        `gorm:"column:resource_type"`
	SyncType           string        `gorm:"column:sync_type"`
	Status             SyncJobStatus `gorm:"column:status"`
	CompletedTimestamp int64         `gorm:"column:completed_timestamp"`
	SyncTaskID         int           `gorm:"column:sync_task_id"`
	CompletePercents   int           `gorm:"column:complete_percents"`
	CreatedAt          time.Time     `gorm:"column:created_at"`
	UpdatedAt          time.Time     `gorm:"column:updated_at"`
}

func (SyncJob) TableName() string {
	return "sync_jobs"
}

func (p *SyncJob) HasSyncTask() bool {
	return p.SyncTaskID > 0
}

type ISyncJobRepo interface {
	CreateSyncJob(ctx context.Context, syncJob *SyncJob) error
	GetSyncJob(ctx context.Context, id int) (*SyncJob, error)
	UpdateSyncJob(ctx context.Context, syncJob *SyncJob) error
	DeleteSyncJob(ctx context.Context, id int) error
	ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status SyncJobStatus, resourceType string) ([]*SyncJob, int64, error)
}
