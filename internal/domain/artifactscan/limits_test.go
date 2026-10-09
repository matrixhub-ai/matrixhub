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
	"time"
)

func TestArtifactLimitsValidation(t *testing.T) {
	defaults, err := (Limits{}).normalized()
	if err != nil || defaults != DefaultLimits() {
		t.Fatal(defaults, err)
	}
	for _, limits := range []Limits{{MaxFileBytes: -1}, {MaxFileBytes: 1<<30 + 1}, {MaxFiles: 4097}, {TaskTimeout: time.Millisecond}, {TaskTimeout: 2 * time.Hour}} {
		if _, err := limits.normalized(); err == nil {
			t.Fatalf("accepted %+v", limits)
		}
	}
}

type failedEvidenceScanner struct{}
type reasonFailure string

func (r reasonFailure) Error() string           { return string(r) }
func (r reasonFailure) ScanFailureCode() string { return string(r) }
func (failedEvidenceScanner) Scan(context.Context, string, []byte) (FileResult, error) {
	return FileResult{Status: Failed, FileType: "zip", Checks: []string{"ClamAV/test"}, Findings: []Finding{{Scanner: "clamav", Rule: "CLAMAV_SIGNATURE_MATCH", Severity: "high"}}}, reasonFailure("pickle_metadata_limit")
}
func TestArtifactFailedScanPreservesEvidence(t *testing.T) {
	s, err := New(&testStore{}, testSource{}, failedEvidenceScanner{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.scanFile(context.Background(), sampleFile(), "p/r", "", false)
	if err == nil || result.Status != Failed || result.Error != "pickle_metadata_limit" || len(result.Findings) != 1 || len(result.Checks) != 1 || result.FileType != "zip" {
		t.Fatal(result, err)
	}
}
func TestArtifactContextReaderStopsBeforeReading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := bytes.NewBufferString("not read")
	_, err := io.ReadAll(contextReader{ctx, source})
	if !errors.Is(err, context.Canceled) || source.Len() != 8 {
		t.Fatal(source.Len(), err)
	}
}
func TestArtifactConfiguredCapacity(t *testing.T) {
	scanner := &testScanner{status: Passed}
	s, err := NewWithLimits(&testStore{}, testSource{}, scanner, Limits{MaxFileBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.scanFile(context.Background(), sampleFile(), "p/r", "", false)
	if err == nil || result.Error != "file_size_limit" || scanner.calls != 0 {
		t.Fatal(result, err)
	}
}
