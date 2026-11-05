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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	updates := map[string]any{
		"name":            reg.Name,
		"description":     reg.Description,
		"url":             reg.URL,
		"insecure":        reg.Insecure,
		"credential_type": reg.CredentialType,
		"auth_info":       reg.AuthInfo,
	}
	return r.db.WithContext(ctx).Model(&registry.Registry{}).Where("id = ?", reg.ID).Updates(updates).Error
}

func (r *RegistryRepo) DeleteRegistry(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var projectCount int64
		if err := tx.Table("projects").Where("registry_id = ?", id).Count(&projectCount).Error; err != nil {
			return err
		}

		var syncPolicyCount int64
		if err := tx.Table("sync_policies").Where("registry_id = ?", id).Count(&syncPolicyCount).Error; err != nil {
			return err
		}

		if projectCount > 0 || syncPolicyCount > 0 {
			return registry.ErrInUse
		}

		err := tx.Where("id = ?", id).Delete(&registry.Registry{}).Error
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			return registry.ErrInUse
		}
		return err
	})
}

func (r *RegistryRepo) PingRegistry(ctx context.Context, reg registry.Registry) (int, string, error) {
	method, endpoint := http.MethodGet, reg.URL
	body := bytes.NewReader(nil)
	basic := registry.AsBasic(reg.GetCredential())
	if reg.Type == registry.TypeModelScope && basic != nil && basic.Password != "" {
		method, endpoint = http.MethodPost, strings.TrimRight(reg.URL, "/")+"/api/v1/login"
		data, err := json.Marshal(map[string]string{"AccessToken": basic.Password})
		if err != nil {
			return 0, "", err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return 0, "", err
	}

	if basic != nil {
		if reg.Type == registry.TypeModelScope {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+basic.Password)
			req.AddCookie(&http.Cookie{Name: "m_session_id", Value: basic.Password})
		} else {
			req.SetBasicAuth(basic.Username, basic.Password)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: reg.Insecure},
		},
	}
	// Do not replay an SDK token in a login body to a redirect target.
	if reg.Type == registry.TypeModelScope && method == http.MethodPost {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if reg.Type == registry.TypeModelScope && method == http.MethodPost && resp.StatusCode == http.StatusOK {
		var result struct{ Code int }
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return 0, "", fmt.Errorf("invalid modelscope login response")
		}
		if result.Code != http.StatusOK {
			return http.StatusUnauthorized, "ModelScope rejected the access token", nil
		}
	}
	return resp.StatusCode, resp.Status, nil
}
