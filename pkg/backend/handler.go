package backend

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/wzshiming/gitd/pkg/lfs"
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
	h.root = h.router()
	return h
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
