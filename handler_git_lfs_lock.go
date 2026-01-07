package gitd

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

var (
	ErrNotOwner = errors.New("Attempt to delete other user's lock")
)

func (h *Handler) registryLFSLock(r *mux.Router) {
	r.HandleFunc("/{repo:.+}/locks", h.requireAuth(h.handleGetLock)).Methods("GET").MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks/verify", h.requireAuth(h.handleLocksVerify)).Methods("POST").MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks", h.requireAuth(h.handleCreateLock)).Methods("POST").MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/locks/{id}/unlock", h.requireAuth(h.handleDeleteLock)).Methods("POST").MatcherFunc(metaMatcher)
}

func (h *Handler) handleGetLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"

	enc := json.NewEncoder(w)
	ll := &lfsLockList{}

	w.Header().Set("Content-Type", metaMediaType)

	locks, nextCursor, err := h.locksStore.Filtered(repo,
		r.FormValue("path"),
		r.FormValue("cursor"),
		r.FormValue("limit"))

	if err != nil {
		ll.Message = err.Error()
	} else {
		ll.Locks = locks
		ll.NextCursor = nextCursor
	}

	enc.Encode(ll)

}

func (h *Handler) handleLocksVerify(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"
	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", metaMediaType)

	reqBody := &lfsVerifiableLockRequest{}
	if err := dec.Decode(reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(&lfsVerifiableLockList{Message: err.Error()})
		return
	}

	// Limit is optional
	limit := reqBody.Limit
	if limit == 0 {
		limit = 100
	}

	ll := &lfsVerifiableLockList{}
	locks, nextCursor, err := h.locksStore.Filtered(repo, "",
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

	enc.Encode(ll)

}

func (h *Handler) handleCreateLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"
	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", metaMediaType)

	var lockRequest lfsLockRequest
	if err := dec.Decode(&lockRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(&lfsLockResponse{Message: err.Error()})
		return
	}

	locks, _, err := h.locksStore.Filtered(repo, lockRequest.Path, "", "1")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(&lfsLockResponse{Message: err.Error()})
		return
	}
	if len(locks) > 0 {
		w.WriteHeader(http.StatusConflict)
		enc.Encode(&lfsLockResponse{Message: "lock already created"})
		return
	}

	lock := &lfsLock{
		Id:       randomLockId(),
		Path:     lockRequest.Path,
		Owner:    lfsUser{Name: user},
		LockedAt: time.Now(),
	}

	if err := h.locksStore.Add(repo, *lock); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(&lfsLockResponse{Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	enc.Encode(&lfsLockResponse{
		Lock: lock,
	})

}

func (h *Handler) handleDeleteLock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repo := vars["repo"] + ".git"
	lockId := vars["id"]
	user := getUserFromRequest(r)

	dec := json.NewDecoder(r.Body)
	enc := json.NewEncoder(w)

	w.Header().Set("Content-Type", metaMediaType)

	var unlockRequest lfsUnlockRequest

	if len(lockId) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(&lfsUnlockResponse{Message: "invalid lock id"})
		return
	}

	if err := dec.Decode(&unlockRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(&lfsUnlockResponse{Message: err.Error()})
		return
	}

	l, err := h.locksStore.Delete(repo, user, lockId, unlockRequest.Force)
	if err != nil {
		if err == ErrNotOwner {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		enc.Encode(&lfsUnlockResponse{Message: err.Error()})
		return
	}
	if l == nil {
		w.WriteHeader(http.StatusNotFound)
		enc.Encode(&lfsUnlockResponse{Message: "unable to find lock"})
		return
	}

	enc.Encode(&lfsUnlockResponse{Lock: l})

}

func randomLockId() string {
	var id [20]byte
	rand.Read(id[:])
	return fmt.Sprintf("%x", id[:])
}

func getUserFromRequest(r *http.Request) string {
	user := context.Get(r, "USER")
	if user == nil {
		return ""
	}
	return user.(string)
}

type lfsUser struct {
	Name string `json:"name"`
}

type lfsLock struct {
	Id       string    `json:"id"`
	Path     string    `json:"path"`
	Owner    lfsUser   `json:"owner"`
	LockedAt time.Time `json:"locked_at"`
}

type lfsLockRequest struct {
	Path string `json:"path"`
}

type lfsLockResponse struct {
	Lock    *lfsLock `json:"lock"`
	Message string   `json:"message,omitempty"`
}

type lfsUnlockRequest struct {
	Force bool `json:"force"`
}

type lfsUnlockResponse struct {
	Lock    *lfsLock `json:"lock"`
	Message string   `json:"message,omitempty"`
}

type lfsLockList struct {
	Locks      []lfsLock `json:"locks"`
	NextCursor string    `json:"next_cursor,omitempty"`
	Message    string    `json:"message,omitempty"`
}

type lfsVerifiableLockRequest struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type lfsVerifiableLockList struct {
	Ours       []lfsLock `json:"ours"`
	Theirs     []lfsLock `json:"theirs"`
	NextCursor string    `json:"next_cursor,omitempty"`
	Message    string    `json:"message,omitempty"`
}
