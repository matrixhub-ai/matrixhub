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
	"time"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
)

type SyncTaskStatus int

// SyncTaskStatus represents the status of a sync task
const (
	SyncTaskStatusRunning SyncTaskStatus = iota + 1
	SyncTaskStatusSucceeded
	SyncTaskStatusFailed
	SyncTaskStatusStopped
)

// SyncTaskStatusPending means the task row exists; sync_task_processor will claim it and run work later.
const SyncTaskStatusPending = 5

type SyncTask struct {
	ID                 int       `gorm:"primarykey"`
	SyncPolicyID       int       `gorm:"column:sync_policy_id"`
	TriggerType        int       `gorm:"column:trigger_type"` // 1: manual, 2: scheduled
	Status             int       `gorm:"column:status"`       // 1: running, 2: succeeded, 3: failed, 4: stopped, 5: pending (queued for processor)
	StartedTimestamp   int64     `gorm:"column:started_timestamp"`
	CompletedTimestamp int64     `gorm:"column:completed_timestamp"`
	TotalItems         int       `gorm:"column:total_items"`
	SuccessfulItems    int       `gorm:"column:successful_items"`
	CompletePercents   int       `gorm:"column:complete_percents"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (SyncTask) TableName() string {
	return "sync_tasks"
}

type ISyncTaskRepo interface {
	CreateSyncTask(ctx context.Context, task *SyncTask) (*SyncTask, error)
	GetSyncTask(ctx context.Context, id int) (*SyncTask, error)
	UpdateSyncTask(ctx context.Context, task *SyncTask) error
	DeleteSyncTask(ctx context.Context, id int) error
	ListSyncTasksByPolicyID(ctx context.Context, policyID int, page, pageSize int, status SyncTaskStatus) ([]*SyncTask, int64, error)
}

func ConvertSyncTaskStatusToProto(status SyncTaskStatus) v1alpha1.SyncTaskStatus {
	return v1alpha1.SyncTaskStatus(status)
}

func ConvertSyncTaskStatusFromProto(status v1alpha1.SyncTaskStatus) SyncTaskStatus {
	return SyncTaskStatus(status)
}
