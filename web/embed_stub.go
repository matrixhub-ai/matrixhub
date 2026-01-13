//go:build !embedweb
// +build !embedweb

package web

import (
	"net/http"
)

var Web http.Handler
