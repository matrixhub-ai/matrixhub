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
	"fmt"
	"regexp"
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
	allow  []*regexp.Regexp
	ignore []*regexp.Regexp
}

func compileSelection(sel Selection) (*compiledSelection, error) {
	c := &compiledSelection{}
	for _, p := range sel.AllowPatterns {
		re, err := fnmatchRegexp(p)
		if err != nil {
			return nil, err
		}
		c.allow = append(c.allow, re)
	}
	for _, p := range sel.IgnorePatterns {
		re, err := fnmatchRegexp(p)
		if err != nil {
			return nil, err
		}
		c.ignore = append(c.ignore, re)
	}
	return c, nil
}

// fnmatchRegexp translates a pattern with huggingface_hub semantics: fnmatch
// rules where '*' also matches '/', and a trailing '/' selects a directory.
func fnmatchRegexp(pattern string) (*regexp.Regexp, error) {
	if strings.HasSuffix(pattern, "/") {
		pattern += "*"
	}
	var b strings.Builder
	runes := []rune(pattern)
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			end := classEnd(runes, i)
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			b.WriteString(translateClass(runes[i+1 : end]))
			i = end
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	re, err := regexp.Compile(`^(?s:` + b.String() + `)$`)
	if err != nil {
		return nil, fmt.Errorf("%w: pattern %q: %v", ErrInvalidSelection, pattern, err)
	}
	return re, nil
}

// classEnd returns the index of the ']' closing the class opened at start, or
// -1 when the class is not closed. As in fnmatch, a ']' right after '[' or
// "[!" is a literal.
func classEnd(runes []rune, start int) int {
	j := start + 1
	if j < len(runes) && runes[j] == '!' {
		j++
	}
	if j < len(runes) && runes[j] == ']' {
		j++
	}
	for ; j < len(runes); j++ {
		if runes[j] == ']' {
			return j
		}
	}
	return -1
}

func translateClass(body []rune) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, r := range body {
		switch {
		case i == 0 && r == '!':
			b.WriteByte('^')
		case i == 0 && r == '^':
			b.WriteString(`\^`)
		case r == '\\' || r == '[' || r == ']':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte(']')
	return b.String()
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
		if len(c.allow) > 0 && !matchAny(c.allow, f.Path) {
			continue
		}
		if matchAny(c.ignore, f.Path) {
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

func hasAnySuffix(p string, suffixes []string) bool {
	for _, s := range suffixes {
		if strings.HasSuffix(p, s) {
			return true
		}
	}
	return false
}

func matchAny(res []*regexp.Regexp, p string) bool {
	for _, re := range res {
		if re.MatchString(p) {
			return true
		}
	}
	return false
}

// taskKey identifies the cache task of a selection: selections that pick the
// same objects of the same commit share a task.
func taskKey(project, name, commit string, objects []string) string {
	sorted := slices.Clone(objects)
	slices.Sort(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\n")))
	return project + "/" + name + "@" + commit + "#" + hex.EncodeToString(sum[:])[:16]
}
