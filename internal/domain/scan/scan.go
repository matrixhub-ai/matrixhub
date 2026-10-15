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

// Package scan implements the model-artifact security scanning and admission
// domain: task lifecycle, file-level scanning orchestration, version verdict
// aggregation, download admission policy and audit events.
package scan

import (
	"context"
	"io"
	"time"
)

// TaskStatus is the lifecycle state of one scan task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskScanning  TaskStatus = "scanning"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// Verdict is the aggregated conclusion of a completed task.
type Verdict string

const (
	VerdictPass    Verdict = "pass"
	VerdictWarning Verdict = "warning"
	VerdictBlocked Verdict = "blocked"
	// VerdictUnknown marks not-yet-completed tasks.
	VerdictUnknown Verdict = ""
)

// VersionStatus is the externally visible security state of one revision,
// derived from its latest task. Unscanned / scanning / failed are NEVER
// collapsed into a safe value.
type VersionStatus string

const (
	StatusUnscanned VersionStatus = "unscanned"
	StatusScanning  VersionStatus = "scanning"
	StatusPass      VersionStatus = "pass"
	StatusWarning   VersionStatus = "warning"
	StatusBlocked   VersionStatus = "blocked"
	StatusFailed    VersionStatus = "failed"
)

// HFStatus maps a VersionStatus onto the Hugging Face securityRepoStatus
// vocabulary ("scanning" | "clean" | "infected" | "error"). Unscanned maps to
// "scanning": the platform intends to scan every revision, and an unscanned
// revision must never be presented as clean.
func (s VersionStatus) HFStatus() string {
	switch s {
	case StatusPass, StatusWarning:
		return "clean"
	case StatusBlocked:
		return "infected"
	case StatusFailed:
		return "error"
	default: // unscanned, scanning
		return "scanning"
	}
}

// Severities for file findings (ordered).
const (
	SevClean    = "clean"
	SevInfo     = "info"
	SevWarning  = "warning"
	SevCritical = "critical"
)

// Task identifies one scan of one repository revision.
type Task struct {
	ID              int64
	RepoType        string // models | datasets
	Project         string
	Name            string
	Revision        string // commit SHA — the scanning boundary
	Trigger         string // upload | sync | manual | rule_update
	Status          TaskStatus
	Verdict         Verdict
	Force           bool // manual rescan: bypass the digest cache
	CreatedBy       string
	Error           string
	FileCount       int
	BytesTotal      int64
	ScannerVersions map[string]string
	StartedAt       *time.Time
	FinishedAt      *time.Time
	ClaimOwner      string
	ClaimDeadline   time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RepoKey is the (type, project, name) repository coordinate.
type RepoKey struct {
	RepoType string
	Project  string
	Name     string
}

// FileResult is one scanner observation about one file within a task. A
// scanner with no findings writes a single SevClean row. Detail never carries
// payload bytes — only rule identifiers, references and bounded summaries.
type FileResult struct {
	ID             int64
	TaskID         int64
	Path           string
	Digest         string // sha256 of content
	FileType       string
	Size           int64
	ScannerID      string
	ScannerVersion string
	Severity       string
	Rule           string
	Detail         string
	CreatedAt      time.Time
}

// CachedFileResult is the reuse record for identical content.
type CachedFileResult struct {
	ScannerID      string
	ScannerVersion string
	Severity       string
	Rule           string
	Detail         string
}

// Policy controls download admission. Platform-wide default plus optional
// per-project override.
type Policy struct {
	ID                   int64
	Project              *string // nil = platform default
	Mode                 string  // enforce | audit | off
	BlockSeverity        string  // lowest severity that blocks: critical | high | medium
	OnPending            string  // block | allow
	OnFailed             string  // block | allow
	OnScannerUnavailable string  // block | allow (alias of OnFailed semantics for transport errors)
	UpdatedBy            string
	UpdatedAt            time.Time
}

// DefaultPolicy is the safe-by-default platform posture.
func DefaultPolicy() Policy {
	return Policy{
		Mode:                 "enforce",
		BlockSeverity:        "critical",
		OnPending:            "block",
		OnFailed:             "block",
		OnScannerUnavailable: "block",
	}
}

// AuditEvent records a governance-relevant action.
type AuditEvent struct {
	ID        int64
	RepoType  string
	Project   string
	Name      string
	Revision  string
	Path      *string
	Actor     string
	Action    string // task_created | task_completed | rescan | policy_decision | admission | review
	Decision  string // e.g. allow / deny / verdict
	Reason    string
	TaskID    int64
	CreatedAt time.Time
}

// AuditFilter narrows audit queries.
type AuditFilter struct {
	RepoType string
	Project  string
	Name     string
	Revision string
	Actor    string
	Action   string
	Limit    int
	Offset   int
}

// FileDesc describes one file in a revision tree.
type FileDesc struct {
	Path     string
	Size     int64
	BlobHash string // git blob sha1 or LFS sha256 oid (advisory)
}

// TreeReader lists and opens files of a repository revision (implemented by
// the repo layer on top of the git storage).
type TreeReader interface {
	// Files lists all files at the revision.
	Files(ctx context.Context, key RepoKey, revision string) ([]FileDesc, error)
	// Open streams one file's content at the revision.
	Open(ctx context.Context, key RepoKey, revision, path string) (io.ReadCloser, error)
	// ResolveRevision maps a branch/tag/short SHA to a full commit SHA
	// (empty revision → default branch).
	ResolveRevision(ctx context.Context, key RepoKey, revision string) (string, error)
}

// FileFinding is what a scanner reports for one file.
type FileFinding struct {
	Severity string
	Rule     string
	Detail   string
}

// Scanner is one detection capability. Implementations MUST be static (no
// deserialization, no import, no execution) and must bound their resource use.
type Scanner interface {
	ID() string
	// Version returns the scanner + rule version used for cache keying and
	// reports. Unavailable engines return an error (task fails, never clean).
	Version(ctx context.Context) (string, error)
	// Applicable reports whether the scanner should run on this file.
	Applicable(path, fileType string, size int64) bool
	// ScanFile analyzes the file. r is seekable; size is the content length.
	ScanFile(ctx context.Context, path, fileType string, size int64, r io.ReadSeeker) ([]FileFinding, error)
}

// Store is the persistence port for the scan domain.
type Store interface {
	CreateTask(ctx context.Context, t *Task) error
	// ClaimNextPending CAS-claims the oldest pending task for owner with the
	// given lease (for crash recovery).
	ClaimNextPending(ctx context.Context, owner string, lease time.Duration) (*Task, error)
	FinishTask(ctx context.Context, t *Task) error
	// ResetStaleClaims re-queues tasks whose claim lease expired (service
	// restart recovery). Returns the number of recovered tasks.
	ResetStaleClaims(ctx context.Context, now time.Time) (int, error)
	CancelTask(ctx context.Context, id int64) error

	SaveFileResult(ctx context.Context, fr FileResult) error
	SaveFileResults(ctx context.Context, frs []FileResult) error
	// CachedResults returns prior results for (digest, scanner, version).
	CachedResults(ctx context.Context, digest, scannerID, scannerVersion string) ([]CachedFileResult, error)

	// LatestTask returns the newest task for a revision (any status).
	LatestTask(ctx context.Context, key RepoKey, revision string) (*Task, error)
	// LatestCompletedTask returns the newest completed task for a revision.
	LatestCompletedTask(ctx context.Context, key RepoKey, revision string) (*Task, error)
	// FindingsOfTask returns the file results of a task grouped for reports.
	FindingsOfTask(ctx context.Context, taskID int64, limit, offset int) ([]FileResult, error)

	// ListPendingRevisions supports "which repos are mid-scan" queries.
	GetPolicy(ctx context.Context, project *string) (Policy, error)
	UpsertPolicy(ctx context.Context, p Policy) error

	AppendAudit(ctx context.Context, e AuditEvent) error
	ListAudit(ctx context.Context, f AuditFilter) ([]AuditEvent, error)
}

// Limits bounds one scan execution.
type Limits struct {
	TaskTimeout     time.Duration
	MaxFiles        int
	MaxFileSize     int64 // per file; larger files get a "skipped" finding
	MaxSpoolBytes   int64 // seeker spool cap for pickle/archive analysis
	MaxConcurrent   int   // processor parallelism (wired via jobserver config)
	ClamAVSocket    string
	ClamAVTCPServer string
}

// DefaultLimits returns production-safe execution bounds.
func DefaultLimits() Limits {
	return Limits{
		TaskTimeout:   30 * time.Minute,
		MaxFiles:      10_000,
		MaxFileSize:   4 << 30, // 4 GiB
		MaxSpoolBytes: 2 << 30, // 2 GiB
		MaxConcurrent: 2,
	}
}

// ServiceAPI is the consumer-facing subset of the scan service (used by API
// handlers so they depend on the port, not the implementation).
type ServiceAPI interface {
	AdmitDownload(ctx context.Context, key RepoKey, revision, actor string) (Decision, error)
	ResolveRevision(ctx context.Context, key RepoKey, revision string) (string, error)
	VersionStatus(ctx context.Context, key RepoKey, revision string) (VersionStatus, error)
	BuildReport(ctx context.Context, key RepoKey, revision string) (*RevisionReport, error)
	FileSeverityMap(ctx context.Context, key RepoKey, revision string) (map[string]string, error)
}

// FileSeverityMap returns the highest finding severity per file path for a
// revision's latest completed scan (empty map when not completed).
func (s *Service) FileSeverityMap(ctx context.Context, key RepoKey, revision string) (map[string]string, error) {
	out := map[string]string{}
	t, err := s.store.LatestTask(ctx, key, revision)
	if err != nil || t == nil || t.Status != TaskCompleted {
		return out, err
	}
	rows, err := s.store.FindingsOfTask(ctx, t.ID, 10000, 0)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if cur, ok := out[r.Path]; !ok || severityRank(r.Severity) > severityRank(cur) {
			out[r.Path] = r.Severity
		}
	}
	return out, nil
}

var _ ServiceAPI = (*Service)(nil)
