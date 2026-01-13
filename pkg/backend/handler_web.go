package backend

import (
	"github.com/gorilla/mux"
	"github.com/matrixhub-ai/matrixhub/web"
)

// registerWebHandlers registers the web interface handlers.
func (h *Handler) registerWeb(r *mux.Router) {
	r.NotFoundHandler = web.Web
}
