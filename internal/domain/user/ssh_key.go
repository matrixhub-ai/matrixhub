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

package user

import (
	"context"
	"time"
)

type SSHKey struct {
	Id          int `gorm:"primary_key"`
	Name        string
	UserId      int
	PublicKey   string
	Fingerprint string
	ExpireAt    *time.Time
	CreatedAt   time.Time
}

func (s SSHKey) IsExpired(t time.Time) bool {
	return s.ExpireAt != nil && s.ExpireAt.Before(t)
}

func (SSHKey) TableName() string {
	return "ssh_keys"
}

type ISSHKeyRepo interface {
	GetByFingerprint(ctx context.Context, fingerprint string) (*SSHKey, error)
	ListSSHKeys(ctx context.Context, userId int) ([]*SSHKey, error)
	CreateSSHKey(ctx context.Context, key SSHKey) error
	DeleteSSHKey(ctx context.Context, userId, id int) error
}
