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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	lfserrors "github.com/git-lfs/git-lfs/v3/errors"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
)

type artifactScanRow struct {
	Repo      string `gorm:"primaryKey;size:255"`
	Revision  string `gorm:"primaryKey;size:40"`
	Status    artifactscan.Status
	Report    string
	UpdatedAt time.Time
	Attempt   int64 `gorm:"not null;default:1"`
	Force     bool
}

func (artifactScanRow) TableName() string { return "mh_prototype_artifact_scans" }

type ArtifactScanStore struct{ db *gorm.DB }

// Schema is installed by SQL migration 2. Store construction never modifies it.
func NewArtifactScanStore(db *gorm.DB) (*ArtifactScanStore, error) {
	for _, table := range []any{&artifactScanRow{}, &artifactScanAttemptRow{}, &artifactAuditRow{}, &artifactPolicyRow{}, &artifactCacheRow{}} {
		if !db.Migrator().HasTable(table) {
			return nil, errors.New("artifact security schema missing: apply database migration 2")
		}
	}
	for _, column := range []string{"attempt", "force"} {
		if !db.Migrator().HasColumn(&artifactScanRow{}, column) {
			return nil, errors.New("artifact security schema outdated: apply database migration 2")
		}
	}
	return &ArtifactScanStore{db: db}, nil
}

func (s *ArtifactScanStore) Get(ctx context.Context, repo, revision string) (artifactscan.Report, error) {
	r := artifactscan.Report{Repo: repo, Revision: revision, Status: artifactscan.Unscanned, Files: []artifactscan.FileResult{}}
	var row artifactScanRow
	query := s.db.WithContext(ctx).Where("repo = ? AND revision = ?", repo, revision).Limit(1).Find(&row)
	err := query.Error
	if err == nil && query.RowsAffected == 0 {
		return r, nil
	}
	if err != nil {
		return r, err
	}
	if row.Report != "" {
		if err = json.Unmarshal([]byte(row.Report), &r); err != nil {
			return r, err
		}
	}
	r.Status, r.UpdatedAt = row.Status, row.UpdatedAt
	r.Attempt, r.Force = row.Attempt, row.Force
	return r, nil
}

func (s *ArtifactScanStore) Enqueue(ctx context.Context, repo, revision string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&artifactScanRow{Repo: repo, Revision: revision, Status: artifactscan.Pending, Attempt: 1, UpdatedAt: time.Now().UTC()})
		if created.Error != nil {
			return created.Error
		}
		if created.RowsAffected == 0 {
			return nil
		}
		return auditTx(tx, artifactscan.AuditEvent{Repo: repo, Revision: revision, Attempt: 1, Actor: "system:upload", Action: "queued", At: time.Now().UTC()})
	})
}

func (s *ArtifactScanStore) Claim(ctx context.Context) (artifactscan.Report, bool, error) {
	var result artifactscan.Report
	var found bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row artifactScanRow
		query := tx.Where("status = ?", artifactscan.Pending).Order("updated_at ASC").Limit(1).Find(&row)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected == 0 {
			return nil
		}
		claimed := tx.Model(&artifactScanRow{}).Where("repo = ? AND revision = ? AND status = ? AND attempt = ?", row.Repo, row.Revision, artifactscan.Pending, row.Attempt).Updates(map[string]any{"status": artifactscan.Scanning, "updated_at": time.Now().UTC()})
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected == 0 {
			return nil
		}
		if err := auditTx(tx, artifactscan.AuditEvent{Repo: row.Repo, Revision: row.Revision, Attempt: row.Attempt, Actor: "system:worker", Action: "scanning", At: time.Now().UTC()}); err != nil {
			return err
		}
		result = artifactscan.Report{Repo: row.Repo, Revision: row.Revision, Status: artifactscan.Scanning, Attempt: row.Attempt, Force: row.Force}
		found = true
		return nil
	})
	return result, found && err == nil, err
}

func (s *ArtifactScanStore) Save(ctx context.Context, r artifactscan.Report) error {
	encoded, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&artifactScanRow{}).Where("repo = ? AND revision = ? AND attempt = ? AND status = ?", r.Repo, r.Revision, r.Attempt, artifactscan.Scanning).Updates(map[string]any{"status": r.Status, "report": string(encoded), "updated_at": r.UpdatedAt})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return nil
		}
		if err := tx.Create(&artifactScanAttemptRow{Repo: r.Repo, Revision: r.Revision, Attempt: r.Attempt, Report: string(encoded), UpdatedAt: r.UpdatedAt}).Error; err != nil {
			return err
		}
		return auditTx(tx, artifactscan.AuditEvent{Repo: r.Repo, Revision: r.Revision, Attempt: r.Attempt, Actor: "system:worker", Action: string(r.Status), Detail: r.Error, At: r.UpdatedAt})
	})
}

func (s *ArtifactScanStore) Recover(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []artifactScanRow
		if err := tx.Where("status = ?", artifactscan.Scanning).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			now := time.Now().UTC()
			if err := tx.Model(&artifactScanRow{}).Where("repo = ? AND revision = ? AND attempt = ? AND status = ?", row.Repo, row.Revision, row.Attempt, artifactscan.Scanning).Updates(map[string]any{"status": artifactscan.Pending, "attempt": row.Attempt + 1, "report": "", "updated_at": now}).Error; err != nil {
				return err
			}
			if err := auditTx(tx, artifactscan.AuditEvent{Repo: row.Repo, Revision: row.Revision, Attempt: row.Attempt + 1, Actor: "system:restart", Action: "recovered", Detail: "interrupted scan queued again", At: now}); err != nil {
				return err
			}
		}
		return nil
	})
}

type ArtifactSource struct {
	Storage *storage.Storage
	LFS     lfs.Storage
}

func (s *ArtifactSource) Files(ctx context.Context, name, revision string) ([]artifactscan.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := s.Storage.ResolvePath(name)
	if path == "" {
		return nil, errors.New("invalid repository path")
	}
	r, err := repository.Open(path)
	if err != nil {
		return nil, err
	}
	entries, err := r.Tree(revision, "", &repository.TreeOptions{Recursive: true})
	if err != nil {
		return nil, err
	}
	files := make([]artifactscan.File, 0, len(entries))
	for _, entry := range entries {
		if entry.Type() != repository.EntryTypeFile {
			continue
		}
		blob, err := entry.Blob()
		if err != nil {
			return nil, err
		}
		file := artifactscan.File{Path: entry.Path(), Size: blob.Size(), Open: blob.NewReader}
		pointer, err := blob.LFSPointer()
		if err != nil && !lfserrors.IsNotAPointerError(err) {
			return nil, err
		}
		if pointer != nil {
			file.Size = pointer.Size()
			file.ExpectedSHA256 = pointer.OID()
			file.Open = func() (io.ReadCloser, error) {
				getter, ok := s.LFS.(lfs.Getter)
				if !ok {
					return nil, errors.New("LFS scanner read unavailable")
				}
				content, _, err := getter.Get(pointer.OID())
				return content, err
			}
		}
		files = append(files, file)
	}
	return files, nil
}

type ArtifactHTTPScanner struct {
	Endpoint string
	Client   *http.Client
}

func (s *ArtifactHTTPScanner) Scan(ctx context.Context, path string, data []byte) (artifactscan.FileResult, error) {
	return s.ScanStream(ctx, path, bytes.NewReader(data), int64(len(data)))
}

func (s *ArtifactHTTPScanner) ScanStream(ctx context.Context, path string, data io.Reader, size int64) (artifactscan.FileResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.Endpoint, "/")+"/scan?path="+url.QueryEscape(path), data)
	if err != nil {
		return artifactscan.FileResult{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = size
	resp, err := s.Client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return artifactscan.FileResult{}, artifactScannerFailure("scanner_timeout")
		}
		return artifactscan.FileResult{}, artifactScannerFailure("scanner_unavailable")
	}
	defer func() { _ = resp.Body.Close() }()
	var result artifactscan.FileResult
	err = json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&result)
	if resp.StatusCode != http.StatusOK {
		if err != nil {
			return artifactscan.FileResult{}, artifactScannerFailure("scanner_incomplete")
		}
		return result, artifactScannerFailure(safeScannerFailure(result.Error))
	}
	return result, err
}

type artifactScannerFailure string

func (e artifactScannerFailure) Error() string           { return string(e) }
func (e artifactScannerFailure) ScanFailureCode() string { return string(e) }

func safeScannerFailure(code string) string {
	switch code {
	case "file_size_limit", "pickle_metadata_limit", "archive_expansion_limit", "archive_member_limit", "archive_ratio_limit", "encrypted_archive", "unsupported_serialization", "static_analysis_incomplete", "scanner_timeout", "scanner_busy", "scanner_limit_exceeded", "scanner_unavailable", "truncated_request", "scan_budget_exceeded":
		return code
	default:
		return "scanner_incomplete"
	}
}
