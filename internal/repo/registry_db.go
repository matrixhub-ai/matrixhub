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

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"gorm.io/gorm"
)

type RegistryDBRepo struct {
	db *gorm.DB
}

func NewRegistryDBRepo(db *gorm.DB) *RegistryDBRepo {
	return &RegistryDBRepo{db}
}

func (r *RegistryDBRepo) ListRegistries(ctx context.Context, page, pageSize int, search string) (rs []*registry.Registry, total int64, err error) {
	query := r.db.WithContext(ctx).Model(&registry.Registry{}).Limit(pageSize).Offset((page - 1) * pageSize)
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if err = query.Count(&total).Error; err != nil {
		return
	}
	err = query.Find(&rs).Error
	return
}

func (r *RegistryDBRepo) GetRegistry(ctx context.Context, id int32) (*registry.Registry, error) {
	var registry registry.Registry
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&registry).Error
	if err != nil {
		return nil, err
	}
	return &registry, nil
}

func (r *RegistryDBRepo) CreateRegistry(ctx context.Context, reg registry.Registry) (*registry.Registry, error) {
	if err := r.db.WithContext(ctx).Create(&reg).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *RegistryDBRepo) UpdateRegistry(ctx context.Context, reg registry.Registry) error {
	return r.db.WithContext(ctx).Model(&registry.Registry{}).Where("id = ?", reg.ID).Updates(reg).Error
}

func (r *RegistryDBRepo) DeleteRegistry(ctx context.Context, id int32) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&registry.Registry{}).Error
}
