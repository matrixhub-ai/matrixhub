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
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/repository"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

// Opt-in REST surface; its reviewable contract is api/openapi/artifact-security.json.
func (h *Handler) handleSecurity(w http.ResponseWriter, r *http.Request) {
	ri := getRepoInformation(r)
	action := mux.Vars(r)["action"]
	write := r.Method != http.MethodGet
	op := permission.OperationReadRepo
	if write {
		op = permission.OperationUpdateRepo
	}
	if h.permissionHookFunc == nil {
		responseJSON(w, "authorization unavailable", 503)
		return
	}
	permitted, err := h.permissionHookFunc(r.Context(), op, ri.RepoName, permission.Context{})
	if err != nil {
		responseJSON(w, "authorization unavailable", 503)
		return
	}
	if !permitted {
		responseJSON(w, "permission denied", 403)
		return
	}
	user, _ := authenticate.GetUserInfo(r.Context())
	if action == "policy" && write {
		identity, err := authcodec.Unmarshal(user.User)
		if err != nil {
			responseJSON(w, "authentication required", 401)
			return
		}
		ctx := auth.WithIdentity(r.Context(), identity)
		allowed, err := h.authzService.VerifyProjectPermissionByName(ctx, ri.Namespace, role.ProjectUpdate)
		if err != nil || !allowed {
			responseJSON(w, "project management permission required", 403)
			return
		}
	}
	if action == "policy" {
		switch r.Method {
		case http.MethodGet:
			policy, err := h.artifactScan.Policy(r.Context(), ri.RepoName)
			if err != nil {
				responseJSON(w, "policy unavailable", 503)
				return
			}
			responseJSON(w, policy, 200)
		case http.MethodPut:
			var policy artifactscan.Policy
			decoder := json.NewDecoder(io.LimitReader(r.Body, 2048))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&policy); err != nil || policy.Validate() != nil {
				responseJSON(w, "invalid policy", 400)
				return
			}
			if err := h.artifactScan.SetPolicy(r.Context(), ri.RepoName, policy, user.User); err != nil {
				responseJSON(w, "policy update failed", 503)
				return
			}
			responseJSON(w, policy, 200)
		default:
			responseJSON(w, "method not allowed", 405)
		}
		return
	}
	if action == "audit" {
		if write {
			responseJSON(w, "method not allowed", 405)
			return
		}
		events, err := h.artifactScan.Events(r.Context(), ri.RepoName)
		if err != nil {
			responseJSON(w, "audit unavailable", 503)
			return
		}
		responseJSON(w, map[string]any{"events": events}, 200)
		return
	}
	path := h.storage.ResolvePath(ri.RepoName)
	repo, err := repository.Open(path)
	if err != nil {
		responseJSON(w, "repository unavailable", 404)
		return
	}
	revision := r.URL.Query().Get("revision")
	if revision == "" {
		revision = repo.DefaultBranch()
	}
	commits, err := repo.Commits(revision, &repository.CommitsOptions{Limit: 1})
	if err != nil || len(commits) == 0 {
		responseJSON(w, "revision unavailable", 404)
		return
	}
	revision = commits[0].Hash().String()
	if action == "report" && !write {
		report, err := h.artifactScan.Get(r.Context(), ri.RepoName, revision)
		if err != nil {
			responseJSON(w, "report unavailable", 503)
			return
		}
		responseJSON(w, report, 200)
		return
	}
	if r.Method != http.MethodPost {
		responseJSON(w, "method not allowed", 405)
		return
	}
	switch action {
	case "rescan":
		var input struct {
			Force *bool `json:"force"`
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, 2048))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil && err != io.EOF {
			responseJSON(w, "invalid rescan input", 400)
			return
		}
		force := true
		if input.Force != nil {
			force = *input.Force
		}
		if err := h.artifactScan.Rescan(r.Context(), ri.RepoName, revision, user.User, force); err != nil {
			responseJSON(w, "rescan failed", 409)
			return
		}
	case "cancel":
		if err := h.artifactScan.Cancel(r.Context(), ri.RepoName, revision, user.User); err != nil {
			responseJSON(w, "task cannot be cancelled", 409)
			return
		}
	default:
		responseJSON(w, "method not allowed", 405)
		return
	}
	report, err := h.artifactScan.Get(r.Context(), ri.RepoName, revision)
	if err != nil {
		responseJSON(w, "report unavailable", 503)
		return
	}
	responseJSON(w, report, 202)
}
