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
package apiserver

import "testing"

func TestArtifactRepoName(t *testing.T) {
	for _, tc := range []struct {
		input, name, kind string
		valid             bool
	}{
		{"p/m", "p/m", "models", true},
		{"/p/m.git", "p/m", "models", true},
		{"datasets/p/m.git", "datasets/p/m", "datasets", true},
		{"/spaces/p/m.git", "spaces/p/m", "spaces", true},
		{"//p/m.git", "", "", false},
		{"p/../m", "", "", false},
		{"../m", "", "", false},
		{"p/", "", "", false},
		{"p/m\\x", "", "", false},
	} {
		name, kind, valid := artifactRepoName(tc.input)
		if name != tc.name || kind != tc.kind || valid != tc.valid {
			t.Errorf("%q -> (%q,%q,%v), want (%q,%q,%v)", tc.input, name, kind, valid, tc.name, tc.kind, tc.valid)
		}
	}
}
