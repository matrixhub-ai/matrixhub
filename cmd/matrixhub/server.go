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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"

	"github.com/matrixhub-ai/matrixhub/pkg/config"
	"github.com/matrixhub-ai/matrixhub/pkg/log"
	"github.com/matrixhub-ai/matrixhub/pkg/server"
)

const configFlag = "config"

var runServerCommand = &cli.Command{
	Name:  "server",
	Usage: "run matrixhub server",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    configFlag,
			Aliases: []string{"c"},
			Value:   "",
			Usage:   "matrixhub config file path",
		},
	},
	Action: func(c *cli.Context) error {
		return runServer(c.String(configFlag))
	},
}

func runServer(configPath string) error {
	cfg, err := config.Init(configPath)
	if err != nil {
		return fmt.Errorf("config init failed: %v", err)
	}

	if err := log.SetLoggerWithConfig(cfg.Debug, log.Config{
		Level:          cfg.Log.Level,
		Encoder:        cfg.Log.Encoder,
		FilePath:       cfg.Log.FilePath,
		FileMaxSize:    cfg.Log.FileMaxSize,
		FileMaxBackups: cfg.Log.FileMaxBackups,
		FileMaxAges:    cfg.Log.FileMaxAges,
		Compress:       cfg.Log.Compress,
	}); err != nil {
		return fmt.Errorf("log init failed: %v", err)
	}

	defer log.Sync()

	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	apiServer, err := server.NewServer(cfg)
	if err != nil {
		return err
	}
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
