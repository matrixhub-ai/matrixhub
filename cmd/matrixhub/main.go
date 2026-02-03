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

	"github.com/urfave/cli/v2"

	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

var Version = "0.1.0"

const configFlag = "config"

func main() {
	app := cli.NewApp()
	app.Name = "matrixhub"
	app.Version = Version
	app.Usage = "MATRIXBUB"
	app.Commands = []*cli.Command{
		runAPIServerCommand,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "matrixhub: %s", err)
		os.Exit(1)
	}
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
