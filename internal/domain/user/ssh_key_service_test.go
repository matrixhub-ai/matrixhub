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

package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	usermocks "github.com/matrixhub-ai/matrixhub/internal/domain/user/mocks"
)

func TestSSHKeyServiceCreateSSHKey(t *testing.T) {
	ctx := context.Background()
	key := user.SSHKey{Name: "laptop", Fingerprint: "SHA256:test"}

	t.Run("creates a new key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		repo.EXPECT().GetByFingerprint(ctx, key.Fingerprint).Return(&user.SSHKey{}, nil)
		repo.EXPECT().CreateSSHKey(ctx, key).Return(nil)

		require.NoError(t, user.NewSSHKeyService(repo).CreateSSHKey(ctx, key))
	})

	t.Run("rejects an active duplicate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		repo.EXPECT().GetByFingerprint(ctx, key.Fingerprint).Return(&user.SSHKey{Id: 1}, nil)

		err := user.NewSSHKeyService(repo).CreateSSHKey(ctx, key)
		require.ErrorIs(t, err, user.ErrSSHKeyAlreadyExists)
	})

	t.Run("rejects an expired duplicate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		expired := time.Now().Add(-time.Minute)
		repo.EXPECT().GetByFingerprint(ctx, key.Fingerprint).Return(&user.SSHKey{Id: 1, ExpireAt: &expired}, nil)

		err := user.NewSSHKeyService(repo).CreateSSHKey(ctx, key)
		require.ErrorIs(t, err, user.ErrSSHKeyExpired)
	})

	t.Run("propagates lookup error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		wantErr := errors.New("lookup failed")
		repo.EXPECT().GetByFingerprint(ctx, key.Fingerprint).Return(nil, wantErr)

		err := user.NewSSHKeyService(repo).CreateSSHKey(ctx, key)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("propagates create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := usermocks.NewMockISSHKeyRepo(ctrl)
		wantErr := errors.New("create failed")
		repo.EXPECT().GetByFingerprint(ctx, key.Fingerprint).Return(nil, nil)
		repo.EXPECT().CreateSSHKey(ctx, key).Return(wantErr)

		err := user.NewSSHKeyService(repo).CreateSSHKey(ctx, key)
		require.ErrorIs(t, err, wantErr)
	})
}
