//go:build !embedweb
// +build !embedweb

package vibecoding

import (
	"net/http"
)

var Web http.Handler
