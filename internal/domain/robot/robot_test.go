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

package robot_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
)

func TestRobotValidity(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	past := now.Add(-time.Second)
	future := now.Add(time.Second)

	tests := []struct {
		name  string
		robot robot.Robot
		valid bool
	}{
		{name: "disabled", robot: robot.Robot{Enabled: false}, valid: false},
		{name: "enabled without expiration", robot: robot.Robot{Enabled: true}, valid: true},
		{name: "enabled before expiration", robot: robot.Robot{Enabled: true, ExpireAt: &future}, valid: true},
		{name: "enabled after expiration", robot: robot.Robot{Enabled: true, ExpireAt: &past}, valid: false},
		{name: "expiration instant remains valid", robot: robot.Robot{Enabled: true, ExpireAt: &now}, valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.valid, tt.robot.IsValid(now))
		})
	}
}

func TestRobotTokenHashAndIdentity(t *testing.T) {
	rb := robot.Robot{TokenHash: "expected"}
	require.True(t, rb.CheckTokenHash("expected"))
	require.False(t, rb.CheckTokenHash("other"))

	identity := robot.NewRobotIdentity(7, "robot$ci")
	require.Equal(t, 7, identity.GetID())
	require.Equal(t, "robot$ci", identity.GetName())
	require.Equal(t, "robot", identity.TypeName())
}
