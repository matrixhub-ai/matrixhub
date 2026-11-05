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

package pickle

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

func hasRule(fs []Finding, rule string) bool {
	for _, f := range fs {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func hasRef(fs []Finding, ref string) bool {
	for _, f := range fs {
		if f.GlobalRef == ref {
			return true
		}
	}
	return false
}

func maxSeverity(fs []Finding) string {
	sev := SeverityInfo
	for _, f := range fs {
		if f.Severity == SeverityCritical {
			return SeverityCritical
		}
		if f.Severity == SeverityWarning {
			sev = SeverityWarning
		}
	}
	return sev
}

// --- crafted pickle streams (STATIC SAMPLES: they are never unpickled by the
// test — only walked as opcode bytes; nothing executes) ---

// pickleGlobal builds a protocol-2 stream with a GLOBAL + REDUCE payload.
func pickleGlobal(module, name string) []byte {
	var b bytes.Buffer
	b.WriteByte(0x80)
	b.WriteByte(2)
	b.WriteByte('c') // GLOBAL
	b.WriteString(module)
	b.WriteByte('\n')
	b.WriteString(name)
	b.WriteByte('\n')
	b.WriteByte(')') // EMPTY_TUPLE (args)
	b.WriteByte('R') // REDUCE
	b.WriteByte('.') // STOP
	return b.Bytes()
}

// pickleStackGlobal builds a protocol-4 STACK_GLOBAL payload (like
// picklescan's evil.py sample).
func pickleStackGlobal(module, name string) []byte {
	var b bytes.Buffer
	b.WriteByte(0x80)
	b.WriteByte(4)
	b.WriteByte(0x95) // FRAME
	_ = binary.Write(&b, binary.LittleEndian, uint64(0))
	writeStr := func(s string) {
		b.WriteByte(0x8c) // SHORT_BINUNICODE
		b.WriteByte(byte(len(s)))
		b.WriteString(s)
	}
	writeStr(module)
	writeStr(name)
	b.WriteByte(0x93) // STACK_GLOBAL
	b.WriteByte(')')
	b.WriteByte('R') // REDUCE
	b.WriteByte('.')
	return b.Bytes()
}

func TestScanStreamSafeTorchPickle(t *testing.T) {
	// torch.save of a plain tensor references torch._utils._rebuild_tensor_v2
	fs := ScanStream(bytes.NewReader(pickleGlobal("torch._utils", "_rebuild_tensor_v2")), DefaultLimits())
	if maxSeverity(fs) != SeverityInfo {
		t.Fatalf("safe torch pickle flagged: %+v", fs)
	}
}

func TestScanStreamDangerousOsSystem(t *testing.T) {
	fs := ScanStream(bytes.NewReader(pickleGlobal("os", "system")), DefaultLimits())
	if !hasRule(fs, "pickle.os-global") || !hasRef(fs, "os.system") {
		t.Fatalf("os.system not detected critical: %+v", fs)
	}
	if maxSeverity(fs) != SeverityCritical {
		t.Fatalf("expected critical, got %s", maxSeverity(fs))
	}
}

func TestScanStreamBuiltinsEval(t *testing.T) {
	fs := ScanStream(bytes.NewReader(pickleGlobal("builtins", "eval")), DefaultLimits())
	if !hasRule(fs, "pickle.builtins-eval") {
		t.Fatalf("builtins.eval not detected: %+v", fs)
	}
}

func TestScanStreamBuiltinsSafeName(t *testing.T) {
	fs := ScanStream(bytes.NewReader(pickleGlobal("builtins", "complex")), DefaultLimits())
	if maxSeverity(fs) != SeverityInfo {
		t.Fatalf("builtins.complex should be clean: %+v", fs)
	}
}

func TestScanStreamStackGlobalProtocol4(t *testing.T) {
	fs := ScanStream(bytes.NewReader(pickleStackGlobal("os", "system")), DefaultLimits())
	if !hasRef(fs, "os.system") || maxSeverity(fs) != SeverityCritical {
		t.Fatalf("STACK_GLOBAL os.system not detected: %+v", fs)
	}
}

func TestScanStreamSubprocess(t *testing.T) {
	fs := ScanStream(bytes.NewReader(pickleStackGlobal("subprocess", "Popen")), DefaultLimits())
	if maxSeverity(fs) != SeverityCritical {
		t.Fatalf("subprocess.Popen not critical: %+v", fs)
	}
}

func TestScanStreamUnknownModuleWarns(t *testing.T) {
	fs := ScanStream(bytes.NewReader(pickleGlobal("mycorp.loader", "rebuild")), DefaultLimits())
	if !hasRule(fs, "pickle.unknown-global") || maxSeverity(fs) != SeverityWarning {
		t.Fatalf("unknown import should warn: %+v", fs)
	}
}

func TestScanStreamTruncated(t *testing.T) {
	full := pickleGlobal("torch._utils", "_rebuild_tensor_v2")
	fs := ScanStream(bytes.NewReader(full[:len(full)-3]), DefaultLimits())
	if !hasRule(fs, "pickle.malformed") {
		t.Fatalf("truncated stream must be reported, got %+v", fs)
	}
}

func TestScanStreamGarbage(t *testing.T) {
	fs := ScanStream(strings.NewReader("this is not a pickle at all"), DefaultLimits())
	if len(fs) == 0 {
		t.Fatal("garbage input must not look clean")
	}
}

func TestScanStreamOpcodeBudget(t *testing.T) {
	var b bytes.Buffer
	b.WriteByte(0x80)
	b.WriteByte(2)
	for i := 0; i < 200; i++ {
		b.WriteByte(0x88) // NEWTRUE pushes; also grows the stack — budget hits first
	}
	lim := DefaultLimits()
	lim.MaxOpcodes = 50
	lim.MaxStackDepth = 10_000_000 // let the opcode budget be the binding cap
	fs := ScanStream(bytes.NewReader(b.Bytes()), lim)
	if !hasRule(fs, "pickle.opcode-budget-exceeded") {
		t.Fatalf("budget not enforced: %+v", fs)
	}
}

func TestScanStreamStackDepthBomb(t *testing.T) {
	var b bytes.Buffer
	b.WriteByte(0x80)
	b.WriteByte(2)
	for i := 0; i < 5_000; i++ {
		b.WriteByte(0x88)
	}
	lim := DefaultLimits()
	lim.MaxOpcodes = 10_000_000
	fs := ScanStream(bytes.NewReader(b.Bytes()), lim)
	if !hasRule(fs, "pickle.malformed") {
		t.Fatalf("stack bomb should end as malformed/depth, got %+v", fs)
	}
}

// --- zip container (.pt/.pth) handling ---

func buildTorchZip(t *testing.T, picklePayload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("data.pkl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(picklePayload); err != nil {
		t.Fatal(err)
	}
	if _, err := zw.Create("version"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestScanModelFileZipWithMaliciousPickle(t *testing.T) {
	data := buildTorchZip(t, pickleGlobal("os", "system"))
	fs := ScanModelFile(bytes.NewReader(data), int64(len(data)), DefaultLimits())
	if maxSeverity(fs) != SeverityCritical || !hasRef(fs, "os.system") {
		t.Fatalf("zip data.pkl os.system not detected: %+v", fs)
	}
}

func TestScanModelFileZipSafe(t *testing.T) {
	data := buildTorchZip(t, pickleGlobal("torch._utils", "_rebuild_tensor_v2"))
	fs := ScanModelFile(bytes.NewReader(data), int64(len(data)), DefaultLimits())
	if maxSeverity(fs) != SeverityInfo {
		t.Fatalf("safe torch zip flagged: %+v", fs)
	}
}

func TestScanModelFileRawPickleAndUnknown(t *testing.T) {
	raw := pickleStackGlobal("builtins", "exec")
	fs := ScanModelFile(bytes.NewReader(raw), int64(len(raw)), DefaultLimits())
	if !hasRule(fs, "pickle.builtins-exec") {
		t.Fatalf("raw pickle exec not detected: %+v", fs)
	}
	other := []byte{0x00, 0x01, 0x02, 0x03}
	fs = ScanModelFile(bytes.NewReader(other), 4, DefaultLimits())
	if !hasRule(fs, "pickle.not-a-pickle") {
		t.Fatalf("non-pickle should be info-skip: %+v", fs)
	}
}

func TestScanModelFileSeekerWithoutReaderAt(t *testing.T) {
	data := buildTorchZip(t, pickleGlobal("posix", "system"))
	rs := struct{ io.ReadSeeker }{bytes.NewReader(data)} // hides ReadAt
	fs := ScanModelFile(rs, int64(len(data)), DefaultLimits())
	if maxSeverity(fs) != SeverityCritical {
		t.Fatalf("ReadSeeker-only zip path failed: %+v", fs)
	}
}

func TestPayloadBytesNeverInFindings(t *testing.T) {
	// The report must never echo payload bytes; findings only carry rules/refs.
	evil := pickleGlobal("os", "system")
	fs := ScanStream(bytes.NewReader(evil), DefaultLimits())
	joined := ""
	for _, f := range fs {
		joined += f.Rule + f.Detail + f.GlobalRef
	}
	// the raw opcode string must not appear verbatim in any finding text
	if strings.Contains(joined, "\x80\x02cos\nsystem\n") {
		t.Fatal("finding text leaks raw payload bytes")
	}
}
