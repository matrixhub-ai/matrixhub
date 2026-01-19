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

package backend

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParseDefaultBranchFromPktLine(t *testing.T) {
	tests := []struct {
		name           string
		buildResponse  func() []byte
		expectedBranch string
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid response with main branch",
			buildResponse: func() []byte {
				return buildPktLineResponse("abc123def456789012345678901234567890abcd", "refs/heads/main", "main")
			},
			expectedBranch: "main",
			expectError:    false,
		},
		{
			name: "valid response with master branch",
			buildResponse: func() []byte {
				return buildPktLineResponse("abc123def456789012345678901234567890abcd", "refs/heads/master", "master")
			},
			expectedBranch: "master",
			expectError:    false,
		},
		{
			name: "valid response with develop branch",
			buildResponse: func() []byte {
				return buildPktLineResponse("abc123def456789012345678901234567890abcd", "refs/heads/develop", "develop")
			},
			expectedBranch: "develop",
			expectError:    false,
		},
		{
			name: "response with multiple capabilities",
			buildResponse: func() []byte {
				return buildPktLineResponseWithCaps(
					"abc123def456789012345678901234567890abcd",
					"refs/heads/main",
					"agent=git/2.34.1 symref=HEAD:refs/heads/main filter object-format=sha1",
				)
			},
			expectedBranch: "main",
			expectError:    false,
		},
		{
			name: "empty response",
			buildResponse: func() []byte {
				return []byte{}
			},
			expectError:   true,
			errorContains: "failed to read",
		},
		{
			name: "missing symref capability",
			buildResponse: func() []byte {
				return buildPktLineResponseWithCaps(
					"abc123def456789012345678901234567890abcd",
					"refs/heads/main",
					"agent=git/2.34.1 filter object-format=sha1",
				)
			},
			expectError:   true,
			errorContains: "could not determine default branch",
		},
		{
			name: "empty repository (no refs)",
			buildResponse: func() []byte {
				// Service announcement + flush + flush (no refs)
				service := "001e# service=git-upload-pack\n"
				flush := "0000"
				return []byte(service + flush + flush)
			},
			expectError:   true,
			errorContains: "empty repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.buildResponse()
			reader := bytes.NewReader(data)

			branch, err := parseDefaultBranchFromPktLine(reader)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if branch != tt.expectedBranch {
				t.Errorf("expected branch %q, got %q", tt.expectedBranch, branch)
			}
		})
	}
}

// buildPktLineResponse creates a valid pkt-line response with the given SHA, ref, and default branch.
func buildPktLineResponse(sha, ref, defaultBranch string) []byte {
	caps := fmt.Sprintf("symref=HEAD:refs/heads/%s", defaultBranch)
	return buildPktLineResponseWithCaps(sha, ref, caps)
}

// pktLineHeaderSize is the size of the pkt-line length header in bytes (4 hex characters).
const pktLineHeaderSize = 4

// buildPktLineResponseWithCaps creates a valid pkt-line response with the given SHA, ref, and capabilities.
func buildPktLineResponseWithCaps(sha, ref, caps string) []byte {
	// Service announcement
	service := "001e# service=git-upload-pack\n"
	flush := "0000"

	// First ref with capabilities (NUL-separated)
	line := fmt.Sprintf("%s %s\x00%s\n", sha, ref, caps)
	linePkt := fmt.Sprintf("%04x%s", len(line)+pktLineHeaderSize, line)

	// HEAD ref
	headLine := fmt.Sprintf("%s HEAD\n", sha)
	headPkt := fmt.Sprintf("%04x%s", len(headLine)+pktLineHeaderSize, headLine)

	return []byte(service + flush + linePkt + headPkt + flush)
}
