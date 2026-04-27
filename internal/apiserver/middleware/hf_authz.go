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

	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type action string

const (
	resourceDataset = "datasets"
	resourceModel   = "models"

	actionRead  action = "read"
	actionWrite action = "write"
)

var (
	hfSkipAuthzMethods = map[string]bool{
		"/api/whoami-v2":     true,
		"/api/repos/create":  true,
		"/api/repos/delete":  true,
		"/api/repos/move":    true,
		"/api/validate-yaml": true,
	}
	readMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	resourcePermissions = map[string]map[action]role.Permission{
		resourceDataset: {
			actionRead:  role.DatasetPull,
			actionWrite: role.DatasetPush,
		},
		resourceModel: {
			actionRead:  role.ModelPull,
			actionWrite: role.ModelPush,
		},
	}
)

func HFAuthzMiddleware(authzSvc authz.IAuthzService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hfSkipAuthzMethods[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			if !checkHFPerm(authzSvc, r) {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkHFPerm(authzSvc authz.IAuthzService, r *http.Request) bool {
	vars := mux.Vars(r)
	projectName, resource := vars["namespace"], vars["repoType"]
	method := r.Method
	act := actionRead
	if !slices.Contains(readMethods, method) {
		act = actionWrite
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
	passed, err := authzSvc.VerifyProjectPermissionByName(r.Context(), projectName, permission)
	if err != nil {
		log.Errorf("Failed to verify project permission: %s", err)
		return false
	}

	return passed
}
