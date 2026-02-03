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

package dao

import (
	"github.com/matrixhub-ai/matrixhub/pkg/db"
	"github.com/matrixhub-ai/matrixhub/pkg/db/dao"
)

// Project represents a project entity in the database.
type Project struct {
	db.Model
	Name        string
	DisplayName string
	Description string
}

func (p Project) GetID() uint {
	return p.ID
}

// DAO is the data access object for Project.
type DAO struct {
	db *db.DB[Project]
	*dao.DAO[Project]
}

func NewDAO(db *db.DB[Project]) *DAO {
	return &DAO{
		db:  db,
		DAO: dao.NewDAO(db),
	}
}
