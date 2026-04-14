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

package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

func GitAuthzMiddleware(authzSvc authz.IAuthzService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// todo: objects API add auth
			if strings.HasPrefix(r.URL.Path, "/objects") {
				next.ServeHTTP(w, r)
				return
			}
			if !checkGitPerm(authzSvc, r) {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func checkGitPerm(authzSvc authz.IAuthzService, r *http.Request) bool {
	vars := mux.Vars(r)
	repo := vars["repo"]
	method := r.Method
	if repo == "" {
		return false
	}
	repo = strings.TrimSuffix(repo, ".git")

	act := actionRead
	if !slices.Contains(readMethods, method) {
		act = actionWrite
	}
	project, resource := parseProject(repo)
	if project == "" {
		return false
	}

	var permission role.Permission
	if resource == "" {
		permission = role.ModelPull
	} else {
		permission = resourcePermissions[resource][act]
	}
	if permission == "" {
		return false
	}

	passed, err := authzSvc.VerifyProjectPermissionByName(r.Context(), project, permission)
	if err != nil {
		log.Errorf("Failed to verify project permission: %s", err)
	}
	return passed
}

func parseProject(repo string) (string, string) {
	s := strings.Split(repo, "/")
	if len(s) < 2 {
		return "", ""
	}
	if s[0] == resourceDataset {
		return s[1], resourceDataset
	}

	return s[0], resourceModel
}
