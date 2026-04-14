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

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

type sshKeyRepo struct {
	db *gorm.DB
}

func (s *sshKeyRepo) GetByFingerprint(ctx context.Context, fingerprint string) (*user.SSHKey, error) {
	var sk user.SSHKey
	err := s.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).Find(&sk).Error
	return &sk, err
}

func (s *sshKeyRepo) ListSSHKeys(ctx context.Context, userId int) (sks []*user.SSHKey, err error) {
	err = s.db.WithContext(ctx).Where("user_id = ?", userId).Find(&sks).Error
	return
}

func (s *sshKeyRepo) CreateSSHKey(ctx context.Context, key user.SSHKey) error {
	return s.db.WithContext(ctx).Create(&key).Error
}

func (s *sshKeyRepo) DeleteSSHKey(ctx context.Context, userId, id int) error {
	return s.db.WithContext(ctx).Where("id = ? and user_id = ?", id, userId).Delete(&user.SSHKey{}).Error
}

func NewSSHKeyRepo(db *gorm.DB) user.ISSHKeyRepo {
	return &sshKeyRepo{
		db: db,
	}
}
