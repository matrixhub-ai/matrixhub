package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
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
