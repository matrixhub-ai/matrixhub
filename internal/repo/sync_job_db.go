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

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
)

type syncJobDB struct {
	db *gorm.DB
}

// NewSyncJobDB creates a new syncJobDB instance
func NewSyncJobDB(db *gorm.DB) syncjob.ISyncJobRepo {
	return &syncJobDB{db: db}
}

// CreateSyncJob creates a new sync job
func (r *syncJobDB) CreateSyncJob(ctx context.Context, job *syncjob.SyncJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

// GetSyncJob gets a sync job by ID
func (r *syncJobDB) GetSyncJob(ctx context.Context, id int) (*syncjob.SyncJob, error) {
	var job syncjob.SyncJob
	err := r.db.WithContext(ctx).First(&job, id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateSyncJob updates a sync job
func (r *syncJobDB) UpdateSyncJob(ctx context.Context, job *syncjob.SyncJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}

// DeleteSyncJob deletes a sync job by ID
func (r *syncJobDB) DeleteSyncJob(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&syncjob.SyncJob{}, id).Error
}

// ListSyncJobsByTaskID lists sync jobs by task ID with pagination and filtering
func (r *syncJobDB) ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status syncjob.SyncJobStatus, resourceType string) ([]*syncjob.SyncJob, int64, error) {
	var jobs []*syncjob.SyncJob
	var total int64

	query := r.db.WithContext(ctx).Model(&syncjob.SyncJob{}).
		Where("sync_task_id = ?", taskID)

	if status > syncjob.SyncJobStatusUnspecified {
		query = query.Where("status = ?", status)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// SelectPendingJobs returns jobs with status = pending.
func (r *syncJobDB) SelectPendingJobs(ctx context.Context, limit int) ([]*syncjob.SyncJob, error) {
	var rows []*syncjob.SyncJob
	err := r.db.WithContext(ctx).
		Where("status = ?", syncjob.SyncJobStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// UpdateJobStatusCAS atomically updates job status from fromStatus to toStatus.
func (r *syncJobDB) UpdateJobStatusCAS(ctx context.Context, jobID int, fromStatus, toStatus syncjob.SyncJobStatus) (bool, error) {
	res := r.db.WithContext(ctx).Exec(
		"UPDATE sync_jobs SET status = ? WHERE id = ? AND status = ?",
		toStatus, jobID, fromStatus,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Ensure syncJobDB implements ISyncJobRepo
var _ syncjob.ISyncJobRepo = (*syncJobDB)(nil)
