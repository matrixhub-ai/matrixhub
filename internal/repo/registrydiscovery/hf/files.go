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

package hf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

// maxTreePages bounds tree pagination so a misbehaving registry cannot keep a
// listing running forever.
const maxTreePages = 100

// maxRevisionFiles bounds the files of one listing, so a misbehaving registry
// cannot fill the server's memory with pages of tiny entries. A tree page of
// huggingface.co holds up to 1,000 entries, so maxTreePages pages hold as
// many files.
const maxRevisionFiles = maxTreePages * 1000

// maxResponseBytes bounds one response body. A tree page of huggingface.co
// holds up to 1,000 entries, a few hundred kilobytes.
const maxResponseBytes = 32 << 20

// maxAdvertisementBytes bounds the git ref advertisement read for the name of
// the default branch. The decoder reads every ref, so the default branch of a
// repository with tens of thousands of refs goes unnamed.
const maxAdvertisementBytes = 1 << 20

// defaultBranchTimeout bounds the lookup of the default branch's name, which
// only labels a listing.
const defaultBranchTimeout = 10 * time.Second

type revisionInfo struct {
	SHA string `json:"sha"`
}

type treeItem struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	LFS  *struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	} `json:"lfs"`
}

// ListRevisionFiles resolves revision on a Hugging Face-compatible registry and
// lists every file of the resolved commit. Both huggingface.co (and mirrors)
// and MatrixHub registries serve this API.
func (d *Discovery) ListRevisionFiles(ctx context.Context, reg *registry.Registry, namespace, name, revision string) (*git.RevisionFiles, error) {
	base, err := endpointBase(reg)
	if err != nil {
		return nil, err
	}
	repoPath := url.PathEscape(namespace) + "/" + url.PathEscape(name)

	infoURL := base + "/api/models/" + repoPath
	if revision != "" {
		infoURL += "/revision/" + url.PathEscape(revision)
	}
	var info revisionInfo
	if _, _, err := d.getJSON(ctx, infoURL, &info); err != nil {
		return nil, err
	}
	if info.SHA == "" {
		return nil, fmt.Errorf("%w: %s/%s@%s has no commit", registrydiscovery.ErrRemoteNotFound, namespace, name, revision)
	}

	files := []git.RevisionFile{}
	next := base + "/api/models/" + repoPath + "/tree/" + url.PathEscape(info.SHA) + "?recursive=true"
	for page := 0; next != ""; page++ {
		if page >= maxTreePages {
			return nil, fmt.Errorf("%w: tree of %s/%s has more than %d pages", registrydiscovery.ErrRemoteTooLarge, namespace, name, maxTreePages)
		}
		var items []treeItem
		pageURL, header, err := d.getJSON(ctx, next, &items)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if it.Type != "file" {
				continue
			}
			f := git.RevisionFile{Path: it.Path, Size: it.Size}
			if it.LFS != nil && it.LFS.OID != "" {
				f.IsLFS = true
				f.OID = it.LFS.OID
				f.Size = it.LFS.Size
			}
			files = append(files, f)
		}
		if len(files) > maxRevisionFiles {
			return nil, fmt.Errorf("%w: %s/%s has more than %d files", registrydiscovery.ErrRemoteTooLarge, namespace, name, maxRevisionFiles)
		}
		if next = nextLink(header.Get("Link")); next != "" {
			// Redirects are followed, as git follows them when the proxy
			// syncs: hf-mirror.com, for example, may redirect to
			// huggingface.co, whose links then name huggingface.co. A link
			// is therefore resolved against the page that sent it, and
			// followed only on that page's origin.
			if next, err = sameOrigin(pageURL, next); err != nil {
				return nil, err
			}
		}
	}

	if revision == "" {
		revision = d.defaultBranch(ctx, base, repoPath, info.SHA)
	}
	return &git.RevisionFiles{Revision: revision, Commit: info.SHA, Files: files}, nil
}

// defaultBranch names the default branch of a repository from its git ref
// advertisement, as `git ls-remote --symref` does when the proxy creates the
// mirror. It returns "" when the registry does not tell, or when its HEAD is
// not at commit, so that the name always matches the listed files. The name
// only labels a listing, so failing to learn it fails nothing.
func (d *Discovery) defaultBranch(ctx context.Context, base, repoPath, commit string) string {
	ctx, cancel := context.WithTimeout(ctx, defaultBranchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/"+repoPath+"/info/refs?service=git-upload-pack", nil)
	if err != nil {
		return ""
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	refs := packp.NewAdvRefs()
	if err := refs.Decode(io.LimitReader(resp.Body, maxAdvertisementBytes)); err != nil {
		return ""
	}
	if refs.Head == nil || refs.Head.String() != commit {
		return ""
	}
	for _, symref := range refs.Capabilities.Get(capability.SymRef) {
		if branch, ok := strings.CutPrefix(symref, "HEAD:refs/heads/"); ok {
			return branch
		}
	}
	return ""
}

// getJSON fetches u and decodes a JSON body into out, mapping HTTP failures to
// the registrydiscovery sentinel errors. It returns the URL that answered,
// after redirects, and the response headers.
//
// Requests carry no credential. The proxy sync that a pull runs sends none
// either, so a preview never lists files that the pull cannot fetch, and a
// project permission is never enough to act with the registry's credential.
func (d *Discovery) getJSON(ctx context.Context, u string, out any) (*url.URL, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build hf request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", registrydiscovery.ErrRemoteUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Follow huggingface_hub: X-Error-Code names the failure, and a 401 on a
	// repo API means the repository does not exist or is private, unless the
	// credential itself was rejected.
	errorCode := resp.Header.Get("X-Error-Code")
	errorMessage := resp.Header.Get("X-Error-Message")
	switch {
	case resp.StatusCode == http.StatusOK:
	case errorCode == "RepoNotFound" || errorCode == "RevisionNotFound" || errorCode == "EntryNotFound":
		return nil, nil, fmt.Errorf("%w: %s (%s)", registrydiscovery.ErrRemoteNotFound, errorCode, u)
	case errorCode == "GatedRepo":
		return nil, nil, fmt.Errorf("%w: %w (%s)", registrydiscovery.ErrRemoteAccessDenied, registrydiscovery.ErrRemoteGated, u)
	case resp.StatusCode == http.StatusNotFound:
		return nil, nil, fmt.Errorf("%w: %s", registrydiscovery.ErrRemoteNotFound, u)
	case resp.StatusCode == http.StatusUnauthorized && errorMessage != "Invalid credentials in Authorization header":
		return nil, nil, fmt.Errorf("%w: repository does not exist, or is private and needs a credential (%s)", registrydiscovery.ErrRemoteNotFound, u)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, nil, fmt.Errorf("%w: status %d for %s", registrydiscovery.ErrRemoteAccessDenied, resp.StatusCode, u)
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, nil, fmt.Errorf("%w: %w%s (%s)", registrydiscovery.ErrRemoteUnavailable, registrydiscovery.ErrRemoteRateLimited, retryAfterHint(resp.Header), u)
	default:
		// Any other answer, such as a 5xx or a 4xx that the Hub API does
		// not use, means the registry cannot serve the listing now.
		return nil, nil, fmt.Errorf("%w: status %d for %s", registrydiscovery.ErrRemoteUnavailable, resp.StatusCode, u)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
		// For example the HTML error page of a proxy in front of the
		// registry.
		return nil, nil, fmt.Errorf("%w: decode response from %s: %w", registrydiscovery.ErrRemoteUnavailable, u, err)
	}
	return resp.Request.URL, resp.Header, nil
}

// sameOrigin resolves ref against the URL of the page that sent it and
// rejects targets on another origin.
func sameOrigin(page *url.URL, ref string) (string, error) {
	r, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("%w: parse pagination link %q: %w", registrydiscovery.ErrRemoteUnavailable, ref, err)
	}
	resolved := page.ResolveReference(r)
	if resolved.Scheme != page.Scheme || resolved.Host != page.Host {
		return "", fmt.Errorf("%w: pagination link %q leaves the origin %s://%s of its page", registrydiscovery.ErrRemoteUnavailable, ref, page.Scheme, page.Host)
	}
	return resolved.String(), nil
}

// nextLink extracts the rel="next" target from an RFC 8288 Link header.
func nextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		segments := strings.Split(part, ";")
		if len(segments) < 2 {
			continue
		}
		target := strings.TrimSpace(segments[0])
		if !strings.HasPrefix(target, "<") || !strings.HasSuffix(target, ">") {
			continue
		}
		for _, param := range segments[1:] {
			if strings.EqualFold(strings.ReplaceAll(strings.TrimSpace(param), " ", ""), `rel="next"`) {
				return strings.TrimSuffix(strings.TrimPrefix(target, "<"), ">")
			}
		}
	}
	return ""
}

// retryAfterHint renders a Retry-After header, in seconds or as an HTTP date.
func retryAfterHint(h http.Header) string {
	after := h.Get("Retry-After")
	if after == "" {
		return ""
	}
	if secs, err := strconv.Atoi(after); err == nil {
		return fmt.Sprintf("; retry after %ds", secs)
	}
	return "; retry after " + after
}
