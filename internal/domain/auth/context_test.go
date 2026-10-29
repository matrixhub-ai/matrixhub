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

package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/auth"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
)

func TestIdentityContext(t *testing.T) {
	identity, ok := auth.IdentityFromContext(context.Background())
	require.False(t, ok)
	require.Nil(t, identity)

	want := user.NewUserIdentity(42, "alice")
	ctx := auth.WithIdentity(context.Background(), want)
	identity, ok = auth.IdentityFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, want, identity)
}
