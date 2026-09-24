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
	"github.com/spf13/cobra"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
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

	apiServer, err := apiserver.NewAPIServer(cfg)
	if err != nil {
		return err
	}
	errorCh := apiServer.Start()

	sign := make(chan os.Signal, 1)
	signal.Notify(sign, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sign:
	case <-errorCh:
	}

	apiServer.Shutdown()
	return nil
}

var apiserverCmd = &cobra.Command{
	Use:   "apiserver",
	Short: "run matrixhub api server",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString(configFlag)
		return runAPIServer(configPath)
	},
}

func init() {
	rootCmd.AddCommand(apiserverCmd)

	apiserverCmd.Flags().StringP(configFlag, "c", "./config/config.yaml", "matrixhub config file path")
}

func runInit(configPath string, sqlPath string) (*config.Config, func(), error) {
	cfg, err := config.Init(configPath, sqlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("config init failed: %v", err)
	}

	if err := log.SetLoggerWithConfig(cfg.Debug, cfg.Log); err != nil {
		return nil, nil, fmt.Errorf("log init failed: %v", err)
	}

	return cfg, func() {
		log.Sync()
	}, nil
}
