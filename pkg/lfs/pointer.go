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
