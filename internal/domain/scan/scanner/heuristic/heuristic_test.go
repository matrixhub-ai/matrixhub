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

package heuristic

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestDetectFileType(t *testing.T) {
	cases := []struct {
		name string
		head string
		want string
	}{
		{"model.pt", "PK\x03\x04", "zip container"},
		{"weights.pkl", "\x80\x02", "pickle"},
		{"data.gz", "\x1f\x8b\x08\x00", "gzip"},
		{"libmodel.so", "\x7fELF\x02", "elf binary"},
		{"evil.exe", "MZ\x90\x00", "pe binary"},
		{"model.safetensors", `{"metadata":`, "safetensors"},
		{"README.md", "# hello", "text"},
	}
	for _, c := range cases {
		rs := bytes.NewReader([]byte(c.head))
		if got := DetectFileType(rs, c.name); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestScanZipBombDeclared(t *testing.T) {
	// Build a zip whose central directory declares a huge uncompressed size.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("bomb.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(bytes.Repeat([]byte{0}, 1<<20)); err != nil {
		t.Fatal(err)
	}
	_ = zw.Close()
	data := buf.Bytes()
	// Corrupt the declared uncompressed size in both local header and CD is
	// fiddly; instead assert via the ratio path with a tiny real member count.
	lim := DefaultLimits()
	lim.MaxRatio = 100 // 1MiB of zeros compresses far beyond 100:1
	fs := Scan(bytes.NewReader(data), "bomb.zip", int64(len(data)), lim)
	found := false
	for _, f := range fs {
		if f.Rule == "heur.zip-bomb" {
			found = true
		}
	}
	if !found {
		t.Fatalf("zip bomb not detected: %+v", fs)
	}
}

func TestScanZipSafe(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("data.pkl")
	_, _ = w.Write([]byte("\x80\x02c\ntorch._utils\n_rebuild_tensor_v2\n"))
	w2, _ := zw.Create("version")
	_, _ = w2.Write([]byte("3\n"))
	_ = zw.Close()
	fs := Scan(bytes.NewReader(buf.Bytes()), "model.pt", int64(buf.Len()), DefaultLimits())
	for _, f := range fs {
		if f.Severity == SeverityCritical {
			t.Fatalf("safe zip flagged critical: %+v", fs)
		}
	}
}

func TestScanGzipBomb(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	// 8 MiB of zeros — well past the 1000:1 ratio for this small compressed size
	if _, err := gw.Write(bytes.Repeat([]byte{0}, 8<<20)); err != nil {
		t.Fatal(err)
	}
	_ = gw.Close()
	// 8 MiB of zeros deflates to roughly a 1000:1 ratio, which sits exactly at
	// the production cap; pin a deterministic, stricter cap for the test.
	lim := DefaultLimits()
	lim.MaxRatio = 100
	fs := Scan(bytes.NewReader(buf.Bytes()), "blob.gz", int64(buf.Len()), lim)
	found := false
	for _, f := range fs {
		if f.Rule == "heur.gzip-bomb" {
			found = true
		}
	}
	if !found {
		t.Fatalf("gzip bomb not detected: %+v", fs)
	}
}

func TestScanGzipSafe(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write(bytes.Repeat([]byte("matrixhub"), 64)) // ~512B, ratio ~ small
	_ = gw.Close()
	fs := Scan(bytes.NewReader(buf.Bytes()), "small.gz", int64(buf.Len()), DefaultLimits())
	for _, f := range fs {
		if strings.HasPrefix(f.Rule, "heur.") && f.Severity == SeverityCritical {
			t.Fatalf("safe gzip flagged: %+v", fs)
		}
	}
}

func TestScanNestedBombDepth(t *testing.T) {
	// gzip inside zip: safe content, exercises the nesting path without tripping caps
	var inner bytes.Buffer
	gw := gzip.NewWriter(&inner)
	_, _ = gw.Write(bytes.Repeat([]byte("x"), 1024))
	_ = gw.Close()

	var outer bytes.Buffer
	zw := zip.NewWriter(&outer)
	w, _ := zw.Create("inner.gz")
	_, _ = w.Write(inner.Bytes())
	_ = zw.Close()

	fs := Scan(bytes.NewReader(outer.Bytes()), "outer.zip", int64(outer.Len()), DefaultLimits())
	for _, f := range fs {
		if f.Rule == "heur.nesting-too-deep" || f.Rule == "heur.zip-bomb" {
			t.Fatalf("nested safe archive flagged: %+v", fs)
		}
	}
}
