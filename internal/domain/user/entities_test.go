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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
)

func TestAccessTokenValidity(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	past := now.Add(-time.Second)
	future := now.Add(time.Second)

	tests := []struct {
		name  string
		token user.AccessToken
		valid bool
	}{
		{name: "disabled", token: user.AccessToken{Enabled: false}, valid: false},
		{name: "enabled without expiration", token: user.AccessToken{Enabled: true}, valid: true},
		{name: "enabled before expiration", token: user.AccessToken{Enabled: true, ExpireAt: &future}, valid: true},
		{name: "enabled after expiration", token: user.AccessToken{Enabled: true, ExpireAt: &past}, valid: false},
		{name: "expiration instant remains valid", token: user.AccessToken{Enabled: true, ExpireAt: &now}, valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.valid, tt.token.IsValid(now))
		})
	}
}

func TestUserCheckPassword(t *testing.T) {
	hash, err := crypto.HashPassword("correct-password")
	require.NoError(t, err)

	u := user.User{Password: hash}
	require.True(t, u.CheckPassword("correct-password"))
	require.False(t, u.CheckPassword("wrong-password"))
}

func TestUserIdentity(t *testing.T) {
	identity := user.NewUserIdentity(42, "alice")

	require.Equal(t, 42, identity.GetID())
	require.Equal(t, "alice", identity.GetName())
	require.Equal(t, "user", identity.TypeName())
}
