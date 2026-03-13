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
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

// Dataset represents a dataset in the system.
type Dataset struct {
	ID            int64         `json:"id" db:"id"`
	Name          string        `json:"name" db:"name"`
	ProjectID     int           `json:"projectId" db:"project_id"`
	ProjectName   string        `json:"projectName" db:"project_name"`
	DefaultBranch string        `json:"defaultBranch" db:"default_branch"`
	NumRows       string        `json:"numRows" db:"num_rows"`
	License       string        `json:"license" db:"license"`
	Size          int64         `json:"size" db:"size"`
	ReadmeContent string        `json:"readmeContent" db:"readme_content"`
	IsPopular     bool          `json:"isPopular" db:"is_popular"`
	Labels        []model.Label `json:"labels" db:"-" gorm:"-"`
	CreatedAt     time.Time     `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time     `json:"updatedAt" db:"updated_at"`
}

// IDatasetRepo defines the repository interface for dataset database operations.
type IDatasetRepo interface {
	// Create creates a new dataset in the database.
	Create(ctx context.Context, d *Dataset) (*Dataset, error)

	// GetByProjectAndName retrieves a dataset by its project and name.
	GetByProjectAndName(ctx context.Context, project, name string) (*Dataset, error)

	// List retrieves a list of datasets based on the provided filter, along with the total count.
	List(ctx context.Context, filter *model.Filter) ([]*Dataset, int64, error)

	// Delete removes a dataset from the database by its project and name.
	Delete(ctx context.Context, project, name string) error
}
