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
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

type ProjectType int

const (
	ProjectTypePrivate ProjectType = iota
	ProjectTypePublic
	ProjectTypeUnspecified
)

type Project struct {
	ID           int         `gorm:"primary_key"`
	Name         string      `gorm:"column:name"`
	Type         ProjectType `gorm:"column:type"`
	RegistryID   *int        `gorm:"column:registry_id"`
	Organization string      `gorm:"column:organization"`
	RegistryURL  string      `gorm:"column:registry_url;<-:false"`
	ModelCount   int         `gorm:"column:model_count;<-:false"`
	DatasetCount int         `gorm:"column:dataset_count;<-:false"`
	CreatedAt    time.Time   `gorm:"column:created_at"`
	UpdatedAt    time.Time   `gorm:"column:updated_at"`
}

func (*Project) TableName() string {
	return "projects"
}

func (p *Project) HasProxy() bool {
	return p.RegistryID != nil
}

func (p *Project) IsPublic() bool {
	return p.Type == ProjectTypePublic
}

type MemberType string

const (
	MemberTypeUser  MemberType = "user"
	MemberTypeGroup MemberType = "group"
)

type ProjectMember struct {
	ID         int           `gorm:"primary_key"`
	ProjectID  *int          `gorm:"column:project_id;index"`
	MemberID   int           `gorm:"column:member_id"`
	MemberType MemberType    `gorm:"column:member_type"`
	MemberName string        `gorm:"column:member_name;<-:false"`
	RoleID     role.RoleType `gorm:"column:role_id"`
	CreatedAt  time.Time     `gorm:"column:created_at"`
	UpdatedAt  time.Time     `gorm:"column:updated_at"`
}

func (ProjectMember) TableName() string {
	return "members_roles_projects"
}

type Member struct {
	MemberID   int
	MemberType MemberType
}

// IProjectRepo defines the project repository interface
type IProjectRepo interface {
	CreateProject(ctx context.Context, project *Project) (*Project, error)
	GetProjectByID(ctx context.Context, id int) (*Project, error)
	GetProjectByName(ctx context.Context, name string) (*Project, error)
	GetProjectIDByName(ctx context.Context, name string) (int, error)
	ListProjects(ctx context.Context, name string, projectType ProjectType, managedOnly bool, page, pageSize int) ([]*Project, int64, error)
	UpdateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, id int) error

	ListProjectMembers(ctx context.Context, projectID int, memberName string, page, pageSize int) ([]*ProjectMember, int64, error)
	AddProjectMemberWithRole(ctx context.Context, projectMember *ProjectMember) error
	RemoveProjectMembers(ctx context.Context, projectID int, members []*Member) error
	UpdateProjectMemberRole(ctx context.Context, projectID int, member Member, role role.RoleType) error
	GetUserProjectRole(ctx context.Context, userID int, projectID int) (int, error)
}
