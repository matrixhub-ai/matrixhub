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
	"net/http"
	"net/url"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

// maxTreePages bounds tree pagination so a misbehaving registry cannot keep a
// listing running forever.
const maxTreePages = 100

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
	if _, err := d.getJSON(ctx, reg, infoURL, &info); err != nil {
		return nil, err
	}
	if info.SHA == "" {
		return nil, fmt.Errorf("%w: %s/%s@%s has no commit", registrydiscovery.ErrRemoteNotFound, namespace, name, revision)
	}

	files := []git.RevisionFile{}
	next := base + "/api/models/" + repoPath + "/tree/" + url.PathEscape(info.SHA) + "?recursive=true"
	for page := 0; next != ""; page++ {
		if page >= maxTreePages {
			return nil, fmt.Errorf("tree of %s/%s has more than %d pages", namespace, name, maxTreePages)
		}
		var items []treeItem
		header, err := d.getJSON(ctx, reg, next, &items)
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
		if next = nextLink(header.Get("Link")); next != "" {
			// The request carries the registry credential, so only follow
			// pagination links on the registry's own origin.
			if next, err = sameOrigin(base, next); err != nil {
				return nil, err
			}
		}
	}

	return &git.RevisionFiles{Revision: revision, Commit: info.SHA, Files: files}, nil
}

// getJSON fetches u and decodes a JSON body into out, mapping HTTP failures to
// the registrydiscovery sentinel errors.
func (d *Discovery) getJSON(ctx context.Context, reg *registry.Registry, u string, out any) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build hf request: %w", err)
	}
	if basic := registry.AsBasic(reg.GetCredential()); basic != nil {
		req.Header.Set("Authorization", "Bearer "+basic.Password)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", registrydiscovery.ErrRemoteUnavailable, err)
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
		return nil, fmt.Errorf("%w: %s (%s)", registrydiscovery.ErrRemoteNotFound, errorCode, u)
	case errorCode == "GatedRepo":
		return nil, fmt.Errorf("%w: gated repository needs an authorized credential (%s)", registrydiscovery.ErrRemoteAccessDenied, u)
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: %s", registrydiscovery.ErrRemoteNotFound, u)
	case resp.StatusCode == http.StatusUnauthorized && errorMessage != "Invalid credentials in Authorization header":
		return nil, fmt.Errorf("%w: repository does not exist, or is private and needs a credential (%s)", registrydiscovery.ErrRemoteNotFound, u)
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: status %d for %s", registrydiscovery.ErrRemoteAccessDenied, resp.StatusCode, u)
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, fmt.Errorf("%w: status %d for %s", registrydiscovery.ErrRemoteUnavailable, resp.StatusCode, u)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("hf request %s returned status %d", u, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decode hf response from %s: %w", u, err)
	}
	return resp.Header, nil
}

// sameOrigin resolves ref against base and rejects targets on another origin.
func sameOrigin(base, ref string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse registry url: %w", err)
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("parse pagination link %q: %w", ref, err)
	}
	resolved := b.ResolveReference(r)
	if resolved.Scheme != b.Scheme || resolved.Host != b.Host {
		return "", fmt.Errorf("pagination link %q leaves registry origin %s://%s", ref, b.Scheme, b.Host)
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
