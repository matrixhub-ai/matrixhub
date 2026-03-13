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
	"time"
)

// Model represents an AI model in the system.
type Model struct {
	ID             int64     `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	ProjectID      int       `json:"projectId" db:"project_id"`
	ProjectName    string    `json:"projectName" db:"project_name" gorm:"<-:false"` // Read-only, not writable
	Size           int64     `json:"size" db:"size"`
	DefaultBranch  string    `json:"defaultBranch" db:"default_branch"`
	ParameterCount int64     `json:"parameterCount" db:"parameter_count"`
	ReadmeContent  string    `json:"readmeContent" db:"readme_content"`
	IsPopular      bool      `json:"isPopular" db:"is_popular"`
	Labels         []Label   `json:"labels" db:"-" gorm:"-"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

// Label represents a category label for models/datasets.
type Label struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Category  string    `json:"category" db:"category"`
	Scope     string    `json:"scope" db:"scope"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// Filter defines query parameters for listing models.
type Filter struct {
	Project  string   // filter by project name
	Label    []string // filter by labels
	Search   string   // project name or model name, prioritize project name matching (supports fuzzy search).
	Sort     string
	Page     int32
	PageSize int32
	Popular  *bool // filter by popular flag (true = only popular, false/nil = all)
}

// IModelRepo defines the repository interface for model operations.
type IModelRepo interface {
	// Create creates a new model in the database.
	Create(ctx context.Context, m *Model) (*Model, error)

	// GetByProjectAndName retrieves a model by its project and name.
	GetByProjectAndName(ctx context.Context, project, name string) (*Model, error)

	// List retrieves a list of models based on the provided filter, along with the total count.
	List(ctx context.Context, filter *Filter) ([]*Model, int64, error)

	// Delete removes a model from the database by its project and name.
	Delete(ctx context.Context, project, name string) error
}

// ILabelRepo defines the repository interface for label operations.
type ILabelRepo interface {
	ListByCategoryAndScope(ctx context.Context, category, scope string) ([]*Label, error)
	GetByModelID(ctx context.Context, modelID int64) ([]*Label, error)
}
