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
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/infra/log"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func shouldMigrate(db *gorm.DB, sqlPath string, migrationsTable string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if sqlPath, err = filepath.Abs(sqlPath); err != nil {
		return err
	}
	dialector := db.Name()
	sqlPath = fmt.Sprintf("file://%s", filepath.Join(sqlPath, dialector))

	var session *migrate.Migrate
	switch dialector {
	case "mysql":
		session, err = newMysqlMigrateSession(sqlDB, sqlPath, migrationsTable)
	case "postgres":
		session, err = newPostgresMigrateSession(sqlDB, sqlPath, migrationsTable)
	default:
		return fmt.Errorf("migration not support database: %s", dialector)
	}
	if err != nil {
		return err
	}

	log.Infof("migrate sql path: %s", sqlPath)
	if err := session.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Info("database has migrated to latest version, nothing changed")
			return nil
		}
		return fmt.Errorf("migrate failed: %w", err)
	}

	version, dirty, err := session.Version()
	log.Infow("database migrated successfully", "version", version, "dirty", dirty, "err", err)
	return nil
}

func newPostgresMigrateSession(db *sql.DB, sqlPath string, migrationsTable string) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(sqlPath, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("migrate.NewWithDatabaseInstance:%v", err)
	}
	return m, nil
}

func newMysqlMigrateSession(db *sql.DB, sqlPath string, migrationsTable string) (*migrate.Migrate, error) {
	driver, err := mysql.WithInstance(db, &mysql.Config{
		MigrationsTable: migrationsTable,
	})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(sqlPath, "publish", driver)
	if err != nil {
		return nil, fmt.Errorf("migrate.NewWithDatabaseInstance:%v", err)
	}
	return m, nil
}
