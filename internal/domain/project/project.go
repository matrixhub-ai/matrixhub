package project

import (
	"context"
)

type Project struct {
	Name string `json:"name"`
}

type IProjectRepository interface {
	GetProject(ctx context.Context, param *Project) (*Project, error)
	CreateProject(ctx context.Context, param *Project) error
}
