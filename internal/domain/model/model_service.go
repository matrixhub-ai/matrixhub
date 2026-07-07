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

package model

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/repository"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// IModelService defines the service interface for model operations.
type IModelService interface {
	// Model CRUD operations
	CreateModel(ctx context.Context, project, name string) (*Model, error)
	GetModel(ctx context.Context, project, name string) (*Model, error)
	ListModels(ctx context.Context, filter *Filter) ([]*Model, int64, error)
	DeleteModel(ctx context.Context, project, name string) error

	// EnsureModel ensures that a model exists in the database for the
	// given namespace and name.
	EnsureModel(ctx context.Context, project, name string) (*Model, error)

	// Label operations
	ListModelTaskLabels(ctx context.Context) ([]*Label, error)
	ListModelFrameLabels(ctx context.Context) ([]*Label, error)

	// Git operations
	ListModelRevisions(ctx context.Context, project, name string) (*git.Revisions, error)
	ListModelCommits(ctx context.Context, project, name, revision string, page, pageSize int) ([]*git.Commit, int64, error)
	GetModelCommit(ctx context.Context, project, name, commitID string) (*git.Commit, error)
	CreateModelCommit(ctx context.Context, project, name, revision string, commit *git.Commit, ops []git.CommitOperation) (string, error)
	GetModelTree(ctx context.Context, project, name, revision, path string) ([]*git.TreeEntry, error)
	GetModelBlob(ctx context.Context, project, name, revision, path string) (*git.TreeEntry, error)

	// Settings
	UpdateModelSetting(ctx context.Context, project, name string, update *SettingUpdate) error

	// Metadata sync
	SyncMetadata(ctx context.Context, project, name string) error

	CheckOrSyncFromRemote(ctx context.Context, project, name string) error
}

// ModelService implements the model service operations.
type ModelService struct {
	modelRepo    IModelRepo
	labelRepo    ILabelRepo
	gitRepo      git.IGitRepo
	projectRepo  project.IProjectRepo
	registryRepo registry.IRegistryRepo
}

// NewModelService creates a new ModelService instance.
func NewModelService(modelRepo IModelRepo, labelRepo ILabelRepo, gitRepo git.IGitRepo, projectRepo project.IProjectRepo, registryRepo registry.IRegistryRepo) IModelService {
	return &ModelService{
		modelRepo:    modelRepo,
		labelRepo:    labelRepo,
		gitRepo:      gitRepo,
		projectRepo:  projectRepo,
		registryRepo: registryRepo,
	}
}

// CreateModel creates a new model in the system.
func (s *ModelService) CreateModel(ctx context.Context, project, name string) (*Model, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid name")
	}

	// Check if model already exists
	_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err == nil {
		return nil, errors.New("model already exists")
	}
	// If error message contains "not found", continue to create
	// Otherwise, return the error
	if !strings.Contains(err.Error(), "not found") {
		return nil, err
	}

	model := &Model{
		Name:        name,
		ProjectName: project,
	}

	if err := s.gitRepo.CreateRepository(ctx, "models", project, name); err != nil {
		return nil, err
	}

	return s.modelRepo.Create(ctx, model)
}

// GetModel retrieves a model by project and name.
func (s *ModelService) GetModel(ctx context.Context, project, name string) (*Model, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	mod, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return mod, nil
}

// ListModels returns a paginated list of models with optional filtering.
func (s *ModelService) ListModels(ctx context.Context, filter *Filter) ([]*Model, int64, error) {
	if filter == nil {
		filter = &Filter{
			Page:     1,
			PageSize: 20,
		}
	}

	return s.modelRepo.List(ctx, filter)
}

// DeleteModel deletes a model by project and name.
func (s *ModelService) DeleteModel(ctx context.Context, project, name string) error {
	if project == "" {
		return errors.New("invalid project")
	}
	if name == "" {
		return errors.New("invalid input")
	}

	// First delete the Git repository, then delete the model record in the database.
	if err := s.gitRepo.DeleteRepository(ctx, "models", project, name); err != nil {
		return err
	}

	return s.modelRepo.Delete(ctx, project, name)
}

// ListModelTaskLabels returns all task labels for models.
func (s *ModelService) ListModelTaskLabels(ctx context.Context) ([]*Label, error) {
	return s.labelRepo.ListByCategoryAndScope(ctx, "task", "model")
}

// ListModelFrameLabels returns all framework labels for models.
func (s *ModelService) ListModelFrameLabels(ctx context.Context) ([]*Label, error) {
	return s.labelRepo.ListByCategoryAndScope(ctx, "library", "model")
}

// ListModelRevisions returns all branches and tags for a model.
func (s *ModelService) ListModelRevisions(ctx context.Context, project, name string) (*git.Revisions, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if model exists in database first
	_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.ListRevisions(ctx, "models", project, name)
}

// ListModelCommits returns the commit history for a model.
func (s *ModelService) ListModelCommits(ctx context.Context, project, name, revision string, page, pageSize int) ([]*git.Commit, int64, error) {
	if project == "" {
		return nil, 0, errors.New("invalid project")
	}
	if name == "" {
		return nil, 0, errors.New("invalid input")
	}

	// Check if model exists in database first
	_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
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

	return s.gitRepo.ListCommits(ctx, "models", project, name, revision, page, pageSize)
}

// GetModelCommit returns a specific commit by ID.
func (s *ModelService) GetModelCommit(ctx context.Context, project, name, commitID string) (*git.Commit, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}
	if commitID == "" {
		return nil, errors.New("invalid input")
	}

	// Check if model exists in database first
	_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.GetCommit(ctx, "models", project, name, commitID)
}

// CreateModelCommit creates a new model commit.
func (s *ModelService) CreateModelCommit(ctx context.Context, project, name, revision string, commit *git.Commit, ops []git.CommitOperation) (string, error) {
	prj, err := s.projectRepo.GetProjectByName(ctx, project)
	if err != nil {
		return "", err
	}
	if prj.HasProxy() {
		return "", fmt.Errorf("it is not allowed to upload file to model %s/%s for proxy project  ", project, name)
	}

	// create model if not exist
	mod, _ := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if mod == nil {
		mod = &Model{
			Name:        name,
			ProjectID:   prj.ID,
			ProjectName: project,
		}
		if _, err = s.modelRepo.Create(ctx, mod); err != nil {
			return "", err
		}
	}

	commitHash, err := s.gitRepo.CreateCommit(ctx, "models", project, name, revision, commit, ops)
	if err != nil {
		return "", err
	}
	if err = s.SyncMetadata(ctx, project, name); err != nil {
		log.Errorf("failed to sync metadata for %s/%s: %v", project, name, err)
	}

	return commitHash, nil
}

// GetModelTree returns the file tree at a specific revision and path.
func (s *ModelService) GetModelTree(ctx context.Context, project, name, revision, path string) ([]*git.TreeEntry, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if model exists in database first
	_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.GetTree(ctx, "models", project, name, revision, path)
}

// GetModelBlob returns the content of a file at a specific revision.
func (s *ModelService) GetModelBlob(ctx context.Context, project, name, revision, path string) (*git.TreeEntry, error) {
	if project == "" {
		return nil, errors.New("invalid project")
	}
	if name == "" {
		return nil, errors.New("invalid input")
	}

	// Check if model exists in database first
	_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return nil, err
	}

	return s.gitRepo.GetBlob(ctx, "models", project, name, revision, path)
}

// UpdateModelSetting updates model settings such as the popular flag.
func (s *ModelService) UpdateModelSetting(ctx context.Context, project, name string, update *SettingUpdate) error {
	if project == "" {
		return errors.New("invalid project")
	}
	if name == "" {
		return errors.New("invalid name")
	}

	m, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return err
	}

	return s.modelRepo.UpdateSetting(ctx, m.ID, update)
}

// SyncMetadata synchronizes Git repository metadata to the database.
func (s *ModelService) SyncMetadata(ctx context.Context, project, name string) error {
	m, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err != nil {
		return fmt.Errorf("model not found: %w", err)
	}

	files, err := s.gitRepo.ExtractMetadata(ctx, "models", project, name)
	if err != nil {
		return fmt.Errorf("failed to read metadata files: %w", err)
	}

	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		return fmt.Errorf("failed to analyze metadata: %w", err)
	}

	update := &MetadataUpdate{ReadmeContent: &metadata.ReadmeContent}
	if metadata.Size > 0 {
		update.Size = &metadata.Size
	}
	if metadata.ParameterCount > 0 {
		update.ParameterCount = &metadata.ParameterCount
	}
	if err := s.modelRepo.UpdateMetadata(ctx, m.ID, update); err != nil {
		return fmt.Errorf("failed to update model metadata: %w", err)
	}

	return s.updateModelLabels(ctx, m.ID, metadata.Tags)
}

// updateModelLabels replaces all labels for a model with classified tags.
func (s *ModelService) updateModelLabels(ctx context.Context, modelID int64, tags []ClassifiedTag) error {
	var labelIDs []int

	for _, tag := range tags {
		label, err := s.labelRepo.GetOrCreateByName(ctx, tag.Name, tag.Category, "model")
		if err != nil {
			return fmt.Errorf("failed to get/create label %s (category=%s): %w", tag.Name, tag.Category, err)
		}
		labelIDs = append(labelIDs, label.ID)
	}

	return s.labelRepo.UpdateModelLabels(ctx, modelID, labelIDs)
}

func (s *ModelService) CheckOrSyncFromRemote(ctx context.Context, project, name string) error {
	prj, err := s.projectRepo.GetProjectByName(ctx, project)
	if err != nil {
		return err
	}
	if !prj.HasProxy() {
		_, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
		if err != nil {
			return repository.ErrRepositoryNotExists
		} else {
			return nil
		}
	}

	reg, err := s.registryRepo.GetRegistry(ctx, *prj.RegistryID)
	if err != nil {
		return fmt.Errorf("get registry(id=%d): %w", prj.RegistryID, err)
	}
	gr := &git.GitRepository{
		RemoteRegistryURL:  reg.URL,
		RemoteProjectName:  prj.Organization,
		RemoteResourceName: name,
		ProjectName:        project,
		ResourceName:       name,
		ResourceType:       "model",
	}
	mod, _ := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if mod == nil {
		mod = &Model{
			Name:        name,
			ProjectID:   prj.ID,
			ProjectName: project,
		}
		if _, err = s.modelRepo.Create(ctx, mod); err != nil {
			return err
		}
	} else if !mod.ShouldSync() {
		return nil
	}
	if err = s.gitRepo.PullFromRemote(context.Background(), gr); err != nil {
		return err
	}
	if err = s.SyncMetadata(context.Background(), mod.ProjectName, mod.Name); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

func (s *ModelService) EnsureModel(ctx context.Context, project, name string) (*Model, error) {
	model, err := s.modelRepo.GetByProjectAndName(ctx, project, name)
	if err == nil {
		return model, nil
	}

	return s.CreateModel(ctx, project, name)
}
