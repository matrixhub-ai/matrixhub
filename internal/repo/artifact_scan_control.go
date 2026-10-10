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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
)

type artifactScanAttemptRow struct {
	Repo      string `gorm:"primaryKey;size:255"`
	Revision  string `gorm:"primaryKey;size:40"`
	Attempt   int64  `gorm:"primaryKey"`
	Report    string
	UpdatedAt time.Time
}

func (artifactScanAttemptRow) TableName() string { return "mh_prototype_scan_attempts" }

type artifactAuditRow struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	Repo     string `gorm:"size:255;index"`
	Revision string `gorm:"size:40"`
	Attempt  int64
	Actor    string
	Action   string
	Detail   string
	At       time.Time
}

func (artifactAuditRow) TableName() string { return "mh_prototype_scan_audit" }

type artifactPolicyRow struct {
	Repo   string `gorm:"primaryKey;size:255"`
	Policy string
}

func (artifactPolicyRow) TableName() string { return "mh_prototype_scan_policies" }

type artifactCacheRow struct {
	Repo    string `gorm:"primaryKey;size:255"`
	Context string `gorm:"primaryKey;size:64"`
	Digest  string `gorm:"primaryKey;size:64"`
	Ruleset string `gorm:"primaryKey;size:64"`
	Result  string
}

func (artifactCacheRow) TableName() string { return "mh_prototype_scan_cache" }
func auditTx(tx *gorm.DB, e artifactscan.AuditEvent) error {
	return tx.Create(&artifactAuditRow{Repo: e.Repo, Revision: e.Revision, Attempt: e.Attempt, Actor: e.Actor, Action: e.Action, Detail: e.Detail, At: e.At}).Error
}
func (s *ArtifactScanStore) Record(ctx context.Context, e artifactscan.AuditEvent) error {
	return auditTx(s.db.WithContext(ctx), e)
}
func (s *ArtifactScanStore) Events(ctx context.Context, repo string, limit int) ([]artifactscan.AuditEvent, error) {
	var rows []artifactAuditRow
	scope := strings.SplitN(repo, "/", 2)[0]
	err := s.db.WithContext(ctx).Where("repo = ? OR (repo = ? AND action = ?)", repo, scope, "policy-updated").Order("id DESC").Limit(limit).Find(&rows).Error
	events := make([]artifactscan.AuditEvent, 0, len(rows))
	for _, r := range rows {
		events = append(events, artifactscan.AuditEvent{ID: r.ID, Repo: r.Repo, Revision: r.Revision, Attempt: r.Attempt, Actor: r.Actor, Action: r.Action, Detail: r.Detail, At: r.At})
	}
	return events, err
}
func (s *ArtifactScanStore) Rescan(ctx context.Context, repo, revision, actor string, force bool) error {
	if err := s.Enqueue(ctx, repo, revision); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row artifactScanRow
		if err := tx.Where("repo = ? AND revision = ?", repo, revision).First(&row).Error; err != nil {
			return err
		}
		if (row.Status == artifactscan.Pending || row.Status == artifactscan.Scanning) && !force {
			return nil
		}
		updated := tx.Model(&artifactScanRow{}).Where("repo = ? AND revision = ? AND attempt = ?", repo, revision, row.Attempt).Updates(map[string]any{"status": artifactscan.Pending, "attempt": row.Attempt + 1, "force": force, "report": "", "updated_at": time.Now().UTC()})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return errors.New("task changed; retry rescan")
		}
		return auditTx(tx, artifactscan.AuditEvent{Repo: repo, Revision: revision, Attempt: row.Attempt + 1, Actor: actor, Action: "rescan", Detail: map[bool]string{true: "forced", false: "reuse-eligible"}[force], At: time.Now().UTC()})
	})
}
func (s *ArtifactScanStore) Cancel(ctx context.Context, repo, revision, actor string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row artifactScanRow
		if err := tx.Where("repo = ? AND revision = ?", repo, revision).First(&row).Error; err != nil {
			return err
		}
		if row.Status != artifactscan.Pending && row.Status != artifactscan.Scanning {
			return errors.New("only pending or scanning tasks can be cancelled")
		}
		r := artifactscan.Report{Repo: repo, Revision: revision, Attempt: row.Attempt, Status: artifactscan.Cancelled, Error: "cancelled by operator", UpdatedAt: time.Now().UTC()}
		encoded, err := json.Marshal(r)
		if err != nil {
			return err
		}
		update := tx.Model(&artifactScanRow{}).Where("repo = ? AND revision = ? AND attempt = ? AND status = ?", repo, revision, row.Attempt, row.Status).Updates(map[string]any{"status": r.Status, "report": string(encoded), "updated_at": r.UpdatedAt})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return errors.New("task changed; retry cancel")
		}
		if err := tx.Create(&artifactScanAttemptRow{Repo: repo, Revision: revision, Attempt: row.Attempt, Report: string(encoded), UpdatedAt: r.UpdatedAt}).Error; err != nil {
			return err
		}
		return auditTx(tx, artifactscan.AuditEvent{Repo: repo, Revision: revision, Attempt: row.Attempt, Actor: actor, Action: "cancelled", At: r.UpdatedAt})
	})
}
func (s *ArtifactScanStore) GetPolicy(ctx context.Context, repo string) (artifactscan.Policy, bool, error) {
	repo = strings.SplitN(repo, "/", 2)[0]
	var row artifactPolicyRow
	query := s.db.WithContext(ctx).Where("repo = ?", repo).Limit(1).Find(&row)
	if query.Error != nil || query.RowsAffected == 0 {
		return artifactscan.Policy{}, false, query.Error
	}
	var p artifactscan.Policy
	err := json.Unmarshal([]byte(row.Policy), &p)
	if err == nil {
		err = p.Validate()
	}
	return p, true, err
}
func (s *ArtifactScanStore) PutPolicy(ctx context.Context, repo string, p artifactscan.Policy, actor string) error {
	scope := strings.SplitN(repo, "/", 2)[0]
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&artifactPolicyRow{Repo: scope, Policy: string(encoded)}).Error; err != nil {
			return err
		}
		return auditTx(tx, artifactscan.AuditEvent{Repo: scope, Actor: actor, Action: "policy-updated", Detail: string(encoded), At: time.Now().UTC()})
	})
}
func (s *ArtifactScanStore) Cached(ctx context.Context, repo, path, digest, ruleset string) (artifactscan.FileResult, bool, error) {
	key := sha256.Sum256([]byte(path))
	var row artifactCacheRow
	q := s.db.WithContext(ctx).Where("repo = ? AND context = ? AND digest = ? AND ruleset = ?", repo, hex.EncodeToString(key[:]), digest, ruleset).Limit(1).Find(&row)
	if q.Error != nil || q.RowsAffected == 0 {
		return artifactscan.FileResult{}, false, q.Error
	}
	var result artifactscan.FileResult
	if err := json.Unmarshal([]byte(row.Result), &result); err != nil {
		return result, false, err
	}
	if result.Path != path || result.SHA256 != digest || result.Ruleset != ruleset || result.Error != "" || (result.Status != artifactscan.Passed && result.Status != artifactscan.Warning && result.Status != artifactscan.Blocked) {
		return result, false, errors.New("invalid cached result")
	}
	return result, true, nil
}
func (s *ArtifactScanStore) Cache(ctx context.Context, repo, ruleset string, r artifactscan.FileResult) error {
	if ruleset == "" || r.Ruleset != ruleset || r.Error != "" || (r.Status != artifactscan.Passed && r.Status != artifactscan.Warning && r.Status != artifactscan.Blocked) {
		return errors.New("incomplete result cannot be cached")
	}
	key := sha256.Sum256([]byte(r.Path))
	r.Reused = false
	encoded, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&artifactCacheRow{Repo: repo, Context: hex.EncodeToString(key[:]), Digest: r.SHA256, Ruleset: ruleset, Result: string(encoded)}).Error
}
func (s *ArtifactSource) Revisions(ctx context.Context, name string, limit int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := s.Storage.ResolvePath(name)
	if path == "" {
		return nil, errors.New("invalid repository path")
	}
	return ArtifactRevisionsAtPath(ctx, path, limit)
}

// ArtifactRevisionsAtPath reads the captured snapshot, never mutable source refs.
func ArtifactRevisionsAtPath(ctx context.Context, path string, limit int) ([]string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	refs, err := repo.References()
	if err != nil {
		return nil, err
	}
	count := 0
	err = refs.ForEach(func(*plumbing.Reference) error {
		count++
		if count > limit {
			return errors.New("repository ref budget exceeded")
		}
		return ctx.Err()
	})
	refs.Close()
	if err != nil {
		return nil, err
	}
	commits, err := repo.Log(&git.LogOptions{All: true})
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer commits.Close()
	result := []string{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		commit, err := commits.Next()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		result = append(result, commit.Hash.String())
		if len(result) > limit {
			return nil, errors.New("reachable history exceeds admission budget")
		}
	}
}
func (s *ArtifactHTTPScanner) Identity(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.Endpoint, "/")+"/identity", nil)
	if err != nil {
		return "", err
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return "", errors.New("scanner identity unavailable")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != 200 {
		return "", errors.New("scanner identity unavailable")
	}
	var identity struct {
		Ruleset string `json:"ruleset"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&identity); err != nil {
		return "", err
	}
	if len(identity.Ruleset) != 64 {
		return "", errors.New("invalid scanner identity")
	}
	if _, err := hex.DecodeString(identity.Ruleset); err != nil {
		return "", errors.New("invalid scanner identity")
	}
	return identity.Ruleset, nil
}
