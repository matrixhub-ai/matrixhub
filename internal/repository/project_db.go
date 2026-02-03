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
