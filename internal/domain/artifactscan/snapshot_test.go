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

	"github.com/stretchr/testify/require"
)

type revisionTestStore struct {
	testStore
	reports map[string]Report
	queued  []string
}

func (s *revisionTestStore) Get(_ context.Context, repo, revision string) (Report, error) {
	if report, ok := s.reports[revision]; ok {
		return report, nil
	}
	return Report{Repo: repo, Revision: revision, Status: Unscanned}, nil
}
func (s *revisionTestStore) Enqueue(_ context.Context, _ string, revision string) error {
	s.queued = append(s.queued, revision)
	return nil
}
func TestArtifactSnapshotAdmissionBindsEveryRevision(t *testing.T) {
	approved, fresh := strings.Repeat("a", 40), strings.Repeat("b", 40)
	store := &revisionTestStore{reports: map[string]Report{approved: {Repo: "models/p/r", Revision: approved, Status: Passed}}}
	service := &Service{store: store}
	decision, err := service.AdmitRevisions(t.Context(), "models/p/r", []string{approved}, "tester", "git-snapshot")
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	decision, err = service.AdmitRevisions(t.Context(), "models/p/r", []string{fresh, approved}, "tester", "git-snapshot")
	require.NoError(t, err)
	require.False(t, decision.Allowed, "approved old content must not authorize a new commit")
	require.Equal(t, fresh, decision.Revision)
	require.Equal(t, []string{fresh}, store.queued)
	store.reports[fresh] = Report{Repo: "models/p/r", Revision: fresh, Status: Passed}
	store.reports[approved] = Report{Repo: "models/p/r", Revision: approved, Status: Blocked}
	decision, err = service.AdmitRevisions(t.Context(), "models/p/r", []string{fresh, approved}, "tester", "git-snapshot")
	require.NoError(t, err)
	require.False(t, decision.Allowed, "a clean tip must not expose blocked reachable history")
	require.Equal(t, approved, decision.Revision)
	_, err = service.AdmitRevisions(t.Context(), "models/p/r", []string{"main"}, "tester", "git-snapshot")
	require.Error(t, err)
	_, err = service.AdmitRevisions(t.Context(), "models/p/r", make([]string, 257), "tester", "git-snapshot")
	require.ErrorContains(t, err, "budget")
}
