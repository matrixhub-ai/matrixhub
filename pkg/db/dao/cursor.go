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
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// TODO: future improvement: add checksum to cursor to detect tampering

// encodeCursor encodes a uint ID into a cursor string.
func encodeCursor(id uint) string {
	return "c-" + base64.URLEncoding.EncodeToString(binary.BigEndian.AppendUint64([]byte{}, uint64(id)))
}

// decodeCursor decodes a cursor string back into a uint ID.
func decodeCursor(cursor string) (uint, error) {
	if cursor == "" {
		return 0, fmt.Errorf("cursor is empty")
	}

	if len(cursor) < 3 || cursor[:2] != "c-" {
		return 0, fmt.Errorf("invalid cursor format")
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor[2:])
	if err != nil {
		return 0, fmt.Errorf("failed to decode cursor: %v", err)
	}

	if len(decoded) != 8 {
		return 0, fmt.Errorf("invalid cursor length")
	}

	parsedId := binary.BigEndian.Uint64(decoded)
	return uint(parsedId), nil
}
