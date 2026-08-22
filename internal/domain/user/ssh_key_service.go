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

type ISSHKeyService interface {
	CreateSSHKey(ctx context.Context, key SSHKey) error
}

type SSHKeyService struct {
	repo ISSHKeyRepo
}

func (s *SSHKeyService) CreateSSHKey(ctx context.Context, key SSHKey) error {
	existing, err := s.repo.GetByFingerprint(ctx, key.Fingerprint)
	if err != nil {
		return err
	}
	if existing != nil && existing.Id != 0 {
		if existing.IsExpired(time.Now()) {
			return ErrSSHKeyExpired
		}
		return ErrSSHKeyAlreadyExists
	}
	return s.repo.CreateSSHKey(ctx, key)
}

func NewSSHKeyService(repo ISSHKeyRepo) ISSHKeyService {
	return &SSHKeyService{
		repo: repo,
	}
}
