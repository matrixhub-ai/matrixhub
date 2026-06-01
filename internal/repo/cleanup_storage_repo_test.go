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

package repo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfinedPathAcceptsRelativePathAlreadyUnderRoot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	root := filepath.Join("data", "lfs")
	path := filepath.Join("data", "lfs", "fd", "f7", "object")
	got, err := confinedPath(root, path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := filepath.Join(cwd, path)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestConfinedPathJoinsObjectRelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	got, err := confinedPath(filepath.Join("data", "lfs"), filepath.Join("fd", "f7", "object"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := filepath.Join(cwd, "data", "lfs", "fd", "f7", "object")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestConfinedPathRejectsEscapedPath(t *testing.T) {
	if _, err := confinedPath(filepath.Join("data", "lfs"), filepath.Join("..", "repositories", "repo.git")); err == nil {
		t.Fatal("expected escaped path error")
	}
}
