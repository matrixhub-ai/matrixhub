package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jinzhu/configor"

	"github.com/matrixhub-ai/matrixhub/internal/infra/db"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type Config struct {
	Debug         bool             `yaml:"debug"`
	Log           log.Config       `yaml:"log"`
	APIServer     *APIServerConfig `yaml:"apiServer" validate:"required"`
	MigrationPath string           `yaml:"migrationPath" validate:"required"`

	Database db.Config `yaml:"database" validate:"required"`
}

type APIServerConfig struct {
	Port int `yaml:"port" env:"APISERVER_PORT" validate:"required"`
}

func Init(configPath, sqlPath string) (*Config, error) {
	cfg := new(Config)
	if err := configor.Load(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to load config(%s): %v", configPath, err)
	}

	dsn, ok := os.LookupEnv(db.MATRIXHUB_DSN_ENV)
	if ok {
		cfg.Database.DSN = dsn
	} else {
		log.Warn("failed to find ghippo dsn from env")
	}

	if cfg.Database.Migrate {
		cfg.Database.SQLPath = filepath.Join(cfg.MigrationPath, sqlPath)
	}
	cfg.Database.Debug = cfg.Debug

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config invalid: %v", err)
	}

	return cfg, nil
}

func (config *Config) Validate() error {
	fileInfo, err := os.Stat(config.MigrationPath)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("%s is not dir", fileInfo.Name())
	}

	return nil
}
