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
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type testStore struct{ report Report }

func (s *testStore) Get(context.Context, string, string) (Report, error) { return s.report, nil }
func (s *testStore) Enqueue(context.Context, string, string) error       { return nil }
func (s *testStore) Claim(context.Context) (Report, bool, error)         { return Report{}, false, nil }
func (s *testStore) Save(_ context.Context, r Report) error              { s.report = r; return nil }
func (s *testStore) Recover(context.Context) error                       { return nil }

type testSource struct {
	files []File
	err   error
}

func (s testSource) Files(context.Context, string, string) ([]File, error) { return s.files, s.err }

type testScanner struct {
	status Status
	err    error
	calls  int
}

func (s *testScanner) Scan(context.Context, string, []byte) (FileResult, error) {
	s.calls++
	return FileResult{Status: s.status}, s.err
}
func sampleFile() File {
	return File{Path: "config.json", Size: 2, Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader([]byte("{}"))), nil }}
}

func TestArtifactAdmission(t *testing.T) {
	for _, status := range []Status{Unscanned, Pending, Scanning, Warning, Blocked, Failed, Status("unknown"), Passed} {
		if Allowed(status) != (status == Passed) {
			t.Fatalf("unexpected admission for %s", status)
		}
	}
}

func TestArtifactScanFailuresRemainClosed(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		sourceErr, errorScan error
		scanStatus, want     Status
		wantError            string
	}{
		{name: "clean", scanStatus: Passed, want: Passed},
		{name: "danger", scanStatus: Blocked, want: Blocked},
		{name: "warning", scanStatus: Warning, want: Warning},
		{name: "missing-source", sourceErr: errors.New("missing"), scanStatus: Passed, want: Failed},
		{name: "transport-error", errorScan: errors.New("offline"), scanStatus: Passed, want: Failed, wantError: "scanner_incomplete"},
		{name: "invalid-verdict", scanStatus: Pending, want: Failed, wantError: "scanner_protocol_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &testStore{}
			scanner := &testScanner{status: tc.scanStatus, err: tc.errorScan}
			service := &Service{store: store, source: testSource{files: []File{sampleFile()}, err: tc.sourceErr}, scanner: scanner, limits: DefaultLimits()}
			service.process(context.Background(), Report{Repo: "models/p/r", Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
			if store.report.Status != tc.want {
				t.Fatalf("%+v", store.report)
			}
			if tc.wantError != "" && store.report.Files[0].Error != tc.wantError {
				t.Fatalf("%+v", store.report.Files[0])
			}
			if tc.want != Passed && Allowed(store.report.Status) {
				t.Fatal("failure admitted")
			}
		})
	}
}

func TestArtifactIntegrityAndBudgets(t *testing.T) {
	for _, name := range []string{"size", "digest", "oversized", "cancelled", "file-count"} {
		t.Run(name, func(t *testing.T) {
			file := sampleFile()
			ctx := context.Background()
			if name == "size" {
				file.Size = 3
			}
			if name == "digest" {
				file.ExpectedSHA256 = "wrong"
			}
			if name == "oversized" {
				file.Size = 8*1024*1024 + 1
			}
			if name == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			files := []File{file}
			if name == "file-count" {
				files = make([]File, 257)
			}
			store := &testStore{}
			scanner := &testScanner{status: Passed}
			service := &Service{store: store, source: testSource{files: files}, scanner: scanner, limits: Limits{MaxFileBytes: 8 * 1024 * 1024, MaxFiles: 256, TaskTimeout: DefaultLimits().TaskTimeout}}
			service.process(ctx, Report{Repo: "models/p/r", Revision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
			if store.report.Status != Failed || scanner.calls != 0 {
				t.Fatalf("status=%s calls=%d", store.report.Status, scanner.calls)
			}
		})
	}
}
