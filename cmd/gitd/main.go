// Command gitd is a git server that uses the git binary to serve repositories over HTTP.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wzshiming/gitd/internal/handlers"
	"github.com/wzshiming/gitd/pkg/backend"
)

var (
	addr    = ":8080"
	repoDir = "./data"
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "HTTP server address")
	flag.StringVar(&repoDir, "repo", "./data", "Directory containing git repositories")
	flag.Parse()
}

func main() {
	absRootDir, err := filepath.Abs(repoDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path of repo directory: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Starting gitd server on %s, serving repositories from %s\n", addr, absRootDir)

	var handler http.Handler
	handler = backend.NewHandler(
		backend.WithRootDir(absRootDir),
	)
	handler = handlers.CompressHandler(handler)
	handler = handlers.LoggingHandler(os.Stderr, handler)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
