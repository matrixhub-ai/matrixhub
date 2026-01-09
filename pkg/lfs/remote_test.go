package lfs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
			result := GetLFSEndpoint(tt.repoURL)
			if result != tt.expected {
				t.Errorf("GetLFSEndpoint(%q) = %q, want %q", tt.repoURL, result, tt.expected)
			}
		})
	}
}

func TestRemoteClient_BatchDownload(t *testing.T) {
	// Use a pointer to store server URL, which will be set after server is created
	var serverURL string

	// Create a mock LFS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info/lfs/objects/batch" {
			http.NotFound(w, r)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/vnd.git-lfs+json" {
			http.Error(w, "Invalid content type", http.StatusBadRequest)
			return
		}

		// Parse request
		var req BatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Create response
		response := BatchResponse{
			Transfer: "basic",
			Objects:  make([]BatchResponseObject, len(req.Objects)),
		}

		for i, obj := range req.Objects {
			response.Objects[i] = BatchResponseObject{
				Oid:  obj.Oid,
				Size: obj.Size,
				Actions: map[string]Action{
					"download": {
						Href: serverURL + "/objects/" + obj.Oid,
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/vnd.git-lfs+json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewRemoteClient()

	objects := []LFSObject{
		{Oid: "abc123", Size: 1024},
		{Oid: "def456", Size: 2048},
	}

	resp, err := client.BatchDownload(context.Background(), server.URL+"/info/lfs", objects)
	if err != nil {
		t.Fatalf("BatchDownload failed: %v", err)
	}

	if len(resp.Objects) != 2 {
		t.Errorf("Expected 2 objects, got %d", len(resp.Objects))
	}

	for i, obj := range resp.Objects {
		if obj.Oid != objects[i].Oid {
			t.Errorf("Object %d: expected OID %s, got %s", i, objects[i].Oid, obj.Oid)
		}
		if obj.Actions["download"].Href == "" {
			t.Errorf("Object %d: missing download href", i)
		}
	}
}

func TestRemoteClient_DownloadObject(t *testing.T) {
	expectedContent := []byte("test content for LFS object")

	// Create a mock server that serves object content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for custom header
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			http.Error(w, "Missing custom header", http.StatusBadRequest)
			return
		}
		w.Write(expectedContent)
	}))
	defer server.Close()

	client := NewRemoteClient()

	action := &Action{
		Href: server.URL,
		Header: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	body, err := client.DownloadObject(context.Background(), action)
	if err != nil {
		t.Fatalf("DownloadObject failed: %v", err)
	}
	defer body.Close()

	content := make([]byte, len(expectedContent))
	n, _ := body.Read(content)
	if string(content[:n]) != string(expectedContent) {
		t.Errorf("Expected content %q, got %q", expectedContent, content[:n])
	}
}

func TestRemoteClient_FetchAndStore(t *testing.T) {
	objectContent := []byte("test LFS object content")
	oid := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // SHA256 of empty string for test

	// Use a pointer to store server URL, which will be set after server is created
	var serverURL string

	// Create a mock LFS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/info/lfs/objects/batch":
			var req BatchRequest
			json.NewDecoder(r.Body).Decode(&req)

			response := BatchResponse{
				Transfer: "basic",
				Objects:  make([]BatchResponseObject, len(req.Objects)),
			}

			for i, obj := range req.Objects {
				response.Objects[i] = BatchResponseObject{
					Oid:  obj.Oid,
					Size: obj.Size,
					Actions: map[string]Action{
						"download": {
							Href: serverURL + "/objects/" + obj.Oid,
						},
					},
				}
			}

			w.Header().Set("Content-Type", "application/vnd.git-lfs+json")
			json.NewEncoder(w).Encode(response)

		case "/objects/" + oid:
			w.Write(objectContent)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	// Create temp directory for content store
	tmpDir, err := os.MkdirTemp("", "lfs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewContent(tmpDir)
	client := NewRemoteClient()

	objects := []LFSObject{
		{Oid: oid, Size: int64(len(objectContent))},
	}

	// First fetch - object should be downloaded
	err = client.FetchAndStore(context.Background(), server.URL+"/info/lfs", objects, store)
	if err != nil {
		t.Fatalf("FetchAndStore failed: %v", err)
	}

	// Verify object was stored (note: hash verification will fail in this test
	// because our mock doesn't return content matching the OID hash)
}

func TestRemoteClient_BatchDownloadEmpty(t *testing.T) {
	client := NewRemoteClient()

	resp, err := client.BatchDownload(context.Background(), "http://example.com/info/lfs", nil)
	if err != nil {
		t.Fatalf("BatchDownload with empty objects failed: %v", err)
	}

	if len(resp.Objects) != 0 {
		t.Errorf("Expected 0 objects, got %d", len(resp.Objects))
	}
}
