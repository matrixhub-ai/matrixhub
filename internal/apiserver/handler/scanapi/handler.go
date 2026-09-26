// Copyright The Matrixhub Authors.
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

// Package scanapi exposes the security-scan REST surface under
// /api/scan/v1alpha1: explainable reports, manual rescan, policy management
// and audit queries. It is a plain mux handler chained like the other
// backends; the gRPC-gateway owns /api/v1alpha1 so this prefix is distinct.
package scanapi

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/domain/scan"
)

// Handler serves the scan REST API.
type Handler struct {
	svc  scan.ServiceAPI
	full *scan.Service // rescan + policy write need the concrete service for now
	root *mux.Router
}

// New builds the handler; full may be nil (read-only deployments).
func New(svc scan.ServiceAPI, full *scan.Service) *Handler {
	h := &Handler{svc: svc, full: full, root: mux.NewRouter()}
	h.register()
	return h
}

// Use appends middlewares to the scan router (e.g. the HF session/token
// authn middleware so the web UI can call these endpoints with cookies).
func (h *Handler) Use(mw ...mux.MiddlewareFunc) {
	h.root.Use(mw...)
}

func (h *Handler) register() {
	r := h.root
	r.Use(h.requireUser)

	r.HandleFunc("/api/scan/v1alpha1/reports/{repoType}/{project}/{name}/revision/{rev}", h.handleReport).Methods(http.MethodGet)
	r.HandleFunc("/api/scan/v1alpha1/reports/{repoType}/{project}/{name}/revision/{rev}/rescan", h.handleRescan).Methods(http.MethodPost)
	r.HandleFunc("/api/scan/v1alpha1/status/{repoType}/{project}/{name}/revision/{rev}", h.handleStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/scan/v1alpha1/policies", h.handlePlatformPolicy).Methods(http.MethodGet, http.MethodPut)
	r.HandleFunc("/api/scan/v1alpha1/policies/{project}", h.handleProjectPolicy).Methods(http.MethodGet, http.MethodPut)
	r.HandleFunc("/api/scan/v1alpha1/audit", h.handleAudit).Methods(http.MethodGet)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.root.ServeHTTP(w, r)
}

// requireUser rejects anonymous access: scan evidence and audit trails are
// for authenticated operators only.
func (h *Handler) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userInfo, ok := authenticate.GetUserInfo(r.Context())
		if !ok || userInfo.User == authenticate.Anonymous {
			writeJSON(w, map[string]string{"error": "Unauthorized"}, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, data any, sc int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(sc)
	_ = json.NewEncoder(w).Encode(data)
}

func actorOf(r *http.Request) string {
	if userInfo, ok := authenticate.GetUserInfo(r.Context()); ok && userInfo.User != "" {
		return string(userInfo.User)
	}
	return "unknown"
}

func repoKeyOf(r *http.Request) scan.RepoKey {
	vars := mux.Vars(r)
	return scan.RepoKey{RepoType: vars["repoType"], Project: vars["project"], Name: vars["name"]}
}

func (h *Handler) handleReport(w http.ResponseWriter, r *http.Request) {
	key := repoKeyOf(r)
	rev := mux.Vars(r)["rev"]
	sha, err := h.svc.ResolveRevision(r.Context(), key, rev)
	if err != nil {
		writeJSON(w, map[string]string{"error": "revision not found"}, http.StatusNotFound)
		return
	}
	rep, err := h.svc.BuildReport(r.Context(), key, sha)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, rep, http.StatusOK)
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	key := repoKeyOf(r)
	rev := mux.Vars(r)["rev"]
	sha, err := h.svc.ResolveRevision(r.Context(), key, rev)
	if err != nil {
		writeJSON(w, map[string]string{"error": "revision not found"}, http.StatusNotFound)
		return
	}
	status, err := h.svc.VersionStatus(r.Context(), key, sha)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"revision": sha,
		"status":   string(status),
		"hfStatus": status.HFStatus(),
	}, http.StatusOK)
}

func (h *Handler) handleRescan(w http.ResponseWriter, r *http.Request) {
	if h.full == nil {
		writeJSON(w, map[string]string{"error": "rescan unavailable"}, http.StatusNotImplemented)
		return
	}
	key := repoKeyOf(r)
	rev := mux.Vars(r)["rev"]
	sha, err := h.svc.ResolveRevision(r.Context(), key, rev)
	if err != nil {
		writeJSON(w, map[string]string{"error": "revision not found"}, http.StatusNotFound)
		return
	}
	actor := actorOf(r)
	taskID, err := h.full.Rescan(r.Context(), key, sha, actor)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"taskId": taskID, "revision": sha}, http.StatusAccepted)
}

func (h *Handler) handlePlatformPolicy(w http.ResponseWriter, r *http.Request) {
	h.handlePolicy(w, r, nil)
}

func (h *Handler) handleProjectPolicy(w http.ResponseWriter, r *http.Request) {
	project := mux.Vars(r)["project"]
	h.handlePolicy(w, r, &project)
}

func (h *Handler) handlePolicy(w http.ResponseWriter, r *http.Request, project *string) {
	if h.full == nil {
		writeJSON(w, map[string]string{"error": "policy management unavailable"}, http.StatusNotImplemented)
		return
	}
	if r.Method == http.MethodPut {
		var body struct {
			Mode                 string `json:"mode"`
			BlockSeverity        string `json:"blockSeverity"`
			OnPending            string `json:"onPending"`
			OnFailed             string `json:"onFailed"`
			OnScannerUnavailable string `json:"onScannerUnavailable"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, map[string]string{"error": "invalid body"}, http.StatusBadRequest)
			return
		}
		p := scan.DefaultPolicy()
		p.Project = project
		if body.Mode != "" {
			p.Mode = body.Mode
		}
		if body.BlockSeverity != "" {
			p.BlockSeverity = body.BlockSeverity
		}
		if body.OnPending != "" {
			p.OnPending = body.OnPending
		}
		if body.OnFailed != "" {
			p.OnFailed = body.OnFailed
		}
		if body.OnScannerUnavailable != "" {
			p.OnScannerUnavailable = body.OnScannerUnavailable
		}
		p.UpdatedBy = actorOf(r)
		if err := h.full.UpsertPolicy(r.Context(), p); err != nil {
			writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
			return
		}
		_ = h.full.AppendAudit(r.Context(), scan.AuditEvent{
			Action: "policy_update", Actor: p.UpdatedBy, Decision: p.Mode,
			Reason: "blockAt=" + p.BlockSeverity, Project: deref(project),
		})
		writeJSON(w, p, http.StatusOK)
		return
	}
	p, err := h.full.GetPolicy(r.Context(), project)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, p, http.StatusOK)
}

func (h *Handler) handleAudit(w http.ResponseWriter, r *http.Request) {
	if h.full == nil {
		writeJSON(w, map[string]string{"error": "audit unavailable"}, http.StatusNotImplemented)
		return
	}
	q := r.URL.Query()
	events, err := h.full.ListAudit(r.Context(), scan.AuditFilter{
		RepoType: q.Get("repoType"),
		Project:  q.Get("project"),
		Name:     q.Get("name"),
		Revision: q.Get("revision"),
		Actor:    q.Get("actor"),
		Action:   q.Get("action"),
		Limit:    intQuery(q.Get("limit"), 100),
		Offset:   intQuery(q.Get("offset"), 0),
	})
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"events": events}, http.StatusOK)
}

func intQuery(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
		if n > 1000 {
			return 1000
		}
	}
	return n
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ChainWith wraps next with the git-style auth chain (basic auth / token /
// anonymous-annotate) so scan REST callers are authenticated like git and
// LFS clients, then dispatches scan-API paths to this handler.
func (h *Handler) ChainWith(authn func(http.Handler) http.Handler, next http.Handler) http.Handler {
	return authn(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= len("/api/scan/") && r.URL.Path[:len("/api/scan/")] == "/api/scan/" {
			h.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
