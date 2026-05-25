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

package repo

import (
	"context"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
)

type syncTaskDB struct {
	db *gorm.DB
}

// NewSyncTaskDB creates a new syncTaskDB instance
func NewSyncTaskDB(db *gorm.DB) syncpolicy.ISyncTaskRepo {
	return &syncTaskDB{db: db}
}

// CreateSyncTask creates a new sync task
func (r *syncTaskDB) CreateSyncTask(ctx context.Context, task *syncpolicy.SyncTask) (*syncpolicy.SyncTask, error) {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

// GetSyncTask gets a sync task by ID
func (r *syncTaskDB) GetSyncTask(ctx context.Context, id int) (*syncpolicy.SyncTask, error) {
	var task syncpolicy.SyncTask
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateSyncTask updates a sync task
func (r *syncTaskDB) UpdateSyncTask(ctx context.Context, task *syncpolicy.SyncTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// DeleteSyncTask deletes a sync task by ID
func (r *syncTaskDB) DeleteSyncTask(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&syncpolicy.SyncTask{}, id).Error
}

// ListSyncTasksByPolicyID lists sync tasks by policy ID with pagination
func (r *syncTaskDB) ListSyncTasksByPolicyID(ctx context.Context, policyID int, page, pageSize int, status syncpolicy.SyncTaskStatus) ([]*syncpolicy.SyncTask, int64, error) {
	var tasks []*syncpolicy.SyncTask
	var total int64

	query := r.db.WithContext(ctx).Model(&syncpolicy.SyncTask{}).
		Where("sync_policy_id = ?", policyID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// SelectPendingTasks returns tasks with status = pending.
func (r *syncTaskDB) SelectPendingTasks(ctx context.Context, limit int) ([]*syncpolicy.SyncTask, error) {
	var rows []*syncpolicy.SyncTask
	err := r.db.WithContext(ctx).
		Where("status = ?", syncpolicy.SyncTaskStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// UpdateTaskStatusCAS atomically updates task status from fromStatus to toStatus.
func (r *syncTaskDB) UpdateTaskStatusCAS(ctx context.Context, taskID int, fromStatus, toStatus syncpolicy.SyncTaskStatus) (bool, error) {
	res := r.db.WithContext(ctx).Exec(
		"UPDATE sync_tasks SET status = ? WHERE id = ? AND status = ?",
		toStatus, taskID, fromStatus,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Ensure syncTaskDB implements ISyncTaskRepo
var _ syncpolicy.ISyncTaskRepo = (*syncTaskDB)(nil)
