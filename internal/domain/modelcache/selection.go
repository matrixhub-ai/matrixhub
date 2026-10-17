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
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// weightExts identify weight-like files, used to warn about selections that
// keep none of them.
var weightExts = []string{
	".safetensors", ".bin", ".pt", ".pth", ".ckpt", ".gguf", ".ggml", ".onnx", ".h5", ".msgpack",
}

type compiledSelection struct {
	allow  []*glob
	ignore []*glob
}

// selectAll selects every file of a revision.
var selectAll = &compiledSelection{}

func compileSelection(sel Selection) (*compiledSelection, error) {
	c := &compiledSelection{}
	for _, p := range sel.AllowPatterns {
		g, err := compileGlob(p)
		if err != nil {
			return nil, err
		}
		c.allow = append(c.allow, g)
	}
	for _, p := range sel.IgnorePatterns {
		g, err := compileGlob(p)
		if err != nil {
			return nil, err
		}
		c.ignore = append(c.ignore, g)
	}
	return c, nil
}

// plan is the result of applying a selection to the files of a revision.
type plan struct {
	// files are the selected files, sorted by path.
	files []git.RevisionFile
	// objects are the unique LFS object ids of files, in path order.
	objects []string
	// sizes maps each object id to its size.
	sizes    map[string]int64
	warnings []string
}

// plan applies the allow patterns (none means everything), then the ignore
// patterns.
func (c *compiledSelection) plan(files []git.RevisionFile) *plan {
	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b git.RevisionFile) int { return strings.Compare(a.Path, b.Path) })

	p := &plan{sizes: map[string]int64{}}
	hasWeights, selectedWeights := false, false
	for _, f := range sorted {
		weight := hasAnySuffix(f.Path, weightExts)
		hasWeights = hasWeights || weight
		if !c.selects(f.Path) {
			continue
		}
		selectedWeights = selectedWeights || weight
		p.files = append(p.files, f)
		if f.IsLFS && f.OID != "" {
			if _, seen := p.sizes[f.OID]; !seen {
				p.objects = append(p.objects, f.OID)
				p.sizes[f.OID] = f.Size
			}
		}
	}

	switch {
	case len(p.files) == 0:
		p.warnings = append(p.warnings, "the selection matched no files")
	case hasWeights && !selectedWeights:
		p.warnings = append(p.warnings, "the allow and ignore patterns exclude every weight file")
	}
	return p
}

func (c *compiledSelection) selects(path string) bool {
	if len(c.allow) == 0 && len(c.ignore) == 0 {
		return true
	}
	runes := []rune(path)
	if len(c.allow) > 0 && !matchAny(c.allow, runes) {
		return false
	}
	return !matchAny(c.ignore, runes)
}

func hasAnySuffix(p string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(p, s) {
			return true
		}
	}
	return false
}

func matchAny(globs []*glob, path []rune) bool {
	for _, g := range globs {
		if g.match(path) {
			return true
		}
	}
	return false
}

// taskKey identifies the files a task pulls: requests that pick the same
// objects of the same commit attach to one running task. It is internal; task
// IDs identify runs.
func taskKey(project, name, commit string, objects []string) string {
	sorted := slices.Clone(objects)
	slices.Sort(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\n")))
	return project + "/" + name + "@" + commit + "#" + hex.EncodeToString(sum[:])[:16]
}
