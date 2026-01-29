package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
)

type ProjectDBRepository struct {
	db *gorm.DB
}

func NewProjectDBRepository(db *gorm.DB) *ProjectDBRepository {
	return &ProjectDBRepository{db}
}

func (r *ProjectDBRepository) GetProject(ctx context.Context, param *project.Project) (*project.Project, error) {
	dbWithCtx := r.db.WithContext(ctx)
	output := &project.Project{}
	err := dbWithCtx.Where(param).First(output).Error
	return output, err
}

func (r *ProjectDBRepository) CreateProject(ctx context.Context, param *project.Project) error {
	dbWithCtx := r.db.WithContext(ctx)
	return dbWithCtx.Create(param).Error
}
