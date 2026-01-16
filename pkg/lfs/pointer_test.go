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

package lfs

import (
	"strings"
	"testing"
)

func TestParsePointer(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectOid   string
		expectSize  int64
		expectError bool
	}{
		{
			name: "valid LFS pointer",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
size 1024
`,
			expectOid:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectSize:  1024,
			expectError: false,
		},
		{
			name:        "invalid pointer - not LFS format",
			content:     "Hello, world!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr, err := DecodePointer(strings.NewReader(tt.content))
			if tt.expectError {
				if err == nil {
					t.Errorf("ParsePointer() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParsePointer() unexpected error: %v", err)
			}

			if ptr.Oid != tt.expectOid {
				t.Errorf("ParsePointer() Oid = %q, want %q", ptr.Oid, tt.expectOid)
			}

			if ptr.Size != tt.expectSize {
				t.Errorf("ParsePointer() Size = %d, want %d", ptr.Size, tt.expectSize)
			}
		})
	}
}
