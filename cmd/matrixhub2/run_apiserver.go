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
