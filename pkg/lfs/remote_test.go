package lfs

import (
	"testing"
)

func TestGetLFSEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "basic URL without .git",
			repoURL:  "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo.git/info/lfs",
		},
		{
			name:     "URL with .git",
			repoURL:  "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git/info/lfs",
		},
		{
			name:     "URL with trailing slash",
			repoURL:  "https://github.com/owner/repo/",
			expected: "https://github.com/owner/repo.git/info/lfs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLFSEndpoint(tt.repoURL)
			if result != tt.expected {
				t.Errorf("getLFSEndpoint(%q) = %q, want %q", tt.repoURL, result, tt.expected)
			}
		})
	}
}
