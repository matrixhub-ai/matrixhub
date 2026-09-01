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

package dataset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

// IDatasetService defines the service interface for dataset operations.
type IDatasetService interface {
	// Dataset CRUD operations
	CreateDataset(ctx context.Context, project, name string) (*Dataset, error)
	GetDataset(ctx context.Context, project, name string) (*Dataset, error)
	ListDatasets(ctx context.Context, filter *model.Filter) ([]*Dataset, int64, error)
	DeleteDataset(ctx context.Context, project, name string) error

	// Label operations
	ListDatasetTaskLabels(ctx context.Context) ([]*model.Label, error)

	// SyncMetadata synchronizes Git repository metadata to the database.
	SyncMetadata(ctx context.Context, project, name string) error

	// Git operations
	ListDatasetRevisions(ctx context.Context, project, name string) (*git.Revisions, error)
	ListDatasetCommits(ctx context.Context, project, name, revision string, page, pageSize int) ([]*git.Commit, int64, error)
	GetDatasetCommit(ctx context.Context, project, name, commitID string) (*git.Commit, error)
	GetDatasetTree(ctx context.Context, project, name, revision, path string) ([]*git.TreeEntry, error)
	GetDatasetBlob(ctx context.Context, project, name, revision, path string) (*git.TreeEntry, error)
}

// DatasetService implements the dataset service operations.
type DatasetService struct {
	datasetRepo IDatasetRepo
	labelRepo   model.ILabelRepo
	gitRepo     git.IGitRepo
}

// NewDatasetService creates a new DatasetService instance.
func NewDatasetService(datasetRepo IDatasetRepo, labelRepo model.ILabelRepo, gitRepo git.IGitRepo) IDatasetService {
	return &DatasetService{
		datasetRepo: datasetRepo,
		labelRepo:   labelRepo,
		gitRepo:     gitRepo,
	}
}

// CreateDataset creates a new dataset in the system.
func (s *DatasetService) CreateDataset(ctx context.Context, project, name string) (*Dataset, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if dataset already exists
	_, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err == nil {
		return nil, errors.New("dataset already exists")
	}
	// If error message contains "not found", continue to create
	// Otherwise, return the error
	if !strings.Contains(err.Error(), "not found") {
		return nil, err
	}

	dataset := &Dataset{
		Name:        name,
		ProjectName: project,
	}

	if err := s.gitRepo.CreateRepository(ctx, "datasets", project, name); err != nil {
		return nil, err
	}

	return s.datasetRepo.Create(ctx, dataset)
}

// GetDataset retrieves a dataset by project and name.
func (s *DatasetService) GetDataset(ctx context.Context, project, name string) (*Dataset, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	dataset, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

// ListDatasets returns a paginated list of datasets with optional filtering.
func (s *DatasetService) ListDatasets(ctx context.Context, filter *model.Filter) ([]*Dataset, int64, error) {
	if filter == nil {
		filter = &model.Filter{
			Page:     1,
			PageSize: 20,
		}
	}

	return s.datasetRepo.List(ctx, filter)
}

// DeleteDataset deletes a dataset by project and name.
func (s *DatasetService) DeleteDataset(ctx context.Context, project, name string) error {
	if project == "" {
		return errors.New("invalid project")
	}
	if name == "" {
		return errors.New("invalid input")
	}

	// First delete the Git repository, then delete the dataset record in the database.
	if err := s.gitRepo.DeleteRepository(ctx, "datasets", project, name); err != nil {
		return err
	}

	return s.datasetRepo.Delete(ctx, project, name)
}

// SyncMetadata synchronizes Git repository metadata to the database.
func (s *DatasetService) SyncMetadata(ctx context.Context, project, name string) error {
	d, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return fmt.Errorf("dataset not found: %w", err)
	}

	files, err := s.gitRepo.ExtractMetadata(ctx, "datasets", project, name)
	if err != nil {
		return fmt.Errorf("failed to read metadata files: %w", err)
	}

	metadata, err := model.AnalyzeRepoMetadata(files)
	if err != nil {
		return fmt.Errorf("failed to analyze metadata: %w", err)
	}

	update := &MetadataUpdate{
		ReadmeContent: &metadata.ReadmeContent,
		Size:          &metadata.Size,
	}
	if err := s.datasetRepo.UpdateMetadata(ctx, d.ID, update); err != nil {
		return fmt.Errorf("failed to update dataset metadata: %w", err)
	}

	return s.updateDatasetLabels(ctx, d.ID, metadata.Tags)
}

// updateDatasetLabels replaces all labels for a dataset with classified tags.
func (s *DatasetService) updateDatasetLabels(ctx context.Context, datasetID int64, tags []model.ClassifiedTag) error {
	var labelIDs []int

	for _, tag := range tags {
		label, err := s.labelRepo.GetOrCreateByName(ctx, tag.Name, tag.Category, "dataset")
		if err != nil {
			return fmt.Errorf("failed to get/create label %s (category=%s): %w", tag.Name, tag.Category, err)
		}
		labelIDs = append(labelIDs, label.ID)
	}

	return s.labelRepo.UpdateDatasetLabels(ctx, datasetID, labelIDs)
}

// ListDatasetTaskLabels returns all task labels for datasets.
func (s *DatasetService) ListDatasetTaskLabels(ctx context.Context) ([]*model.Label, error) {
	return s.labelRepo.ListByCategoryAndScope(ctx, "task", "dataset")
}

// ListDatasetRevisions returns all branches and tags for a dataset.
func (s *DatasetService) ListDatasetRevisions(ctx context.Context, project, name string) (*git.Revisions, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if dataset exists in database first
	_, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.ListRevisions(ctx, "datasets", project, name)
}

// ListDatasetCommits returns the commit history for a dataset.
func (s *DatasetService) ListDatasetCommits(ctx context.Context, project, name, revision string, page, pageSize int) ([]*git.Commit, int64, error) {
	if project == "" {
		return nil, 0, errors.New("invalid project")
	}
	if name == "" {
		return nil, 0, errors.New("invalid input")
	}

	// Check if dataset exists in database first
	_, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, 0, err
	}

	// Set default values
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	return s.gitRepo.ListCommits(ctx, "datasets", project, name, revision, page, pageSize)
}

// GetDatasetCommit returns a specific commit by ID.
func (s *DatasetService) GetDatasetCommit(ctx context.Context, project, name, commitID string) (*git.Commit, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}
	if commitID == "" {
		return nil, errors.New("invalid input")
	}

	// Check if dataset exists in database first
	_, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.GetCommit(ctx, "datasets", project, name, commitID)
}

// GetDatasetTree returns the file tree at a specific revision and path.
func (s *DatasetService) GetDatasetTree(ctx context.Context, project, name, revision, path string) ([]*git.TreeEntry, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if dataset exists in database first
	_, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.GetTree(ctx, "datasets", project, name, revision, path)
}

// GetDatasetBlob returns the content of a file at a specific revision.
func (s *DatasetService) GetDatasetBlob(ctx context.Context, project, name, revision, path string) (*git.TreeEntry, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if dataset exists in database first
	_, err := s.datasetRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.GetBlob(ctx, "datasets", project, name, revision, path)
}
