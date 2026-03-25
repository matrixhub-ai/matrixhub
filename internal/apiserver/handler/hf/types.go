package hf

import (
	"encoding/json"

	"github.com/matrixhub-ai/hfd/pkg/repository"
)

// whoamiResponse represents the response for the /api/whoami-v2 endpoint.
type whoamiResponse struct {
	Type          string   `json:"type"`
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Fullname      string   `json:"fullname"`
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"emailVerified"`
	IsPro         bool     `json:"isPro"`
	CanPay        bool     `json:"canPay"`
	AvatarURL     string   `json:"avatarUrl,omitempty"`
	Orgs          []any    `json:"orgs"`
	Auth          authInfo `json:"auth"`
}

// authInfo represents the auth section of the whoami response.
type authInfo struct {
	AccessToken accessToken `json:"accessToken"`
}

// accessToken represents the access token info in the whoami response.
type accessToken struct {
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

// treeEntry is the API response type for a tree entry, with JSON annotations.
type treeEntry struct {
	OID        string               `json:"oid"`
	Path       string               `json:"path"`
	Type       repository.EntryType `json:"type"`
	Size       int64                `json:"size"`
	LFS        *lfsPointer          `json:"lfs,omitempty"`
	LastCommit *treeLastCommit      `json:"lastCommit,omitempty"`
}

// lfsPointer is the API response type for an LFS pointer, with JSON annotations.
type lfsPointer struct {
	OID         string `json:"oid"`
	Size        int64  `json:"size"`
	PointerSize int64  `json:"pointerSize"`
}

// treeLastCommit is the API response type for the last commit of a tree entry, with JSON annotations.
type treeLastCommit struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Date  string `json:"date"`
}

// treeSize represents the response for the Get folder size API.
type treeSize struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// repoListItem represents a repository item in the list response for models, datasets, and spaces.
type repoListItem struct {
	RepoID        string   `json:"id"`
	Likes         int      `json:"likes"`
	TrendingScore int      `json:"trendingScore"`
	Private       bool     `json:"private"`
	Downloads     int      `json:"downloads"`
	Tags          []string `json:"tags,omitempty"`
	PipelineTag   string   `json:"pipeline_tag,omitempty"`
	LibraryName   string   `json:"library_name,omitempty"`
	ModelID       string   `json:"modelId,omitempty"`
}

// repoInfo represents the info response for HuggingFace API
type repoInfo struct {
	ID           string    `json:"id"`
	ModelID      string    `json:"modelId,omitempty"`
	SHA          string    `json:"sha"`
	Private      bool      `json:"private"`
	Disabled     bool      `json:"disabled"`
	Gated        bool      `json:"gated"`
	Downloads    int       `json:"downloads"`
	Likes        int       `json:"likes"`
	Tags         []string  `json:"tags"` // This is not git tags, but the tags in HuggingFace card metadata
	CardData     any       `json:"cardData,omitempty"`
	Siblings     []sibling `json:"siblings"`
	CreatedAt    string    `json:"createdAt,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	UsedStorage  int64     `json:"usedStorage"`
}

// sibling represents a file in the model repository
type sibling struct {
	RFilename string `json:"rfilename"`
}

// deleteRepoRequest represents the delete repo request body.
type deleteRepoRequest struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
}

// moveRepoRequest represents the move repo request body.
type moveRepoRequest struct {
	FromRepo string `json:"fromRepo"`
	ToRepo   string `json:"toRepo"`
	Type     string `json:"type"`
}

// repoSettingsRequest represents the repo settings update request body.
type repoSettingsRequest struct {
	Private *bool `json:"private,omitempty"`
	Gated   any   `json:"gated,omitempty"`
}

// createBranchRequest represents the create branch request body.
type createBranchRequest struct {
	StartingPoint string `json:"startingPoint,omitempty"`
}

// createTagRequest represents the create tag request body.
type createTagRequest struct {
	Tag     string `json:"tag"`
	Message string `json:"message,omitempty"`
}

// gitRefInfo represents a single git ref (branch or tag).
type gitRefInfo struct {
	Name         string `json:"name"`
	Ref          string `json:"ref"`
	TargetCommit string `json:"targetCommit"`
}

// gitRefs represents the response for listing repo refs.
type gitRefs struct {
	Branches []gitRefInfo `json:"branches"`
	Converts []gitRefInfo `json:"converts"`
	Tags     []gitRefInfo `json:"tags"`
}

// commitAuthor represents the author entry in a commit's authors list.
type commitAuthor struct {
	User string `json:"user"`
}

// commitInfo represents a single commit in the list commits response.
type commitInfo struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Message string         `json:"message"`
	Authors []commitAuthor `json:"authors"`
	Date    string         `json:"date"`
}

// superSquashRequest represents the super-squash request body.
type superSquashRequest struct {
	Message string `json:"message,omitempty"`
}

// preuploadRequest represents the preupload request body.
type preuploadRequest struct {
	Files []preuploadFile `json:"files"`
}

// preuploadFile represents a file in the preupload request.
type preuploadFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Sample string `json:"sample"`
}

// preuploadResponse represents the preupload response body.
type preuploadResponse struct {
	Files []preuploadResponseFile `json:"files"`
}

// preuploadResponseFile represents a file in the preupload response.
type preuploadResponseFile struct {
	Path         string `json:"path"`
	UploadMode   string `json:"uploadMode"`
	ShouldIgnore bool   `json:"shouldIgnore"`
}

// createRepoRequest represents the create repo request body.
type createRepoRequest struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	Organization string `json:"organization,omitempty"`
	Private      bool   `json:"private"`
}

// createRepoResponse represents the create repo response body.
type createRepoResponse struct {
	URL string `json:"url"`
}

// commitResponse represents the commit response body.
type commitResponse struct {
	CommitURL     string `json:"commitUrl"`
	CommitOid     string `json:"commitOid"`
	CommitMessage string `json:"commitMessage"`
}

// commitOperation represents a single operation in the NDJSON commit request.
type commitOperation struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// commitHeader represents the header operation in the commit request.
type commitHeader struct {
	Summary      string `json:"summary"`
	Description  string `json:"description"`
	ParentCommit string `json:"parentCommit"`
}

// commitFile represents a regular file operation in the commit request.
type commitFile struct {
	Content  string `json:"content"`
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
}

// commitLFSFile represents an LFS file operation in the commit request.
type commitLFSFile struct {
	Path string `json:"path"`
	Algo string `json:"algo"`
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

// commitDeletedFile represents a delete file operation in the commit request.
type commitDeletedFile struct {
	Path string `json:"path"`
}
