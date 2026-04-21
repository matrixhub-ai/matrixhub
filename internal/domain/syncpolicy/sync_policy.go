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
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
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
	Cron              string    `gorm:"column:cron"`        // cron expression when TriggerType is scheduled
	LastRunAt         int64     `gorm:"column:last_run_at"` // ms since epoch; last claim / run bookkeeping
	NextRunAt         int64     `gorm:"column:next_run_at"` // ms; 0 = not scheduled
}

func (SyncPolicy) TableName() string {
	return "sync_policies"
}

// ApplyScheduleNextRun sets NextRunAt based on current policy settings.
func (p *SyncPolicy) ApplyScheduleNextRun(now time.Time) error {
	if p.IsDisabled {
		p.NextRunAt = 0
		return nil
	}
	if p.TriggerType == TriggerTypeScheduled {
		if p.Cron == "" {
			p.NextRunAt = 0
			return nil
		}
		if err := utils.ValidateCronExpr(p.Cron); err != nil {
			return err
		}
		next, err := utils.NextAfter(p.Cron, now)
		if err != nil {
			return err
		}
		p.NextRunAt = next.UnixMilli()
		return nil
	}
	p.NextRunAt = 0
	return nil
}

// nextRunAtAfterClaim returns the next next_run_at (ms) after a successful claim at nowMs, or ok=false to skip advancing.
func (p *SyncPolicy) nextRunAtAfterClaim(nowMs int64) (nextNext int64, ok bool) {
	switch p.TriggerType {
	case TriggerTypeScheduled:
		if p.Cron == "" {
			log.Warnw("sync: scheduled policy missing cron, skipping claim advance", "policyId", p.ID)
			return 0, false
		}
		n, err := utils.NextAfter(p.Cron, time.UnixMilli(nowMs))
		if err != nil {
			log.Warnw("sync: invalid cron on policy, skipping", "policyId", p.ID, "error", err)
			return 0, false
		}
		return n.UnixMilli(), true
	case TriggerTypeManual:
		return 0, true
	default:
		return 0, true
	}
}

type ISyncPolicyRepo interface {
	CreateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	GetSyncPolicy(ctx context.Context, id int) (*SyncPolicy, error)
	UpdateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	DeleteSyncPolicy(ctx context.Context, id int) error
	ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]*SyncPolicy, int64, error)
	GenerateSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error)

	SelectDuePolicies(ctx context.Context, nowMs int64, limit int) ([]*SyncPolicy, error)
	AdvanceNextRunAtCAS(ctx context.Context, policyID int, snapshotMs, nextNextMs, nowMs int64) (claimed bool, err error)
}
