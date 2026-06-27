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

package hf

import (
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/infra/authcodec"
)

func TestCommitAuthorName(t *testing.T) {
	robotIdentity := robot.NewRobotIdentity(4, "robot$test-pro-select")
	encodedRobot, err := authcodec.Marshal(robotIdentity)
	if err != nil {
		t.Fatalf("marshal robot identity: %v", err)
	}

	tests := []struct {
		name string
		user authenticate.UserInfo
		want string
	}{
		{
			name: "typed identity",
			user: authenticate.UserInfo{User: encodedRobot},
			want: "robot$test-pro-select",
		},
		{
			name: "plain user name",
			user: authenticate.UserInfo{User: "HuggingFace"},
			want: "HuggingFace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commitAuthorName(tt.user); got != tt.want {
				t.Fatalf("commitAuthorName() = %q, want %q", got, tt.want)
			}
		})
	}
}
