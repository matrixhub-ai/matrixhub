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
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/matrixhub-ai/matrixhub/internal/domain/scan"
)

// GORM row types for the scan tables.

type ScanTaskRow struct {
	ID              int64      `gorm:"column:id;primaryKey"`
	RepoType        string     `gorm:"column:repo_type"`
	Project         string     `gorm:"column:project"`
	Name            string     `gorm:"column:name"`
	Revision        string     `gorm:"column:revision"`
	Trigger         string     `gorm:"column:trigger"`
	Status          string     `gorm:"column:status"`
	Verdict         string     `gorm:"column:verdict"`
	Force           bool       `gorm:"column:force"`
	CreatedBy       string     `gorm:"column:created_by"`
	Error           string     `gorm:"column:error"`
	FileCount       int        `gorm:"column:file_count"`
	BytesTotal      int64      `gorm:"column:bytes_total"`
	ScannerVersions string     `gorm:"column:scanner_versions"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	ClaimOwner      string     `gorm:"column:claim_owner"`
	ClaimDeadline   *time.Time `gorm:"column:claim_deadline"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (ScanTaskRow) TableName() string { return "scan_tasks" }

type ScanFileResultRow struct {
	ID             int64     `gorm:"column:id;primaryKey"`
	TaskID         int64     `gorm:"column:task_id"`
	Path           string    `gorm:"column:path"`
	Digest         string    `gorm:"column:digest"`
	FileType       string    `gorm:"column:file_type"`
	Size           int64     `gorm:"column:size"`
	ScannerID      string    `gorm:"column:scanner_id"`
	ScannerVersion string    `gorm:"column:scanner_version"`
	Severity       string    `gorm:"column:severity"`
	Rule           string    `gorm:"column:rule"`
	Detail         string    `gorm:"column:detail"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (ScanFileResultRow) TableName() string { return "scan_file_results" }

type ScanPolicyRow struct {
	ID                   int64     `gorm:"column:id;primaryKey"`
	Project              *string   `gorm:"column:project"`
	Mode                 string    `gorm:"column:mode"`
	BlockSeverity        string    `gorm:"column:block_severity"`
	OnPending            string    `gorm:"column:on_pending"`
	OnFailed             string    `gorm:"column:on_failed"`
	OnScannerUnavailable string    `gorm:"column:on_scanner_unavailable"`
	UpdatedBy            string    `gorm:"column:updated_by"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
	CreatedAt            time.Time `gorm:"column:created_at"`
}

func (ScanPolicyRow) TableName() string { return "scan_policies" }

type ScanAuditEventRow struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	RepoType  string    `gorm:"column:repo_type"`
	Project   string    `gorm:"column:project"`
	Name      string    `gorm:"column:name"`
	Revision  string    `gorm:"column:revision"`
	Path      *string   `gorm:"column:path"`
	Actor     string    `gorm:"column:actor"`
	Action    string    `gorm:"column:action"`
	Decision  string    `gorm:"column:decision"`
	Reason    string    `gorm:"column:reason"`
	TaskID    int64     `gorm:"column:task_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (ScanAuditEventRow) TableName() string { return "scan_audit_events" }

// ScanStore implements scan.Store on GORM.
type ScanStore struct {
	db *gorm.DB
}

func NewScanStore(db *gorm.DB) *ScanStore {
	return &ScanStore{db: db}
}

func taskToRow(t *scan.Task) ScanTaskRow {
	versions := ""
	if t.ScannerVersions != nil {
		b, _ := json.Marshal(t.ScannerVersions)
		versions = string(b)
	}
	return ScanTaskRow{
		ID: t.ID, RepoType: t.RepoType, Project: t.Project, Name: t.Name,
		Revision: t.Revision, Trigger: t.Trigger, Status: string(t.Status),
		Verdict: string(t.Verdict), Force: t.Force, CreatedBy: t.CreatedBy,
		Error: t.Error, FileCount: t.FileCount, BytesTotal: t.BytesTotal,
		ScannerVersions: versions, StartedAt: t.StartedAt, FinishedAt: t.FinishedAt,
		ClaimOwner: t.ClaimOwner, ClaimDeadline: &t.ClaimDeadline,
	}
}

func rowToTask(r ScanTaskRow) *scan.Task {
	t := &scan.Task{
		ID: r.ID, RepoType: r.RepoType, Project: r.Project, Name: r.Name,
		Revision: r.Revision, Trigger: r.Trigger, Status: scan.TaskStatus(r.Status),
		Verdict: scan.Verdict(r.Verdict), Force: r.Force, CreatedBy: r.CreatedBy,
		Error: r.Error, FileCount: r.FileCount, BytesTotal: r.BytesTotal,
		StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
		ClaimOwner: r.ClaimOwner, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.ClaimDeadline != nil {
		t.ClaimDeadline = *r.ClaimDeadline
	}
	if r.ScannerVersions != "" {
		_ = json.Unmarshal([]byte(r.ScannerVersions), &t.ScannerVersions)
	}
	return t
}

func (s *ScanStore) CreateTask(ctx context.Context, t *scan.Task) error {
	row := taskToRow(t)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	t.ID = row.ID
	t.CreatedAt = row.CreatedAt
	return nil
}

func (s *ScanStore) ClaimNextPending(ctx context.Context, owner string, lease time.Duration) (*scan.Task, error) {
	var row ScanTaskRow
	deadline := time.Now().Add(lease)
	err := s.db.WithContext(ctx).
		Where("status = ?", string(scan.TaskPending)).
		Order("id ASC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// CAS the claim so two workers cannot take the same task.
	res := s.db.WithContext(ctx).Model(&ScanTaskRow{}).
		Where("id = ? AND status = ?", row.ID, string(scan.TaskPending)).
		Updates(map[string]any{
			"status":         string(scan.TaskScanning),
			"claim_owner":    owner,
			"claim_deadline": deadline,
			"started_at":     time.Now(),
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil // lost the race
	}
	return s.GetTask(ctx, row.ID)
}

func (s *ScanStore) GetTask(ctx context.Context, id int64) (*scan.Task, error) {
	var row ScanTaskRow
	if err := s.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return rowToTask(row), nil
}

func (s *ScanStore) FinishTask(ctx context.Context, t *scan.Task) error {
	row := taskToRow(t)
	return s.db.WithContext(ctx).Model(&ScanTaskRow{}).
		Where("id = ?", t.ID).
		Updates(map[string]any{
			"status":           row.Status,
			"verdict":          row.Verdict,
			"error":            row.Error,
			"file_count":       row.FileCount,
			"bytes_total":      row.BytesTotal,
			"scanner_versions": row.ScannerVersions,
			"started_at":       row.StartedAt,
			"finished_at":      row.FinishedAt,
			"claim_owner":      row.ClaimOwner,
			"claim_deadline":   row.ClaimDeadline,
		}).Error
}

func (s *ScanStore) ResetStaleClaims(ctx context.Context, now time.Time) (int, error) {
	res := s.db.WithContext(ctx).Model(&ScanTaskRow{}).
		Where("status = ? AND claim_deadline IS NOT NULL AND claim_deadline < ?",
			string(scan.TaskScanning), now).
		Updates(map[string]any{
			"status":         string(scan.TaskPending),
			"claim_owner":    "",
			"claim_deadline": nil,
		})
	return int(res.RowsAffected), res.Error
}

func (s *ScanStore) CancelTask(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Model(&ScanTaskRow{}).
		Where("id = ? AND status IN ?", id, []string{string(scan.TaskPending), string(scan.TaskScanning)}).
		Update("status", string(scan.TaskCancelled)).Error
}

func (s *ScanStore) SaveFileResult(ctx context.Context, fr scan.FileResult) error {
	row := fileResultToRow(fr)
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *ScanStore) SaveFileResults(ctx context.Context, frs []scan.FileResult) error {
	if len(frs) == 0 {
		return nil
	}
	rows := make([]ScanFileResultRow, 0, len(frs))
	for _, fr := range frs {
		rows = append(rows, fileResultToRow(fr))
	}
	return s.db.WithContext(ctx).CreateInBatches(rows, 500).Error
}

func fileResultToRow(fr scan.FileResult) ScanFileResultRow {
	return ScanFileResultRow{
		TaskID: fr.TaskID, Path: fr.Path, Digest: fr.Digest, FileType: fr.FileType,
		Size: fr.Size, ScannerID: fr.ScannerID, ScannerVersion: fr.ScannerVersion,
		Severity: fr.Severity, Rule: fr.Rule, Detail: fr.Detail, CreatedAt: fr.CreatedAt,
	}
}

func (s *ScanStore) CachedResults(ctx context.Context, digest, scannerID, scannerVersion string) ([]scan.CachedFileResult, error) {
	var rows []ScanFileResultRow
	err := s.db.WithContext(ctx).
		Where("digest = ? AND scanner_id = ? AND scanner_version = ?", digest, scannerID, scannerVersion).
		Order("id DESC").Limit(64).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]scan.CachedFileResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, scan.CachedFileResult{
			ScannerID: r.ScannerID, ScannerVersion: r.ScannerVersion,
			Severity: r.Severity, Rule: r.Rule, Detail: r.Detail,
		})
	}
	return out, nil
}

func (s *ScanStore) LatestTask(ctx context.Context, key scan.RepoKey, revision string) (*scan.Task, error) {
	return s.latestTaskWhere(ctx, key, revision, "status IS NOT NULL")
}

func (s *ScanStore) LatestCompletedTask(ctx context.Context, key scan.RepoKey, revision string) (*scan.Task, error) {
	return s.latestTaskWhere(ctx, key, revision, "status = ?", string(scan.TaskCompleted))
}

func (s *ScanStore) latestTaskWhere(ctx context.Context, key scan.RepoKey, revision, cond string, args ...any) (*scan.Task, error) {
	q := s.db.WithContext(ctx).
		Where("repo_type = ? AND project = ? AND name = ? AND revision = ?",
			key.RepoType, key.Project, key.Name, revision)
	if len(args) > 0 {
		q = q.Where(cond, args...)
	}
	var row ScanTaskRow
	err := q.Order("id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rowToTask(row), nil
}

func (s *ScanStore) FindingsOfTask(ctx context.Context, taskID int64, limit, offset int) ([]scan.FileResult, error) {
	var rows []ScanFileResultRow
	q := s.db.WithContext(ctx).Where("task_id = ?", taskID).Order("path ASC, id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]scan.FileResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, scan.FileResult{
			ID: r.ID, TaskID: r.TaskID, Path: r.Path, Digest: r.Digest, FileType: r.FileType,
			Size: r.Size, ScannerID: r.ScannerID, ScannerVersion: r.ScannerVersion,
			Severity: r.Severity, Rule: r.Rule, Detail: r.Detail, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (s *ScanStore) GetPolicy(ctx context.Context, project *string) (scan.Policy, error) {
	var row ScanPolicyRow
	err := s.db.WithContext(ctx).
		Where("project IS ?", project).
		First(&row).Error // project NULL matches the platform default via `IS NULL`
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if project != nil {
			// fall back to the platform default
			return s.GetPolicy(ctx, nil)
		}
		def := scan.DefaultPolicy()
		return def, nil
	}
	if err != nil {
		return scan.Policy{}, err
	}
	return policyFromRow(row), nil
}

func policyFromRow(r ScanPolicyRow) scan.Policy {
	return scan.Policy{
		ID: r.ID, Project: r.Project, Mode: r.Mode, BlockSeverity: r.BlockSeverity,
		OnPending: r.OnPending, OnFailed: r.OnFailed, OnScannerUnavailable: r.OnScannerUnavailable,
		UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt,
	}
}

func (s *ScanStore) UpsertPolicy(ctx context.Context, p scan.Policy) error {
	row := ScanPolicyRow{
		Project: p.Project, Mode: p.Mode, BlockSeverity: p.BlockSeverity,
		OnPending: p.OnPending, OnFailed: p.OnFailed,
		OnScannerUnavailable: p.OnScannerUnavailable, UpdatedBy: p.UpdatedBy,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "project"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"mode", "block_severity", "on_pending", "on_failed",
			"on_scanner_unavailable", "updated_by", "updated_at",
		}),
	}).Create(&row).Error
}

func (s *ScanStore) AppendAudit(ctx context.Context, e scan.AuditEvent) error {
	row := ScanAuditEventRow{
		RepoType: e.RepoType, Project: e.Project, Name: e.Name, Revision: e.Revision,
		Path: e.Path, Actor: e.Actor, Action: e.Action, Decision: e.Decision,
		Reason: e.Reason, TaskID: e.TaskID, CreatedAt: time.Now().UTC(),
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *ScanStore) ListAudit(ctx context.Context, f scan.AuditFilter) ([]scan.AuditEvent, error) {
	q := s.db.WithContext(ctx).Model(&ScanAuditEventRow{})
	if f.RepoType != "" {
		q = q.Where("repo_type = ?", f.RepoType)
	}
	if f.Project != "" {
		q = q.Where("project = ?", f.Project)
	}
	if f.Name != "" {
		q = q.Where("name = ?", f.Name)
	}
	if f.Revision != "" {
		q = q.Where("revision = ?", f.Revision)
	}
	if f.Actor != "" {
		q = q.Where("actor = ?", f.Actor)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []ScanAuditEventRow
	if err := q.Order("id DESC").Limit(limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]scan.AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, scan.AuditEvent{
			ID: r.ID, RepoType: r.RepoType, Project: r.Project, Name: r.Name,
			Revision: r.Revision, Path: r.Path, Actor: r.Actor, Action: r.Action,
			Decision: r.Decision, Reason: r.Reason, TaskID: r.TaskID, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

var _ scan.Store = (*ScanStore)(nil)
