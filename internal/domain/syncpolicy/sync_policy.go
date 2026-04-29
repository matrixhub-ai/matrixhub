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

	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// ==================== Custom Types ====================

type SyncPolicyType int

const (
	SyncPolicyTypeUnspecified SyncPolicyType = iota
	SyncPolicyTypePull
	SyncPolicyTypePush
)

func (t SyncPolicyType) Value() (driver.Value, error) {
	return int64(t), nil
}

func (t *SyncPolicyType) Scan(value any) error {
	if value == nil {
		*t = SyncPolicyTypeUnspecified
		return nil
	}
	switch v := value.(type) {
	case int64:
		*t = SyncPolicyType(v)
	case int:
		*t = SyncPolicyType(v)
	case int32:
		*t = SyncPolicyType(v)
	case uint8:
		*t = SyncPolicyType(v)
	default:
		return fmt.Errorf("cannot scan %T into SyncPolicyType", value)
	}
	return nil
}

type TriggerType int

const (
	TriggerTypeUnspecified TriggerType = iota
	TriggerTypeManual
	TriggerTypeScheduled
)

func (t TriggerType) Value() (driver.Value, error) {
	return int64(t), nil
}

func (t *TriggerType) Scan(value any) error {
	if value == nil {
		*t = TriggerTypeUnspecified
		return nil
	}
	switch v := value.(type) {
	case int64:
		*t = TriggerType(v)
	case int:
		*t = TriggerType(v)
	case int32:
		*t = TriggerType(v)
	case uint8:
		*t = TriggerType(v)
	default:
		return fmt.Errorf("cannot scan %T into TriggerType", value)
	}
	return nil
}

// ==================== Entity ====================

type SyncPolicy struct {
	ID                 int            `gorm:"primarykey"`
	Name               string         `gorm:"column:name"`
	Description        string         `gorm:"column:description"`
	PolicyType         SyncPolicyType `gorm:"column:policy_type"`
	TriggerType        TriggerType    `gorm:"column:trigger_type"`
	RegistryID         int            `gorm:"column:registry_id"`
	LocalResourceName  string         `gorm:"column:local_resource_name"`
	LocalProjectName   string         `gorm:"column:local_project_name"`
	RemoteResourceName string         `gorm:"column:remote_resource_name"`
	RemoteProjectName  string         `gorm:"column:remote_project_name"`
	ResourceTypes      string         `gorm:"column:resource_types"`
	Bandwidth          string         `gorm:"column:bandwidth"`
	Cron               string         `gorm:"column:cron"`        // cron expression when TriggerType is scheduled
	LastRunAt          int64          `gorm:"column:last_run_at"` // ms since epoch; last claim / run bookkeeping
	NextRunAt          int64          `gorm:"column:next_run_at"` // ms; 0 = not scheduled
	IsOverwrite        bool           `gorm:"column:is_overwrite"`
	IsDisabled         bool           `gorm:"column:is_disabled"`
	CreatedAt          time.Time      `gorm:"column:created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at"`
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

// GetRemoteRegistryID returns the registry ID associated with the remote side.
func (p *SyncPolicy) GetRemoteRegistryID() int {
	return p.RegistryID
}

// GetLocalResourcePath returns the local resource path for job execution.
func (p *SyncPolicy) GetLocalResourcePath() string {
	if p.LocalProjectName == "" {
		return p.LocalResourceName
	}
	return p.LocalProjectName + "/" + p.LocalResourceName
}

// GetRemoteResourcePath returns the remote resource path for job execution.
func (p *SyncPolicy) GetRemoteResourcePath() string {
	if p.RemoteProjectName == "" {
		return p.RemoteResourceName
	}
	return p.RemoteProjectName + "/" + p.RemoteResourceName
}

// IsPullBase returns true if this is a pull-based policy.
func (p *SyncPolicy) IsPullBase() bool {
	return p.PolicyType == SyncPolicyTypePull
}

// IsPushBase returns true if this is a push-based policy.
func (p *SyncPolicy) IsPushBase() bool {
	return p.PolicyType == SyncPolicyTypePush
}

// HasWildcardResourceName returns true when the policy targets all resources in the remote project.
func (p *SyncPolicy) HasWildcardResourceName() bool {
	return p.RemoteResourceName == "**" || p.RemoteResourceName == "*"
}

type ISyncPolicyRepo interface {
	CreateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	GetSyncPolicy(ctx context.Context, id int) (*SyncPolicy, error)
	UpdateSyncPolicy(ctx context.Context, policy *SyncPolicy) error
	DeleteSyncPolicy(ctx context.Context, id int) error
	ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]*SyncPolicy, int64, error)

	SelectDuePolicies(ctx context.Context, nowMs int64, limit int) ([]*SyncPolicy, error)
	AdvanceNextRunAtCAS(ctx context.Context, policyID int, snapshotMs, nextNextMs, nowMs int64) (claimed bool, err error)
}
