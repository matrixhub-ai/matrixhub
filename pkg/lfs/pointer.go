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
	"io"

	"github.com/git-lfs/git-lfs/v3/lfs"
)

// LFS pointers are small (typically < 200 bytes) and have a specific format
const MaxLFSPointerSize = 1024

// DecodePointer parses an LFS pointer from a reader
// Returns nil if the content is not a valid LFS pointer
func DecodePointer(r io.Reader) (*lfs.Pointer, error) {
	return lfs.DecodePointer(r)
}
