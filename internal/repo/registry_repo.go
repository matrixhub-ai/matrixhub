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
	"crypto/tls"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
)

type RegistryRepo struct {
	db *gorm.DB
}

func NewRegistryRepo(db *gorm.DB) *RegistryRepo {
	return &RegistryRepo{db}
}

func (r *RegistryRepo) ListRegistries(ctx context.Context, page, pageSize int, search string) (rs []*registry.Registry, total int64, err error) {
	baseQuery := r.db.WithContext(ctx).Model(&registry.Registry{})
	if search != "" {
		baseQuery = baseQuery.Where("name LIKE ?", "%"+search+"%")
	}
	if err = baseQuery.Count(&total).Error; err != nil {
		return
	}
	pagedQuery := baseQuery.Order("id ASC").Limit(pageSize).Offset((page - 1) * pageSize)
	err = pagedQuery.Find(&rs).Error
	return
}

func (r *RegistryRepo) GetRegistry(ctx context.Context, id int) (*registry.Registry, error) {
	var registry registry.Registry
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&registry).Error
	if err != nil {
		return nil, err
	}
	return &registry, nil
}

func (r *RegistryRepo) CreateRegistry(ctx context.Context, reg registry.Registry) (*registry.Registry, error) {
	if err := r.db.WithContext(ctx).Create(&reg).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *RegistryRepo) UpdateRegistry(ctx context.Context, reg registry.Registry) error {
	return r.db.WithContext(ctx).Model(&registry.Registry{}).Where("id = ?", reg.ID).Updates(reg).Error
}

func (r *RegistryRepo) DeleteRegistry(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&registry.Registry{}).Error
}

func (r *RegistryRepo) PingRegistry(ctx context.Context, reg registry.Registry) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reg.URL, nil)
	if err != nil {
		return 0, "", err
	}

	if basic := registry.AsBasic(reg.GetCredential()); basic != nil {
		req.SetBasicAuth(basic.Username, basic.Password)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: reg.Insecure},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return resp.StatusCode, resp.Status, nil
}

func (r *RegistryRepo) GetRegistryByName(ctx context.Context, name string) (*registry.Registry, error) {
	var reg registry.Registry
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&reg).Error
	if err != nil {
		return nil, err
	}
	return &reg, nil
}
