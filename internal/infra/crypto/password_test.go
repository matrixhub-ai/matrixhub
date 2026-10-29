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

package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	infra "github.com/matrixhub-ai/matrixhub/internal/infra/crypto"
)

func TestPasswordHash(t *testing.T) {
	hash, err := infra.HashPassword("correct horse battery staple")
	require.NoError(t, err)
	require.NotEqual(t, "correct horse battery staple", hash)
	require.True(t, infra.CheckPasswordHash("correct horse battery staple", hash))
	require.False(t, infra.CheckPasswordHash("wrong password", hash))
	require.False(t, infra.CheckPasswordHash("password", "not-a-bcrypt-hash"))
}
