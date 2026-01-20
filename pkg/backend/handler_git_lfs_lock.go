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

package backend

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
)

var (
	ErrNotOwner = errors.New("attempt to delete other user's lock")
)

func (h *Handler) registryLFSLock(r *mux.Router) {
	r.HandleFunc("/{repo:.+}.git/locks", h.handleGetLock).Methods(http.MethodGet).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks", h.handleGetLock).Methods(http.MethodGet).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}.git/locks/verify", h.handleLocksVerify).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks/verify", h.handleLocksVerify).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}.git/locks", h.handleCreateLock).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks", h.handleCreateLock).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}.git/locks/{id}/unlock", h.handleDeleteLock).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks/{id}/unlock", h.handleDeleteLock).Methods(http.MethodPost).MatcherFunc(metaMatcher)
}

func (h *Handler) handleGetLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	ll := &lfs.LockList{}

	w.Header().Set("Content-Type", metaMediaType)

	locks, nextCursor, err := h.locksStore.Filtered(repoName,
		r.FormValue("path"),
		r.FormValue("cursor"),
		r.FormValue("limit"))

	if err != nil {
		ll.Message = err.Error()
	} else {
		ll.Locks = locks
		ll.NextCursor = nextCursor
	}

	h.JSON(w, ll, http.StatusOK)
}

func (h *Handler) handleLocksVerify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", metaMediaType)

	reqBody := &lfs.VerifiableLockRequest{}
	if err := dec.Decode(reqBody); err != nil {
		h.JSON(w, &lfs.VerifiableLockList{Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Limit is optional
	limit := reqBody.Limit
	if limit == 0 {
		limit = 100
	}

	ll := &lfs.VerifiableLockList{}
	locks, nextCursor, err := h.locksStore.Filtered(repoName, "",
		reqBody.Cursor,
		strconv.Itoa(limit))
	if err != nil {
		ll.Message = err.Error()
	} else {
		ll.NextCursor = nextCursor

		for _, l := range locks {
			if l.Owner.Name == user {
				ll.Ours = append(ll.Ours, l)
			} else {
				ll.Theirs = append(ll.Theirs, l)
			}
		}
	}

	h.JSON(w, ll, http.StatusOK)
}

func (h *Handler) handleCreateLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", metaMediaType)

	var lockRequest lfs.LockRequest
	if err := dec.Decode(&lockRequest); err != nil {
		h.JSON(w, &lfs.LockResponse{Message: err.Error()}, http.StatusBadRequest)
		return
	}

	locks, _, err := h.locksStore.Filtered(repoName, lockRequest.Path, "", "1")
	if err != nil {
		h.JSON(w, &lfs.LockResponse{Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	if len(locks) > 0 {
		h.JSON(w, &lfs.LockResponse{Message: "lock already created"}, http.StatusConflict)
		return
	}

	lock := &lfs.Lock{
		Id:       randomLockId(),
		Path:     lockRequest.Path,
		Owner:    lfs.User{Name: user},
		LockedAt: time.Now(),
	}

	if err := h.locksStore.Add(repoName, *lock); err != nil {
		h.JSON(w, &lfs.LockResponse{Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	h.JSON(w, &lfs.LockResponse{Lock: lock}, http.StatusCreated)
}

func (h *Handler) handleDeleteLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	lockId := vars["id"]
	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", metaMediaType)

	var unlockRequest lfs.UnlockRequest

	if len(lockId) == 0 {
		h.JSON(w, &lfs.UnlockResponse{Message: "invalid lock id"}, http.StatusBadRequest)
		return
	}

	if err := dec.Decode(&unlockRequest); err != nil {
		h.JSON(w, &lfs.UnlockResponse{Message: err.Error()}, http.StatusBadRequest)
		return
	}

	l, err := h.locksStore.Delete(repoName, user, lockId, unlockRequest.Force)
	if err != nil {
		if err == ErrNotOwner {
			h.JSON(w, &lfs.UnlockResponse{Message: err.Error()}, http.StatusForbidden)
		} else {
			h.JSON(w, &lfs.UnlockResponse{Message: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	if l == nil {
		h.JSON(w, &lfs.UnlockResponse{Message: "unable to find lock"}, http.StatusNotFound)
		return
	}

	h.JSON(w, &lfs.UnlockResponse{Lock: l}, http.StatusOK)
}

func randomLockId() string {
	var id [20]byte
	_, _ = rand.Read(id[:])
	return hex.EncodeToString(id[:])
}

func getUserFromRequest(r *http.Request) string {
	user := context.Get(r, "USER")
	if user == nil {
		return ""
	}
	return user.(string)
}
