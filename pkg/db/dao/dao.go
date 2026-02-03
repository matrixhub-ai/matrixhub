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

package dao

import (
	"context"

	"github.com/matrixhub-ai/matrixhub/pkg/db"
)

// Model represents a database model with an ID.
type Model interface {
	GetID() uint
}

// DAO is a generic data access object for models of type T.
type DAO[T Model] struct {
	db *db.DB[T]
}

// NewDAO creates a new DAO for models of type T.
func NewDAO[T Model](db *db.DB[T]) *DAO[T] {
	return &DAO[T]{
		db: db,
	}
}

// MigrateTable migrates the database table for the model T.
func (d *DAO[T]) MigrateTable(ctx context.Context) error {
	return d.db.
		MigrateTable(ctx)
}

// QueryFunc defines a function type for modifying a database query chain.
type QueryFunc[T Model] func(db.ChainInterface[T]) db.ChainInterface[T]

// List retrieves a list of models of type T with pagination and optional query modifications.
func (d *DAO[T]) List(ctx context.Context, cursor string, limit int, queries ...QueryFunc[T]) (outputs []T, nextCursor string, err error) {
	query := d.db.
		Limit(limit)

	if cursor != "" {
		id, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		query = query.Where("id > ?", id)
	}

	for _, q := range queries {
		query = q(query)
	}

	items, err := query.Find(ctx)
	if err != nil {
		return nil, "", err
	}

	if len(items) == limit {
		nextCursor = encodeCursor(items[len(items)-1].GetID())
	}

	return items, nextCursor, nil

}

// Count returns the count of models of type T with optional query modifications.
func (d *DAO[T]) Count(ctx context.Context, queries ...QueryFunc[T]) (count int64, err error) {
	query := d.db.
		Offset(0)
	for _, q := range queries {
		query = q(query)
	}
	return query.Count(ctx, "id")
}

// Get retrieves a model of type T by its ID.
func (d *DAO[T]) Get(ctx context.Context, id uint) (output T, err error) {
	return d.db.
		Where("id = ?", id).
		First(ctx)
}

// GetByName retrieves a model of type T by its name.
func (d *DAO[T]) GetByName(ctx context.Context, name string) (output T, err error) {
	return d.db.
		Where("name = ?", name).
		First(ctx)
}

// Create creates a new model of type T in the database.
func (d *DAO[T]) Create(ctx context.Context, param T) error {
	return d.db.
		Create(ctx, &param)
}

// Delete deletes a model of type T by its ID.
func (d *DAO[T]) Delete(ctx context.Context, id uint) (rowsAffected int, err error) {
	return d.db.
		Where("id = ?", id).
		Limit(1).
		Delete(ctx)
}

// Update updates a model of type T in the database.
func (d *DAO[T]) Update(ctx context.Context, param T) (rowsAffected int, err error) {
	return d.db.
		Limit(1).
		Updates(ctx, param)
}

// UpdateName updates the name of a model of type T by its ID.
func (d *DAO[T]) UpdateName(ctx context.Context, id uint, name string) (rowsAffected int, err error) {
	return d.db.
		Where("id = ?", id).
		Limit(1).
		Update(ctx, "name", name)
}

// ClearDeleted permanently removes soft-deleted models of type T from the database.
func (d *DAO[T]) ClearDeleted(ctx context.Context) (rowsAffected int, err error) {
	return d.db.
		ClearDeleted(ctx)
}
