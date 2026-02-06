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

	"github.com/jinzhu/configor"
)

type Config struct {
	Debug     bool             `yaml:"debug"`
	Log       *Log             `yaml:"log"`
	APIServer *APIServerConfig `yaml:"apiServer" validate:"required"`

	Database *Database `yaml:"database" validate:"required"`
}

type Log struct {
	Level   string `yaml:"level"`
	Encoder string `yaml:"encoder"`

	FilePath       string `yaml:"filePath"`       // Old log files are in the same directory
	FileMaxSize    int    `yaml:"fileMaxSize"`    // Maximum size of a single log file (MB)
	FileMaxBackups int    `yaml:"fileMaxBackups"` // Maximum number of old log files to retain
	FileMaxAges    int    `yaml:"fileMaxAges"`    // Maximum days to retain old files, determined by filename and timestamp
	Compress       bool   `yaml:"compress"`       // Whether to compress archived old files
}

type Database struct {
	Debug                  bool   `yaml:"debug"`
	Driver                 string `yaml:"driver"`
	AccessType             string `yaml:"accessType"`
	DSN                    string `yaml:"dsn"`
	MaxOpenConns           int    `yaml:"maxOpenConns"`
	MaxIdleConns           int    `yaml:"maxIdleConns"`
	ConnMaxLifetimeSeconds int    `yaml:"connMaxLifetimeSeconds"`
	ConnMaxIdleSeconds     int    `yaml:"connMaxIdleSeconds"`
}

type APIServerConfig struct {
	Port int `yaml:"port" env:"SERVER_PORT" validate:"required"`
}

func Init(configPath string) (*Config, error) {
	cfg := new(Config)
	if configPath != "" {
		if err := configor.Load(cfg, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config(%s): %v", configPath, err)
		}
	}

	if cfg.APIServer == nil {
		cfg.APIServer = &APIServerConfig{
			Port: 9527,
		}
	}

	if cfg.Database == nil {
		cfg.Database = &Database{
			Driver: "sqlite3",
			DSN:    "matrixhub.db",
		}
	}

	if cfg.Log == nil {
		cfg.Log = &Log{
			Level:   "info",
			Encoder: "json",
		}
	}

	cfg.Database.Debug = cfg.Debug

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config invalid: %v", err)
	}

	return cfg, nil
}

func (config *Config) Validate() error {

	return nil
}
