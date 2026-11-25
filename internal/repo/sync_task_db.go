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
	"gorm.io/gorm/clause"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
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
	res := r.db.WithContext(ctx).
		Model(&syncpolicy.SyncTask{}).
		Where("id = ? AND status = ?", taskID, fromStatus).
		Update("status", toStatus)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// RecountTask locks the task row before it counts the jobs. A recount that
// starts while another one is in progress therefore waits for it to commit,
// and then counts the job statuses committed in the meantime. The locking read
// must stay the first statement of the transaction: under REPEATABLE READ,
// MySQL takes the snapshot at the first plain read, so a plain read before the
// lock would count the job statuses from before the wait. SQLite has no row
// locks, and its driver drops the locking clause, but it runs on a single
// connection, so the transaction alone serializes recounts there. The
// transaction then also holds that connection, so a query inside update would
// wait for it forever.
func (r *syncTaskDB) RecountTask(ctx context.Context, taskID int, update func(*syncpolicy.SyncTask, map[syncjob.SyncJobStatus]int)) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task syncpolicy.SyncTask
		if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).First(&task, taskID).Error; err != nil {
			return err
		}

		var rows []struct {
			Status syncjob.SyncJobStatus
			Jobs   int
		}
		if err := tx.Model(&syncjob.SyncJob{}).
			Select("status, COUNT(*) AS jobs").
			Where("sync_task_id = ?", taskID).
			Group("status").
			Scan(&rows).Error; err != nil {
			return err
		}
		counts := make(map[syncjob.SyncJobStatus]int, len(rows))
		for _, row := range rows {
			counts[row.Status] = row.Jobs
		}

		update(&task, counts)
		return tx.Model(&task).
			Select("status", "completed_timestamp", "total_items", "successful_items",
				"stopped_items", "failed_items", "complete_percents").
			Updates(&task).Error
	})
}

// Ensure syncTaskDB implements ISyncTaskRepo
var _ syncpolicy.ISyncTaskRepo = (*syncTaskDB)(nil)
