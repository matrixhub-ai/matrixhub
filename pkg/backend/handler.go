package backend

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/pkg/lfs"
	"github.com/wzshiming/gitd/pkg/queue"
)

type Authenticator interface {
	Authenticate(user, password string) (string, bool)
}

// Handler
type Handler struct {
	rootDir string

	authenticate Authenticator

	locksStore   *lfs.LockDB
	contentStore *lfs.Content
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

// NewHandler creates a new Handler with the given repository directory.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		rootDir: "./data",
	}

	for _, opt := range opts {
		opt(h)
	}

	h.locksStore = lfs.NewLock(filepath.Join(h.rootDir, "lfs", "locks.db"))
	h.contentStore = lfs.NewContent(filepath.Join(h.rootDir, "lfs"))

	// Initialize queue store
	queueStore, err := queue.NewStore(filepath.Join(h.rootDir, "queue", "queue.db"))
	if err == nil {
		h.queueStore = queueStore
		h.queueWorker = queue.NewWorker(queueStore, 2)
		h.registerTaskHandlers()
		h.queueWorker.Start()
	}

	h.root = h.router()
	return h
}

// Close closes all resources used by the handler
func (h *Handler) Close() {
	if h.queueWorker != nil {
		h.queueWorker.Stop()
	}
	if h.queueStore != nil {
		h.queueStore.Close()
	}
	if h.locksStore != nil {
		h.locksStore.Close()
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// Queue management endpoints
	h.registryQueue(r)

	return r
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.authenticate == nil {
			context.Set(r, "USER", "anonymous")
			next(w, r)
			return
		}

		user, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="gitd"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		name, ret := h.authenticate.Authenticate(user, password)
		if !ret {
			w.Header().Set("WWW-Authenticate", `Basic realm="gitd"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		context.Set(r, "USER", name)
		next(w, r)
	}
}
