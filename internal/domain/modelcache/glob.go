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
// pattern.
const maxPatternWildcards = 16

// maxMatchSteps bounds the comparisons that matching makes for one plan:
// about half a second of work for the costliest comparisons, a rune against a
// range of a class, and about a third for most. Finding a segment costs up to
// the path length times the segment length, and a character class compares a
// rune with each of its ranges, so long patterns against many long paths
// could otherwise keep the server busy for minutes. Ordinary selections stay
// far below it: ten patterns that scan whole paths fit for 100,000 files.
const maxMatchSteps = 100_000_000

// matchBudget counts the comparisons that matching makes.
type matchBudget struct {
	steps, max int
}

func (b *matchBudget) exceeded() bool { return b.steps > b.max }

// glob matches a path against a pattern with huggingface_hub semantics:
// fnmatch rules where '*' and '?' also match '/', a trailing '/' selects a
// directory, and '\' counts as '/' in patterns and paths, as huggingface_hub
// normalizes it. It does not use regular expressions: the pattern is split at its
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

// cost is the number of comparisons that match makes at most.
func (m *runeMatcher) cost() int {
	if m.kind == matchClass && len(m.ranges) > 1 {
		return len(m.ranges)
	}
	return 1
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
	body := normalizeSeparators(pattern)
	wildcards := 0
	if strings.HasSuffix(body, "/") {
		// The '*' that selects the directory's files is not the user's.
		body += "*"
		wildcards--
	}
	runes := []rune(body)
	g := &glob{segments: [][]runeMatcher{nil}}
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
			g.segments[last] = append(g.segments[last], parseClass(runes[i+1:end]))
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
// a range. As in fnmatch, '^' is a literal, and a reversed range such as "z-a"
// is dropped: a class left empty matches nothing, or, negated, any rune. A '\'
// was turned into '/' already.
func parseClass(body []rune) runeMatcher {
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
			continue
		}
		m.ranges = append(m.ranges, runeRange{lo: lo, hi: hi})
	}
	return m
}

// match reports whether path matches the pattern. It charges the comparisons
// it makes to b and gives up, returning false, once b is exceeded.
func (g *glob) match(path []rune, b *matchBudget) bool {
	if len(g.segments) == 1 {
		return len(path) == len(g.segments[0]) && matchAt(g.segments[0], path, 0, b)
	}
	first, last := g.segments[0], g.segments[len(g.segments)-1]
	if len(first)+len(last) > len(path) ||
		!matchAt(first, path, 0, b) ||
		!matchAt(last, path, len(path)-len(last), b) {
		return false
	}
	// Placing each middle segment as early as possible leaves the most room
	// for the segments after it.
	pos, end := len(first), len(path)-len(last)
	for _, seg := range g.segments[1 : len(g.segments)-1] {
		i := find(seg, path[:end], pos, b)
		if i < 0 {
			return false
		}
		pos = i + len(seg)
	}
	return true
}

func matchAt(seg []runeMatcher, path []rune, at int, b *matchBudget) bool {
	for i := range seg {
		b.steps += seg[i].cost()
		if !seg[i].match(path[at+i]) {
			return false
		}
	}
	return true
}

// find returns the first index from which seg matches path, or -1.
func find(seg []runeMatcher, path []rune, from int, b *matchBudget) int {
	for i := from; i+len(seg) <= len(path) && !b.exceeded(); i++ {
		if matchAt(seg, path, i, b) {
			return i
		}
	}
	return -1
}
