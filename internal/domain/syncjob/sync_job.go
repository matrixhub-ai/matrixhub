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
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/datatypes"
)

type SyncJobStatus int

const (
	SyncJobStatusUnspecified SyncJobStatus = iota
	SyncJobStatusRunning
	SyncJobStatusSucceeded
	SyncJobStatusFailed
	SyncJobStatusStopped
	SyncJobStatusPending
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

// Job types. The sync_jobs table holds the jobs of every type.
const (
	// JobTypeSync pulls or pushes a resource for a sync task.
	JobTypeSync = "sync"
	// JobTypeModelCache pulls a model of a proxy project from upstream and
	// checks that upstream has a revision. It belongs to no sync task, and
	// its parameters are ModelCacheParams.
	JobTypeModelCache = "model_cache"
)

// SyncJob is a job of any job type. JobType is one of the job types; a job
// created without one is a sync job. SyncTaskID is nil for a job that belongs
// to no sync task.
type SyncJob struct {
	ID                 int           `gorm:"primarykey"`
	JobType            string        `gorm:"column:job_type;default:sync"`
	RemoteRegistryID   int           `gorm:"column:remote_registry_id"`
	RemoteProjectName  string        `gorm:"column:remote_project_name"`
	RemoteResourceName string        `gorm:"column:remote_resource_name"`
	ProjectName        string        `gorm:"column:project_name"`
	ResourceName       string        `gorm:"column:resource_name"`
	ResourceType       string        `gorm:"column:resource_type"`
	SyncType           string        `gorm:"column:sync_type"`
	Status             SyncJobStatus `gorm:"column:status"`
	CompletedTimestamp int64         `gorm:"column:completed_timestamp"`
	SyncTaskID         *int          `gorm:"column:sync_task_id"`
	CompletePercents   int           `gorm:"column:complete_percents"`
	CreatedAt          time.Time     `gorm:"column:created_at"`
	UpdatedAt          time.Time     `gorm:"column:updated_at"`

	// Params holds the parameters of the job type as JSON. Sync jobs have
	// none.
	Params datatypes.JSON `gorm:"column:params"`
}

func (SyncJob) TableName() string {
	return "sync_jobs"
}

func (p *SyncJob) HasSyncTask() bool {
	return p.SyncTaskID != nil && *p.SyncTaskID > 0
}

// ModelCacheParams are the parameters of a model cache job.
type ModelCacheParams struct {
	// Revision is a branch, tag or commit; empty means the default branch.
	Revision string `json:"revision,omitempty"`
}

// ModelCacheParams decodes the parameters of a model cache job.
func (p *SyncJob) ModelCacheParams() (ModelCacheParams, error) {
	var params ModelCacheParams
	if len(p.Params) == 0 {
		return params, nil
	}
	if err := json.Unmarshal(p.Params, &params); err != nil {
		return params, fmt.Errorf("decode model cache params of job %d: %w", p.ID, err)
	}
	return params, nil
}

//go:generate go tool mockgen -source=sync_job.go -destination=mocks/sync_job_repo_mock.go -package=mocks
type ISyncJobRepo interface {
	CreateSyncJob(ctx context.Context, syncJob *SyncJob) error
	CreateSyncJobs(ctx context.Context, syncJobs []*SyncJob) error
	GetSyncJob(ctx context.Context, id int) (*SyncJob, error)
	UpdateSyncJob(ctx context.Context, syncJob *SyncJob) error
	DeleteSyncJob(ctx context.Context, id int) error
	ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status SyncJobStatus, resourceType string) ([]*SyncJob, int64, error)

	SelectPendingJobs(ctx context.Context, limit int) ([]*SyncJob, error)
	UpdateJobStatusCAS(ctx context.Context, jobID int, fromStatus, toStatus SyncJobStatus) (bool, error)
}
