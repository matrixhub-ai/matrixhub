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
	"database/sql/driver"
	"fmt"
	"time"
)

type SyncTaskStatus int

const (
	SyncTaskStatusUnspecified SyncTaskStatus = iota
	SyncTaskStatusRunning
	SyncTaskStatusSucceeded
	SyncTaskStatusFailed
	SyncTaskStatusStopped
	SyncTaskStatusPending
)

func (t SyncTaskStatus) Value() (driver.Value, error) {
	return int64(t), nil
}

func (t *SyncTaskStatus) Scan(value any) error {
	if value == nil {
		*t = SyncTaskStatusUnspecified
		return nil
	}
	switch v := value.(type) {
	case int64:
		*t = SyncTaskStatus(v)
	case int:
		*t = SyncTaskStatus(v)
	case int32:
		*t = SyncTaskStatus(v)
	case uint8:
		*t = SyncTaskStatus(v)
	default:
		return fmt.Errorf("cannot scan %T into SyncTaskStatus", value)
	}
	return nil
}

type SyncTask struct {
	ID                 int            `gorm:"primarykey"`
	SyncPolicyID       int            `gorm:"column:sync_policy_id"`
	TriggerType        TriggerType    `gorm:"column:trigger_type"`
	Status             SyncTaskStatus `gorm:"column:status"`
	StartedTimestamp   int64          `gorm:"column:started_timestamp"`
	CompletedTimestamp int64          `gorm:"column:completed_timestamp"`
	TotalItems         int            `gorm:"column:total_items"`
	SuccessfulItems    int            `gorm:"column:successful_items"`
	StoppedItems       int            `gorm:"column:stopped_items"`
	FailedItems        int            `gorm:"column:failed_items"`
	CompletePercents   int            `gorm:"column:complete_percents"`
	CreatedAt          time.Time      `gorm:"column:created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at"`
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

	SelectPendingTasks(ctx context.Context, limit int) ([]*SyncTask, error)
	UpdateTaskStatusCAS(ctx context.Context, taskID int, fromStatus, toStatus SyncTaskStatus) (bool, error)
}
