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
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

type labelDB struct {
	db *gorm.DB
}

// NewLabelDB creates a new LabelRepo instance
func NewLabelDB(db *gorm.DB) model.ILabelRepo {
	return &labelDB{db: db}
}

// ListByCategoryAndScope retrieves labels by category and scope
func (l *labelDB) ListByCategoryAndScope(ctx context.Context, category, scope string) ([]*model.Label, error) {
	var labels []*model.Label
	err := l.db.WithContext(ctx).
		Where("category = ? AND scope = ?", category, scope).
		Find(&labels).Error
	return labels, err
}

// GetByModelID retrieves labels for a specific model
func (l *labelDB) GetByModelID(ctx context.Context, modelID int64) ([]*model.Label, error) {
	var labels []*model.Label
	err := l.db.WithContext(ctx).
		Table("labels l").
		Joins("INNER JOIN models_labels ml ON l.id = ml.label_id").
		Where("ml.model_id = ?", modelID).
		Find(&labels).Error
	return labels, err
}

// GetOrCreateByName finds or creates a label by name, category and scope.
func (l *labelDB) GetOrCreateByName(ctx context.Context, name, category, scope string) (*model.Label, error) {
	label := model.Label{
		Name:     name,
		Category: category,
		Scope:    scope,
	}

	result := l.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&label)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to create label: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		return &label, nil
	}

	var existing model.Label
	err := l.db.WithContext(ctx).
		Where("name = ? AND category = ? AND scope = ?", name, category, scope).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("label disappeared after conflict for name=%s category=%s scope=%s", name, category, scope)
		}
		return nil, fmt.Errorf("failed to load existing label: %w", err)
	}

	return &existing, nil
}

// UpdateModelLabels replaces all label associations for a model.
func (l *labelDB) UpdateModelLabels(ctx context.Context, modelID int64, labelIDs []int) error {
	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete old associations
		if err := tx.Exec(
			"DELETE FROM models_labels WHERE model_id = ?",
			modelID,
		).Error; err != nil {
			return fmt.Errorf("failed to delete old labels: %w", err)
		}

		// Insert new associations
		if len(labelIDs) > 0 {
			type modelLabel struct {
				ModelID int64 `gorm:"column:model_id"`
				LabelID int   `gorm:"column:label_id"`
			}
			rows := make([]modelLabel, len(labelIDs))
			for i, id := range labelIDs {
				rows[i] = modelLabel{ModelID: modelID, LabelID: id}
			}
			if err := tx.Table("models_labels").CreateInBatches(rows, 100).Error; err != nil {
				return fmt.Errorf("failed to insert new labels: %w", err)
			}
		}

		return nil
	})
}

// UpdateDatasetLabels replaces all label associations for a dataset.
func (l *labelDB) UpdateDatasetLabels(ctx context.Context, datasetID int64, labelIDs []int) error {
	return l.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete old associations
		if err := tx.Exec(
			"DELETE FROM datasets_labels WHERE dataset_id = ?",
			datasetID,
		).Error; err != nil {
			return fmt.Errorf("failed to delete old labels: %w", err)
		}

		// Insert new associations
		if len(labelIDs) > 0 {
			type datasetLabel struct {
				DatasetID int64 `gorm:"column:dataset_id"`
				LabelID   int   `gorm:"column:label_id"`
			}
			rows := make([]datasetLabel, len(labelIDs))
			for i, id := range labelIDs {
				rows[i] = datasetLabel{DatasetID: datasetID, LabelID: id}
			}
			if err := tx.Table("datasets_labels").CreateInBatches(rows, 100).Error; err != nil {
				return fmt.Errorf("failed to insert new labels: %w", err)
			}
		}

		return nil
	})
}
