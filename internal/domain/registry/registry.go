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

package registry

import (
	"context"
	"time"
)

type Registry struct {
	ID             int `gorm:"primarykey"`
	Name           string
	Description    string
	Type           string
	URL            string
	CredentialType string
	AuthInfo       string
	Insecure       bool
	Status         int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Registry) TableName() string {
	return "registries"
}

// GetCredential returns the registry credential built from persistence fields; nil when none or invalid.
func (r *Registry) GetCredential() ICredential {
	return ParseCredentialInfo(r.CredentialType, r.AuthInfo)
}

// SetCredential sets or clears the credential; pass nil to clear.
func (r *Registry) SetCredential(c ICredential) {
	if c == nil {
		r.CredentialType = ""
		r.AuthInfo = ""
		return
	}
	r.CredentialType = c.Type()
	r.AuthInfo = c.String()
}

type IRegistryRepo interface {
	ListRegistries(ctx context.Context, page, pageSize int, search string) ([]*Registry, int64, error)
	GetRegistry(ctx context.Context, id int32) (*Registry, error)
	CreateRegistry(ctx context.Context, registry Registry) (*Registry, error)
	UpdateRegistry(ctx context.Context, registry Registry) error
	DeleteRegistry(ctx context.Context, id int32) error
}
