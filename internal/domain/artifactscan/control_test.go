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
	"strings"
	"testing"
)

func TestArtifactPolicies(t *testing.T) {
	strict := StrictPolicy()
	for _, status := range []Status{Unscanned, Pending, Scanning, Warning, Failed, Cancelled, Blocked, "unknown"} {
		if Evaluate(Report{Status: status}, strict).Allowed {
			t.Fatalf("strict admitted %s", status)
		}
	}
	lenient := Policy{"high", "allow", "allow"}
	for _, status := range []Status{Passed, Warning, Failed, Pending, Scanning} {
		if !Evaluate(Report{Status: status}, lenient).Allowed {
			t.Fatalf("explicit policy denied %s", status)
		}
	}
	for _, status := range []Status{Blocked, Cancelled, "unknown"} {
		if Evaluate(Report{Status: status}, lenient).Allowed {
			t.Fatalf("lenient admitted %s", status)
		}
	}
	if Evaluate(Report{Status: Failed, Files: []FileResult{{Status: Blocked}}}, lenient).Allowed {
		t.Fatal("failure policy erased known blocked file")
	}
	if Evaluate(Report{Status: Passed}, Policy{}).Allowed {
		t.Fatal("corrupt policy admitted content")
	}
}

func TestKnownFindingsSurviveIncompleteStatus(t *testing.T) {
	for _, status := range []Status{Failed, Pending, Scanning, Passed, Warning} {
		for _, threshold := range []string{"medium", "high"} {
			t.Run(string(status)+"/"+threshold, func(t *testing.T) {
				policy := Policy{threshold, "allow", "allow"}
				report := Report{Status: status, Files: []FileResult{
					{Status: Failed, Findings: []Finding{{Severity: "medium"}}},
					{Status: Failed, Error: "pickle_metadata_limit", Findings: []Finding{{Severity: "high"}}},
				}}
				decision := Evaluate(report, policy)
				if decision.Allowed || decision.Reason != "high_risk_blocked" || decision.ScanStatus != status {
					t.Fatalf("known high risk erased by completion status: %+v", decision)
				}
				report.Files = report.Files[:1]
				decision = Evaluate(report, policy)
				if threshold == "medium" && (decision.Allowed || decision.Reason != "warning_policy") {
					t.Fatalf("known medium risk erased by completion status: %+v", decision)
				}
				if threshold == "high" && !decision.Allowed {
					t.Fatalf("explicit medium-risk policy was ignored: %+v", decision)
				}
			})
		}
	}
}

func TestIncompleteRiskRecommendation(t *testing.T) {
	file := FileResult{Status: Failed, Findings: []Finding{{Severity: "medium"}, {Severity: "high"}}}
	if got := fileRecommendation(file); got != "quarantine_and_replace" {
		t.Fatal(got)
	}
	file.Findings = file.Findings[:1]
	if got := fileRecommendation(file); got != "manual_review" {
		t.Fatal(got)
	}
	file.Findings = nil
	if got := fileRecommendation(file); got != "repair_and_rescan" {
		t.Fatal(got)
	}
}

type changingScanner struct {
	before, during string
	findings       []Finding
}

func (s changingScanner) Identity(context.Context) (string, error) { return s.before, nil }
func (s changingScanner) Scan(context.Context, string, []byte) (FileResult, error) {
	return FileResult{Status: Passed, Ruleset: s.during, Findings: s.findings}, nil
}
func TestArtifactRulesChangingDuringScan(t *testing.T) {
	store := &testStore{}
	s := &Service{store: store, source: testSource{files: []File{sampleFile()}}, scanner: changingScanner{before: strings.Repeat("a", 64), during: strings.Repeat("b", 64)}, limits: DefaultLimits()}
	s.process(context.Background(), Report{Repo: "p/r", Revision: strings.Repeat("a", 40)})
	if store.report.Status != Failed || store.report.Files[0].Error != "scanner_ruleset_changed" {
		t.Fatal(store.report)
	}
}

func TestRulesetFailurePreservesRisk(t *testing.T) {
	store := &testStore{}
	s := &Service{store: store, source: testSource{files: []File{sampleFile()}}, scanner: changingScanner{
		before: strings.Repeat("a", 64), during: strings.Repeat("b", 64), findings: []Finding{{Severity: "high"}},
	}, limits: DefaultLimits()}
	s.process(context.Background(), Report{Repo: "p/r", Revision: strings.Repeat("a", 40)})
	decision := Evaluate(store.report, Policy{"high", "allow", "allow"})
	if store.report.Status != Failed || len(store.report.Files[0].Findings) != 1 || decision.Allowed || decision.Reason != "high_risk_blocked" || store.report.Files[0].RecommendedAction != "quarantine_and_replace" {
		t.Fatal(store.report, decision)
	}
}
