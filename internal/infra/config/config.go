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
		log.Warn("failed to find matrixhub dsn from env")
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
