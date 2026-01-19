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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matrixhub-ai/matrixhub/internal/handlers"
	"github.com/matrixhub-ai/matrixhub/pkg/backend"
	"github.com/matrixhub-ai/matrixhub/pkg/version"
)

var (
	addr        = ":9527"
	dataDir     = "./data"
	showVersion = false
)

func init() {
	flag.StringVar(&addr, "addr", ":9527", "HTTP server address")
	flag.StringVar(&dataDir, "data", "./data", "Directory containing git repositories")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()
}

func main() {
	if showVersion {
		version.PrintVersion()
		os.Exit(0)
	}

	absRootDir, err := filepath.Abs(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path of repo directory: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Starting matrixhub server on %s, serving repositories from %s\n", addr, absRootDir)

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
