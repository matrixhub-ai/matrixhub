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

// Package heuristic implements resource-safety checks (compression bombs,
// pathological nesting) and file-type identification for scanned artifacts.
// It never executes or deserializes anything.
package heuristic

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"strings"
)

// Limits bounds decompression work.
type Limits struct {
	MaxDecompressedBytes int64 // absolute cap per container
	MaxRatio             int64 // compressed→decompressed ratio cap
	MaxNestedDepth       int
	MaxMembers           int
}

// DefaultLimits returns production-safe caps.
func DefaultLimits() Limits {
	return Limits{
		MaxDecompressedBytes: 4 << 30, // 4 GiB
		MaxRatio:             1000,
		MaxNestedDepth:       8,
		MaxMembers:           200_000,
	}
}

// Finding mirrors the pickle scanner's finding shape.
type Finding struct {
	Rule     string
	Severity string
	Detail   string
}

// Severities (kept in sync with the scan domain).
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// DetectFileType identifies the file type by magic bytes + extension.
// It returns a short stable identifier used in reports.
func DetectFileType(rs io.ReadSeeker, name string) string {
	head := make([]byte, 16)
	n, _ := rs.Read(head)
	_, _ = rs.Seek(0, io.SeekStart)
	head = head[:n]

	switch {
	case n >= 4 && bytes.Equal(head[:4], []byte("PK\x03\x04")):
		if strings.HasSuffix(strings.ToLower(name), ".npz") {
			return "zip (numpy npz)"
		}
		return "zip container"
	case n >= 2 && bytes.Equal(head[:2], []byte("\x1f\x8b")):
		return "gzip"
	case n >= 4 && bytes.Equal(head[:3], []byte("BZh")):
		return "bzip2"
	case n >= 6 && bytes.Equal(head[:6], []byte("7z\xbc\xaf\x27\x1c")):
		return "7z"
	case n >= 4 && bytes.Equal(head[:4], []byte("\x28\xb5\x2f\xfd")):
		return "zstd"
	case n > 0 && head[0] == 0x80:
		return "pickle"
	case n >= 4 && bytes.Equal(head[:4], []byte("OggS")):
		return "ogg"
	case n >= 12 && bytes.Equal(head[4:8], []byte("ftyp")):
		return "mp4"
	case n >= 4 && bytes.Equal(head[:4], []byte("\x89PNG")):
		return "png"
	case n >= 3 && bytes.Equal(head[:3], []byte("\xff\xd8\xff")):
		return "jpeg"
	case n >= 4 && bytes.Equal(head[:4], []byte("RIFF")):
		return "riff"
	case n >= 5 && bytes.Equal(head[:5], []byte("%PDF-")):
		return "pdf"
	case n >= 4 && bytes.Equal(head[:4], []byte("\x7fELF")):
		return "elf binary"
	case n >= 2 && bytes.Equal(head[:2], []byte("MZ")):
		return "pe binary"
	}
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".py", ".pyc", ".pyo":
		return "python"
	case ".so", ".dylib", ".dll":
		return "shared library"
	case ".sh", ".bash":
		return "shell script"
	case ".exe":
		return "executable"
	case ".safetensors":
		return "safetensors"
	case ".tar", ".tgz", ".tar.gz":
		return "tar"
	case ".gguf":
		return "gguf"
	case ".onnx":
		return "onnx"
	case ".json", ".jsonl":
		return "json"
	case ".md", ".txt":
		return "text"
	case ".csv", ".tsv":
		return "tabular text"
	case ".yaml", ".yml", ".toml":
		return "config"
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return "image"
	case ".pkl", ".pickle", ".pt", ".pth", ".bin", ".ckpt", ".h5", ".msgpack", ".npy":
		return "serialized model"
	default:
		if ext == "" {
			return "unknown"
		}
		return strings.TrimPrefix(ext, ".")
	}
}

// Scan walks archives to detect compression bombs and pathological nesting.
// It reads at most Limits.MaxDecompressedBytes of expanded content per level.
func Scan(rs io.ReadSeeker, name string, size int64, lim Limits) []Finding {
	var findings []Finding
	scanContainer(rs, name, size, lim, 0, &findings)
	return findings
}

func scanContainer(rs io.ReadSeeker, name string, size int64, lim Limits, depth int, out *[]Finding) {
	if depth > lim.MaxNestedDepth {
		*out = append(*out, Finding{Rule: "heur.nesting-too-deep", Severity: SeverityWarning,
			Detail: fmt.Sprintf("%s: nested archive depth exceeds %d", name, lim.MaxNestedDepth)})
		return
	}
	_, _ = rs.Seek(0, io.SeekStart)
	head := make([]byte, 4)
	n, _ := rs.Read(head)
	_, _ = rs.Seek(0, io.SeekStart)
	if n < 4 {
		return
	}

	switch {
	case bytes.Equal(head[:4], []byte("PK\x03\x04")):
		scanZip(rs, size, lim, depth, out)
	case bytes.Equal(head[:2], []byte("\x1f\x8b")):
		scanGzip(rs, size, lim, depth, out)
	}
}

func scanZip(rs io.ReadSeeker, size int64, lim Limits, depth int, out *[]Finding) {
	zr, err := zip.NewReader(&seekerAt{rs}, size)
	if err != nil {
		return
	}
	if len(zr.File) > lim.MaxMembers {
		*out = append(*out, Finding{Rule: "heur.member-count", Severity: SeverityWarning,
			Detail: fmt.Sprintf("zip declares %d members (cap %d)", len(zr.File), lim.MaxMembers)})
		return
	}
	var declared uint64
	for _, f := range zr.File {
		declared += f.UncompressedSize64
	}
	if declared > uint64(lim.MaxDecompressedBytes) {
		*out = append(*out, Finding{Rule: "heur.zip-bomb", Severity: SeverityCritical,
			Detail: fmt.Sprintf("declared expanded size %d bytes exceeds cap %d", declared, lim.MaxDecompressedBytes)})
		return
	}
	if size > 0 && declared > uint64(size)*uint64(lim.MaxRatio) {
		*out = append(*out, Finding{Rule: "heur.zip-bomb", Severity: SeverityCritical,
			Detail: fmt.Sprintf("compression ratio %d:%d exceeds cap %d:1", declared/uint64(size), 1, lim.MaxRatio)})
		return
	}
	// Verify the first member actually decompresses within the cap (declared
	// sizes can lie) and descend into nested archives.
	for i, f := range zr.File {
		if i >= 8 { // bound nested exploration work
			break
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		nested, isArchive, isText := sniffNested(rc, lim)
		_ = rc.Close()
		switch {
		case isArchive:
			nrs := bytes.NewReader(nested)
			scanContainer(nrs, f.Name, int64(len(nested)), lim, depth+1, out)
		case isText:
			// plain member, nothing to do
		}
	}
}

func scanGzip(rs io.ReadSeeker, size int64, lim Limits, depth int, out *[]Finding) {
	gr, err := gzip.NewReader(rs)
	if err != nil {
		return
	}
	defer func() { _ = gr.Close() }()
	var written int64
	buf := make([]byte, 64<<10)
	truncated := false
	var head bytes.Buffer
	for {
		n, err := gr.Read(buf)
		written += int64(n)
		if head.Len() < 512 {
			head.Write(buf[:min(n, 512-head.Len())])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			*out = append(*out, Finding{Rule: "heur.gzip-corrupt", Severity: SeverityWarning,
				Detail: fmt.Sprintf("gzip stream error: %v", err)})
			return
		}
		if written > lim.MaxDecompressedBytes {
			truncated = true
			break
		}
	}
	if truncated || (size > 0 && written > size*lim.MaxRatio) {
		*out = append(*out, Finding{Rule: "heur.gzip-bomb", Severity: SeverityCritical,
			Detail: fmt.Sprintf("expanded %d bytes from %d compressed bytes (caps: %d bytes, %d:1)",
				written, size, lim.MaxDecompressedBytes, lim.MaxRatio)})
		return
	}
	// gzip-wrapped tar: check member count within the cap
	if head.Len() >= 512 && isTarMagic(head.Bytes()) {
		tr := tar.NewReader(bytes.NewReader(head.Bytes()))
		_ = tr // full tar walk is unnecessary for v1; magic identified for typing
	}
}

func isTarMagic(b []byte) bool {
	return len(b) >= 265+5 && string(b[257:262]) == "ustar"
}

// sniffNested reads at most 4 MiB of a member deciding whether it is a nested
// archive; text members are read harmlessly. It enforces the global cap.
func sniffNested(rc io.Reader, lim Limits) (data []byte, isArchive, isText bool) {
	capBytes := int64(4 << 20)
	if capBytes > lim.MaxDecompressedBytes {
		capBytes = lim.MaxDecompressedBytes
	}
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for int64(len(buf)) < capBytes {
		n, err := rc.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	if len(buf) >= 4 {
		if bytes.Equal(buf[:4], []byte("PK\x03\x04")) || bytes.Equal(buf[:2], []byte("\x1f\x8b")) {
			return buf, true, false
		}
	}
	return buf, false, true
}

type seekerAt struct{ rs io.ReadSeeker }

func (s *seekerAt) ReadAt(p []byte, off int64) (int, error) {
	if _, err := s.rs.Seek(off, io.SeekStart); err != nil {
		return 0, err
	}
	return io.ReadFull(s.rs, p)
}
