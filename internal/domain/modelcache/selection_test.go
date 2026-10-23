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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestGlob(t *testing.T) {
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
		{"model?safetensors", "model/safetensors", true},
		{"*", "", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "aXbYbZc", true},
		{"a*b*c", "acb", false},
		{"a**b", "a/x/b", true},
		{"*a*a*", "a", false},
		{"*a*a*", "aa", true},
		{"[a-c]x", "bx", true},
		{"[a-c]x", "dx", false},
		{"[!a-c]x", "dx", true},
		{"[a-]x", "-x", true},
		{"[^a]", "^", true},
		{"[^a]", "b", false},
		{`a\b`, `a\b`, true},
		{"[!]", "[!]", true},
		{"模型/*.bin", "模型/权重.bin", true},
		{"???", "ab", false},
		// As in fnmatch, a reversed range is dropped: a class left empty
		// matches nothing, or, negated, anything.
		{"[z-a]x", "ax", false},
		{"[z-a]x", "zx", false},
		{"[!b-a]x", "qx", true},
		{"[b-aX]", "X", true},
		{"[b-aX]", "b", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"~"+tt.path, func(t *testing.T) {
			g, err := compileGlob(tt.pattern)
			require.NoError(t, err)
			require.Equal(t, tt.want, g.match([]rune(tt.path), &matchBudget{max: maxMatchSteps}))
		})
	}
}

func TestGlobRejectsInvalidPatterns(t *testing.T) {
	for _, pattern := range []string{"\xff*", strings.Repeat("?", maxPatternWildcards+1)} {
		_, err := compileGlob(pattern)
		require.ErrorIs(t, err, ErrInvalidSelection, pattern)
	}
	// A run of '*' counts as one wildcard.
	_, err := compileGlob(strings.Repeat("*", 500) + "zz")
	require.NoError(t, err)

	// The '*' that a trailing '/' implies is not counted, and errors quote
	// the pattern as given.
	dir := strings.Repeat("?", maxPatternWildcards) + "/"
	_, err = compileGlob(dir)
	require.NoError(t, err)
	_, err = compileGlob("?" + dir)
	require.ErrorContains(t, err, fmt.Sprintf("%q", "?"+dir))
}

// A class compares a rune with each of its ranges, and the budget counts
// every comparison.
func TestGlobChargesEachRangeOfAClass(t *testing.T) {
	g, err := compileGlob("[" + cjkRunes(200) + "]")
	require.NoError(t, err)
	b := &matchBudget{max: maxMatchSteps}

	require.False(t, g.match([]rune("x"), b))
	require.Equal(t, 200, b.steps)
}

// cjkRunes returns n distinct runes that paths of ASCII never contain.
func cjkRunes(n int) string {
	var sb strings.Builder
	for i := range n {
		sb.WriteRune(rune(0x4e00 + 2*i))
	}
	return sb.String()
}

// Classes of many ranges cost a comparison per range, so such patterns
// against many paths exhaust the budget instead of keeping the server busy
// for a minute.
func TestPlanFailsWhenClassesAreTooExpensive(t *testing.T) {
	var allow []string
	for range 16 {
		allow = append(allow, "*["+cjkRunes(252)+"]*")
	}
	sel, err := compileSelection(Selection{AllowPatterns: allow})
	require.NoError(t, err)
	// Matching makes about 6.4 million comparisons of a rune with a class,
	// each against 252 ranges: within this budget only if a class counted
	// as one comparison.
	sel.maxSteps = 10_000_000
	files := make([]git.RevisionFile, 2000)
	for i := range files {
		files[i] = git.RevisionFile{Path: fmt.Sprintf("%05d/", i) + strings.Repeat("abcdefgh/", 21)}
	}

	start := time.Now()
	_, err = sel.plan(files)
	require.ErrorIs(t, err, ErrInvalidSelection)
	require.Less(t, time.Since(start), 2*time.Second)
}

// The regular expressions that patterns used to compile to took seconds to
// plan such a selection.
func TestPlanStaysCheapWithPathologicalPatterns(t *testing.T) {
	var allow, ignore []string
	for i := 0; i < 16; i++ {
		allow = append(allow, strings.Repeat("*[!/]", 7)+"*zz")
		ignore = append(ignore, strings.Repeat("*?", 8)+"zz")
	}
	sel, err := compileSelection(Selection{AllowPatterns: allow, IgnorePatterns: ignore})
	require.NoError(t, err)
	files := make([]git.RevisionFile, 1000)
	for i := range files {
		files[i] = git.RevisionFile{Path: fmt.Sprintf("%05d/", i) + strings.Repeat("abcdefgh/", 22)}
	}

	start := time.Now()
	_, err = sel.plan(files)
	require.NoError(t, err)
	require.Less(t, time.Since(start), 2*time.Second)
}

// Long literal segments make finding a segment cost the path length times
// the segment length, so such patterns against long paths exhaust the budget.
func TestPlanFailsWhenMatchingIsTooExpensive(t *testing.T) {
	var allow []string
	for i := 0; i < 16; i++ {
		allow = append(allow, "*"+strings.Repeat("a", 100)+"b*")
	}
	sel, err := compileSelection(Selection{AllowPatterns: allow})
	require.NoError(t, err)
	sel.maxSteps = 10_000_000
	files := make([]git.RevisionFile, 1000)
	for i := range files {
		files[i] = git.RevisionFile{Path: fmt.Sprintf("%05d", i) + strings.Repeat("a", 1000)}
	}

	start := time.Now()
	_, err = sel.plan(files)
	require.ErrorIs(t, err, ErrInvalidSelection)
	require.Less(t, time.Since(start), 2*time.Second)

	// The same patterns are cheap on paths without long runs of 'a'.
	for i := range files {
		files[i].Path = fmt.Sprintf("%05d/", i) + strings.Repeat("model/", 30) + "model.safetensors"
	}
	_, err = sel.plan(files)
	require.NoError(t, err)
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

	p, err := sel.plan([]git.RevisionFile{
		{Path: "config.json", Size: 10},
		lfs("model.safetensors", "a", 100),
		lfs("original/consolidated.00.pth", "b", 100),
		lfs("onnx/model.onnx", "c", 100),
		lfs("model-q4.gguf", "d", 100),
	})
	require.NoError(t, err)

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

	p, err := sel.plan([]git.RevisionFile{
		{Path: "README.md"},
		{Path: "config.json"},
		lfs("model-00001-of-00002.safetensors", "a", 1),
		lfs("model-00002-of-00002.safetensors", "b", 1),
		lfs("original/consolidated.00.pth", "c", 1),
	})
	require.NoError(t, err)

	require.Equal(t, []string{"config.json", "model-00001-of-00002.safetensors"}, paths(p.files))
}

func TestPlanWarnsWhenNoWeightIsSelected(t *testing.T) {
	sel, err := compileSelection(Selection{AllowPatterns: []string{"*.md"}})
	require.NoError(t, err)

	p, err := sel.plan([]git.RevisionFile{{Path: "README.md"}, lfs("model-q4_k_m.gguf", "a", 1)})
	require.NoError(t, err)

	require.Equal(t, []string{"README.md"}, paths(p.files))
	require.Len(t, p.warnings, 1)
}

func TestPlanWarnsWhenNothingIsSelected(t *testing.T) {
	sel, err := compileSelection(Selection{AllowPatterns: []string{"*.nothing"}})
	require.NoError(t, err)

	p, err := sel.plan([]git.RevisionFile{{Path: "README.md"}})
	require.NoError(t, err)

	require.Empty(t, p.files)
	require.Len(t, p.warnings, 1)
}

func TestPlanDeduplicatesObjects(t *testing.T) {
	sel, err := compileSelection(Selection{})
	require.NoError(t, err)

	p, err := sel.plan([]git.RevisionFile{lfs("a/model.safetensors", "x", 1), lfs("b/model.safetensors", "x", 1)})
	require.NoError(t, err)

	require.Len(t, p.files, 2)
	require.Equal(t, []string{"x"}, p.objects)
}

func TestTaskKey(t *testing.T) {
	require.Equal(t, taskKey("p", "m", []string{"a", "b"}), taskKey("p", "m", []string{"b", "a"}))
	require.NotEqual(t, taskKey("p", "m", []string{"a"}), taskKey("p", "m", []string{"a", "b"}))
	require.NotEqual(t, taskKey("p", "m", []string{"a"}), taskKey("p", "other", []string{"a"}))
}
