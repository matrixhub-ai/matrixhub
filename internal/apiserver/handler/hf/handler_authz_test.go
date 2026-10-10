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

package hf

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

func TestHuggingFaceRepoAuthorizationFailure(t *testing.T) {
	for _, repoType := range []string{"model", "dataset", "space"} {
		for _, operation := range []string{"create", "delete"} {
			for _, failure := range []string{"denied", "service-error"} {
				t.Run(repoType+"/"+operation+"/"+failure, func(t *testing.T) {
					store := storage.NewStorage(storage.WithRootDir(t.TempDir()))
					storageName := "test-project/protected-repo"
					if prefix := repoTypePrefix(repoType); prefix != "" {
						storageName = prefix + "/" + storageName
					}
					repoPath := store.ResolvePath(storageName)
					if operation == "delete" {
						if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
							t.Fatal(err)
						}
						if _, err := repository.Init(context.Background(), repoPath, "main"); err != nil {
							t.Fatal(err)
						}
					}
					wantPermission := role.ModelPush
					if repoType == "dataset" {
						wantPermission = role.DatasetPush
					}
					calls := 0
					auth := &transportTestAuthz{verify: func(name string, perm role.Permission) {
						calls++
						if name != "test-project" || perm != wantPermission {
							t.Errorf("authorization received %q / %v, want test-project / %v", name, perm, wantPermission)
						}
					}}
					wantStatus := http.StatusForbidden
					if failure == "service-error" {
						auth.allowed = true // An error must prevail even if a service returns true.
						auth.err = errors.New("authorization service unavailable")
						wantStatus = http.StatusInternalServerError
					}
					handler := NewHandler(WithStorage(store), WithServices(nil, nil, auth))
					body, err := json.Marshal(map[string]string{
						"type": repoType, "name": "protected-repo", "organization": "test-project",
					})
					if err != nil {
						t.Fatal(err)
					}
					method := http.MethodPost
					if operation == "delete" {
						method = http.MethodDelete
					}
					req := httptest.NewRequest(method, "/api/repos/"+operation, strings.NewReader(string(body)))
					req.Header.Set("Content-Type", "application/json")
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, req)
					if response.Code != wantStatus {
						t.Fatalf("status %d, want %d; body: %s", response.Code, wantStatus, response.Body.String())
					}
					if calls != 1 {
						t.Fatalf("authorization calls = %d, want 1", calls)
					}
					if operation == "delete" {
						if !repository.IsRepository(repoPath) {
							t.Fatal("repository was removed despite authorization failure")
						}
					} else if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
						t.Fatalf("repository path created despite authorization failure: %v", err)
					}
				})
			}
		}
	}
}
