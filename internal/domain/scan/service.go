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

package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// Service orchestrates the scan domain.
type Service struct {
	store    Store
	tree     TreeReader
	scanners []Scanner
	lim      Limits
}

func NewService(store Store, tree TreeReader, scanners []Scanner, lim Limits) *Service {
	return &Service{store: store, tree: tree, scanners: scanners, lim: lim}
}

// EnqueueRevision creates a scan task for one revision. Idempotent per
// revision while a task is pending or scanning.
func (s *Service) EnqueueRevision(ctx context.Context, key RepoKey, revision, trigger, actor string, force bool) (int64, error) {
	if latest, err := s.store.LatestTask(ctx, key, revision); err == nil && latest != nil && !force {
		if latest.Status == TaskPending || latest.Status == TaskScanning {
			return latest.ID, nil // already queued
		}
	}
	t := &Task{
		RepoType: key.RepoType, Project: key.Project, Name: key.Name,
		Revision: revision, Trigger: trigger, Status: TaskPending,
		Force: force, CreatedBy: actor,
	}
	if err := s.store.CreateTask(ctx, t); err != nil {
		return 0, err
	}
	_ = s.store.AppendAudit(ctx, AuditEvent{
		RepoType: key.RepoType, Project: key.Project, Name: key.Name,
		Revision: revision, Actor: actor, Action: "task_created",
		Decision: string(TaskPending), Reason: "trigger=" + trigger, TaskID: t.ID,
	})
	log.Infow("scan task enqueued", "repo", key.Project+"/"+key.Name,
		"revision", revision, "trigger", trigger, "task", t.ID)
	return t.ID, nil
}

// ExecuteNext claims and runs the oldest pending task. Returns the task when
// one ran (nil when the queue is empty).
func (s *Service) ExecuteNext(ctx context.Context, owner string) (*Task, error) {
	lease := s.lim.TaskTimeout + time.Minute
	t, err := s.store.ClaimNextPending(ctx, owner, lease)
	if err != nil || t == nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, s.lim.TaskTimeout)
	defer cancel()
	s.execute(runCtx, t)
	return t, nil
}

// execute runs one claimed task to completion and persists the outcome. Any
// scanner failure fails the whole task — a scan that could not run is never
// recorded as safe.
func (s *Service) execute(ctx context.Context, t *Task) {
	now := time.Now()
	t.Status = TaskScanning
	t.StartedAt = &now
	_ = s.store.FinishTask(ctx, t) // persist the scanning transition

	key := RepoKey{RepoType: t.RepoType, Project: t.Project, Name: t.Name}

	versions := map[string]string{}
	for _, sc := range s.scanners {
		v, err := sc.Version(ctx)
		if err != nil {
			s.failTask(ctx, t, fmt.Sprintf("scanner %s unavailable: %v", sc.ID(), err))
			return
		}
		versions[sc.ID()] = v
	}
	t.ScannerVersions = versions

	files, err := s.tree.Files(ctx, key, t.Revision)
	if err != nil {
		s.failTask(ctx, t, fmt.Sprintf("list revision tree: %v", err))
		return
	}
	if len(files) > s.lim.MaxFiles {
		s.failTask(ctx, t, fmt.Sprintf("revision has %d files, above cap %d", len(files), s.lim.MaxFiles))
		return
	}

	var (
		verdict     = VerdictPass
		fileCount   int
		bytesTotal  int64
		taskFailure error
	)
	for _, fd := range files {
		if ctx.Err() != nil {
			taskFailure = fmt.Errorf("cancelled: %w", ctx.Err())
			break
		}
		digest, size, err := s.digestFile(ctx, key, t.Revision, fd.Path)
		if err != nil {
			taskFailure = fmt.Errorf("digest %s: %w", fd.Path, err)
			break
		}
		fileCount++
		bytesTotal += size

		for _, sc := range s.scanners {
			results := s.runScanner(ctx, sc, t, fd.Path, digest, size)
			if results.taskErr != nil {
				taskFailure = fmt.Errorf("scanner %s on %s: %w", sc.ID(), fd.Path, results.taskErr)
				break
			}
			rowVerdict := SevClean
			for _, f := range results.findings {
				s.saveResult(t, fd.Path, digest, size, sc, f)
				if severityRank(f.Severity) > severityRank(rowVerdict) {
					rowVerdict = f.Severity
				}
			}
			if len(results.findings) == 0 {
				s.saveResult(t, fd.Path, digest, size, sc, FileFinding{Severity: SevClean})
			}
			if rowVerdict == SevCritical {
				verdict = VerdictBlocked
			} else if rowVerdict == SevWarning && verdict != VerdictBlocked {
				verdict = VerdictWarning
			}
		}
		if taskFailure != nil {
			break
		}
	}

	fin := time.Now()
	t.FinishedAt = &fin
	t.FileCount = fileCount
	t.BytesTotal = bytesTotal
	if taskFailure != nil {
		if errors.Is(taskFailure, context.DeadlineExceeded) || errors.Is(taskFailure, context.Canceled) {
			t.Status = TaskCancelled
		} else {
			t.Status = TaskFailed
		}
		t.Error = taskFailure.Error()
	} else {
		t.Status = TaskCompleted
		t.Verdict = verdict
	}
	if err := s.store.FinishTask(ctx, t); err != nil {
		log.Warnw("persist scan outcome failed", "task", t.ID, "error", err)
	}
	_ = s.store.AppendAudit(ctx, AuditEvent{
		RepoType: t.RepoType, Project: t.Project, Name: t.Name, Revision: t.Revision,
		Actor: t.CreatedBy, Action: "task_completed",
		Decision: string(t.Status) + "/" + string(t.Verdict),
		Reason:   t.Error, TaskID: t.ID,
	})
}

func (s *Service) failTask(ctx context.Context, t *Task, msg string) {
	now := time.Now()
	t.Status = TaskFailed
	t.Error = msg
	t.FinishedAt = &now
	_ = s.store.FinishTask(ctx, t)
	_ = s.store.AppendAudit(ctx, AuditEvent{
		RepoType: t.RepoType, Project: t.Project, Name: t.Name, Revision: t.Revision,
		Actor: t.CreatedBy, Action: "task_completed", Decision: string(TaskFailed),
		Reason: msg, TaskID: t.ID,
	})
	log.Warnw("scan task failed", "task", t.ID, "error", msg)
}

func (s *Service) saveResult(t *Task, path, digest string, size int64, sc Scanner, f FileFinding) {
	_ = s.store.SaveFileResult(context.Background(), FileResult{
		TaskID: t.ID, Path: path, Digest: digest, Size: size,
		ScannerID: sc.ID(), ScannerVersion: s.scannerVersion(t, sc),
		Severity: f.Severity, Rule: f.Rule, Detail: f.Detail,
	})
}

func (s *Service) scannerVersion(t *Task, sc Scanner) string {
	if t.ScannerVersions != nil {
		if v, ok := t.ScannerVersions[sc.ID()]; ok {
			return v
		}
	}
	return "unknown"
}

type scannerOutcome struct {
	findings []FileFinding
	taskErr  error
}

// runScanner executes one scanner for one file, honoring the digest cache.
// Cached results are copied into the current task so reports stay complete.
func (s *Service) runScanner(ctx context.Context, sc Scanner, t *Task, path, digest string, size int64) scannerOutcome {
	ver := s.scannerVersion(t, sc)
	if !t.Force {
		if cached, err := s.store.CachedResults(ctx, digest, sc.ID(), ver); err == nil && len(cached) > 0 {
			out := scannerOutcome{}
			for _, c := range cached {
				if c.Severity == SevClean && len(cached) == 1 {
					return scannerOutcome{} // single clean row is implicit
				}
				out.findings = append(out.findings, FileFinding{
					Severity: c.Severity, Rule: c.Rule, Detail: c.Detail + " (reused)",
				})
			}
			return out
		}
	}

	// Cheap applicability pre-check by extension happens inside scanners, but
	// opening is needed for content checks — scanners receive the stream.
	rc, err := s.tree.Open(ctx, RepoKey{RepoType: t.RepoType, Project: t.Project, Name: t.Name}, t.Revision, path)
	if err != nil {
		return scannerOutcome{taskErr: fmt.Errorf("open: %w", err)}
	}
	defer func() { _ = rc.Close() }()

	seeker, cleanup, err := s.ensureSeeker(rc, path)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return scannerOutcome{taskErr: fmt.Errorf("spool: %w", err)}
	}

	fileType := DetectFileType(seeker, path)
	if !sc.Applicable(path, fileType, size) {
		return scannerOutcome{} // not applicable: no rows, not a failure
	}
	findings, err := sc.ScanFile(ctx, path, fileType, size, seeker)
	if err != nil {
		return scannerOutcome{taskErr: err}
	}
	return scannerOutcome{findings: findings}
}

// ensureSeeker returns a seekable view of r, spooling to a temp file when the
// source is not seekable (bounded by MaxSpoolBytes).
func (s *Service) ensureSeeker(r io.Reader, path string) (io.ReadSeeker, func(), error) {
	if rs, ok := r.(io.ReadSeeker); ok {
		return rs, nil, nil
	}
	tmp, err := os.CreateTemp("", "mxscan-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.Remove(tmp.Name()) }
	n, err := io.Copy(tmp, io.LimitReader(r, s.lim.MaxSpoolBytes+1))
	if err != nil {
		_ = tmp.Close()
		cleanup()
		return nil, nil, err
	}
	if n > s.lim.MaxSpoolBytes {
		_ = tmp.Close()
		cleanup()
		return nil, nil, fmt.Errorf("%s exceeds spool cap %d bytes", path, s.lim.MaxSpoolBytes)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		_ = tmp.Close()
		cleanup()
		return nil, nil, err
	}
	return tmp, func() { _ = tmp.Close(); cleanup() }, nil
}

// DetectFileType is wired to the heuristic scanner's identification so the
// whole domain shares one typing function (overridden in scanners.go).
var DetectFileType = func(r io.ReadSeeker, name string) string { return "unknown" }

// digestFile streams the file once to compute its sha256 content digest.
func (s *Service) digestFile(ctx context.Context, key RepoKey, revision, path string) (string, int64, error) {
	rc, err := s.tree.Open(ctx, key, revision, path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = rc.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(rc, s.lim.MaxFileSize+1))
	if err != nil {
		return "", 0, err
	}
	if n > s.lim.MaxFileSize {
		return "", 0, fmt.Errorf("exceeds per-file cap %d bytes", s.lim.MaxFileSize)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// ResolveRevision maps a branch/tag/short name to a full commit SHA.
func (s *Service) ResolveRevision(ctx context.Context, key RepoKey, revision string) (string, error) {
	return s.tree.ResolveRevision(ctx, key, revision)
}

// VersionStatus derives the externally visible status of one revision from
// its latest task. No task → unscanned (never clean).
func (s *Service) VersionStatus(ctx context.Context, key RepoKey, revision string) (VersionStatus, error) {
	t, err := s.store.LatestTask(ctx, key, revision)
	if err != nil {
		return StatusUnscanned, err
	}
	if t == nil {
		return StatusUnscanned, nil
	}
	switch t.Status {
	case TaskPending, TaskScanning:
		return StatusScanning, nil
	case TaskFailed, TaskCancelled:
		return StatusFailed, nil
	case TaskCompleted:
		switch t.Verdict {
		case VerdictBlocked:
			return StatusBlocked, nil
		case VerdictWarning:
			return StatusWarning, nil
		default:
			return StatusPass, nil
		}
	}
	return StatusUnscanned, nil
}

// Decision is an admission outcome for one download attempt.
type Decision struct {
	Allow      bool
	Status     VersionStatus
	HTTPStatus int
	Code       string
	Message    string
	Revision   string
}

// AdmitDownload evaluates the admission policy for one (repo, revision).
// The revision must already be resolved to a commit SHA by the caller.
func (s *Service) AdmitDownload(ctx context.Context, key RepoKey, revision, actor string) (Decision, error) {
	d := Decision{Revision: revision}
	status, err := s.VersionStatus(ctx, key, revision)
	if err != nil {
		return d, err
	}
	d.Status = status

	policy, err := s.store.GetPolicy(ctx, &key.Project)
	if err != nil {
		return d, err
	}
	if policy.Mode == "off" {
		d.Allow = true
		d.Code = "ScanOff"
		s.auditAdmission(ctx, key, revision, actor, d, policy)
		return d, nil
	}

	blockAt := severityRank(policy.BlockSeverity)
	warnRank := severityRank(SevWarning)
	switch status {
	case StatusPass:
		d.Allow = true
		d.Code = "ScanPass"
	case StatusWarning:
		if warnRank >= blockAt {
			d.Allow = false
			d.HTTPStatus = 403
			d.Code = "RevisionBlocked"
			d.Message = fmt.Sprintf("revision %s is blocked by security policy: scan verdict is warning (policy blocks at %s severity)", shortRev(revision), policy.BlockSeverity)
		} else {
			d.Allow = true
			d.Code = "ScanWarning"
		}
	case StatusBlocked:
		d.Allow = false
		d.HTTPStatus = 403
		d.Code = "RevisionBlocked"
		d.Message = fmt.Sprintf("revision %s is blocked by security policy: malicious or dangerous content detected (see scan report)", shortRev(revision))
	case StatusScanning, StatusUnscanned:
		if policy.OnPending == "block" {
			d.Allow = false
			d.HTTPStatus = 403
			d.Code = "RevisionPendingScan"
			d.Message = fmt.Sprintf("revision %s has not completed security scanning yet; downloads are held by policy until the scan finishes", shortRev(revision))
		} else {
			d.Allow = true
			d.Code = "ScanPendingControlledAllow"
		}
	case StatusFailed:
		if policy.OnFailed == "block" {
			d.Allow = false
			d.HTTPStatus = 403
			d.Code = "RevisionScanFailed"
			d.Message = fmt.Sprintf("revision %s could not be scanned (scanner unavailable or scan error); downloads are held by policy", shortRev(revision))
		} else {
			d.Allow = true
			d.Code = "ScanFailedControlledAllow"
		}
	}
	s.auditAdmission(ctx, key, revision, actor, d, policy)
	return d, nil
}

// auditAdmission records policy decisions, deduplicating consecutive identical
// allows for the same revision to keep the audit trail useful, not noisy.
func (s *Service) auditAdmission(ctx context.Context, key RepoKey, revision, actor string, d Decision, policy Policy) {
	if d.Allow {
		prev, err := s.store.ListAudit(ctx, AuditFilter{
			RepoType: key.RepoType, Project: key.Project, Name: key.Name,
			Revision: revision, Action: "admission", Limit: 1,
		})
		if err == nil && len(prev) == 1 && prev[0].Decision == "allow/"+d.Code {
			return // unchanged
		}
	}
	decision := "deny/" + d.Code
	if d.Allow {
		decision = "allow/" + d.Code
	}
	_ = s.store.AppendAudit(ctx, AuditEvent{
		RepoType: key.RepoType, Project: key.Project, Name: key.Name, Revision: revision,
		Actor: actor, Action: "admission", Decision: decision,
		Reason: fmt.Sprintf("status=%s mode=%s blockAt=%s onPending=%s onFailed=%s",
			d.Status, policy.Mode, policy.BlockSeverity, policy.OnPending, policy.OnFailed),
	})
}

// FileSummary is the per-file security info surfaced through the HF API and
// the report API.
type FileSummary struct {
	Path     string   `json:"path"`
	Digest   string   `json:"digest"`
	FileType string   `json:"fileType,omitempty"`
	Size     int64    `json:"size"`
	Severity string   `json:"severity"`
	Rules    []string `json:"rules,omitempty"`
}

// RevisionReport assembles the explainable report of one revision.
type RevisionReport struct {
	RepoType        string
	Project         string
	Name            string
	Revision        string
	Status          VersionStatus
	Verdict         Verdict
	TaskID          int64
	Files           []FileSummary
	ScannerVersions map[string]string
	Counts          map[string]int
	ScannedAt       *time.Time
	Error           string
}

// BuildReport assembles the version-level report with per-file evidence.
func (s *Service) BuildReport(ctx context.Context, key RepoKey, revision string) (*RevisionReport, error) {
	rep := &RevisionReport{
		RepoType: key.RepoType, Project: key.Project, Name: key.Name, Revision: revision,
	}
	status, err := s.VersionStatus(ctx, key, revision)
	if err != nil {
		return nil, err
	}
	rep.Status = status
	t, err := s.store.LatestTask(ctx, key, revision)
	if err != nil {
		return nil, err
	}
	if t != nil {
		rep.TaskID = t.ID
		rep.ScannerVersions = t.ScannerVersions
		rep.ScannedAt = t.FinishedAt
		rep.Error = t.Error
		rep.Verdict = t.Verdict
	}
	if status == StatusPass || status == StatusWarning || status == StatusBlocked {
		rows, err := s.store.FindingsOfTask(ctx, t.ID, 10000, 0)
		if err != nil {
			return nil, err
		}
		byPath := map[string]*FileSummary{}
		order := []string{}
		rep.Counts = map[string]int{}
		for _, r := range rows {
			fs, ok := byPath[r.Path]
			if !ok {
				fs = &FileSummary{Path: r.Path, Digest: r.Digest, FileType: r.FileType, Size: r.Size, Severity: SevClean}
				byPath[r.Path] = fs
				order = append(order, r.Path)
			}
			if severityRank(r.Severity) > severityRank(fs.Severity) {
				fs.Severity = r.Severity
			}
			if r.Rule != "" {
				fs.Rules = append(fs.Rules, r.Rule)
			}
			if r.Severity != SevClean {
				rep.Counts[r.Severity]++
			}
		}
		for _, p := range order {
			rep.Files = append(rep.Files, *byPath[p])
		}
	}
	return rep, nil
}

// Rescan re-queues a revision bypassing the digest cache.
func (s *Service) Rescan(ctx context.Context, key RepoKey, revision, actor string) (int64, error) {
	return s.EnqueueRevision(ctx, key, revision, "manual", actor, true)
}

// severityRank orders severities for aggregation comparisons.
func severityRank(sev string) int {
	switch sev {
	case SevCritical:
		return 3
	case "high":
		return 3
	case SevWarning:
		return 2
	case "medium":
		return 2
	case SevInfo:
		return 1
	default:
		return 0
	}
}

func shortRev(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// GetPolicy / UpsertPolicy / AppendAudit / ListAudit expose governance
// operations used by the scan REST API.
func (s *Service) GetPolicy(ctx context.Context, project *string) (Policy, error) {
	return s.store.GetPolicy(ctx, project)
}

func (s *Service) UpsertPolicy(ctx context.Context, p Policy) error {
	return s.store.UpsertPolicy(ctx, p)
}

func (s *Service) AppendAudit(ctx context.Context, e AuditEvent) error {
	return s.store.AppendAudit(ctx, e)
}

func (s *Service) ListAudit(ctx context.Context, f AuditFilter) ([]AuditEvent, error) {
	return s.store.ListAudit(ctx, f)
}
