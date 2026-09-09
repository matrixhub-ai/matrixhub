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

package authenticator

import (
	"context"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

type SSHKeyAuthenticator struct {
	sshKeyRepo user.ISSHKeyRepo
	userRepo   user.IUserRepo
}

func NewSSHKeyAuthenticator(sshKeyRepo user.ISSHKeyRepo, userRepo user.IUserRepo) *SSHKeyAuthenticator {
	return &SSHKeyAuthenticator{
		sshKeyRepo: sshKeyRepo,
		userRepo:   userRepo,
	}
}

func (s *SSHKeyAuthenticator) Authenticate(ctx context.Context, fingerprint string) (auth.Identity, bool, bool, error) {
	key, err := s.sshKeyRepo.GetByFingerprint(ctx, fingerprint)
	if err != nil {
		return nil, false, false, err
	}
	if key == nil || key.Id == 0 {
		return nil, false, false, nil
	}
	if key.IsExpired(time.Now()) {
		return nil, false, false, nil
	}
	u, err := s.userRepo.GetUser(ctx, key.UserId)
	if err != nil {
		return nil, false, false, err
	}

	return user.NewUserIdentity(key.UserId, u.Username), false, true, nil
}
