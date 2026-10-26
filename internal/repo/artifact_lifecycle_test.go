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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
	"github.com/stretchr/testify/require"
)

type lifecycleFixtureSource struct{}

func (lifecycleFixtureSource) Files(context.Context, string, string) ([]artifactscan.File, error) {
	return []artifactscan.File{{Path: "config.json", Size: 2, Open: func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewBufferString("{}")), nil }}}, nil
}

// Tests orchestration and persistence against a controllable scanner protocol,
// not ClamAV/Fickling detection accuracy.
func TestArtifactScannerLifecycleOfflineRulesAndRecovery(t *testing.T) {
	store := newArtifactTestStore(t)
	var offline atomic.Bool
	var identity atomic.Value
	var scans atomic.Int32
	offline.Store(true)
	identity.Store(strings.Repeat("a", 64))
	scanner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if offline.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/identity" {
			json.NewEncoder(w).Encode(map[string]string{"ruleset": identity.Load().(string)})
			return
		}
		digest := sha256.New()
		size, err := io.Copy(digest, r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		scans.Add(1)
		json.NewEncoder(w).Encode(artifactscan.FileResult{Status: artifactscan.Passed, FileType: "json-by-name", SHA256: hex.EncodeToString(digest.Sum(nil)), Size: size, Ruleset: identity.Load().(string), Checks: []string{"controlled-protocol-fixture"}})
	}))
	defer scanner.Close()
	transport := &ArtifactHTTPScanner{Endpoint: scanner.URL, Client: scanner.Client()}
	service, err := artifactscan.New(store, lifecycleFixtureSource{}, transport)
	require.NoError(t, err)
	name, revision := "models/test/lifecycle", strings.Repeat("a", 40)
	require.NoError(t, service.Enqueue(t.Context(), name, revision))
	require.NoError(t, service.ProcessNext(t.Context()))
	report, err := service.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Failed, report.Status)
	require.Equal(t, "scanner identity unavailable", report.Error)
	require.Zero(t, scans.Load())
	decision, err := service.Decide(t.Context(), name, revision, "tester", "hf")
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	offline.Store(false)
	require.NoError(t, service.Rescan(t.Context(), name, revision, "tester", true))
	require.NoError(t, service.ProcessNext(t.Context()))
	report, err = service.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Passed, report.Status)
	require.Equal(t, int32(1), scans.Load())
	require.Len(t, report.Files, 1)
	require.Equal(t, "json-by-name", report.Files[0].FileType)
	require.False(t, report.Files[0].CheckedAt.IsZero())
	require.Equal(t, "distribute_by_policy", report.Files[0].RecommendedAction)
	encoded, err := json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "\"file_type\":\"json-by-name\"")

	identity.Store(strings.Repeat("b", 64))
	pending, err := service.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Pending, pending.Status)
	require.Greater(t, pending.Attempt, report.Attempt)
	decision, err = service.Decide(t.Context(), name, revision, "tester", "hf")
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.NoError(t, service.ProcessNext(t.Context()))
	report, err = service.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Passed, report.Status)
	require.Equal(t, strings.Repeat("b", 64), report.Ruleset)
	require.Equal(t, int32(2), scans.Load())
	require.NoError(t, service.Rescan(t.Context(), name, revision, "tester", true))
	interrupted, found, err := store.Claim(t.Context())
	require.NoError(t, err)
	require.True(t, found)
	restarted, err := artifactscan.New(store, lifecycleFixtureSource{}, transport)
	require.NoError(t, err)
	resumed, err := store.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Pending, resumed.Status)
	require.Greater(t, resumed.Attempt, interrupted.Attempt)
	interrupted.Status = artifactscan.Passed
	require.NoError(t, store.Save(t.Context(), interrupted))
	resumed, err = store.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Pending, resumed.Status)
	require.NoError(t, restarted.ProcessNext(t.Context()))
	report, err = restarted.Get(t.Context(), name, revision)
	require.NoError(t, err)
	require.Equal(t, artifactscan.Passed, report.Status)
	require.Equal(t, int32(3), scans.Load())
	events, err := store.Events(t.Context(), name, 100)
	require.NoError(t, err)
	actions := []string{}
	for _, event := range events {
		actions = append(actions, event.Action)
	}
	require.Contains(t, actions, "recovered")
	require.Contains(t, actions, "rescan")
}
