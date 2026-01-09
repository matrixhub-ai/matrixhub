package lfs

import (
	"bufio"
	"io"
	"strings"

	"github.com/git-lfs/git-lfs/v3/lfs"
)

// Pointer represents a parsed LFS pointer
type Pointer struct {
	Oid  string
	Size int64
}

// ParsePointer parses an LFS pointer from a reader
// Returns nil if the content is not a valid LFS pointer
func ParsePointer(r io.Reader) (*Pointer, error) {
	ptr, err := lfs.DecodePointer(r)
	if err != nil {
		return nil, err
	}
	return &Pointer{
		Oid:  ptr.Oid,
		Size: ptr.Size,
	}, nil
}

// IsLFSPointerContent checks if the given content is an LFS pointer
// LFS pointers are small (typically < 200 bytes) and have a specific format
const maxLFSPointerSize = 1024

func IsLFSPointerContent(content []byte) bool {
	if len(content) > maxLFSPointerSize {
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	hasVersion := false
	hasOid := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "version https://git-lfs.github.com/spec/v") {
			hasVersion = true
		}
		if strings.HasPrefix(line, "oid sha256:") {
			hasOid = true
		}
	}

	return hasVersion && hasOid
}
