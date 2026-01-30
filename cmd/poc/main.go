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
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/matrixhub-ai/matrixhub/internal/handlers"
	"github.com/matrixhub-ai/matrixhub/pkg/backend"
	"github.com/matrixhub-ai/matrixhub/pkg/lfs"
	"github.com/matrixhub-ai/matrixhub/pkg/s3fs"
	"github.com/matrixhub-ai/matrixhub/pkg/version"
)

var (
	addr           = ":9527"
	dataDir        = "./data"
	s3Repositories = false
	s3SignEndpoint = ""
	s3Endpoint     = ""
	s3AccessKey    = ""
	s3SecretKey    = ""
	s3Bucket       = ""
	s3UsePathStyle = false
	showVersion    = false
)

func init() {
	flag.StringVar(&addr, "addr", ":9527", "HTTP server address")
	flag.StringVar(&dataDir, "data", "./data", "Directory containing git repositories")
	flag.BoolVar(&s3Repositories, "s3-repositories", false, "Store repositories in S3")
	flag.StringVar(&s3Endpoint, "s3-endpoint", "", "S3 endpoint")
	flag.StringVar(&s3SignEndpoint, "s3-sign-endpoint", "", "S3 signing endpoint (if different from s3-endpoint)")
	flag.StringVar(&s3AccessKey, "s3-access-key", "", "S3 access key")
	flag.StringVar(&s3SecretKey, "s3-secret-key", "", "S3 secret key")
	flag.StringVar(&s3Bucket, "s3-bucket", "", "S3 bucket name")
	flag.BoolVar(&s3UsePathStyle, "s3-use-path-style", false, "Use path style for S3 URLs")
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

	opts := []backend.Option{
		backend.WithRootDir(absRootDir),
	}

	log.Printf("Starting matrixhub server on %s, serving repositories from %s\n", addr, absRootDir)

	if s3Endpoint != "" && s3Bucket != "" {
		if s3Repositories {
			repositoriesDir := filepath.Join(absRootDir, "repositories")
			log.Printf("Mounting S3 bucket %s at %s\n", s3Bucket, repositoriesDir)
			err := s3fs.Mount(
				context.Background(),
				repositoriesDir,
				s3Endpoint,
				s3AccessKey,
				s3SecretKey,
				s3Bucket,
				"/repositories/",
				s3UsePathStyle,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error mounting S3 bucket: %v\n", err)
				os.Exit(1)
			}
			defer func() {
				log.Printf("Unmounting S3 bucket from %s\n", repositoriesDir)
				if err := s3fs.Unmount(context.Background(), repositoriesDir); err != nil {
					fmt.Fprintf(os.Stderr, "Error unmounting S3 bucket: %v\n", err)
				}
			}()
		}

		opts = append(opts,
			backend.WithLFSS3(
				lfs.NewS3(
					"lfs",
					s3Endpoint,
					s3AccessKey,
					s3SecretKey,
					s3Bucket,
					s3UsePathStyle,
					s3SignEndpoint,
				),
			),
		)
	}

	var handler http.Handler
	handler = backend.NewHandler(
		opts...,
	)

	handler = handlers.CompressHandler(handler)
	handler = handlers.LoggingHandler(os.Stderr, handler)
	if err := http.ListenAndServe(addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}
}
