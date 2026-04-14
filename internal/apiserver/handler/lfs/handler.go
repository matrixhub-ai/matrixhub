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
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/storage"
)

// Handler handles HTTP requests for Git LFS API endpoints, including batch operations and object content management.
type Handler struct {
	storage            *storage.Storage
	root               *mux.Router
	next               http.Handler
	middlewares        []mux.MiddlewareFunc
	lfsStorage         lfs.Storage
	locksStorage       *lfs.LockStorage
	permissionHookFunc permission.PermissionHookFunc
	tokenSignValidator authenticate.TokenSignValidator
	mirror             *mirror.Mirror
}

// Option defines a functional option for configuring the Handler.
type Option func(*Handler)

// WithMiddlewares sets the middlewares for the router.
func WithMiddlewares(middlewares ...mux.MiddlewareFunc) Option {
	return func(h *Handler) {
		h.middlewares = middlewares
	}
}

// WithStorage sets the storage backend for the handler. This is required.
func WithStorage(storage *storage.Storage) Option {
	return func(h *Handler) {
		h.storage = storage
	}
}

// WithNext sets the next http.Handler to call if the request is not handled by this handler.
func WithNext(next http.Handler) Option {
	return func(h *Handler) {
		h.next = next
	}
}

// WithPermissionHookFunc sets the authentication hook for verifying operations.
func WithPermissionHookFunc(fn permission.PermissionHookFunc) Option {
	return func(h *Handler) {
		h.permissionHookFunc = fn
	}
}

// WithTokenSignValidator sets the token signer for signing LFS batch response action headers.
func WithTokenSignValidator(signer authenticate.TokenSignValidator) Option {
	return func(h *Handler) {
		h.tokenSignValidator = signer
	}
}

// WithLFSStorage configures the LFS storage backend.
func WithLFSStorage(storage lfs.Storage) Option {
	return func(h *Handler) {
		h.lfsStorage = storage
	}
}

// WithMirror sets the mirror to use for repository synchronization. If not provided,
// a mirror will be created when mirrorSourceFunc is set.
func WithMirror(m *mirror.Mirror) Option {
	return func(h *Handler) {
		h.mirror = m
	}
}

// NewHandler creates a new Handler with the given repository directory.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		root:         mux.NewRouter(),
		locksStorage: lfs.NewLock(),
	}

	for _, opt := range opts {
		opt(h)
	}

	h.register()
	return h
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.root.ServeHTTP(w, r)
}

func (h *Handler) register() {
	// Git LFS protocol endpoints
	h.registryLFS(h.root)
	h.registryLFSLock(h.root)

	h.root.Use(h.middlewares...)
	h.root.NotFoundHandler = h.next
}

func (h *Handler) registryLFS(r *mux.Router) {
	r.HandleFunc("/{repo:.+}.git/info/lfs/objects/batch", h.handleBatch).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/{repo:.+}/info/lfs/objects/batch", h.handleBatch).Methods(http.MethodPost).MatcherFunc(metaMatcher)
	r.HandleFunc("/objects/{oid}", h.handleGetContent).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/objects/{oid}", h.handlePutContent).Methods(http.MethodPut)
	r.HandleFunc("/objects/{oid}/verify", h.handleVerifyObject).Methods(http.MethodPost)
}

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

func responseJSON(w http.ResponseWriter, data any, sc int) {
	header := w.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json; charset=utf-8")
	}

	if sc >= http.StatusBadRequest {
		header.Del("Content-Length")
		header.Set("X-Content-Type-Options", "nosniff")
	}

	if sc != 0 {
		w.WriteHeader(sc)
	}

	if data == nil {
		_, _ = w.Write([]byte("{}"))
		return
	}

	switch t := data.(type) {
	case error:
		var dataErr struct {
			Error string `json:"error"`
		}
		dataErr.Error = t.Error()
		data = dataErr
	case string:
		var dataErr struct {
			Error string `json:"error"`
		}
		dataErr.Error = t
		data = dataErr
	}

	enc := json.NewEncoder(w)
	_ = enc.Encode(data)
}
