package lfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/wzshiming/httpseek"
)

// RemoteClient handles fetching LFS objects from remote Git LFS servers
type RemoteClient struct {
	httpClient *http.Client
}

// NewRemoteClient creates a new RemoteClient for fetching LFS objects
func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		httpClient: &http.Client{
			Transport: httpseek.NewMustReaderTransport(http.DefaultTransport,
				func(r *http.Request, retry int, err error) error {
					log.Printf("Retry %d for %s due to error: %v\n", retry+1, r.URL.String(), err)
					if retry >= 5 {
						return fmt.Errorf("max retries reached for %s: %w", r.URL.String(), err)
					}
					// Simple backoff strategy
					time.Sleep(time.Duration(retry+1) * time.Second)
					return nil
				}),
			Timeout: 30 * time.Minute, // Long timeout for large files
		},
	}
}

// BatchRequest represents a request to the LFS batch API
type BatchRequest struct {
	Operation string        `json:"operation"`
	Transfers []string      `json:"transfers,omitempty"`
	Objects   []BatchObject `json:"objects"`
}

// BatchObject represents an object in a batch request
type BatchObject struct {
	Oid  string `json:"oid"`
	Size int64  `json:"size"`
}

// BatchResponse represents a response from the LFS batch API
type BatchResponse struct {
	Transfer string                `json:"transfer,omitempty"`
	Objects  []BatchResponseObject `json:"objects"`
}

// BatchResponseObject represents an object in a batch response
type BatchResponseObject struct {
	Oid           string            `json:"oid"`
	Size          int64             `json:"size"`
	Authenticated bool              `json:"authenticated,omitempty"`
	Actions       map[string]Action `json:"actions,omitempty"`
	Error         *ObjectError      `json:"error,omitempty"`
}

// Action represents an action in a batch response
type Action struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in,omitempty"`
	ExpiresAt time.Time         `json:"expires_at,omitempty"`
}

// ObjectError represents an error for an object in a batch response
type ObjectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LFSObject represents an LFS object to be fetched
type LFSObject struct {
	Oid  string
	Size int64
}

// GetLFSEndpoint derives the LFS endpoint from a Git repository URL
func GetLFSEndpoint(repoURL string) string {
	// Remove .git suffix if present and add /info/lfs
	endpoint := strings.TrimSuffix(repoURL, "/")
	if !strings.HasSuffix(endpoint, ".git") {
		endpoint += ".git"
	}
	return endpoint + "/info/lfs"
}

// BatchDownload requests download URLs for LFS objects using the batch API
func (c *RemoteClient) BatchDownload(ctx context.Context, lfsEndpoint string, objects []LFSObject) (*BatchResponse, error) {
	if len(objects) == 0 {
		return &BatchResponse{}, nil
	}

	batchURL := lfsEndpoint + "/objects/batch"

	batchObjects := make([]BatchObject, len(objects))
	for i, obj := range objects {
		batchObjects[i] = BatchObject{
			Oid:  obj.Oid,
			Size: obj.Size,
		}
	}

	reqBody := BatchRequest{
		Operation: "download",
		Transfers: []string{"basic"},
		Objects:   batchObjects,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", batchURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create batch request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Accept", "application/vnd.git-lfs+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute batch request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode batch response: %w", err)
	}

	return &batchResp, nil
}

// DownloadObject downloads an LFS object from the given action
func (c *RemoteClient) DownloadObject(ctx context.Context, action *Action) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, action.Href, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	for key, value := range action.Header {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// FetchAndStore fetches missing LFS objects and stores them in the content store
func (c *RemoteClient) FetchAndStore(ctx context.Context, lfsEndpoint string, objects []LFSObject, store *Content) error {
	if len(objects) == 0 {
		return nil
	}

	// Filter out objects that already exist
	var missing []LFSObject
	for _, obj := range objects {
		if !store.Exists(obj.Oid) {
			missing = append(missing, obj)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	// Request download URLs for missing objects
	batchResp, err := c.BatchDownload(ctx, lfsEndpoint, missing)
	if err != nil {
		return fmt.Errorf("batch download request failed: %w", err)
	}

	// Download and store each object
	for _, obj := range batchResp.Objects {
		if obj.Error != nil {
			log.Printf("Skipping object %s due to error: %s\n", obj.Oid, obj.Error.Message)
			continue
		}

		downloadAction, ok := obj.Actions["download"]
		if !ok {
			continue
		}

		log.Printf("Downloading LFS object %s\n", obj.Oid)
		body, err := c.DownloadObject(ctx, &downloadAction)
		if err != nil {
			log.Printf("Failed to download LFS object %s: %v\n", obj.Oid, err)
			continue
		}

		err = store.Put(obj.Oid, body, obj.Size)
		body.Close()
		if err != nil {
			log.Printf("Failed to store LFS object %s: %v\n", obj.Oid, err)
			continue
		}
	}

	return nil
}
