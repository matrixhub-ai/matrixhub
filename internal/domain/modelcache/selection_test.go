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

package modelcache

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestFnmatchRegexp(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.safetensors", "model.safetensors", true},
		{"*.safetensors", "nested/model.safetensors", true},
		{"*.safetensors", "model.bin", false},
		{"original/", "original/consolidated.00.pth", true},
		{"original/", "original/sub/params.json", true},
		{"original/", "models/original/params.json", false},
		{"model-0000?-of-00002.safetensors", "model-00001-of-00002.safetensors", true},
		{"[!a]*.json", "config.json", true},
		{"[!a]*.json", "added_tokens.json", false},
		{"[ab]*", "b.txt", true},
		{"[ab]*", "c.txt", false},
		{"[]]x", "]x", true},
		{"file[.txt", "file[.txt", true},
		{"a.b", "a.b", true},
		{"a.b", "axb", false},
		{"*.JSON", "config.json", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"~"+tt.path, func(t *testing.T) {
			re, err := fnmatchRegexp(tt.pattern)
			require.NoError(t, err)
			require.Equal(t, tt.want, re.MatchString(tt.path))
		})
	}
}

func TestFnmatchRegexpRejectsInvalidClass(t *testing.T) {
	_, err := fnmatchRegexp("[z-a]")
	require.ErrorIs(t, err, ErrInvalidSelection)
}

func paths(files []git.RevisionFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}

func lfs(path, oid string, size int64) git.RevisionFile {
	return git.RevisionFile{Path: path, Size: size, IsLFS: true, OID: oid}
}

func TestPlanSelectsEveryFileByDefault(t *testing.T) {
	sel, err := compileSelection(Selection{})
	require.NoError(t, err)

	p := sel.plan([]git.RevisionFile{
		{Path: "config.json", Size: 10},
		lfs("model.safetensors", "a", 100),
		lfs("original/consolidated.00.pth", "b", 100),
		lfs("onnx/model.onnx", "c", 100),
		lfs("model-q4.gguf", "d", 100),
	})

	require.Equal(t, []string{"config.json", "model-q4.gguf", "model.safetensors", "onnx/model.onnx", "original/consolidated.00.pth"}, paths(p.files))
	require.Equal(t, []string{"d", "a", "c", "b"}, p.objects)
	require.Equal(t, map[string]int64{"a": 100, "b": 100, "c": 100, "d": 100}, p.sizes)
	require.Empty(t, p.warnings)
}

func TestPlanAppliesPatterns(t *testing.T) {
	sel, err := compileSelection(Selection{
		AllowPatterns:  []string{"*.safetensors", "config.json"},
		IgnorePatterns: []string{"model-00002*"},
	})
	require.NoError(t, err)

	p := sel.plan([]git.RevisionFile{
		{Path: "README.md"},
		{Path: "config.json"},
		lfs("model-00001-of-00002.safetensors", "a", 1),
		lfs("model-00002-of-00002.safetensors", "b", 1),
		lfs("original/consolidated.00.pth", "c", 1),
	})

	require.Equal(t, []string{"config.json", "model-00001-of-00002.safetensors"}, paths(p.files))
}

func TestPlanWarnsWhenNoWeightIsSelected(t *testing.T) {
	sel, err := compileSelection(Selection{AllowPatterns: []string{"*.md"}})
	require.NoError(t, err)

	p := sel.plan([]git.RevisionFile{{Path: "README.md"}, lfs("model-q4_k_m.gguf", "a", 1)})

	require.Equal(t, []string{"README.md"}, paths(p.files))
	require.Len(t, p.warnings, 1)
}

func TestPlanWarnsWhenNothingIsSelected(t *testing.T) {
	sel, err := compileSelection(Selection{AllowPatterns: []string{"*.nothing"}})
	require.NoError(t, err)

	p := sel.plan([]git.RevisionFile{{Path: "README.md"}})

	require.Empty(t, p.files)
	require.Len(t, p.warnings, 1)
}

func TestPlanDeduplicatesObjects(t *testing.T) {
	sel, err := compileSelection(Selection{})
	require.NoError(t, err)

	p := sel.plan([]git.RevisionFile{lfs("a/model.safetensors", "x", 1), lfs("b/model.safetensors", "x", 1)})

	require.Len(t, p.files, 2)
	require.Equal(t, []string{"x"}, p.objects)
}

func TestTaskKey(t *testing.T) {
	require.Equal(t, taskKey("p", "m", "c1", []string{"a", "b"}), taskKey("p", "m", "c1", []string{"b", "a"}))
	require.NotEqual(t, taskKey("p", "m", "c1", []string{"a"}), taskKey("p", "m", "c2", []string{"a"}))
	require.NotEqual(t, taskKey("p", "m", "c1", []string{"a"}), taskKey("p", "m", "c1", []string{"a", "b"}))
}
