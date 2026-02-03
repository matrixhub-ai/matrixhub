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

package dao

import (
	"testing"
)

func TestCursorEncodingDecoding(t *testing.T) {
	testCases := []struct {
		id     uint
		cursor string
	}{
		{1, "c-AAAAAAAAAAE="},
		{12345, "c-AAAAAAAAMDk="},
		{67890, "c-AAAAAAABCTI="},
		{4294967295, "c-AAAAAP____8="},
	}

	for _, tc := range testCases {
		encoded := encodeCursor(tc.id)
		if encoded != tc.cursor {
			t.Errorf("encodeCursor(%d) = %s; want %s", tc.id, encoded, tc.cursor)
		}

		decoded, err := decodeCursor(tc.cursor)
		if err != nil {
			t.Errorf("decodeCursor(%s) returned error: %v", tc.cursor, err)
		}
		if decoded != tc.id {
			t.Errorf("decodeCursor(%s) = %d; want %d", tc.cursor, decoded, tc.id)
		}
	}
}
