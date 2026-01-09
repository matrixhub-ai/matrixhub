package lfs

import (
	"strings"
	"testing"
)

func TestIsLFSPointerContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "valid LFS pointer",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
size 1024
`,
			expected: true,
		},
		{
			name: "valid LFS pointer v2",
			content: `version https://git-lfs.github.com/spec/v2
oid sha256:abc123
size 512
`,
			expected: true,
		},
		{
			name:     "not an LFS pointer - missing version",
			content:  "oid sha256:abc123\nsize 1024\n",
			expected: false,
		},
		{
			name:     "not an LFS pointer - missing oid",
			content:  "version https://git-lfs.github.com/spec/v1\nsize 1024\n",
			expected: false,
		},
		{
			name:     "not an LFS pointer - regular text",
			content:  "Hello, world!",
			expected: false,
		},
		{
			name:     "not an LFS pointer - large content",
			content:  strings.Repeat("a", 2000),
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLFSPointerContent([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("IsLFSPointerContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParsePointer(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectOid   string
		expectSize  int64
		expectError bool
	}{
		{
			name: "valid LFS pointer",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
size 1024
`,
			expectOid:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectSize:  1024,
			expectError: false,
		},
		{
			name:        "invalid pointer - not LFS format",
			content:     "Hello, world!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr, err := ParsePointer(strings.NewReader(tt.content))
			if tt.expectError {
				if err == nil {
					t.Errorf("ParsePointer() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ParsePointer() unexpected error: %v", err)
			}

			if ptr.Oid != tt.expectOid {
				t.Errorf("ParsePointer() Oid = %q, want %q", ptr.Oid, tt.expectOid)
			}

			if ptr.Size != tt.expectSize {
				t.Errorf("ParsePointer() Size = %d, want %d", ptr.Size, tt.expectSize)
			}
		})
	}
}
