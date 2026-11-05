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

package scan

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/scan/scanner/clamav"
	"github.com/matrixhub-ai/matrixhub/internal/domain/scan/scanner/heuristic"
	"github.com/matrixhub-ai/matrixhub/internal/domain/scan/scanner/pickle"
)

func init() {
	// Wire the shared file-type identification to the heuristic scanner's
	// implementation so every component types files identically.
	DetectFileType = heuristic.DetectFileType
}

// pickleScanner adapts the static pickle walker to the Scanner port.
type pickleScanner struct {
	lim pickle.Limits
}

// NewPickleScanner returns the dangerous-serialization static analyzer.
func NewPickleScanner() Scanner {
	return &pickleScanner{lim: pickle.DefaultLimits()}
}

func (p *pickleScanner) ID() string { return "pickle-static" }

func (p *pickleScanner) Version(_ context.Context) (string, error) {
	return "opcode-walker/1", nil
}

func (p *pickleScanner) Applicable(name, fileType string, _ int64) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".pkl", ".pickle", ".pt", ".pth", ".bin", ".ckpt", ".h5", ".joblib":
		return true
	}
	return fileType == "pickle" || fileType == "zip container" || fileType == "serialized model"
}

func (p *pickleScanner) ScanFile(_ context.Context, name, _ string, size int64, r io.ReadSeeker) ([]FileFinding, error) {
	fs := pickle.ScanModelFile(r, size, p.lim)
	out := make([]FileFinding, 0, len(fs))
	for _, f := range fs {
		out = append(out, FileFinding{Severity: f.Severity, Rule: f.Rule, Detail: f.Detail})
	}
	return out, nil
}

// heuristicScanner adapts the resource-safety checks.
type heuristicScanner struct {
	lim heuristic.Limits
}

// NewHeuristicScanner returns the compression-bomb / typing scanner.
func NewHeuristicScanner() Scanner {
	return &heuristicScanner{lim: heuristic.DefaultLimits()}
}

func (h *heuristicScanner) ID() string { return "heuristics" }

func (h *heuristicScanner) Version(_ context.Context) (string, error) {
	return "heur/1", nil
}

func (h *heuristicScanner) Applicable(_, fileType string, _ int64) bool {
	switch fileType {
	case "zip container", "zip (numpy npz)", "gzip", "bzip2", "7z", "tar", "zstd":
		return true
	}
	return false
}

func (h *heuristicScanner) ScanFile(_ context.Context, name, _ string, size int64, r io.ReadSeeker) ([]FileFinding, error) {
	fs := heuristic.Scan(r, name, size, h.lim)
	out := make([]FileFinding, 0, len(fs))
	for _, f := range fs {
		out = append(out, FileFinding{Severity: f.Severity, Rule: f.Rule, Detail: f.Detail})
	}
	return out, nil
}

// clamAVScanner adapts the clamd client. Unavailability surfaces as an error
// so the task fails instead of reporting a false clean.
type clamAVScanner struct {
	client *clamav.Client
}

// NewClamAVScanner returns the generic malware scanner backed by clamd.
func NewClamAVScanner(cfg clamav.Config) Scanner {
	return &clamAVScanner{client: clamav.NewClient(cfg)}
}

func (c *clamAVScanner) ID() string { return "clamav" }

func (c *clamAVScanner) Version(ctx context.Context) (string, error) {
	v, err := c.client.Version()
	if err != nil {
		return "", err
	}
	return v, nil
}

func (c *clamAVScanner) Applicable(_, _ string, size int64) bool {
	return size <= c.client.MaxScanBytes()
}

func (c *clamAVScanner) ScanFile(ctx context.Context, name, _ string, size int64, r io.ReadSeeker) ([]FileFinding, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}
	res, err := c.client.ScanStream(r)
	if err != nil {
		return nil, err
	}
	if res.Clean {
		return nil, nil
	}
	return []FileFinding{{
		Severity: SevCritical,
		Rule:     "clamav." + res.Signature,
		Detail:   fmt.Sprintf("ClamAV signature %s matched; the file is presumed malicious", res.Signature),
	}}, nil
}
