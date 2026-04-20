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

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
)

// SyncPolicyType represents the type of sync policy
const (
	SyncPolicyTypePull int = iota + 1
	SyncPolicyTypePush
)

// TriggerType represents how the sync is triggered
const (
	TriggerTypeManual int = iota + 1
	TriggerTypeScheduled
)

type SyncPolicy struct {
	ID                int       `gorm:"primarykey"`
	Name              string    `gorm:"column:name"`
	Description       string    `gorm:"column:description"`
	PolicyType        int       `gorm:"column:policy_type"`         // 1: pull, 2: push
	TriggerType       int       `gorm:"column:trigger_type"`        // 1: manual, 2: scheduled
	SourceRegistryID  int       `gorm:"column:source_registry_id"`  // for pull policy
	ResourceName      string    `gorm:"column:resource_name"`       // source resource name
	ResourceTypes     string    `gorm:"column:resource_types"`      // comma separated: model,dataset
	TargetProjectName string    `gorm:"column:target_project_name"` // target local project name
	Bandwidth         string    `gorm:"column:bandwidth"`
	IsOverwrite       bool      `gorm:"column:is_overwrite"`
	IsDisabled        bool      `gorm:"column:is_disabled"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (SyncPolicy) TableName() string {
	return "sync_policies"
}

type ISyncPolicyRepo interface {
	CreateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	GetSyncPolicy(ctx context.Context, id int) (*SyncPolicy, error)
	UpdateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	DeleteSyncPolicy(ctx context.Context, id int) error
	ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]*SyncPolicy, int64, error)
	GenerateSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error)
}
