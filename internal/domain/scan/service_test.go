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
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

// ---- test doubles ----

type memStore struct {
	mu         sync.Mutex
	nextID     int64
	tasks      []*Task
	results    []FileResult
	policies   map[string]Policy // key: "" platform, else project
	audit      []AuditEvent
	cachedHits int
}

func newMemStore() *memStore { return &memStore{policies: map[string]Policy{}} }

func (m *memStore) CreateTask(_ context.Context, t *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	t.ID = m.nextID
	m.tasks = append(m.tasks, t)
	return nil
}

func (m *memStore) ClaimNextPending(_ context.Context, owner string, _ time.Duration) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.Status == TaskPending {
			t.Status = TaskScanning
			t.ClaimOwner = owner
			now := time.Now()
			t.StartedAt = &now
			return t, nil
		}
	}
	return nil, nil
}

func (m *memStore) FinishTask(_ context.Context, t *Task) error { return nil }

func (m *memStore) ResetStaleClaims(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

func (m *memStore) CancelTask(_ context.Context, id int64) error { return nil }

func (m *memStore) SaveFileResult(_ context.Context, fr FileResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.results = append(m.results, fr)
	return nil
}

func (m *memStore) SaveFileResults(_ context.Context, frs []FileResult) error {
	for _, fr := range frs {
		if err := m.SaveFileResult(context.Background(), fr); err != nil {
			return err
		}
	}
	return nil
}

func (m *memStore) CachedResults(_ context.Context, digest, scannerID, version string) ([]CachedFileResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []CachedFileResult
	for i := len(m.results) - 1; i >= 0; i-- {
		r := m.results[i]
		if r.Digest == digest && r.ScannerID == scannerID && r.ScannerVersion == version {
			out = append(out, CachedFileResult{
				ScannerID: r.ScannerID, ScannerVersion: r.ScannerVersion,
				Severity: r.Severity, Rule: r.Rule, Detail: r.Detail,
			})
		}
	}
	if len(out) > 0 {
		m.cachedHits++
	}
	return out, nil
}

func (m *memStore) LatestTask(_ context.Context, key RepoKey, rev string) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest *Task
	for _, t := range m.tasks {
		if t.RepoType == key.RepoType && t.Project == key.Project && t.Name == key.Name && t.Revision == rev {
			latest = t
		}
	}
	return latest, nil
}

func (m *memStore) LatestCompletedTask(_ context.Context, key RepoKey, rev string) (*Task, error) {
	t, _ := m.LatestTask(context.Background(), key, rev)
	if t != nil && t.Status == TaskCompleted {
		return t, nil
	}
	return nil, nil
}

func (m *memStore) FindingsOfTask(_ context.Context, taskID int64, _, _ int) ([]FileResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []FileResult
	for _, r := range m.results {
		if r.TaskID == taskID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memStore) GetPolicy(_ context.Context, project *string) (Policy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ""
	if project != nil {
		key = *project
	}
	if p, ok := m.policies[key]; ok {
		return p, nil
	}
	if project != nil {
		return m.policies[""], nil // project falls back to platform default
	}
	return DefaultPolicy(), nil
}

func (m *memStore) UpsertPolicy(_ context.Context, p Policy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ""
	if p.Project != nil {
		key = *p.Project
	}
	m.policies[key] = p
	return nil
}

func (m *memStore) AppendAudit(_ context.Context, e AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audit = append(m.audit, e)
	return nil
}

func (m *memStore) ListAudit(_ context.Context, f AuditFilter) ([]AuditEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []AuditEvent
	for i := len(m.audit) - 1; i >= 0 && len(out) < f.Limit; i-- {
		e := m.audit[i]
		if f.Revision != "" && e.Revision != f.Revision {
			continue
		}
		if f.Action != "" && e.Action != f.Action {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// fakeTree serves deterministic file content.
type fakeTree struct {
	files map[string]string // path → content
}

func (f *fakeTree) Files(_ context.Context, _ RepoKey, _ string) ([]FileDesc, error) {
	var out []FileDesc
	for p, c := range f.files {
		out = append(out, FileDesc{Path: p, Size: int64(len(c))})
	}
	return out, nil
}

func (f *fakeTree) Open(_ context.Context, _ RepoKey, _, path string) (io.ReadCloser, error) {
	c, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("no such file %s", path)
	}
	return io.NopCloser(bytes.NewReader([]byte(c))), nil
}

func (f *fakeTree) ResolveRevision(_ context.Context, _ RepoKey, rev string) (string, error) {
	return rev, nil
}

// fakeScanner returns configured findings and counts invocations.
type fakeScanner struct {
	id       string
	ver      string
	findings map[string][]FileFinding // path → findings
	calls    int
}

func (f *fakeScanner) ID() string                                { return f.id }
func (f *fakeScanner) Version(_ context.Context) (string, error) { return f.ver, nil }
func (f *fakeScanner) Applicable(_, _ string, _ int64) bool      { return true }
func (f *fakeScanner) ScanFile(_ context.Context, path, _ string, _ int64, _ io.ReadSeeker) ([]FileFinding, error) {
	f.calls++
	return f.findings[path], nil
}

// unavailScanner simulates a dead engine (clamd down).
type unavailScanner struct{}

func (u *unavailScanner) ID() string { return "clam" }
func (u *unavailScanner) Version(_ context.Context) (string, error) {
	return "", fmt.Errorf("connection refused")
}
func (u *unavailScanner) Applicable(_, _ string, _ int64) bool { return true }
func (u *unavailScanner) ScanFile(_ context.Context, _, _ string, _ int64, _ io.ReadSeeker) ([]FileFinding, error) {
	return nil, nil
}

func newTestService(store Store, tree TreeReader, scanners []Scanner) *Service {
	lim := DefaultLimits()
	lim.TaskTimeout = 5 * time.Second
	return NewService(store, tree, scanners, lim)
}

// ---- tests ----

func TestVersionStatusMapping(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{"a": "x"}}
	svc := newTestService(store, tree, []Scanner{&fakeScanner{id: "s", ver: "1"}})
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}

	if st, _ := svc.VersionStatus(context.Background(), key, "r1"); st != StatusUnscanned {
		t.Fatalf("no task → unscanned, got %s", st)
	}

	cases := []struct {
		status  TaskStatus
		verdict Verdict
		want    VersionStatus
	}{
		{TaskPending, VerdictUnknown, StatusScanning},
		{TaskScanning, VerdictUnknown, StatusScanning},
		{TaskFailed, VerdictUnknown, StatusFailed},
		{TaskCancelled, VerdictUnknown, StatusFailed},
		{TaskCompleted, VerdictPass, StatusPass},
		{TaskCompleted, VerdictWarning, StatusWarning},
		{TaskCompleted, VerdictBlocked, StatusBlocked},
	}
	for i, c := range cases {
		rev := fmt.Sprintf("r%d", i)
		store.tasks = append(store.tasks, &Task{
			RepoType: "models", Project: "p", Name: "n", Revision: rev,
			Status: c.status, Verdict: c.verdict,
		})
		if st, _ := svc.VersionStatus(context.Background(), key, rev); st != c.want {
			t.Fatalf("task %s/%s → %s, want %s", c.status, c.verdict, st, c.want)
		}
	}
}

func TestHFStatusNeverCollapsesUnsafeToClean(t *testing.T) {
	for _, s := range []VersionStatus{StatusUnscanned, StatusScanning, StatusFailed} {
		if s.HFStatus() == "clean" {
			t.Fatalf("%s must never map to clean", s)
		}
	}
	if StatusBlocked.HFStatus() != "infected" || StatusPass.HFStatus() != "clean" ||
		StatusFailed.HFStatus() != "error" || StatusScanning.HFStatus() != "scanning" {
		t.Fatal("HF status vocabulary mismatch")
	}
}

func TestExecuteAggregatesVerdictAndPersistsRows(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{
		"safe.txt": "hello",
		"evil.pkl": "picklestuff",
	}}
	sc := &fakeScanner{id: "static", ver: "v1", findings: map[string][]FileFinding{
		"evil.pkl": {{Severity: SevCritical, Rule: "test.evil"}},
	}}
	svc := newTestService(store, tree, []Scanner{sc})
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}

	if _, err := svc.EnqueueRevision(context.Background(), key, "rev1", "upload", "tester", false); err != nil {
		t.Fatal(err)
	}
	task, err := svc.ExecuteNext(context.Background(), "worker-1")
	if err != nil || task == nil {
		t.Fatalf("execute: %v %v", task, err)
	}
	if task.Status != TaskCompleted || task.Verdict != VerdictBlocked {
		t.Fatalf("expected completed/blocked, got %s/%s", task.Status, task.Verdict)
	}
	if task.FileCount != 2 {
		t.Fatalf("file count %d", task.FileCount)
	}
	rows, _ := store.FindingsOfTask(context.Background(), task.ID, 0, 0)
	if len(rows) != 2 { // safe.txt clean + evil.pkl critical
		t.Fatalf("file rows %d: %+v", len(rows), rows)
	}
	// audit trail must contain creation + completion
	if len(store.audit) < 2 {
		t.Fatalf("audit events %d", len(store.audit))
	}
}

func TestExecuteFailsWhenScannerUnavailable(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{"a": "x"}}
	svc := newTestService(store, tree, []Scanner{&unavailScanner{}})
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}
	_, _ = svc.EnqueueRevision(context.Background(), key, "rev1", "upload", "t", false)
	task, _ := svc.ExecuteNext(context.Background(), "w")
	if task == nil || task.Status != TaskFailed {
		t.Fatalf("scanner down must fail the task, got %+v", task)
	}
	if st, _ := svc.VersionStatus(context.Background(), key, "rev1"); st != StatusFailed {
		t.Fatalf("status %s, want failed", st)
	}
}

func TestDigestReuseSkipsScannerAndForceBypasses(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{"same.bin": "identical-bytes"}}
	sc := &fakeScanner{id: "s", ver: "v1"}
	svc := newTestService(store, tree, []Scanner{sc})
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}

	_, _ = svc.EnqueueRevision(context.Background(), key, "rev1", "upload", "t", false)
	task1, _ := svc.ExecuteNext(context.Background(), "w")
	if sc.calls != 1 {
		t.Fatalf("first scan calls = %d", sc.calls)
	}
	// Same content in a new revision: digest cache must avoid the scanner.
	_, _ = svc.EnqueueRevision(context.Background(), key, "rev2", "upload", "t", false)
	task2, _ := svc.ExecuteNext(context.Background(), "w")
	if task2 == nil || task2.Verdict != task1.Verdict {
		t.Fatalf("reused verdict mismatch: %+v vs %+v", task2, task1)
	}
	if sc.calls != 1 {
		t.Fatalf("digest reuse failed: scanner ran %d times", sc.calls)
	}
	// Manual rescan with force re-runs the scanner.
	if _, err := svc.Rescan(context.Background(), key, "rev2", "tester"); err != nil {
		t.Fatal(err)
	}
	if _, _ = svc.ExecuteNext(context.Background(), "w"); sc.calls != 2 {
		t.Fatalf("force rescan must re-run scanner, calls=%d", sc.calls)
	}
}

func TestEnqueueDedupesPendingTask(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{"a": "1"}}
	svc := newTestService(store, tree, nil)
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}
	id1, _ := svc.EnqueueRevision(context.Background(), key, "rev", "upload", "t", false)
	id2, _ := svc.EnqueueRevision(context.Background(), key, "rev", "upload", "t", false)
	if id1 != id2 {
		t.Fatalf("dedup failed: %d vs %d", id1, id2)
	}
}

func TestAdmitDownloadPolicyMatrix(t *testing.T) {
	cases := []struct {
		name      string
		status    TaskStatus
		verdict   Verdict
		policy    Policy
		wantAllow bool
		wantCode  string
	}{
		{"blocked enforce", TaskCompleted, VerdictBlocked, DefaultPolicy(), false, "RevisionBlocked"},
		{"pass enforce", TaskCompleted, VerdictPass, DefaultPolicy(), true, "ScanPass"},
		{"warning blocks at warning", TaskCompleted, VerdictWarning, pol("warning"), false, "RevisionBlocked"},
		{"warning passes at critical", TaskCompleted, VerdictWarning, DefaultPolicy(), true, "ScanWarning"},
		{"pending blocked", TaskScanning, VerdictUnknown, DefaultPolicy(), false, "RevisionPendingScan"},
		{"pending controlled allow", TaskScanning, VerdictUnknown, polOnPending("allow"), true, "ScanPendingControlledAllow"},
		{"failed blocked", TaskFailed, VerdictUnknown, DefaultPolicy(), false, "RevisionScanFailed"},
		{"failed controlled allow", TaskFailed, VerdictUnknown, polOnFailed("allow"), true, "ScanFailedControlledAllow"},
		{"mode off", TaskCompleted, VerdictBlocked, polMode("off"), true, "ScanOff"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := newMemStore()
			tree := &fakeTree{files: map[string]string{"a": "1"}}
			store.policies[""] = c.policy
			store.tasks = append(store.tasks, &Task{
				RepoType: "models", Project: "p", Name: "n", Revision: "rev",
				Status: c.status, Verdict: c.verdict,
			})
			svc := newTestService(store, tree, nil)
			d, err := svc.AdmitDownload(context.Background(),
				RepoKey{RepoType: "models", Project: "p", Name: "n"}, "rev", "user")
			if err != nil {
				t.Fatal(err)
			}
			if d.Allow != c.wantAllow || d.Code != c.wantCode {
				t.Fatalf("got allow=%v code=%s, want %v/%s", d.Allow, d.Code, c.wantAllow, c.wantCode)
			}
			if !d.Allow && d.HTTPStatus != 403 {
				t.Fatalf("denials must be 403, got %d", d.HTTPStatus)
			}
		})
	}
}

func TestBuildReportPerFileSeverity(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{"a": "1", "b": "2"}}
	sc := &fakeScanner{id: "s", ver: "v1", findings: map[string][]FileFinding{
		"a": {{Severity: SevWarning, Rule: "w1"}, {Severity: SevCritical, Rule: "c1"}},
	}}
	svc := newTestService(store, tree, []Scanner{sc})
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}
	_, _ = svc.EnqueueRevision(context.Background(), key, "rev", "upload", "t", false)
	task, _ := svc.ExecuteNext(context.Background(), "w")
	if task == nil || task.Verdict != VerdictBlocked {
		t.Fatalf("task %+v", task)
	}
	rep, err := svc.BuildReport(context.Background(), key, "rev")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != StatusBlocked || rep.Counts[SevCritical] != 1 || rep.Counts[SevWarning] != 1 {
		t.Fatalf("report %+v", rep)
	}
	for _, f := range rep.Files {
		if f.Path == "a" && f.Severity != SevCritical {
			t.Fatalf("file a severity %s", f.Severity)
		}
	}
}

func TestFileSeverityMap(t *testing.T) {
	store := newMemStore()
	tree := &fakeTree{files: map[string]string{"a": "1"}}
	sc := &fakeScanner{id: "s", ver: "v1", findings: map[string][]FileFinding{
		"a": {{Severity: SevWarning, Rule: "w"}},
	}}
	svc := newTestService(store, tree, []Scanner{sc})
	key := RepoKey{RepoType: "models", Project: "p", Name: "n"}
	_, _ = svc.EnqueueRevision(context.Background(), key, "rev", "upload", "t", false)
	_, _ = svc.ExecuteNext(context.Background(), "w")
	m, err := svc.FileSeverityMap(context.Background(), key, "rev")
	if err != nil {
		t.Fatal(err)
	}
	if m["a"] != SevWarning {
		t.Fatalf("severity map %+v", m)
	}
}

// policy helpers
func pol(sev string) Policy        { p := DefaultPolicy(); p.BlockSeverity = sev; return p }
func polMode(m string) Policy      { p := DefaultPolicy(); p.Mode = m; return p }
func polOnPending(v string) Policy { p := DefaultPolicy(); p.OnPending = v; return p }
func polOnFailed(v string) Policy  { p := DefaultPolicy(); p.OnFailed = v; return p }
