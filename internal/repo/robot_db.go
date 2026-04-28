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

package repo

import (
	"context"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type robotRepo struct {
	db *gorm.DB
}

func (r *robotRepo) GetRobotByName(ctx context.Context, name string) (*robot.Robot, error) {
	var rb robot.Robot
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&rb).Error
	if err != nil {
		return nil, err
	}
	return &rb, nil
}

func (r *robotRepo) UpdateRobot(ctx context.Context, robot *robot.Robot) error {
	return r.db.WithContext(ctx).Where("id = ?", robot.ID).Updates(robot).Error
}

func (r *robotRepo) CreateRobot(ctx context.Context, robot *robot.Robot) error {
	return r.db.WithContext(ctx).Omit("Projects.*").Create(robot).Error
}

func (r *robotRepo) GetRobot(ctx context.Context, id int) (*robot.Robot, error) {
	var rb robot.Robot
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rb).Error
	if err != nil {
		return nil, err
	}
	return &rb, nil
}

func (r *robotRepo) DeleteRobot(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(robot.Robot{}).Error
}

func (r *robotRepo) ListSystemRobots(ctx context.Context, page, pageSize int, search string) (rs []*robot.Robot, total int64, err error) {
	query := r.db.WithContext(ctx).Model(&robot.Robot{})
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if err = query.Count(&total).Error; err != nil {
		return
	}
	query = query.Order("name ASC")
	if !utils.IsFullPageData(page, pageSize) {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}
	err = query.Find(&rs).Error
	return
}

func (r *robotRepo) GetRobotByTokenHash(ctx context.Context, tokenHash string) (*robot.Robot, error) {
	var rb robot.Robot
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&rb).Error
	if err != nil {
		return nil, err
	}
	return &rb, nil
}

func NewRobotRepo(db *gorm.DB) robot.IRobotRepo {
	return &robotRepo{db: db}
}
