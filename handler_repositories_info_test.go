package gitd

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

func TestDetectLFSPointer(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedIsLFS bool
		expectedSHA   string
	}{
		{
			name: "Valid LFS pointer",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393
size 12345
`,
			expectedIsLFS: true,
			expectedSHA:   "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393",
		},
		{
			name: "Valid LFS pointer with different format",
			content: `version https://git-lfs.github.com/spec/v1
oid sha256:abc123def456
size 999
`,
			expectedIsLFS: true,
			expectedSHA:   "abc123def456",
		},
		{
			name:          "Regular text file",
			content:       "This is just a regular text file.\nNothing special here.",
			expectedIsLFS: false,
			expectedSHA:   "",
		},
		{
			name:          "Empty file",
			content:       "",
			expectedIsLFS: false,
			expectedSHA:   "",
		},
		{
			name: "File with version but no oid",
			content: `version https://git-lfs.github.com/spec/v1
size 12345
`,
			expectedIsLFS: false,
			expectedSHA:   "",
		},
		{
			name: "File with oid but no version",
			content: `oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393
size 12345
`,
			expectedIsLFS: false,
			expectedSHA:   "",
		},
		{
			name: "Large file (not LFS pointer)",
			content: `This is a large file that contains version https://git-lfs.github.com/spec/v1
and oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393
but it's too large to be a pointer.` + string(make([]byte, 2000)),
			expectedIsLFS: false,
			expectedSHA:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create an in-memory storage
			storer := memory.NewStorage()

			// Create a blob with the test content
			blob := &object.Blob{}
			blob.Hash = plumbing.NewHash("0000000000000000000000000000000000000000")
			blob.Size = int64(len(tt.content))
			
			// Store the blob
			obj := storer.NewEncodedObject()
			obj.SetType(plumbing.BlobObject)
			obj.SetSize(int64(len(tt.content)))
			w, err := obj.Writer()
			if err != nil {
				t.Fatalf("Failed to get writer: %v", err)
			}
			_, err = w.Write([]byte(tt.content))
			if err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			w.Close()
			
			hash, err := storer.SetEncodedObject(obj)
			if err != nil {
				t.Fatalf("Failed to store object: %v", err)
			}

			// Get the blob back
			blob, err = object.GetBlob(storer, hash)
			if err != nil {
				t.Fatalf("Failed to get blob: %v", err)
			}

			// Test detectLFSPointer
			isLFS, sha := detectLFSPointer(blob)

			if isLFS != tt.expectedIsLFS {
				t.Errorf("Expected isLFS=%v, got %v", tt.expectedIsLFS, isLFS)
			}

			if sha != tt.expectedSHA {
				t.Errorf("Expected SHA=%q, got %q", tt.expectedSHA, sha)
			}
		})
	}
}
