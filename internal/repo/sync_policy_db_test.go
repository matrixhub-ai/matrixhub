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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRemoteResourceName(t *testing.T) {
	tests := []struct {
		name             string
		fullPath         string
		wantProjectName  string
		wantResourceName string
	}{
		{
			name:             "nested path with multiple slashes",
			fullPath:         "HuggingFaceTB/test/SmolLM2-135M-Instruct",
			wantProjectName:  "HuggingFaceTB",
			wantResourceName: "test/SmolLM2-135M-Instruct",
		},
		{
			name:             "simple path with single slash",
			fullPath:         "org/model",
			wantProjectName:  "org",
			wantResourceName: "model",
		},
		{
			name:             "no slash - only project name",
			fullPath:         "model",
			wantProjectName:  "model",
			wantResourceName: "",
		},
		{
			name:             "deeply nested path",
			fullPath:         "org/team/project/model",
			wantProjectName:  "org",
			wantResourceName: "team/project/model",
		},
		{
			name:             "empty string",
			fullPath:         "",
			wantProjectName:  "",
			wantResourceName: "",
		},
		{
			name:             "trailing slash",
			fullPath:         "org/",
			wantProjectName:  "org",
			wantResourceName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProjectName, gotResourceName := parseRemoteResourceName(tt.fullPath)
			assert.Equal(t, tt.wantProjectName, gotProjectName)
			assert.Equal(t, tt.wantResourceName, gotResourceName)
		})
	}
}
