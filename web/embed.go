//go:build embedweb

package web

import (
	"embed"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed dist
var embedFS embed.FS

var fs = http.FS(embedFS)
var index http.File
var logo http.File

func init() {
	var err error
	index, err = fs.Open("dist/index.html")
	if err != nil {
		log.Fatal("Failed to open embedded index.html: %v", err)
	}

	logo, err = fs.Open("dist/vite.svg")
	if err != nil {
		log.Fatal("Failed to open embedded vite.svg: %v", err)
	}
}

var Web http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeContent(w, r, "index.html", time.Time{}, index)
		return
	}

	if logo != nil && r.URL.Path == "/vite.svg" {
		http.ServeContent(w, r, "vite.svg", time.Time{}, logo)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/assets/") {
		file, err := fs.Open(path.Join("dist", r.URL.Path))
		if err == nil {
			http.ServeContent(w, r, path.Base(r.URL.Path), time.Time{}, file)
			return
		}
	}
	http.ServeContent(w, r, "index.html", time.Time{}, index)
})
