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
	"unicode/utf8"
)

// maxPatternWildcards bounds the wildcards ('*', '?' and classes) of one
// pattern. Finding a segment costs up to the path length times the wildcards
// it starts with, so the bound keeps selections cheap on repositories with
// many long paths.
const maxPatternWildcards = 16

// glob matches a path against a pattern with huggingface_hub semantics:
// fnmatch rules where '*' and '?' also match '/', and a trailing '/' selects a
// directory. It does not use regular expressions: the pattern is split at its
// '*'s into segments of single-rune matchers, the first and last segments are
// anchored to the ends of the path, and the segments between are found from
// left to right, each at its earliest position.
type glob struct {
	segments [][]runeMatcher
}

type matcherKind uint8

const (
	matchLiteral matcherKind = iota
	matchAnyRune
	matchClass
)

type runeRange struct{ lo, hi rune }

type runeMatcher struct {
	kind   matcherKind
	r      rune
	negate bool
	ranges []runeRange
}

func (m *runeMatcher) match(r rune) bool {
	switch m.kind {
	case matchAnyRune:
		return true
	case matchClass:
		for _, rg := range m.ranges {
			if rg.lo <= r && r <= rg.hi {
				return !m.negate
			}
		}
		return m.negate
	default:
		return r == m.r
	}
}

func compileGlob(pattern string) (*glob, error) {
	if !utf8.ValidString(pattern) {
		return nil, fmt.Errorf("%w: pattern %q is not valid UTF-8", ErrInvalidSelection, pattern)
	}
	if strings.HasSuffix(pattern, "/") {
		pattern += "*"
	}
	runes := []rune(pattern)
	g := &glob{segments: [][]runeMatcher{nil}}
	wildcards := 0
	for i := 0; i < len(runes); i++ {
		last := len(g.segments) - 1
		switch r := runes[i]; r {
		case '*':
			// A run of '*' matches like a single one.
			for i+1 < len(runes) && runes[i+1] == '*' {
				i++
			}
			g.segments = append(g.segments, nil)
			wildcards++
		case '?':
			g.segments[last] = append(g.segments[last], runeMatcher{kind: matchAnyRune})
			wildcards++
		case '[':
			end := classEnd(runes, i)
			if end < 0 {
				g.segments[last] = append(g.segments[last], runeMatcher{r: r})
				continue
			}
			m, err := parseClass(runes[i+1 : end])
			if err != nil {
				return nil, fmt.Errorf("%w: pattern %q: %v", ErrInvalidSelection, pattern, err)
			}
			g.segments[last] = append(g.segments[last], m)
			wildcards++
			i = end
		default:
			g.segments[last] = append(g.segments[last], runeMatcher{r: r})
		}
	}
	if wildcards > maxPatternWildcards {
		return nil, fmt.Errorf("%w: pattern %q has more than %d wildcards", ErrInvalidSelection, pattern, maxPatternWildcards)
	}
	return g, nil
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

// parseClass parses the body of a class: a leading '!' negates it and "a-z" is
// a range. As in fnmatch, '^' and '\' are literals.
func parseClass(body []rune) (runeMatcher, error) {
	m := runeMatcher{kind: matchClass}
	if len(body) > 0 && body[0] == '!' {
		m.negate = true
		body = body[1:]
	}
	for i := 0; i < len(body); i++ {
		lo, hi := body[i], body[i]
		if i+2 < len(body) && body[i+1] == '-' {
			hi = body[i+2]
			i += 2
		}
		if hi < lo {
			return m, fmt.Errorf("invalid range %c-%c", lo, hi)
		}
		m.ranges = append(m.ranges, runeRange{lo: lo, hi: hi})
	}
	return m, nil
}

func (g *glob) match(path []rune) bool {
	if len(g.segments) == 1 {
		return len(path) == len(g.segments[0]) && matchAt(g.segments[0], path, 0)
	}
	first, last := g.segments[0], g.segments[len(g.segments)-1]
	if len(first)+len(last) > len(path) ||
		!matchAt(first, path, 0) ||
		!matchAt(last, path, len(path)-len(last)) {
		return false
	}
	// Placing each middle segment as early as possible leaves the most room
	// for the segments after it.
	pos, end := len(first), len(path)-len(last)
	for _, seg := range g.segments[1 : len(g.segments)-1] {
		i := find(seg, path[:end], pos)
		if i < 0 {
			return false
		}
		pos = i + len(seg)
	}
	return true
}

func matchAt(seg []runeMatcher, path []rune, at int) bool {
	for i := range seg {
		if !seg[i].match(path[at+i]) {
			return false
		}
	}
	return true
}

// find returns the first index from which seg matches path, or -1.
func find(seg []runeMatcher, path []rune, from int) int {
	for i := from; i+len(seg) <= len(path); i++ {
		if matchAt(seg, path, i) {
			return i
		}
	}
	return -1
}
