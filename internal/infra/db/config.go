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

package db

import (
	"fmt"
	"net/url"
	"path/filepath"
)

const (
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
	DriverSQLite   = "sqlite"

	MATRIXHUB_DSN_ENV = "MATRIXHUB_DATABASE_DSN"
)

type Config struct {
	Debug                  bool   `yaml:"debug"`
	Driver                 string `yaml:"driver"`
	AccessType             string `yaml:"accessType"`
	DSN                    string `yaml:"dsn"`
	MaxOpenConns           int    `yaml:"maxOpenConns"`
	MaxIdleConns           int    `yaml:"maxIdleConns"`
	ConnMaxLifetimeSeconds int    `yaml:"connMaxLifetimeSeconds"`
	ConnMaxIdleSeconds     int    `yaml:"connMaxIdleSeconds"`

	SQLPath string `yaml:"sqlPath"`
	Migrate bool   `yaml:"migrate"`
}

func (config *Config) applyDefaults() {
	if config.Driver != DriverSQLite {
		return
	}

	if config.MaxOpenConns == 0 {
		config.MaxOpenConns = 1
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 1
	}
}

func (config *Config) validate() error {
	switch config.Driver {
	case DriverMySQL, DriverPostgres, DriverSQLite:
	default:
		return fmt.Errorf("unsupported database driver %q", config.Driver)
	}

	if config.DSN == "" {
		return fmt.Errorf("database DSN is required for driver %q", config.Driver)
	}
	if config.MaxOpenConns < 0 {
		return fmt.Errorf("maxOpenConns must not be negative")
	}
	if config.MaxIdleConns < 0 {
		return fmt.Errorf("maxIdleConns must not be negative")
	}
	if config.ConnMaxLifetimeSeconds < 0 {
		return fmt.Errorf("connMaxLifetimeSeconds must not be negative")
	}
	if config.ConnMaxIdleSeconds < 0 {
		return fmt.Errorf("connMaxIdleSeconds must not be negative")
	}
	if config.MaxOpenConns > 0 && config.MaxIdleConns > config.MaxOpenConns {
		return fmt.Errorf("maxIdleConns must not exceed maxOpenConns")
	}
	if config.Migrate && config.SQLPath == "" {
		return fmt.Errorf("sqlPath is required when database migration is enabled")
	}

	if config.Driver == DriverSQLite {
		if config.MaxOpenConns != 1 {
			return fmt.Errorf("sqlite requires maxOpenConns to be 1")
		}
		if config.MaxIdleConns > 1 {
			return fmt.Errorf("sqlite requires maxIdleConns to be at most 1")
		}
	}

	return nil
}

func DefaultSQLiteDSN(dataDir string) (string, error) {
	databasePath, err := filepath.Abs(filepath.Join(dataDir, "matrixhub.db"))
	if err != nil {
		return "", fmt.Errorf("resolve sqlite database path: %w", err)
	}

	query := url.Values{}
	query.Set("_busy_timeout", "5000")
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_synchronous", "FULL")
	query.Set("_txlock", "immediate")

	return (&url.URL{
		Scheme:   "file",
		Path:     databasePath,
		RawQuery: query.Encode(),
	}).String(), nil
}
