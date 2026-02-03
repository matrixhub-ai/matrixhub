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
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver"
)

func runAPIServer(configPath string) error {
	cfg, deferFunc, err := runInit(configPath, "sql")
	if err != nil {
		return err
	}
	defer deferFunc()

	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	apiServer := apiserver.NewAPIServer(cfg)
	errorCh := apiServer.Run()

	sign := make(chan os.Signal, 1)
	signal.Notify(sign, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sign:
	case <-errorCh:
	}

	apiServer.Shutdown()
	return nil
}

var runAPIServerCommand = &cli.Command{
	Name:  "apiserver",
	Usage: "run matrixhub api server",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    configFlag,
			Aliases: []string{"c"},
			Value:   "/etc/matrixhub/config.yaml",
			Usage:   "matrixhub config file path",
		},
	},
	Action: func(c *cli.Context) error {
		return runAPIServer(c.String(configFlag))
	},
}
