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
	"errors"
	"net/http"
	"slices"

	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type action string

const (
	resourceDataset = "datasets"
	resourceModel   = "models"
	resourceSpace   = "spaces"

	actionRead  action = "read"
	actionWrite action = "write"
)

var (
	hfPublicMethods = map[string]bool{
		"/api/whoami-v2": true,
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

func HFAuthzMiddleware(projectRepo project.IProjectRepo, authzSvc authz.IAuthzService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hfPublicMethods[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			if !checkPerm(projectRepo, authzSvc, r) {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkPerm(projectRepo project.IProjectRepo, authzSvc authz.IAuthzService, r *http.Request) bool {
	vars := mux.Vars(r)
	projectName, resource := vars["namespace"], vars["repoType"]
	method := r.Method
	act := actionRead
	if !slices.Contains(readMethods, method) {
		act = actionWrite
	}

	prj, err := projectRepo.GetProjectByName(r.Context(), projectName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true
		}
		log.Errorf("Failed to get project by name: %s", err)
		return false
	}

	if prj.IsPublic() && act == actionRead {
		return true
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
	passed, err := authzSvc.VerifyProjectPermission(r.Context(), prj.ID, permission)
	if err != nil {
		log.Errorf("Failed to verify project permission: %s", err)
		return false
	}

	return passed
}
