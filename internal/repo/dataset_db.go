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
	"fmt"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

type datasetDB struct {
	db *gorm.DB
}

// NewDatasetDB creates a new DatasetRepo instance
func NewDatasetDB(db *gorm.DB) dataset.IDatasetRepo {
	return &datasetDB{db: db}
}

// List retrieves datasets with filtering and pagination
func (d *datasetDB) List(ctx context.Context, filter *model.Filter) ([]*dataset.Dataset, int64, error) {
	// Set defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// Build base query with JOIN to projects
	query := d.db.WithContext(ctx).
		Table("datasets d").
		Select(`d.id, d.name, d.project_id, d.default_branch,
				d.num_rows, d.size, d.readme_content,
				d.is_popular,
				d.created_at, d.updated_at,
				p.name as project_name`).
		Joins("INNER JOIN projects p ON d.project_id = p.id")

	// Apply filters
	if filter.Project != "" {
		query = query.Where("p.name = ?", filter.Project)
	}

	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		query = query.Where("p.name LIKE ? OR d.name LIKE ?", pattern, pattern)
	}

	if len(filter.Label) > 0 {
		for _, label := range filter.Label {
			query = query.Where(`EXISTS (
				SELECT 1 FROM datasets_labels dl
				INNER JOIN labels l ON dl.label_id = l.id
				WHERE dl.dataset_id = d.id AND l.name = ?
			)`, label)
		}
	}

	// Apply popular filter
	if filter.Popular != nil && *filter.Popular {
		query = query.Where("d.is_popular = ?", 1)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count datasets: %w", err)
	}

	// Apply sorting
	orderBy := "d.updated_at DESC"
	if filter.Sort == "asc" || filter.Sort == "updated_at_asc" {
		orderBy = "d.updated_at ASC"
	}

	// Apply pagination
	offset := (filter.Page - 1) * filter.PageSize
	query = query.Order(orderBy).Limit(int(filter.PageSize)).Offset(int(offset))

	// Execute query - get datasets first
	var datasets []*dataset.Dataset
	if err := query.Find(&datasets).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list datasets: %w", err)
	}

	if len(datasets) == 0 {
		return datasets, total, nil
	}

	// Collect all dataset IDs for batch label query
	datasetIDs := make([]int64, len(datasets))
	for i, d := range datasets {
		datasetIDs[i] = d.ID
	}

	// Second query: fetch all labels in one batch
	type labelResult struct {
		DatasetID int64 `db:"dataset_id"`
		model.Label
	}
	var labelResults []labelResult
	err := d.db.WithContext(ctx).
		Table("datasets_labels dl").
		Select("dl.dataset_id, l.*").
		Joins("INNER JOIN labels l ON dl.label_id = l.id").
		Where("dl.dataset_id IN ?", datasetIDs).
		Find(&labelResults).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch labels: %w", err)
	}

	// Build label map for efficient lookup
	labelMap := make(map[int64][]model.Label)
	for _, lr := range labelResults {
		labelMap[lr.DatasetID] = append(labelMap[lr.DatasetID], lr.Label)
	}

	// Attach labels to datasets
	for _, d := range datasets {
		d.Labels = labelMap[d.ID]
	}

	return datasets, total, nil
}

// Create creates a new dataset
func (d *datasetDB) Create(ctx context.Context, ds *dataset.Dataset) (*dataset.Dataset, error) {
	if err := d.db.WithContext(ctx).Create(ds).Error; err != nil {
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}
	return ds, nil
}

// GetByProjectAndName retrieves a dataset by project and name
func (d *datasetDB) GetByProjectAndName(ctx context.Context, project, name string) (*dataset.Dataset, error) {
	var ds dataset.Dataset
	err := d.db.WithContext(ctx).
		Table("datasets d").
		Select(`d.id, d.name, d.project_id, d.default_branch,
				d.num_rows, d.size, d.readme_content,
				d.created_at, d.updated_at, p.name as project_name`).
		Joins("INNER JOIN projects p ON d.project_id = p.id").
		Where("p.name = ? AND d.name = ?", project, name).
		First(&ds).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	return &ds, nil
}

// Delete removes a dataset by project and name
func (d *datasetDB) Delete(ctx context.Context, project, name string) error {
	// First get the dataset to obtain its ID
	ds, err := d.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return err
	}

	// Delete by ID (datasets_labels will be cascade deleted)
	err = d.db.WithContext(ctx).
		Delete(&dataset.Dataset{}, ds.ID).Error

	if err != nil {
		return fmt.Errorf("failed to delete dataset: %w", err)
	}

	return nil
}
