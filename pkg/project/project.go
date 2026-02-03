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

package project

import (
	"context"
	"fmt"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/pkg/db"
	cdao "github.com/matrixhub-ai/matrixhub/pkg/db/dao"
	"github.com/matrixhub-ai/matrixhub/pkg/project/dao"
)

func NewProject(rawDB *db.RawDB) *Project {
	db := db.G[dao.Project](rawDB)

	return &Project{
		rawDB: rawDB,
		db:    db,
		dao:   dao.NewDAO(db),
	}
}

type Project struct {
	rawDB *db.RawDB
	db    *db.DB[dao.Project]
	dao   *dao.DAO
}

// MigrateTable migrates the project table
func (p *Project) MigrateTable(ctx context.Context) error {
	return p.dao.MigrateTable(ctx)
}

// Create a new project
func (p *Project) Create(ctx context.Context, input *v1alpha1.CreateProjectRequest) error {
	return db.Transaction(ctx, p.rawDB, func(ctx context.Context, txDB *db.DB[dao.Project]) error {
		d := dao.NewDAO(txDB)
		_, err := d.GetByName(ctx, input.Name)
		if err != nil {
			if !db.IsNotFoundError(err) {
				return err
			}
		} else {
			return fmt.Errorf("project with name %s already exists", input.Name)
		}

		err = d.Create(ctx, dao.Project{
			Name:        input.Name,
			DisplayName: input.DisplayName,
			Description: input.Description,
		})
		if err != nil {
			return err
		}
		return nil
	})
}

// Get project by name
func (p *Project) Get(ctx context.Context, input *v1alpha1.GetProjectRequest) (*v1alpha1.ProjectItem, error) {
	project, err := p.dao.GetByName(ctx, input.Name)
	if err != nil {
		return nil, err
	}

	return daoToModel(project), nil
}

// List projects
func (p *Project) List(ctx context.Context, input *v1alpha1.ListProjectsRequest) ([]*v1alpha1.ProjectItem, string, error) {
	query, err := cdao.BuildOrderByQueries[dao.Project](input.OrderBy, "name", "created_at")
	if err != nil {
		return nil, "", err
	}
	projects, nextCursor, err := p.dao.List(ctx, input.Cursor, int(input.Limit), query...)
	if err != nil {
		return nil, "", err
	}

	var items []*v1alpha1.ProjectItem
	for _, project := range projects {
		items = append(items, daoToModel(project))
	}

	return items, nextCursor, nil
}

func (p *Project) Count(ctx context.Context, input *v1alpha1.ListProjectsRequest) (int64, error) {
	query, err := cdao.BuildOrderByQueries[dao.Project](input.OrderBy, "name", "created_at")
	if err != nil {
		return 0, err
	}
	return p.dao.Count(ctx, query...)
}
