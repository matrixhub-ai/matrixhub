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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/permission"
)

func (h *Handler) handleGetLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	if h.permissionHookFunc != nil {
		op := permission.OperationReadRepo
		if ok, err := h.permissionHookFunc(r.Context(), op, repoName, permission.Context{}); err != nil {
			responseJSON(w, &lfs.VerifiableLockList{Message: err.Error()}, http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, &lfs.VerifiableLockList{Message: "permission denied"}, http.StatusForbidden)
			return
		}
	}

	ll := &lfs.LockList{}

	w.Header().Set("Content-Type", metaMediaType)

	limit := 0
	if limitStr := r.FormValue("limit"); limitStr != "" {
		strtLimit, err := strconv.Atoi(limitStr)
		if err != nil || strtLimit < 0 {
			responseJSON(w, &lfs.LockList{Message: "invalid limit parameter"}, http.StatusBadRequest)
			return
		}
		limit = strtLimit
	}

	locks, nextCursor, err := h.locksStorage.Filtered(repoName,
		r.FormValue("path"),
		r.FormValue("cursor"),
		limit,
	)

	if err != nil {
		ll.Message = err.Error()
	} else {
		ll.Locks = locks
		ll.NextCursor = nextCursor
	}

	responseJSON(w, ll, http.StatusOK)
}

func (h *Handler) handleLocksVerify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	if h.permissionHookFunc != nil {
		op := permission.OperationReadRepo
		if ok, err := h.permissionHookFunc(r.Context(), op, repoName, permission.Context{}); err != nil {
			responseJSON(w, &lfs.VerifiableLockList{Message: err.Error()}, http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, &lfs.VerifiableLockList{Message: "permission denied"}, http.StatusForbidden)
			return
		}
	}

	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", metaMediaType)

	reqBody := &lfs.VerifiableLockRequest{}
	if err := dec.Decode(reqBody); err != nil {
		responseJSON(w, &lfs.VerifiableLockList{Message: err.Error()}, http.StatusBadRequest)
		return
	}

	// Limit is optional
	limit := reqBody.Limit
	if limit == 0 {
		limit = 100
	}

	ll := &lfs.VerifiableLockList{}
	locks, nextCursor, err := h.locksStorage.Filtered(repoName,
		"",
		reqBody.Cursor,
		limit,
	)
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

	responseJSON(w, ll, http.StatusOK)
}

func (h *Handler) handleCreateLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]

	if h.permissionHookFunc != nil {
		op := permission.OperationUpdateRepo
		if ok, err := h.permissionHookFunc(r.Context(), op, repoName, permission.Context{}); err != nil {
			responseJSON(w, &lfs.VerifiableLockList{Message: err.Error()}, http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, &lfs.VerifiableLockList{Message: "permission denied"}, http.StatusForbidden)
			return
		}
	}

	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", metaMediaType)

	var lockRequest lfs.LockRequest
	if err := dec.Decode(&lockRequest); err != nil {
		responseJSON(w, &lfs.LockResponse{Message: err.Error()}, http.StatusBadRequest)
		return
	}

	locks, _, err := h.locksStorage.Filtered(repoName, lockRequest.Path, "", 1)
	if err != nil {
		responseJSON(w, &lfs.LockResponse{Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	if len(locks) > 0 {
		responseJSON(w, &lfs.LockResponse{Message: "lock already created"}, http.StatusConflict)
		return
	}

	lock := &lfs.Lock{
		Id:       randomLockId(),
		Path:     lockRequest.Path,
		Owner:    lfs.User{Name: user},
		LockedAt: time.Now(),
	}

	if err := h.locksStorage.Add(repoName, *lock); err != nil {
		responseJSON(w, &lfs.LockResponse{Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	responseJSON(w, &lfs.LockResponse{Lock: lock}, http.StatusCreated)
}

func (h *Handler) handleDeleteLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	lockId := vars["id"]

	if h.permissionHookFunc != nil {
		op := permission.OperationUpdateRepo
		if ok, err := h.permissionHookFunc(r.Context(), op, repoName, permission.Context{}); err != nil {
			responseJSON(w, &lfs.VerifiableLockList{Message: err.Error()}, http.StatusInternalServerError)
			return
		} else if !ok {
			responseJSON(w, &lfs.VerifiableLockList{Message: "permission denied"}, http.StatusForbidden)
			return
		}
	}

	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)

	w.Header().Set("Content-Type", metaMediaType)

	var unlockRequest lfs.UnlockRequest

	if len(lockId) == 0 {
		responseJSON(w, &lfs.UnlockResponse{Message: "invalid lock id"}, http.StatusBadRequest)
		return
	}

	if err := dec.Decode(&unlockRequest); err != nil {
		responseJSON(w, &lfs.UnlockResponse{Message: err.Error()}, http.StatusBadRequest)
		return
	}

	l, err := h.locksStorage.Delete(repoName, user, lockId, unlockRequest.Force)
	if err != nil {
		if err == lfs.ErrNotOwner {
			responseJSON(w, &lfs.UnlockResponse{Message: err.Error()}, http.StatusForbidden)
		} else {
			responseJSON(w, &lfs.UnlockResponse{Message: err.Error()}, http.StatusInternalServerError)
		}
		return
	}
	if l == nil {
		responseJSON(w, &lfs.UnlockResponse{Message: "unable to find lock"}, http.StatusNotFound)
		return
	}

	responseJSON(w, &lfs.UnlockResponse{Lock: l}, http.StatusOK)
}

func randomLockId() string {
	var id [20]byte
	_, _ = rand.Read(id[:])
	return hex.EncodeToString(id[:])
}

func getUserFromRequest(r *http.Request) string {
	userInfo, _ := authenticate.GetUserInfo(r.Context())
	return userInfo.User
}
