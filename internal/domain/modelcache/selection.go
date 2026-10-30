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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// weightExts identify weight-like files, used to warn about selections that
// keep none of them.
var weightExts = []string{
	".safetensors", ".bin", ".pt", ".pth", ".ckpt", ".gguf", ".ggml", ".onnx", ".onnx_data", ".h5",
	".msgpack", ".ot", ".tflite", ".mlmodel", ".nemo", ".pdparams", ".engine",
}

// planCheckEvery is how many files a plan matches between checks of its
// context.
const planCheckEvery = 256

type compiledSelection struct {
	allow  []*glob
	ignore []*glob
	// maxSteps bounds the runes that matching compares for one plan.
	maxSteps int
}

// selectAll selects every file of a revision.
var selectAll = &compiledSelection{}

func compileSelection(sel Selection) (*compiledSelection, error) {
	c := &compiledSelection{maxSteps: maxMatchSteps}
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
// patterns. It fails with ErrInvalidSelection when matching the patterns
// against the files would take too long, and stops when ctx ends.
func (c *compiledSelection) plan(ctx context.Context, files []git.RevisionFile) (*plan, error) {
	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b git.RevisionFile) int { return strings.Compare(a.Path, b.Path) })

	p := &plan{sizes: map[string]int64{}}
	budget := &matchBudget{max: c.maxSteps}
	hasWeights, selectedWeights := false, false
	for i, f := range sorted {
		if i%planCheckEvery == 0 && ctx.Err() != nil {
			return nil, context.Cause(ctx)
		}
		weight := isWeight(f.Path)
		hasWeights = hasWeights || weight
		selected := c.selects(f.Path, budget)
		if budget.exceeded() {
			return nil, fmt.Errorf("%w: the patterns are too expensive to match against the %d files of this revision", ErrInvalidSelection, len(files))
		}
		if !selected {
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
	return p, nil
}

func (c *compiledSelection) selects(path string, b *matchBudget) bool {
	if len(c.allow) == 0 && len(c.ignore) == 0 {
		return true
	}
	runes := []rune(normalizeSeparators(path))
	if len(c.allow) > 0 && !matchAny(c.allow, runes, b) {
		return false
	}
	return !matchAny(c.ignore, runes, b)
}

// normalizeSeparators treats '\' as '/' in patterns and paths, as
// huggingface_hub does: repository paths always use '/', and '\' is no escape
// in fnmatch, so patterns written with Windows separators match as intended.
func normalizeSeparators(s string) string {
	return strings.ReplaceAll(s, `\`, "/")
}

// isWeight reports whether p looks like a weight file. training_args.bin, which
// the Trainer of transformers saves next to the weights, holds none.
func isWeight(p string) bool {
	if path.Base(p) == "training_args.bin" {
		return false
	}
	for _, ext := range weightExts {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return false
}

func matchAny(globs []*glob, path []rune, b *matchBudget) bool {
	for _, g := range globs {
		if g.match(path, b) {
			return true
		}
	}
	return false
}

// taskKey identifies the files a task pulls: requests that pick the same
// objects of one model attach to one running task, also when the branch moved
// meanwhile to a commit with the same objects. It is internal; task IDs
// identify runs.
func taskKey(project, name string, objects []string) string {
	sorted := slices.Clone(objects)
	slices.Sort(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\n")))
	return project + "/" + name + "#" + hex.EncodeToString(sum[:])
}
