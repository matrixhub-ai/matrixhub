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
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	"github.com/matrixhub-ai/hfd/pkg/storage"

	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

// Handler handles HTTP requests for Git operations, including service discovery and upload/receive pack endpoints.
type Handler struct {
	storage             *storage.Storage
	root                *mux.Router
	next                http.Handler
	middlewares         []mux.MiddlewareFunc
	permissionHookFunc  permission.PermissionHookFunc
	preReceiveHookFunc  receive.PreReceiveHookFunc
	postReceiveHookFunc receive.PostReceiveHookFunc
	mirror              *mirror.Mirror
	modelService        model.IModelService
	gitRepo             git.IGitRepo
	authzService        authz.IAuthzService
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

// WithPermissionHookFunc sets the permission hook for verifying operations.
func WithPermissionHookFunc(fn permission.PermissionHookFunc) Option {
	return func(h *Handler) {
		h.permissionHookFunc = fn
	}
}

// WithPreReceiveHookFunc sets the pre-receive hook called before a git push is processed.
// If the hook returns an error, the push is rejected.
func WithPreReceiveHookFunc(fn receive.PreReceiveHookFunc) Option {
	return func(h *Handler) {
		h.preReceiveHookFunc = fn
	}
}

// WithPostReceiveHookFunc sets the post-receive hook called after a git push is processed.
// Errors from this hook are logged but do not affect the push result.
func WithPostReceiveHookFunc(fn receive.PostReceiveHookFunc) Option {
	return func(h *Handler) {
		h.postReceiveHookFunc = fn
	}
}

// WithMirror sets the mirror to use for repository synchronization. If not provided,
// a mirror will be created when mirrorSourceFunc is set.
func WithMirror(m *mirror.Mirror) Option {
	return func(h *Handler) {
		h.mirror = m
	}
}

// WithServices sets the services for the router.
func WithServices(model model.IModelService, git git.IGitRepo, authz authz.IAuthzService) Option {
	return func(h *Handler) {
		h.modelService = model
		h.gitRepo = git
		h.authzService = authz
	}
}

// NewHandler creates a new Handler with the given repository directory.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		root: mux.NewRouter(),
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
	// Git protocol endpoints
	h.registryGit(h.root)

	h.root.Use(h.middlewares...)
	h.root.NotFoundHandler = h.next
}

func (h *Handler) registryGit(r *mux.Router) {
	r.HandleFunc("/{repo:.+}.git/info/refs", h.handleInfoRefs).Methods(http.MethodGet)
	r.HandleFunc("/{repo:.+}/info/refs", h.handleInfoRefs).Methods(http.MethodGet)
	r.HandleFunc("/{repo:.+}.git/git-upload-pack", h.handleUploadPack).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}/git-upload-pack", h.handleUploadPack).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}.git/git-receive-pack", h.handleReceivePack).Methods(http.MethodPost)
	r.HandleFunc("/{repo:.+}/git-receive-pack", h.handleReceivePack).Methods(http.MethodPost)
}

func responseText(w http.ResponseWriter, text string, sc int) {
	header := w.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/plain; charset=utf-8")
	}

	if sc >= http.StatusBadRequest {
		header.Del("Content-Length")
		header.Set("X-Content-Type-Options", "nosniff")
	}

	if sc == http.StatusUnauthorized {
		header.Set("WWW-Authenticate", `Basic realm="hfd"`)
	}
	if sc != 0 {
		w.WriteHeader(sc)
	}

	if text == "" {
		return
	}

	_, _ = io.WriteString(w, text)
}
