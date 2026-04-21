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
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
)

type syncPolicyDB struct {
	db *gorm.DB
}

// NewSyncPolicyDB creates a new syncPolicyDB instance
func NewSyncPolicyDB(db *gorm.DB) syncpolicy.ISyncPolicyRepo {
	return &syncPolicyDB{db: db}
}

// CreateSyncPolicy creates a new sync policy
func (r *syncPolicyDB) CreateSyncPolicy(ctx context.Context, policy *syncpolicy.SyncPolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

// GetSyncPolicy gets a sync policy by ID
func (r *syncPolicyDB) GetSyncPolicy(ctx context.Context, id int) (*syncpolicy.SyncPolicy, error) {
	var policy syncpolicy.SyncPolicy
	err := r.db.WithContext(ctx).First(&policy, id).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

// UpdateSyncPolicy updates a sync policy
func (r *syncPolicyDB) UpdateSyncPolicy(ctx context.Context, policy *syncpolicy.SyncPolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

// DeleteSyncPolicy deletes a sync policy by ID
func (r *syncPolicyDB) DeleteSyncPolicy(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&syncpolicy.SyncPolicy{}, id).Error
}

// ListSyncPolicies lists sync policies with pagination and search
func (r *syncPolicyDB) ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]*syncpolicy.SyncPolicy, int64, error) {
	var policies []*syncpolicy.SyncPolicy
	var total int64

	query := r.db.WithContext(ctx).Model(&syncpolicy.SyncPolicy{})

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&policies).Error; err != nil {
		return nil, 0, err
	}

	return policies, total, nil
}

// GenerateSyncTaskAndSyncJobs generates a sync task and associated jobs from a policy
func (r *syncPolicyDB) GenerateSyncTaskAndSyncJobs(ctx context.Context, policy *syncpolicy.SyncPolicy) (*syncpolicy.SyncTask, []*syncjob.SyncJob, error) {
	// Create the sync task
	task := &syncpolicy.SyncTask{
		SyncPolicyID:       policy.ID,
		TriggerType:        policy.TriggerType,
		Status:             int(syncpolicy.SyncTaskStatusRunning),
		StartedTimestamp:   time.Now().Unix(),
		CompletedTimestamp: 0,
		TotalItems:         0,
		SuccessfulItems:    0,
		CompletePercents:   0,
	}

	// Parse resource types
	resourceTypes := parseResourceTypes(policy.ResourceTypes)
	task.TotalItems = len(resourceTypes)

	// Create sync jobs based on resource types
	var jobs []*syncjob.SyncJob

	// Parse remote project and resource name from policy.ResourceName
	// e.g., "HuggingFaceTB/test/SmolLM2-135M-Instruct" -> project="HuggingFaceTB", resource="test/SmolLM2-135M-Instruct"
	remoteProjectName, remoteResourceName := parseRemoteResourceName(policy.ResourceName)

	for _, resourceType := range resourceTypes {
		job := &syncjob.SyncJob{
			RemoteRegistryID:   policy.SourceRegistryID,
			RemoteProjectName:  remoteProjectName,
			RemoteResourceName: remoteResourceName,
			ProjectName:        policy.TargetProjectName,
			ResourceName:       remoteResourceName,
			ResourceType:       resourceType,
			SyncType:           "pull",
			CompletePercents:   0,
		}
		jobs = append(jobs, job)
	}

	return task, jobs, nil
}

// parseResourceTypes parses comma-separated resource types
func parseResourceTypes(resourceTypes string) []string {
	if resourceTypes == "" {
		return []string{"model"} // default to model
	}

	var result []string
	types := strings.Split(resourceTypes, ",")
	for _, t := range types {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "all" {
			return []string{"model", "dataset"}
		}
		if t == "model" || t == "dataset" {
			result = append(result, t)
		}
	}

	if len(result) == 0 {
		return []string{"model"}
	}

	return result
}

// parseRemoteResourceName parses a full resource path into project and resource names
// e.g., "HuggingFaceTB/test/SmolLM2-135M-Instruct" -> project="HuggingFaceTB", resource="test/SmolLM2-135M-Instruct"
func parseRemoteResourceName(fullPath string) (string, string) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return fullPath, ""
}

// SelectDuePolicies returns policies that are due (enabled, next_run_at in (0, now]).
func (r *syncPolicyDB) SelectDuePolicies(ctx context.Context, nowMs int64, limit int) ([]*syncpolicy.SyncPolicy, error) {
	var rows []*syncpolicy.SyncPolicy
	err := r.db.WithContext(ctx).
		Where("is_disabled = 0 AND next_run_at > 0 AND next_run_at <= ?", nowMs).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// AdvanceNextRunAtCAS atomically claims a due slot by advancing next_run_at from snapshotMs.
func (r *syncPolicyDB) AdvanceNextRunAtCAS(
	ctx context.Context, policyID int, snapshotMs, nextNextMs, nowMs int64,
) (bool, error) {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE sync_policies
		   SET next_run_at = ?, last_run_at = ?
		 WHERE id = ? AND next_run_at = ?`,
		nextNextMs, nowMs, policyID, snapshotMs,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Ensure syncPolicyDB implements ISyncPolicyRepo
var _ syncpolicy.ISyncPolicyRepo = (*syncPolicyDB)(nil)
