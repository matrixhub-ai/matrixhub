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

package lfs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
)

const (
	contentMediaType = "application/vnd.git-lfs"
	metaMediaType    = contentMediaType + "+json"
)

// handleBatch provides the batch api
func (h *Handler) handleBatch(w http.ResponseWriter, r *http.Request) {
	bv := unpackBatch(r)

	if h.permissionHookFunc != nil {
		op := permission.OperationReadRepo
		if bv.Operation == "upload" {
			op = permission.OperationUpdateRepo
		}
		repoName := bv.repoName()
		if ok, err := h.permissionHookFunc(r.Context(), op, repoName, permission.Context{}); err != nil {
			responseJSON(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, "permission denied", http.StatusForbidden)
			return
		}
	}

	var responseObjects []*lfsRepresentation

	// Create a response object
	for _, object := range bv.Objects {
		if h.lfsStorage.Exists(object.Oid) {
			responseObjects = append(responseObjects, h.lfsRepresent(r.Context(), bv.Operation, object, true, false))
			continue
		}

		if h.mirror != nil {
			if pf := h.mirror.Get(object.Oid); pf != nil {
				responseObjects = append(responseObjects, h.lfsRepresent(r.Context(), bv.Operation, object, true, false))
				continue
			}
		}

		// Object is not found
		if bv.Operation == "upload" {
			responseObjects = append(responseObjects, h.lfsRepresent(r.Context(), bv.Operation, object, false, true))
		} else {
			rep := &lfsRepresentation{
				Oid:  object.Oid,
				Size: object.Size,
				Error: &lfsObjectError{
					Code:    404,
					Message: "Not found",
				},
			}
			responseObjects = append(responseObjects, rep)
		}
	}

	w.Header().Set("Content-Type", metaMediaType)

	respobj := &lfsBatchResponse{
		Transfer: "basic",
		Objects:  responseObjects,
	}

	responseJSON(w, respobj, http.StatusOK)
}

// handlePutContent receives data from the client and puts it into the content store
func (h *Handler) handlePutContent(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	if signer, ok := h.lfsStorage.(lfs.SignPutter); ok {
		url, err := signer.SignPut(rv.Oid)
		if err != nil {
			responseJSON(w, fmt.Sprintf("failed to sign URL for LFS object %q: %v", rv.Oid, err), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return
	}
	if err := h.lfsStorage.Put(rv.Oid, r.Body, r.ContentLength); err != nil {
		responseJSON(w, fmt.Sprintf("failed to put LFS object %s: %v", rv.Oid, err), http.StatusInternalServerError)
		return
	}
}

// handleGetContent gets the content from the content store
func (h *Handler) handleGetContent(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	if !h.lfsStorage.Exists(rv.Oid) {
		if h.mirror != nil {
			pf := h.mirror.Get(rv.Oid)
			if pf != nil {
				rs := pf.NewReadSeeker()
				defer func() {
					_ = rs.Close()
				}()
				http.ServeContent(w, r, rv.Oid, pf.ModTime(), rs)
				return
			}
		}
		responseJSON(w, fmt.Sprintf("LFS object %s not found", rv.Oid), http.StatusNotFound)
		return
	}
	if signer, ok := h.lfsStorage.(lfs.SignGetter); ok {
		url, err := signer.SignGet(rv.Oid)
		if err != nil {
			responseJSON(w, fmt.Sprintf("failed to sign URL for LFS object %q: %v", rv.Oid, err), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return
	}
	if getter, ok := h.lfsStorage.(lfs.Getter); ok {
		content, stat, err := getter.Get(rv.Oid)
		if err != nil {
			if os.IsNotExist(err) {
				responseJSON(w, fmt.Sprintf("LFS object %s not found", rv.Oid), http.StatusNotFound)
				return
			}
			responseJSON(w, fmt.Sprintf("failed to get LFS object %s: %v", rv.Oid, err), http.StatusInternalServerError)
			return
		}
		defer func() {
			_ = content.Close()
		}()

		w.Header().Set("ETag", fmt.Sprintf("\"%s\"", rv.Oid))
		http.ServeContent(w, r, rv.Oid, stat.ModTime(), content)
		return
	}
	responseJSON(w, fmt.Sprintf("LFS storage does not support direct content retrieval for object %s", rv.Oid), http.StatusNotImplemented)
}

func (h *Handler) handleVerifyObject(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	info, err := h.lfsStorage.Info(rv.Oid)
	if err != nil {
		if os.IsNotExist(err) {
			responseJSON(w, fmt.Sprintf("LFS object %s not found", rv.Oid), http.StatusNotFound)
			return
		}
		responseJSON(w, fmt.Sprintf("failed to get LFS object %s info: %v", rv.Oid, err), http.StatusInternalServerError)
		return
	}

	if info.Size() != rv.Size {
		responseJSON(w, "Size mismatch", http.StatusBadRequest)
		return
	}
}

const tokenExpiration = time.Hour

// lfsRepresent takes a RequestVars and Meta and turns it into a Representation suitable
// for json encoding
func (h *Handler) lfsRepresent(ctx context.Context, op string, rv *lfsRequestVars, download, upload bool) *lfsRepresentation {
	rep := &lfsRepresentation{
		Oid:     rv.Oid,
		Size:    rv.Size,
		Actions: make(map[string]*lfsLink),
	}

	user, _ := authenticate.GetUserInfo(ctx)

	if download && op == "download" {
		link := rv.objectsLink()
		header := map[string]string{"Accept": contentMediaType}
		if h.tokenSignValidator != nil {
			if token, err := h.tokenSignValidator.Sign(ctx, http.MethodGet, link, user.User, tokenExpiration); err != nil {
				slog.WarnContext(ctx, "failed to sign token for LFS download link", "oid", rv.Oid, "error", err)
			} else if token != "" {
				header["Authorization"] = "Bearer " + token
			}
		} else if len(rv.Authorization) > 0 {
			header["Authorization"] = rv.Authorization
		}
		rep.Actions["download"] = &lfsLink{Href: link, Header: header}
	}

	if upload && op == "upload" {
		link := rv.objectsLink()
		header := map[string]string{"Accept": contentMediaType}
		if h.tokenSignValidator != nil {
			if token, err := h.tokenSignValidator.Sign(ctx, http.MethodPut, link, user.User, tokenExpiration); err != nil {
				slog.WarnContext(ctx, "failed to sign token for LFS upload link", "oid", rv.Oid, "error", err)
			} else if token != "" {
				header["Authorization"] = "Bearer " + token
			}
		} else if len(rv.Authorization) > 0 {
			header["Authorization"] = rv.Authorization
		}
		rep.Actions["upload"] = &lfsLink{Href: link, Header: header}

		verifyHeader := make(map[string]string)
		verifyLink := rv.verifyLink()
		if h.tokenSignValidator != nil {
			if token, err := h.tokenSignValidator.Sign(ctx, http.MethodPost, verifyLink, user.User, tokenExpiration); err != nil {
				slog.WarnContext(ctx, "failed to sign token for LFS verify link", "oid", rv.Oid, "error", err)
			} else if token != "" {
				verifyHeader["Authorization"] = "Bearer " + token
			}
		} else if len(rv.Authorization) > 0 {
			verifyHeader["Authorization"] = rv.Authorization
		}
		rep.Actions["verify"] = &lfsLink{Href: verifyLink, Header: verifyHeader}
	}

	if len(rep.Actions) == 0 {
		rep.Actions = nil
	}

	return rep
}

func unpack(r *http.Request) *lfsRequestVars {
	vars := mux.Vars(r)
	rv := &lfsRequestVars{
		Repo:          vars["repo"],
		Oid:           vars["oid"],
		Authorization: r.Header.Get("Authorization"),
	}

	if r.Method == http.MethodPost {
		var p lfsRequestVars
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&p)
		if err != nil {
			return rv
		}

		rv.Oid = p.Oid
		rv.Size = p.Size
	}

	return rv
}

func unpackBatch(r *http.Request) *lfsBatchVars {
	vars := mux.Vars(r)

	var bv lfsBatchVars

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&bv)
	if err != nil {
		return &bv
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
		scheme = fwdProto
	}
	origin := fmt.Sprintf("%s://%s", scheme, r.Host)

	for i := range len(bv.Objects) {
		bv.Objects[i].Repo = vars["repo"]
		bv.Objects[i].Authorization = r.Header.Get("Authorization")
		bv.Objects[i].Origin = origin
	}

	return &bv
}

// lfsRequestVars contain variables from the HTTP request. Variables from routing, json body decoding, and
// some headers are stored.
type lfsRequestVars struct {
	Origin string
	Oid    string
	Size   int64

	Repo          string
	Authorization string
}

func (v *lfsRequestVars) objectsLink() string {
	return fmt.Sprintf("%s/objects/%s", v.Origin, v.Oid)
}

func (v *lfsRequestVars) verifyLink() string {
	return fmt.Sprintf("%s/objects/%s/verify", v.Origin, v.Oid)
}

type lfsBatchVars struct {
	Transfers []string          `json:"transfers,omitempty"`
	Operation string            `json:"operation"`
	Objects   []*lfsRequestVars `json:"objects"`
}

func (bv *lfsBatchVars) repoName() string {
	if len(bv.Objects) == 0 {
		return ""
	}
	return bv.Objects[0].Repo
}

type lfsBatchResponse struct {
	Transfer string               `json:"transfer,omitempty"`
	Objects  []*lfsRepresentation `json:"objects"`
}

// lfsRepresentation is object medata as seen by clients of the lfs server.
type lfsRepresentation struct {
	Oid     string              `json:"oid"`
	Size    int64               `json:"size"`
	Actions map[string]*lfsLink `json:"actions,omitempty"`
	Error   *lfsObjectError     `json:"error,omitempty"`
}

type lfsObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// lfsLink provides a structure used to build a hypermedia representation of an HTTP lfsLink.
type lfsLink struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// metaMatcher provides a mux.MatcherFunc that only allows requests that contain
// an Accept header with the metaMediaType
func metaMatcher(r *http.Request, m *mux.RouteMatch) bool {
	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	return mt == metaMediaType
}
