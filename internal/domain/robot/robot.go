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

package robot

import (
	"context"
	"time"

	"gorm.io/datatypes"

	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
)

type ProjectScope string

const (
	RobotPrefix = "robot$"

	ProjectScopeSelected ProjectScope = "selected"
	ProjectScopeAll      ProjectScope = "all"
)

type Robot struct {
	ID                  int `gorm:"primary_key"`
	Name                string
	Description         string
	Enabled             bool
	ProjectId           int
	ExpireAt            *time.Time
	Duration            int
	TokenHash           string
	ProjectScope        ProjectScope
	Projects            []*project.Project          `gorm:"many2many:robots_projects"`
	PlatformPermissions datatypes.JSONSlice[string] `gorm:"type:json"`
	ProjectPermissions  datatypes.JSONSlice[string] `gorm:"type:json"`
	CreateBy            int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (Robot) TableName() string {
	return "robots"
}

type RobotPoject struct {
	ID        int `gorm:"primary_key"`
	RobotId   int
	ProjectId int
	CreatedAt time.Time
}

func (RobotPoject) TableName() string {
	return "robots_projects"
}

type IRobotRepo interface {
	CreateRobot(ctx context.Context, robot *Robot) error
	GetRobot(ctx context.Context, id int) (*Robot, error)
	GetRobotByName(ctx context.Context, name string) (*Robot, error)
	UpdateRobot(ctx context.Context, robot *Robot) error
	DeleteRobot(ctx context.Context, id int) error
	ListSystemRobots(ctx context.Context, page, pageSize int, search string) ([]*Robot, int64, error)
}
