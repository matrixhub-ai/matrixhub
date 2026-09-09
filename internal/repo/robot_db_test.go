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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
)

func TestRobotRepoGetRobotByTokenHash(t *testing.T) {
	database, _ := newSQLiteRepositoryTestDatabase(t)
	ctx := context.Background()
	repo := NewRobotRepo(database)

	rb, err := repo.GetRobotByTokenHash(ctx, "unknown")
	require.NoError(t, err)
	require.Nil(t, rb)

	require.NoError(t, repo.CreateRobot(ctx, &robot.Robot{Name: "robot$ci", TokenHash: "known", ProjectScope: robot.ProjectScopeAll}))
	rb, err = repo.GetRobotByTokenHash(ctx, "known")
	require.NoError(t, err)
	require.Equal(t, "robot$ci", rb.Name)
}
