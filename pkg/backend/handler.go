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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/wzshiming/httpseek"

	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
	"github.com/matrixhub-ai/matrixhub/pkg/queue"
)

type Authenticator interface {
	Authenticate(user, password string) (string, bool)
}

// Handler
type Handler struct {
	rootDir         string
	repositoriesDir string

	httpClient *http.Client
	lfsClient  *lfs.Client

	authenticate Authenticator

	locksStore   *lfs.LockDB
	contentStore *lfs.Content
	s3Store      *lfs.S3
	queueStore   *queue.Store
	queueWorker  *queue.Worker
	root         *mux.Router
}

type Option func(*Handler)

func WithAuthenticate(auth Authenticator) Option {
	return func(h *Handler) {
		h.authenticate = auth
	}
}

func WithRootDir(rootDir string) Option {
	return func(h *Handler) {
		h.rootDir = rootDir
	}
}

// WithQueueWorkers sets the number of concurrent queue workers (default: 2)
func WithQueueWorkers(count int) Option {
	return func(h *Handler) {
		if h.queueWorker != nil {
			h.queueWorker.Stop()
		}
		if h.queueStore != nil {
			h.queueWorker = queue.NewWorker(h.queueStore, count)
		}
	}
}

// WithLFSS3 configures the LFS S3 storage backend.
func WithLFSS3(s3Store *lfs.S3) Option {
	return func(h *Handler) {
		h.s3Store = s3Store
	}
}

// NewHandler creates a new Handler with the given repository directory.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		rootDir: "./data",
		httpClient: &http.Client{
			Transport: httpseek.NewMustReaderTransport(http.DefaultTransport,
				func(r *http.Request, retry int, err error) error {
					log.Printf("Retry %d for %s due to error: %v\n", retry+1, r.URL.String(), err)
					if retry >= 5 {
						return fmt.Errorf("max retries reached for %s: %w", r.URL.String(), err)
					}
					// Simple backoff strategy
					time.Sleep(time.Duration(retry+1) * time.Second)
					return nil
				}),
			Timeout: 30 * time.Minute, // Long timeout for large files
		},
	}

	for _, opt := range opts {
		opt(h)
	}

	h.locksStore = lfs.NewLock(filepath.Join(h.rootDir, "lfs", "locks.db"))
	h.contentStore = lfs.NewContent(filepath.Join(h.rootDir, "lfs"))

	// Initialize queue store
	queueStore, err := queue.NewStore(filepath.Join(h.rootDir, "queue", "queue.db"))
	if err != nil {
		log.Printf("Failed to initialize queue store: %v\n", err)
	} else {
		h.queueStore = queueStore
		h.queueWorker = queue.NewWorker(queueStore, 2)
		h.registerTaskHandlers()
		h.queueWorker.Start()
	}
	h.repositoriesDir = filepath.Join(h.rootDir, "repositories")
	h.lfsClient = lfs.NewClient(h.httpClient)
	h.root = h.router()
	return h
}

// Close closes all resources used by the handler
func (h *Handler) Close() {
	if h.queueWorker != nil {
		h.queueWorker.Stop()
	}
	if h.queueStore != nil {
		_ = h.queueStore.Close()
	}
	if h.locksStore != nil {
		_ = h.locksStore.Close()
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.authenticate == nil {
		context.Set(r, "USER", "anonymous")
		h.root.ServeHTTP(w, r)
		return
	}

	user, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="matrixhub"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name, ret := h.authenticate.Authenticate(user, password)
	if !ret {
		w.Header().Set("WWW-Authenticate", `Basic realm="matrixhub"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	context.Set(r, "USER", name)
	h.root.ServeHTTP(w, r)
}

func (h *Handler) router() *mux.Router {
	r := mux.NewRouter()

	// Git protocol endpoints
	h.registryGit(r)

	// Git LFS protocol endpoints
	h.registryLFS(r)
	h.registryLFSLock(r)

	// Repository management endpoints
	h.registryRepositoriesImport(r)
	h.registryRepositoriesInfo(r)
	h.registryRepositories(r)

	// HuggingFace-compatible API endpoints
	h.registryHuggingFace(r)

	// Queue management endpoints
	h.registryQueue(r)

	h.registerWeb(r)

	return r
}

func (h *Handler) Text(w http.ResponseWriter, text string, sc int) {
	header := w.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/plain; charset=utf-8")
	}

	if sc >= http.StatusBadRequest {
		header.Del("Content-Length")
		header.Set("X-Content-Type-Options", "nosniff")
	}

	if sc != 0 {
		w.WriteHeader(sc)
	}

	if text == "" {
		return
	}

	_, _ = io.WriteString(w, text)
}

func (h *Handler) JSON(w http.ResponseWriter, data any, sc int) {
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
