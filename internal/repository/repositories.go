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
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/db"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"

	_ "github.com/lib/pq"
)

type Repositories struct {
	DB      *gorm.DB
	Project project.IProjectRepository
}

func NewRepositories(conf *config.Config) *Repositories {
	log.Debug("init database")
	database, err := db.New(conf.Database)
	if err != nil {
		log.Fatalw("create database failed", "error", err)
	}

	repositories := &Repositories{
		DB: database,
	}

	repositories.Project = NewProjectDBRepository(repositories.DB)

	return repositories
}

func (r *Repositories) Close() error {
	dbconn, err := r.DB.DB()
	if err != nil {
		return err
	}
	if err := dbconn.Close(); err != nil {
		return err
	}

	return nil
}
