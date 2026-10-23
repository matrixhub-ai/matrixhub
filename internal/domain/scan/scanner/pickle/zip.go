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

package pickle

import (
	"archive/zip"
	"fmt"
	"io"
)

// zipMember adapts *zip.File for the walker without exposing archive/zip.
type zipMember struct {
	f *zip.File
}

func (m zipMember) name() string { return m.f.Name }

func (m zipMember) open() (io.ReadCloser, error) {
	r, err := m.f.Open()
	if err != nil {
		return nil, err
	}
	return r, nil
}

type zipReader struct {
	zr *zip.Reader
}

// seekerReaderAt adapts a plain ReadSeeker for archive/zip when the
// underlying stream does not already implement ReaderAt. Sequential use only.
type seekerReaderAt struct{ rs io.ReadSeeker }

func (s seekerReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if _, err := s.rs.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(s.rs, p)
}

// newZipReader builds a bounded zip reader over a seekable stream.
func newZipReader(rs io.ReadSeeker, size int64) (*zipReader, error) {
	if size <= 0 {
		return nil, fmt.Errorf("empty stream")
	}
	var at io.ReaderAt
	if v, ok := rs.(io.ReaderAt); ok {
		at = v
	} else {
		at = seekerReaderAt{rs: rs}
	}
	zr, err := zip.NewReader(at, size)
	if err != nil {
		return nil, err
	}
	return &zipReader{zr: zr}, nil
}

func (z *zipReader) files() []zipMember {
	out := make([]zipMember, 0, len(z.zr.File))
	for _, f := range z.zr.File {
		out = append(out, zipMember{f: f})
	}
	return out
}
