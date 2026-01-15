package backend

import (
	"github.com/gorilla/mux"
	vibecoding "github.com/matrixhub-ai/matrixhub/vibe-coding"
)

// registerWebHandlers registers the web interface handlers.
func (h *Handler) registerWeb(r *mux.Router) {
	r.NotFoundHandler = vibecoding.Web
}
