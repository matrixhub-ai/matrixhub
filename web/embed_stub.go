//go:build !embedweb

package web

import (
	"net/http"
)

// Web is a stub handler when web embedding is disabled.
var Web http.Handler
