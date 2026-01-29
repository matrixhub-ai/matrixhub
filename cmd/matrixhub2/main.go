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
