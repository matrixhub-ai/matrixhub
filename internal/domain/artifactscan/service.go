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

package artifactscan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

type Status string

const (
	Unscanned Status = "unscanned"
	Pending   Status = "pending"
	Scanning  Status = "scanning"
	Passed    Status = "passed"
	Warning   Status = "warning"
	Blocked   Status = "blocked"
	Failed    Status = "failed"
	Cancelled Status = "cancelled"
)

type Finding struct {
	Scanner  string `json:"scanner"`
	Version  string `json:"version"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
}

type FileResult struct {
	FileType          string    `json:"file_type,omitempty"`
	CheckedAt         time.Time `json:"checked_at,omitempty"`
	RecommendedAction string    `json:"recommended_action,omitempty"`
	Path              string    `json:"path"`
	SHA256            string    `json:"sha256"`
	Size              int64     `json:"size"`
	Status            Status    `json:"status"`
	Findings          []Finding `json:"findings"`
	Checks            []string  `json:"checks"`
	Error             string    `json:"error,omitempty"`
	Reused            bool      `json:"reused"`
	Ruleset           string    `json:"ruleset,omitempty"`
}

type Report struct {
	Repo      string       `json:"repo"`
	Revision  string       `json:"revision"`
	Status    Status       `json:"status"`
	Files     []FileResult `json:"files"`
	Error     string       `json:"error,omitempty"`
	UpdatedAt time.Time    `json:"updated_at"`
	Attempt   int64        `json:"attempt"`
	Ruleset   string       `json:"ruleset,omitempty"`
	Force     bool         `json:"force"`
}

// Ports keep the domain independent of GORM, Git storage and scan transports.
type Store interface {
	Get(context.Context, string, string) (Report, error)
	Enqueue(context.Context, string, string) error
	Claim(context.Context) (Report, bool, error)
	Save(context.Context, Report) error
	Recover(context.Context) error
}

type File struct {
	Path           string
	Size           int64
	ExpectedSHA256 string
	Open           func() (io.ReadCloser, error)
}

type Source interface {
	Files(context.Context, string, string) ([]File, error)
}
type Scanner interface {
	Scan(context.Context, string, []byte) (FileResult, error)
}
type StreamingScanner interface {
	ScanStream(context.Context, string, io.Reader, int64) (FileResult, error)
}

type Service struct {
	store   Store
	source  Source
	scanner Scanner
	limits  Limits
}

// Limits bound I/O and task lifetime independently of static-analyzer metadata.
type Limits struct {
	MaxFileBytes int64
	MaxFiles     int
	TaskTimeout  time.Duration
}

func DefaultLimits() Limits {
	return Limits{MaxFileBytes: 1 << 30, MaxFiles: 256, TaskTimeout: 10 * time.Minute}
}

func (l Limits) normalized() (Limits, error) {
	d := DefaultLimits()
	if l.MaxFileBytes == 0 {
		l.MaxFileBytes = d.MaxFileBytes
	}
	if l.MaxFiles == 0 {
		l.MaxFiles = d.MaxFiles
	}
	if l.TaskTimeout == 0 {
		l.TaskTimeout = d.TaskTimeout
	}
	if l.MaxFileBytes < 1 || l.MaxFileBytes > 1<<30 || l.MaxFiles < 1 || l.MaxFiles > 4096 || l.TaskTimeout < time.Second || l.TaskTimeout > time.Hour {
		return l, fmt.Errorf("invalid artifact scan limits (file <= 1 GiB, files <= 4096, timeout 1s..1h)")
	}
	return l, nil
}

func New(store Store, source Source, scanner Scanner) (*Service, error) {
	return NewWithLimits(store, source, scanner, DefaultLimits())
}

func NewWithLimits(store Store, source Source, scanner Scanner, limits Limits) (*Service, error) {
	limits, err := limits.normalized()
	if err != nil {
		return nil, err
	}
	if err := store.Recover(context.Background()); err != nil {
		return nil, err
	}
	s := &Service{store: store, source: source, scanner: scanner, limits: limits}
	return s, nil
}

func (s *Service) Get(ctx context.Context, repo, revision string) (Report, error) {
	r, err := s.store.Get(ctx, repo, revision)
	if err == nil && r.Ruleset != "" && r.Status != Pending && r.Status != Scanning && r.Status != Cancelled {
		if scanner, ok := s.scanner.(VersionedScanner); ok {
			identity, identityErr := scanner.Identity(ctx)
			if identityErr == nil && identity != r.Ruleset {
				if err = s.Rescan(ctx, repo, revision, "system:ruleset-change", true); err == nil {
					return s.store.Get(ctx, repo, revision)
				}
			}
		}
	}
	return r, err
}
func (s *Service) Enqueue(ctx context.Context, repo, revision string) error {
	if err := validRevision(revision); err != nil {
		return err
	}
	return s.store.Enqueue(ctx, repo, revision)
}
func validRevision(revision string) error {
	if len(revision) != 40 {
		return errors.New("scan requires a commit SHA")
	}
	if _, err := hex.DecodeString(revision); err != nil {
		return errors.New("invalid commit SHA")
	}
	return nil
}

// Strict prototype policy: only a completed, passed revision is admitted.
func Allowed(status Status) bool { return status == Passed }

func HFStatus(status Status) string {
	switch status {
	case Passed:
		return "clean"
	case Blocked, Warning:
		return "infected"
	case Failed, Cancelled:
		return "error"
	default:
		return "scanning"
	}
}

func (s *Service) ProcessNext(ctx context.Context) error {
	r, found, err := s.store.Claim(ctx)
	if err != nil || !found {
		return err
	}
	s.process(ctx, r)
	return nil
}

func (s *Service) process(parent context.Context, r Report) {
	ctx, cancel := context.WithTimeout(parent, s.limits.TaskTimeout)
	defer cancel()
	r.Status, r.Files = Passed, []FileResult{}
	if scanner, ok := s.scanner.(VersionedScanner); ok {
		identity, identityErr := scanner.Identity(ctx)
		if identityErr != nil {
			r.Status, r.Error = Failed, "scanner identity unavailable"
		} else {
			r.Ruleset = identity
		}
	}
	files, err := s.source.Files(ctx, r.Repo, r.Revision)
	if err != nil {
		r.Status, r.Error = Failed, "revision contents unavailable"
	}
	if len(files) > s.limits.MaxFiles {
		r.Status, r.Error = Failed, "prototype file-count limit exceeded"
	}
	if r.Status != Failed {
		for _, file := range files {
			current, stateErr := s.store.Get(ctx, r.Repo, r.Revision)
			if stateErr != nil {
				r.Status, r.Error = Failed, "task state unavailable"
				break
			}
			if r.Attempt > 0 && (current.Attempt != r.Attempt || current.Status == Cancelled) {
				return
			}
			result, scanErr := s.scanFile(ctx, file, r.Repo, r.Ruleset, r.Force)
			result.RecommendedAction = fileRecommendation(result)
			r.Files = append(r.Files, result)
			if scanErr != nil {
				r.Status, r.Error = Failed, "file scan incomplete; inspect per-file results"
				break
			}
			if result.Status == Blocked {
				r.Status = Blocked
			} else if result.Status == Warning && r.Status != Blocked {
				r.Status = Warning
			}
		}
	}
	r.UpdatedAt = time.Now().UTC()
	// Save terminal failure even if shutdown cancelled the scanner context.
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer saveCancel()
	if err := s.store.Save(saveCtx, r); err != nil {
		slog.Error("artifact scan result persistence failed", "repo", r.Repo, "revision", r.Revision, "error", err)
	}
}

func (s *Service) scanFile(ctx context.Context, file File, repo, ruleset string, force bool) (FileResult, error) {
	maxBytes := s.limits.MaxFileBytes
	stream, streaming := s.scanner.(StreamingScanner)
	result := FileResult{Path: file.Path, Size: file.Size, Status: Failed, Findings: []Finding{}, Checks: []string{}, FileType: "unknown", CheckedAt: time.Now().UTC()}
	if err := ctx.Err(); err != nil {
		result.Error = "scan_budget_exceeded"
		return result, err
	}
	if file.Size < 0 || file.Size > maxBytes || (!streaming && file.Size > 8*1024*1024) {
		result.Error = "file_size_limit"
		return result, errors.New("prototype file size limit")
	}
	reader, err := file.Open()
	if err != nil {
		result.Error = "artifact_unavailable"
		return result, err
	}
	defer func() { _ = reader.Close() }()
	hasher := sha256.New()
	var data []byte
	var actualSize int64
	if streaming {
		actualSize, err = io.Copy(hasher, io.LimitReader(contextReader{ctx, reader}, maxBytes+1))
	} else {
		data, err = io.ReadAll(io.LimitReader(reader, 8*1024*1024+1))
		actualSize = int64(len(data))
		_, _ = hasher.Write(data)
	}
	if err != nil || actualSize > maxBytes {
		result.Error = "read_failed_or_oversized"
		return result, errors.New("file unreadable or oversized")
	}
	result.SHA256 = hex.EncodeToString(hasher.Sum(nil))
	result.Size = actualSize
	if result.Size != file.Size || (file.ExpectedSHA256 != "" && result.SHA256 != file.ExpectedSHA256) {
		result.Error = "artifact_integrity_mismatch"
		return result, errors.New("artifact contents do not match immutable metadata")
	}
	if cache, ok := s.store.(ControlStore); ok && ruleset != "" && !force {
		cached, found, cacheErr := cache.Cached(ctx, repo, file.Path, result.SHA256, ruleset)
		if cacheErr != nil {
			result.Error = "cache_unavailable"
			return result, cacheErr
		}
		if found {
			cached.Reused = true
			return cached, nil
		}
	}
	var scanned FileResult
	if streaming {
		var scanReader io.ReadCloser
		scanReader, err = file.Open()
		if err == nil {
			scanned, err = stream.ScanStream(ctx, file.Path, io.LimitReader(scanReader, result.Size+1), result.Size)
			_ = scanReader.Close()
		}
	} else {
		scanned, err = s.scanner.Scan(ctx, file.Path, data)
	}
	// Completion, identity and persistence failures must not erase findings
	// already returned by the scanner. Admission treats them conservatively.
	result.Findings, result.Checks = scanned.Findings, scanned.Checks
	if scanned.FileType != "" {
		result.FileType = scanned.FileType
	}
	if err != nil {
		result.Error = "scanner_incomplete"
		var coded interface{ ScanFailureCode() string }
		if errors.As(err, &coded) {
			result.Error = coded.ScanFailureCode()
		}
		return result, err
	}
	if scanned.Status != Passed && scanned.Status != Warning && scanned.Status != Blocked {
		result.Error = "scanner_protocol_invalid"
		return result, errors.New("invalid scanner result")
	}
	if ruleset != "" && scanned.Ruleset != ruleset {
		result.Error = "scanner_ruleset_changed"
		return result, errors.New("scanner rules changed during this attempt")
	}
	if streaming && (scanned.SHA256 != result.SHA256 || scanned.Size != result.Size) {
		result.Error = "scanner_content_mismatch"
		return result, errors.New("scanner analyzed different content")
	}
	result.Status, result.Findings, result.Checks = scanned.Status, scanned.Findings, scanned.Checks
	result.Ruleset = scanned.Ruleset
	if scanned.FileType != "" {
		result.FileType = scanned.FileType
	}
	if cache, ok := s.store.(ControlStore); ok && ruleset != "" {
		if err := cache.Cache(ctx, repo, ruleset, result); err != nil {
			result.Status, result.Error = Failed, "cache_persistence_failed"
			return result, err
		}
	}
	return result, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
