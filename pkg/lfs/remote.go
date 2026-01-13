package lfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles fetching LFS objects from remote Git LFS servers
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new RemoteClient for fetching LFS objects
func NewClient(httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
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

// getLFSEndpoint derives the LFS endpoint from a Git repository URL
func getLFSEndpoint(repoURL string) string {
	// Remove .git suffix if present and add /info/lfs
	endpoint := strings.TrimSuffix(repoURL, "/")
	if !strings.HasSuffix(endpoint, ".git") {
		endpoint += ".git"
	}
	return endpoint + "/info/lfs/objects/batch"
}

// GetBatch requests download URLs for LFS objects using the batch API
func (c *Client) GetBatch(ctx context.Context, lfsEndpoint string, objects []LFSObject) (*BatchResponse, error) {
	if len(objects) == 0 {
		return &BatchResponse{}, nil
	}

	batchURL := getLFSEndpoint(lfsEndpoint)

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

func (a Action) Request(ctx context.Context) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.Href, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range a.Header {
		req.Header.Set(key, value)
	}

	return req, nil
}
