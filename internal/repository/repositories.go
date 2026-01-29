// Copyright 2026 MatrixHub

package repository

import (
	_ "github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/db"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
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
