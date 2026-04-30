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

package repo

import (
	"context"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/cleanup"
)

type cleanupDB struct {
	db *gorm.DB
}

// NewCleanupDB creates a new CleanupRepo instance.
func NewCleanupDB(db *gorm.DB) cleanup.ICleanupRepo {
	return &cleanupDB{db: db}
}

// ListAllModelPaths returns all valid model paths (project/name format) from database.
// Models without a valid project reference are excluded (they are not orphaned repos,
// but rather data integrity issues that should be handled separately).
func (c *cleanupDB) ListAllModelPaths(ctx context.Context) ([]string, error) {
	var paths []string
	// Use LEFT JOIN and only return paths where project exists
	// MySQL uses CONCAT() for string concatenation
	if err := c.db.WithContext(ctx).
		Table("models m").
		Select("CONCAT(p.name, '/', m.name)").
		Joins("LEFT JOIN projects p ON m.project_id = p.id").
		Where("p.name IS NOT NULL").
		Find(&paths).Error; err != nil {
		return nil, err
	}
	return paths, nil
}

// ListAllDatasetPaths returns all valid dataset paths from database.
// Datasets without a valid project reference are excluded (they are not orphaned repos,
// but rather data integrity issues that should be handled separately).
func (c *cleanupDB) ListAllDatasetPaths(ctx context.Context) ([]string, error) {
	var paths []string
	// Use LEFT JOIN and only return paths where project exists
	// MySQL uses CONCAT() for string concatenation
	if err := c.db.WithContext(ctx).
		Table("datasets d").
		Select("CONCAT(p.name, '/', d.name)").
		Joins("LEFT JOIN projects p ON d.project_id = p.id").
		Where("p.name IS NOT NULL").
		Find(&paths).Error; err != nil {
		return nil, err
	}
	return paths, nil
}
