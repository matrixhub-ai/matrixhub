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
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DB is a generic wrapper around gorm.DB for a specific model type T.
type DB[T any] struct {
	gorm.Interface[T]
	db *RawDB
}

type RawDB = gorm.DB

// New initializes a new GORM DB and returns a context containing it.
func New(driver, dsn string, debug bool) (*RawDB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite3":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("not support storage driver: %s", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, err
	}

	if debug {
		db = db.Debug()
	}

	return db, nil
}

// G returns a generic DB[T] instance for the given GormDB.
func G[T any](db *RawDB) *DB[T] {
	return &DB[T]{gorm.G[T](db), db}
}

// Transaction executes the given function within a database transaction.
func Transaction[T any](ctx context.Context, db *RawDB, fun func(ctx context.Context, db *DB[T]) error) error {
	return db.Transaction(func(tx *RawDB) error {
		txDB := G[T](tx)
		return fun(ctx, txDB)
	})
}

// MigrateTable migrates or creates the table for the model T.
func (d *DB[T]) MigrateTable(ctx context.Context) error {
	return d.db.AutoMigrate(new(T))
}

// ClearDeleted permanently deletes all records that have been soft deleted.
func (d *DB[T]) ClearDeleted(ctx context.Context) (rowsAffected int, err error) {
	r := d.db.
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Delete(new(T))
	return int(r.RowsAffected), r.Error
}

// Model is an alias for gorm.Model.
type Model = gorm.Model

// ChainInterface is an alias for gorm.ChainInterface.
type ChainInterface[T any] = gorm.ChainInterface[T]

// IsNotFoundError checks if the error is a record not found error.
func IsNotFoundError(err error) bool {
	return err == gorm.ErrRecordNotFound
}
