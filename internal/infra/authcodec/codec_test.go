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

package authcodec_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

type invalidIdentity struct {
	Value chan int `json:"value"`
}

func (invalidIdentity) GetID() int       { return 0 }
func (invalidIdentity) GetName() string  { return "invalid" }
func (invalidIdentity) TypeName() string { return "invalid" }

func TestIdentityCodecRoundTrip(t *testing.T) {
	identities := []struct {
		name string
		id   interface {
			GetID() int
			GetName() string
			TypeName() string
		}
	}{
		{name: "user", id: user.NewUserIdentity(7, "alice")},
		{name: "robot", id: robot.NewRobotIdentity(9, "builder")},
	}
	for _, tt := range identities {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := authcodec.Marshal(tt.id)
			require.NoError(t, err)
			decoded, err := authcodec.Unmarshal(encoded)
			require.NoError(t, err)
			require.Equal(t, tt.id.GetID(), decoded.GetID())
			require.Equal(t, tt.id.GetName(), decoded.GetName())
			require.Equal(t, tt.id.TypeName(), decoded.TypeName())
		})
	}
}

func TestIdentityCodecErrors(t *testing.T) {
	_, err := authcodec.Marshal(invalidIdentity{Value: make(chan int)})
	require.ErrorContains(t, err, "marshal payload")

	for _, encoded := range []string{
		`{`,
		`{"type":"unknown","payload":{}}`,
		`{"type":"user","payload":"invalid"}`,
		`{"type":"robot","payload":"invalid"}`,
	} {
		_, err = authcodec.Unmarshal(encoded)
		require.Error(t, err)
	}
}
