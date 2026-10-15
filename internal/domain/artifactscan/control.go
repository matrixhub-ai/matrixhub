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
	"errors"
	"time"
)

type Policy struct {
	BlockSeverity string `json:"block_severity"`
	OnPending     string `json:"on_pending"`
	OnFailure     string `json:"on_failure"`
}

func StrictPolicy() Policy { return Policy{"medium", "block", "block"} }
func (p Policy) Validate() error {
	if p.BlockSeverity != "medium" && p.BlockSeverity != "high" {
		return errors.New("block_severity must be medium or high")
	}
	if (p.OnPending != "block" && p.OnPending != "allow") || (p.OnFailure != "block" && p.OnFailure != "allow") {
		return errors.New("failure and pending actions must be block or allow")
	}
	return nil
}

type Decision struct {
	Allowed    bool   `json:"allowed"`
	Revision   string `json:"revision"`
	ScanStatus Status `json:"scan_status"`
	Reason     string `json:"reason"`
	Policy     Policy `json:"policy"`
}

func Evaluate(r Report, p Policy) Decision {
	d := Decision{Revision: r.Revision, ScanStatus: r.Status, Policy: p, Reason: "scan_not_admitted"}
	if p.Validate() != nil {
		d.Reason = "invalid_policy"
		return d
	}
	// Incomplete scans retain findings. Failure/pending overrides may relax the
	// completion requirement, but must not erase an established risk match.
	mediumRisk := false
	for _, file := range r.Files {
		if file.Status == Blocked {
			d.Reason = "high_risk_blocked"
			return d
		}
		for _, finding := range file.Findings {
			if finding.Severity == "high" {
				d.Reason = "high_risk_blocked"
				return d
			}
			mediumRisk = mediumRisk || finding.Severity == "medium"
		}
	}
	if mediumRisk && p.BlockSeverity == "medium" {
		d.Reason = "warning_policy"
		return d
	}
	switch r.Status {
	case Passed:
		d.Allowed = true
		d.Reason = "scan_passed"
	case Warning:
		d.Allowed = p.BlockSeverity == "high"
		d.Reason = "warning_policy"
	case Unscanned, Pending, Scanning:
		d.Allowed = p.OnPending == "allow"
		d.Reason = "pending_policy"
	case Failed:
		d.Allowed = p.OnFailure == "allow"
		d.Reason = "failure_policy"
	case Blocked:
		d.Reason = "high_risk_blocked"
	case Cancelled:
		d.Reason = "scan_cancelled"
	}
	return d
}

// Dataset scanning is outside this opt-in model feature. Still bind shared LFS
// objects to an authorized immutable dataset tree, instead of exposing raw OIDs.
func (s *Service) AdmitDatasetObject(ctx context.Context, repo, revision, oid, actor string) (Decision, error) {
	if err := validRevision(revision); err != nil {
		return Decision{}, err
	}
	files, err := s.source.Files(ctx, repo, revision)
	if err != nil {
		return Decision{}, err
	}
	d := Decision{Revision: revision, Reason: "object_not_bound_to_dataset_revision"}
	for _, file := range files {
		if file.ExpectedSHA256 == oid {
			d.Allowed = true
			d.Reason = "authorized_dataset_object"
			break
		}
	}
	err = s.record(ctx, AuditEvent{Repo: repo, Revision: revision, Actor: actor, Action: "admission:dataset-lfs", Detail: d.Reason + ":" + map[bool]string{true: "allow", false: "deny"}[d.Allowed]})
	return d, err
}

type AuditEvent struct {
	ID       uint64    `json:"id"`
	Repo     string    `json:"repo"`
	Revision string    `json:"revision"`
	Attempt  int64     `json:"attempt"`
	Actor    string    `json:"actor"`
	Action   string    `json:"action"`
	Detail   string    `json:"detail"`
	At       time.Time `json:"at"`
}
type ControlStore interface {
	Rescan(context.Context, string, string, string, bool) error
	Cancel(context.Context, string, string, string) error
	GetPolicy(context.Context, string) (Policy, bool, error)
	PutPolicy(context.Context, string, Policy, string) error
	Record(context.Context, AuditEvent) error
	Events(context.Context, string, int) ([]AuditEvent, error)
	Cached(context.Context, string, string, string, string) (FileResult, bool, error)
	Cache(context.Context, string, string, FileResult) error
}
type VersionedScanner interface {
	Identity(context.Context) (string, error)
}
type HistorySource interface {
	Revisions(context.Context, string, int) ([]string, error)
}

func (s *Service) Policy(ctx context.Context, repo string) (Policy, error) {
	if c, ok := s.store.(ControlStore); ok {
		p, found, err := c.GetPolicy(ctx, repo)
		if err != nil {
			return Policy{}, err
		}
		if found {
			return p, nil
		}
	}
	return StrictPolicy(), nil
}
func (s *Service) SetPolicy(ctx context.Context, repo string, p Policy, actor string) error {
	if err := p.Validate(); err != nil {
		return err
	}
	c, ok := s.store.(ControlStore)
	if !ok {
		return errors.New("policy store unavailable")
	}
	return c.PutPolicy(ctx, repo, p, actor)
}
func (s *Service) record(ctx context.Context, event AuditEvent) error {
	c, ok := s.store.(ControlStore)
	if !ok {
		return nil
	}
	event.At = time.Now().UTC()
	return c.Record(ctx, event)
}
func (s *Service) Decide(ctx context.Context, repo, revision, actor, channel string) (Decision, error) {
	r, err := s.Get(ctx, repo, revision)
	if err != nil {
		return Decision{}, err
	}
	p, err := s.Policy(ctx, repo)
	if err != nil {
		return Decision{}, err
	}
	d := Evaluate(r, p)
	err = s.record(ctx, AuditEvent{Repo: repo, Revision: revision, Attempt: r.Attempt, Actor: actor, Action: "admission:" + channel, Detail: d.Reason + ":" + map[bool]string{true: "allow", false: "deny"}[d.Allowed]})
	return d, err
}
func (s *Service) Rescan(ctx context.Context, repo, revision, actor string, force bool) error {
	if err := validRevision(revision); err != nil {
		return err
	}
	c, ok := s.store.(ControlStore)
	if !ok {
		return errors.New("rescan store unavailable")
	}
	return c.Rescan(ctx, repo, revision, actor, force)
}
func (s *Service) Cancel(ctx context.Context, repo, revision, actor string) error {
	c, ok := s.store.(ControlStore)
	if !ok {
		return errors.New("cancel store unavailable")
	}
	return c.Cancel(ctx, repo, revision, actor)
}
func (s *Service) Events(ctx context.Context, repo string) ([]AuditEvent, error) {
	c, ok := s.store.(ControlStore)
	if !ok {
		return nil, errors.New("audit store unavailable")
	}
	return c.Events(ctx, repo, 100)
}
func (s *Service) AdmitHistory(ctx context.Context, repo, actor, channel string) (Decision, error) {
	history, ok := s.source.(HistorySource)
	if !ok {
		return Decision{}, errors.New("history reader unavailable")
	}
	revs, err := history.Revisions(ctx, repo, 256)
	if err != nil {
		return Decision{}, err
	}
	return s.AdmitRevisions(ctx, repo, revs, actor, channel)
}

// AdmitRevisions checks the history of the exact immutable transport snapshot.
func (s *Service) AdmitRevisions(ctx context.Context, repo string, revs []string, actor, channel string) (Decision, error) {
	if len(revs) > 256 {
		return Decision{}, errors.New("reachable history exceeds admission budget")
	}
	for _, revision := range revs {
		if err := validRevision(revision); err != nil {
			return Decision{}, err
		}
	}
	if len(revs) == 0 {
		d := Decision{Allowed: true, Reason: "empty_repository", Policy: StrictPolicy()}
		err := s.record(ctx, AuditEvent{Repo: repo, Actor: actor, Action: "admission:" + channel, Detail: "empty_repository:allow"})
		return d, err
	}
	p, err := s.Policy(ctx, repo)
	if err != nil {
		return Decision{}, err
	}
	final := Decision{Allowed: true, Reason: "all_reachable_revisions_admitted", Policy: p}
	for _, revision := range revs {
		r, err := s.Get(ctx, repo, revision)
		if err != nil {
			return Decision{}, err
		}
		if r.Status == Unscanned {
			if err := s.Enqueue(ctx, repo, revision); err != nil {
				return Decision{}, err
			}
		}
		d := Evaluate(r, p)
		if !d.Allowed && final.Allowed {
			final = d
		}
	}
	err = s.record(ctx, AuditEvent{Repo: repo, Revision: final.Revision, Actor: actor, Action: "admission:" + channel, Detail: final.Reason + ":" + map[bool]string{true: "allow", false: "deny"}[final.Allowed]})
	return final, err
}
func (s *Service) AdmitObject(ctx context.Context, repo, revision, oid, actor string) (Decision, error) {
	if err := validRevision(revision); err != nil {
		return Decision{}, err
	}
	r, err := s.Get(ctx, repo, revision)
	if err != nil {
		return Decision{}, err
	}
	linked := false
	for _, file := range r.Files {
		if file.SHA256 == oid {
			linked = true
			break
		}
	}
	if !linked {
		d := Decision{Revision: revision, ScanStatus: r.Status, Reason: "object_not_bound_to_scanned_revision"}
		err := s.record(ctx, AuditEvent{Repo: repo, Revision: revision, Attempt: r.Attempt, Actor: actor, Action: "admission:lfs-object", Detail: d.Reason + ":deny"})
		return d, err
	}
	return s.Decide(ctx, repo, revision, actor, "lfs-object")
}

// Stable action identifiers let API consumers and translated UI use the same advice.
func fileRecommendation(file FileResult) string {
	mediumRisk := false
	for _, finding := range file.Findings {
		if finding.Severity == "high" {
			return "quarantine_and_replace"
		}
		mediumRisk = mediumRisk || finding.Severity == "medium"
	}
	if file.Status == Blocked {
		return "quarantine_and_replace"
	}
	if mediumRisk {
		return "manual_review"
	}
	switch file.Status {
	case Blocked:
		return "quarantine_and_replace"
	case Warning:
		return "manual_review"
	case Passed:
		return "distribute_by_policy"
	default:
		return "repair_and_rescan"
	}
}
