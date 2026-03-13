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

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

type modelDB struct {
	db *gorm.DB
}

// NewModelDB creates a new ModelRepo instance
func NewModelDB(db *gorm.DB) model.IModelRepo {
	return &modelDB{db: db}
}

// List retrieves models with filtering and pagination
func (m *modelDB) List(ctx context.Context, filter *model.Filter) ([]*model.Model, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// Build base query with JOIN to projects
	query := m.db.WithContext(ctx).
		Table("models m").
		Select(`m.id, m.name, m.project_id, m.size, m.parameter_count,
				m.readme_content, m.is_popular, m.default_branch,
				m.created_at, m.updated_at,
				p.name as project_name`).
		Joins("INNER JOIN projects p ON m.project_id = p.id")

	// Apply filters
	if filter.Project != "" {
		query = query.Where("p.name = ?", filter.Project)
	}

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		query = query.Where("p.name LIKE ? OR m.name LIKE ?", pattern, pattern)
	}

	if len(filter.Label) > 0 {
		for _, label := range filter.Label {
			query = query.Where(`EXISTS (
				SELECT 1 FROM models_labels ml
				INNER JOIN labels l ON ml.label_id = l.id
				WHERE ml.model_id = m.id AND l.name = ?
			)`, label)
		}
	}

	// Apply popular filter
	if filter.Popular != nil && *filter.Popular {
		query = query.Where("m.is_popular = ?", 1)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count models: %w", err)
	}

	// Apply sorting
	orderBy := "m.updated_at DESC"
	if filter.Sort == "asc" || filter.Sort == "updated_at_asc" {
		orderBy = "m.updated_at ASC"
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Order(orderBy).Limit(int(filter.PageSize)).Offset(int(offset))

	// Execute query - get models first
	var models []*model.Model
	if err := query.Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return models, total, nil
	}

	// Collect all model IDs for batch label query
	modelIDs := make([]int64, len(models))
	for i, m := range models {
		modelIDs[i] = m.ID
	}

	// Second query: fetch all labels in one batch
	type labelResult struct {
		ModelID int64 `db:"model_id"`
		model.Label
	}
	var labelResults []labelResult
	err := m.db.WithContext(ctx).
		Table("models_labels ml").
		Select("ml.model_id, l.*").
		Joins("INNER JOIN labels l ON ml.label_id = l.id").
		Where("ml.model_id IN ?", modelIDs).
		Find(&labelResults).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch labels: %w", err)
	}

	// Build label map for efficient lookup
	labelMap := make(map[int64][]model.Label)
	for _, lr := range labelResults {
		labelMap[lr.ModelID] = append(labelMap[lr.ModelID], lr.Label)
	}

	// Attach labels to models
	for _, m := range models {
		m.Labels = labelMap[m.ID]
	}

	return models, total, nil
}

// Create creates a new model (placeholder - not implemented for this task)
func (m *modelDB) Create(ctx context.Context, mod *model.Model) (*model.Model, error) {
	// Get project ID from project name
	if mod.ProjectName == "" {
		return nil, fmt.Errorf("project name is required")
	}

	var result struct {
		ID int `db:"id"`
	}
	err := m.db.WithContext(ctx).
		Table("projects").
		Select("id").
		Where("name = ?", mod.ProjectName).
		First(&result).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project not found: %s", mod.ProjectName)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Set project ID and create model
	mod.ProjectID = result.ID
	mod.DefaultBranch = "main" // Set default branch

	if err := m.db.WithContext(ctx).Create(mod).Error; err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}
	return mod, nil
}

// GetByProjectAndName retrieves a model by project and name
func (m *modelDB) GetByProjectAndName(ctx context.Context, project, name string) (*model.Model, error) {
	var mod model.Model
	err := m.db.WithContext(ctx).
		Table("models m").
		Select(`m.id, m.name, m.project_id, m.size, m.parameter_count,
				m.readme_content, m.is_popular, m.default_branch,
				m.created_at, m.updated_at, p.name as project_name`).
		Joins("INNER JOIN projects p ON m.project_id = p.id").
		Where("p.name = ? AND m.name = ?", project, name).
		First(&mod).Error

	if err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("failed to get model: %w", err)
	}
	return &mod, nil
}

// Delete removes a model by project and name (cascade delete models_labels)
func (m *modelDB) Delete(ctx context.Context, project, name string) error {
	// First get the model to obtain its ID
	mod, err := m.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return err
	}

	// Delete by ID (models_labels will be cascade deleted)
	err = m.db.WithContext(ctx).
		Delete(&model.Model{}, mod.ID).Error

	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}
