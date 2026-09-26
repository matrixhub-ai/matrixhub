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

// Package pickle implements a static analyzer for Python pickle streams.
//
// It walks the pickle opcode stream WITHOUT building any objects, importing
// any module or executing any code: the only state it keeps is a bounded
// shadow stack of string operands so that STACK_GLOBAL (protocol 4+) targets
// can be resolved. Every GLOBAL / INST / STACK_GLOBAL operand is matched
// against a denylist of executable entrypoints (critical), an allowlist of
// common model-framework globals (clean) and everything else is reported as a
// warning for manual review.
//
// The walker is hardened against malicious inputs: bounded opcode count,
// bounded memo puts, bounded stack depth and bounded string operands. A
// malformed or truncated stream is reported as a finding, never as clean and
// never as a crash.
package pickle

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Limits bounds the walker so crafted streams cannot exhaust CPU or memory.
// The zero value is NOT safe; use DefaultLimits.
type Limits struct {
	MaxOpcodes      int64
	MaxStackDepth   int
	MaxStringLength int
	MaxFindings     int
}

// DefaultLimits returns production-safe walker bounds.
func DefaultLimits() Limits {
	return Limits{
		MaxOpcodes:      5_000_000,
		MaxStackDepth:   10_000,
		MaxStringLength: 8 << 20,
		MaxFindings:     64,
	}
}

// Severity levels mirror the scan domain's file-finding severities.
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// Finding is one static observation about a pickle stream. It never carries
// payload bytes, only rule identifiers and resolved references.
type Finding struct {
	Rule      string // e.g. "pickle.dangerous-global"
	Severity  string
	GlobalRef string // "module.name" when applicable
	Detail    string
}

// Classification of a resolved module.name reference.
type verdict int

const (
	verdictAllow verdict = iota
	verdictDeny
	verdictWarn
)

// Modules whose ANY referenced name means arbitrary code execution at unpickle
// time. Loading such a pickle with torch.load / pickle.load runs attacker code.
var criticalModules = map[string]string{
	"os":              "pickle.os-global",
	"posix":           "pickle.posix-global",
	"nt":              "pickle.nt-global",
	"subprocess":      "pickle.subprocess-global",
	"ctypes":          "pickle.ctypes-global",
	"pty":             "pickle.pty-global",
	"_winapi":         "pickle.winapi-global",
	"multiprocessing": "pickle.multiprocessing-global",
	"importlib":       "pickle.importlib-global",
	"runpy":           "pickle.runpy-global",
	"code":            "pickle.code-global",
	"codeop":          "pickle.codeop-global",
	"shutil":          "pickle.shutil-global",
}

// Dangerous names inside otherwise-benign modules.
var criticalNames = map[string]map[string]string{
	"builtins": {
		"eval": "pickle.builtins-eval",
		"exec": "pickle.builtins-exec",
		// py2 sibling of exec
		"execfile":     "pickle.builtins-execfile",
		"compile":      "pickle.builtins-compile",
		"__import__":   "pickle.builtins-import",
		"open":         "pickle.builtins-open",
		"breakpoint":   "pickle.builtins-breakpoint",
		"input":        "pickle.builtins-input",
		"__builtins__": "pickle.builtins-dunder",
	},
	"io":       {"open": "pickle.io-open"},
	"platform": {"popen": "pickle.platform-popen"},
}

// Modules (with wildcards for sub-packages) that model frameworks legitimately
// reference from pickles. Anything matched here is clean.
var allowModules = newStringTree(
	"torch", "torchvision", "torchaudio", "torchtext",
	"tensorflow", "keras", "jax", "flax",
	"numpy", "scipy", "pandas", "sklearn",
	"transformers", "tokenizers", "safetensors", "diffusers",
	"collections", "re", "copyreg", "_codecs", "_collections_abc",
	"functools", "itertools", "operator", "enum", "types",
	"decimal", "fractions", "statistics", "datetime", "math", "random",
)

// classify returns the verdict for a resolved module.name pair.
func classify(module, name string) (verdict, string) {
	if module == "" || name == "" {
		return verdictWarn, ""
	}
	if rule, ok := criticalModules[module]; ok {
		return verdictDeny, rule
	}
	if names := criticalNames[module]; names != nil {
		if rule, ok := names[name]; ok {
			return verdictDeny, rule
		}
		if module == "builtins" {
			return verdictAllow, ""
		}
	}
	if allowModules.contains(module) {
		return verdictAllow, ""
	}
	switch module {
	case "sys", "socket", "shutil", "pathlib", "tempfile", "glob", "os.path":
		return verdictWarn, "pickle.review-module"
	}
	return verdictWarn, "pickle.unknown-global"
}

// stringTree matches module names including sub-packages ("torch._utils"
// matches because "torch" is registered).
type stringTree map[string]struct{}

func newStringTree(modules ...string) stringTree {
	t := make(stringTree, len(modules))
	for _, m := range modules {
		t[m] = struct{}{}
	}
	return t
}

func (t stringTree) contains(module string) bool {
	if _, ok := t[module]; ok {
		return true
	}
	// longest-prefix match on package boundaries: torch._utils → torch
	for {
		i := strings.LastIndexByte(module, '.')
		if i < 0 {
			return false
		}
		module = module[:i]
		if _, ok := t[module]; ok {
			return true
		}
	}
}

// stackEntry is a shadow value: non-nil only for string operands we tracked.
type stackEntry *string

type walker struct {
	r    *bufio.Reader
	lim  Limits
	ops  int64
	stk  []stackEntry
	find []Finding
	done bool
}

func (w *walker) add(f Finding) {
	if len(w.find) >= w.lim.MaxFindings {
		w.done = true
		return
	}
	w.find = append(w.find, f)
	if f.Severity == SeverityCritical {
		// keep scanning a bounded number of extra ops for reporting, but the
		// verdict is already decided; stop early to bound work on bombs.
		w.done = true
	}
}

func (w *walker) push(s stackEntry) error {
	if len(w.stk) >= w.lim.MaxStackDepth {
		return fmt.Errorf("stack depth exceeds %d", w.lim.MaxStackDepth)
	}
	w.stk = append(w.stk, s)
	return nil
}

func (w *walker) pop() stackEntry {
	if n := len(w.stk); n > 0 {
		v := w.stk[n-1]
		w.stk = w.stk[:n-1]
		return v
	}
	return nil
}

func (w *walker) popMark() {
	if i := len(w.stk) - 1; i >= 0 {
		for i >= 0 && w.stk[i] != marker {
			i--
		}
		if i >= 0 {
			w.stk = w.stk[:i]
			return
		}
	}
	w.stk = w.stk[:0]
}

// marker is a sentinel pushed by MARK.
var marker stackEntry = (*string)(nil)

// readStringRaw reads a newline-terminated protocol-0 string operand.
func (w *walker) readStringRaw() (string, error) {
	var sb strings.Builder
	for {
		c, err := w.r.ReadByte()
		if err != nil {
			return "", err
		}
		if c == '\n' {
			break
		}
		if sb.Len() > w.lim.MaxStringLength {
			return "", fmt.Errorf("string operand exceeds %d bytes", w.lim.MaxStringLength)
		}
		if c != '\\' {
			sb.WriteByte(c)
			continue
		}
		// protocol-0 escapes: \\ \' \" \a \b \f \n \r \t \v \0 + \xHH
		e, err := w.r.ReadByte()
		if err != nil {
			return "", err
		}
		switch e {
		case '\\', '\'', '"', 'a', 'b', 'f', 'n', 'r', 't', 'v', '0':
			sb.WriteByte(e)
		case 'x':
			h := make([]byte, 2)
			if _, err := io.ReadFull(w.r, h); err != nil {
				return "", err
			}
			sb.WriteByte('\\')
			sb.Write(h)
		default:
			sb.WriteByte(e)
		}
	}
	return sb.String(), nil
}

// readCount reads a little-endian length prefix of n bytes.
func (w *walker) readCount(n int) (int64, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(w.r, b); err != nil {
		return 0, err
	}
	v := int64(0)
	for i := n - 1; i >= 0; i-- {
		v = v<<8 | int64(b[i])
	}
	return v, nil
}

// readLenString reads a length-prefixed string operand, enforcing the cap.
// Payloads beyond the cap are skipped without allocation.
func (w *walker) readLenString(n int) (string, error) {
	ln, err := w.readCount(n)
	if err != nil {
		return "", err
	}
	if ln > int64(w.lim.MaxStringLength) {
		if _, err := io.CopyN(io.Discard, w.r, ln); err != nil {
			return "", err
		}
		return "", nil // oversized: treat as opaque non-string
	}
	b := make([]byte, ln)
	if _, err := io.ReadFull(w.r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

// reportGlobal records a resolved module.name reference.
func (w *walker) reportGlobal(module, name string) {
	ref := module + "." + name
	v, rule := classify(module, name)
	switch v {
	case verdictDeny:
		w.add(Finding{
			Rule:      rule,
			Severity:  SeverityCritical,
			GlobalRef: ref,
			Detail:    fmt.Sprintf("pickle references executable entrypoint %s; unpickling runs attacker-controlled code", ref),
		})
	case verdictWarn:
		w.add(Finding{
			Rule:      rule,
			Severity:  SeverityWarning,
			GlobalRef: ref,
			Detail:    fmt.Sprintf("pickle imports %s which is outside the known-safe model framework set; manual review recommended", ref),
		})
	}
}

// step decodes one opcode. Returns (ok, error); ok=false means clean STOP or
// budget cap reached.
func (w *walker) step() (bool, error) {
	w.ops++
	if w.ops > w.lim.MaxOpcodes {
		w.add(Finding{
			Rule:     "pickle.opcode-budget-exceeded",
			Severity: SeverityWarning,
			Detail:   fmt.Sprintf("stream exceeds %d opcodes; analysis stopped at the cap", w.lim.MaxOpcodes),
		})
		return false, nil
	}
	op, err := w.r.ReadByte()
	if err != nil {
		return false, err
	}
	switch op {
	case 0x80: // PROTO
		_, err := w.r.ReadByte()
		return err == nil, err
	case 0x81: // NEWOBJ — consumes args + the resolved global
		w.pop()
		w.pop()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x96: // NEWOBJ_EX — kwargs, args, global
		w.pop()
		w.pop()
		w.pop()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x82, 0x83, 0x84: // EXT1/2/4
		n := 1 + int(op-0x82)
		if _, err := w.readCount(n); err != nil {
			return false, err
		}
	case 0x85, 0x86, 0x87: // TUPLE1/2/3
		w.pop()
		if op >= 0x86 {
			w.pop()
		}
		if op == 0x87 {
			w.pop()
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x88, 0x89, 0x4E, 0x8F, 0x91, 0x5D, 0x7D: // NEWTRUE/FALSE/NONE/EMPTY_SET/FROZENSET/EMPTY_LIST/EMPTY_DICT
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x8A, 0x8B: // LONG1/LONG4
		n := 1
		if op == 0x8B {
			n = 4
		}
		ln, err := w.readCount(n)
		if err != nil {
			return false, err
		}
		if _, err := io.CopyN(io.Discard, w.r, ln); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x42, 0x43: // BINBYTES, SHORT_BINBYTES
		n := 4
		if op == 0x43 {
			n = 1
		}
		ln, err := w.readCount(n)
		if err != nil {
			return false, err
		}
		if _, err := io.CopyN(io.Discard, w.r, ln); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x8E: // BINBYTES8
		ln, err := w.readCount(8)
		if err != nil {
			return false, err
		}
		if _, err := io.CopyN(io.Discard, w.r, ln); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x96 + 1: // BYTEARRAY8 (0x97)
		ln, err := w.readCount(8)
		if err != nil {
			return false, err
		}
		if _, err := io.CopyN(io.Discard, w.r, ln); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x98, 0x99: // NEXT_BUFFER, READONLY_BUFFER
	case 0x8C, 0x58, 0x55, 0x54: // SHORT_BINUNICODE, BINUNICODE, SHORT_BINSTRING, BINSTRING
		n := map[byte]int{0x8C: 1, 0x58: 4, 0x55: 1, 0x54: 4}[op]
		s, err := w.readLenString(n)
		if err != nil {
			return false, err
		}
		if s == "" {
			if err := w.push(nil); err != nil {
				return false, err
			}
		} else {
			sp := s
			if err := w.push(&sp); err != nil {
				return false, err
			}
		}
	case 0x8D: // BINUNICODE8
		s, err := w.readLenString(8)
		if err != nil {
			return false, err
		}
		if s == "" {
			err = w.push(nil)
		} else {
			sp := s
			err = w.push(&sp)
		}
		if err != nil {
			return false, err
		}
	case 0x53: // STRING (protocol 0, escapes)
		s, err := w.readStringRaw()
		if err != nil {
			return false, err
		}
		sp := s
		if err := w.push(&sp); err != nil {
			return false, err
		}
	case 0x56: // UNICODE (raw-unicode-escape, line terminated)
		s, err := w.readStringRaw()
		if err != nil {
			return false, err
		}
		sp := s
		if err := w.push(&sp); err != nil {
			return false, err
		}
	case 0x46, 0x47: // FLOAT (raw), BINFLOAT
		n := 0
		if op == 0x46 {
			n = -1 // line
		} else {
			n = 8
		}
		if n > 0 {
			if _, err := io.CopyN(io.Discard, w.r, int64(n)); err != nil {
				return false, err
			}
		} else if _, err := w.readStringRaw(); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x49: // INT (line: "123\n" or "01\n" bool)
		if _, err := w.readStringRaw(); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x4A, 0x4B, 0x4D: // BININT, BININT1, BININT2
		n := 4
		switch op {
		case 0x4B:
			n = 1
		case 0x4D:
			n = 2
		default:
		}
		if _, err := io.CopyN(io.Discard, w.r, int64(n)); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x4C: // LONG (line)
		if _, err := w.readStringRaw(); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x50: // PERSID (line)
		if _, err := w.readStringRaw(); err != nil {
			return false, err
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x51: // BINPERSID
		w.pop()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x52: // REDUCE — the actual "call" of the last GLOBAL
		w.pop()
		w.pop()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x28: // MARK
		if err := w.push(marker); err != nil {
			return false, err
		}
	case 0x29: // EMPTY_TUPLE ')'
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x30: // POP
		w.pop()
	case 0x31: // POP_MARK
		w.popMark()
	case 0x32: // DUP
		if n := len(w.stk); n > 0 {
			if err := w.push(w.stk[n-1]); err != nil {
				return false, err
			}
		}
	case 0x63, 0x69: // GLOBAL 'c', INST 'i' (module\nname\n)
		module, err1 := w.readStringRaw()
		name, err2 := w.readStringRaw()
		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("truncated GLOBAL/INST operand")
		}
		w.reportGlobal(module, name)
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x64: // DICT
		w.popMark()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x61: // APPEND
		w.pop()
	case 0x65: // APPENDS
		w.popMark()
	case 0x62: // BUILD
		w.pop()
	case 0x67, 0x68, 0x6A: // GET, BINGET, LONG_BINGET
		if op == 0x67 {
			if _, err := w.readStringRaw(); err != nil {
				return false, err
			}
		} else {
			n := 1
			if op == 0x6A {
				n = 4
			}
			if _, err := w.readCount(n); err != nil {
				return false, err
			}
		}
		// memo GET returns an unknown value (we do not track memo contents)
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x6C: // LIST
		w.popMark()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x6F: // OBJ
		w.popMark()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x70, 0x71, 0x72: // PUT, BINPUT, LONG_BINPUT (memo puts, value stays)
		if op == 0x70 {
			if _, err := w.readStringRaw(); err != nil {
				return false, err
			}
		} else {
			n := 1
			if op == 0x72 {
				n = 4
			}
			if _, err := w.readCount(n); err != nil {
				return false, err
			}
		}
	case 0x94: // MEMOIZE (protocol 4, memoizes TOS)
	case 0x73: // SETITEM
		w.pop()
		w.pop()
	case 0x75: // SETITEMS
		w.popMark()
	case 0x74: // TUPLE
		w.popMark()
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x93: // STACK_GLOBAL — module & name from stack (protocol 4)
		name := w.pop()
		module := w.pop()
		if module != nil && name != nil {
			w.reportGlobal(*module, *name)
		} else {
			w.add(Finding{
				Rule:     "pickle.stack-global-unresolved",
				Severity: SeverityWarning,
				Detail:   "STACK_GLOBAL operands are not statically resolvable (built via computed values); review manually",
			})
		}
		if err := w.push(nil); err != nil {
			return false, err
		}
	case 0x95: // FRAME (8-byte length prefix)
		if _, err := w.readCount(8); err != nil {
			return false, err
		}
	case 0x2E: // STOP
		return false, nil
	default:
		return false, fmt.Errorf("unknown pickle opcode 0x%02X at op #%d", op, w.ops)
	}
	return !w.done, nil
}

// ScanStream statically analyzes one pickle stream. A malformed input yields
// the findings collected so far plus a "malformed" finding — never an empty
// clean result for a stream that could not be fully walked.
func ScanStream(r io.Reader, lim Limits) []Finding {
	w := &walker{r: bufio.NewReaderSize(r, 64<<10), lim: lim}
	for {
		more, err := w.step()
		if err != nil {
			w.add(Finding{
				Rule:     "pickle.malformed",
				Severity: SeverityWarning,
				Detail:   fmt.Sprintf("stream could not be fully walked (%v); treated as unsafe-to-assume-clean", err),
			})
			break
		}
		if !more {
			break
		}
	}
	return w.find
}

// ScanModelFile analyzes a model file: a raw pickle stream (.pkl/.pickle) or a
// zip container (.pt/.pth/.bin/.ckpt) whose data.pkl members are scanned.
// Neither path ever deserializes objects or imports referenced modules.
func ScanModelFile(rs io.ReadSeeker, size int64, lim Limits) []Finding {
	head := make([]byte, 4)
	n, _ := rs.Read(head)
	if n >= 2 && head[0] == 'P' && head[1] == 'K' {
		_, _ = rs.Seek(0, io.SeekStart)
		return scanZipContainer(rs, size, lim)
	}
	_, _ = rs.Seek(0, io.SeekStart)
	if n > 0 && head[0] == 0x80 {
		return ScanStream(rs, lim)
	}
	// Not a pickle and not a zip: nothing to analyze statically.
	return []Finding{{Rule: "pickle.not-a-pickle", Severity: SeverityInfo,
		Detail: "file does not start with a pickle PROTO or zip magic; pickle analysis skipped"}}
}

// scanZipContainer walks zip members and scans pickle members (torch saves
// weights as a zip whose data.pkl drives reconstruction).
func scanZipContainer(rs io.ReadSeeker, size int64, lim Limits) []Finding {
	zr, err := newZipReader(rs, size)
	if err != nil {
		return []Finding{{Rule: "pickle.zip-unreadable", Severity: SeverityWarning,
			Detail: fmt.Sprintf("zip container could not be opened: %v", err)}}
	}
	var findings []Finding
	for _, zf := range zr.files() {
		name := strings.ToLower(zf.name())
		if !strings.HasSuffix(name, ".pkl") && !strings.HasSuffix(name, ".pickle") {
			continue
		}
		rc, err := zf.open()
		if err != nil {
			findings = append(findings, Finding{Rule: "pickle.zip-member-unreadable",
				Severity: SeverityWarning, Detail: fmt.Sprintf("member %s: %v", zf.name(), err)})
			continue
		}
		findings = append(findings, ScanStream(rc, lim)...)
		_ = rc.Close()
	}
	if len(findings) == 0 {
		findings = append(findings, Finding{Rule: "pickle.zip-no-pickle-member",
			Severity: SeverityInfo, Detail: "zip container has no .pkl members"})
	}
	return findings
}
