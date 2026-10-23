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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

const testSHA = "c1899de289a04d12100db370d81485cdf75e47ca"

func TestDiscovery_ListRevisionFiles(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The registry has a credential, but the listing must not send it.
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization = %q, want none", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.EscapedPath() == "/api/models/Qwen/Qwen3-0.6B/revision/refs%2Fpr%2F1":
			fmt.Fprintf(w, `{"sha":%q}`, testSHA)
		case r.URL.Path == "/api/models/Qwen/Qwen3-0.6B/tree/"+testSHA && r.URL.Query().Get("cursor") == "":
			if r.URL.Query().Get("recursive") != "true" {
				t.Errorf("tree request is not recursive: %s", r.URL)
			}
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/models/Qwen/Qwen3-0.6B/tree/%s?recursive=true&cursor=p2>; rel="next"`, ts.URL, testSHA))
			fmt.Fprint(w, `[
				{"type":"directory","path":"nested","size":0},
				{"type":"file","path":"config.json","size":726},
				{"type":"file","path":"model.safetensors","size":134,"lfs":{"oid":"f47f","size":1503300328,"pointerSize":134}}
			]`)
		case r.URL.Query().Get("cursor") == "p2":
			fmt.Fprint(w, `[{"type":"file","path":"nested/tokenizer.json","size":11422654,"lfs":{"oid":"aeb1","size":11422654}}]`)
		default:
			t.Errorf("unexpected request %s", r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	reg := &registry.Registry{ID: 1, URL: ts.URL}
	reg.SetCredential(registry.NewBasicCredential("user", "secret"))

	got, err := New().ListRevisionFiles(context.Background(), reg, "Qwen", "Qwen3-0.6B", "refs/pr/1")
	if err != nil {
		t.Fatalf("ListRevisionFiles() error = %v", err)
	}

	if got.Commit != testSHA || got.Revision != "refs/pr/1" {
		t.Fatalf("got %q@%q", got.Revision, got.Commit)
	}
	want := []git.RevisionFile{
		{Path: "config.json", Size: 726},
		{Path: "model.safetensors", Size: 1503300328, IsLFS: true, OID: "f47f"},
		{Path: "nested/tokenizer.json", Size: 11422654, IsLFS: true, OID: "aeb1"},
	}
	if len(got.Files) != len(want) {
		t.Fatalf("got %d files: %+v", len(got.Files), got.Files)
	}
	for i := range want {
		if got.Files[i] != want[i] {
			t.Errorf("file %d = %+v, want %+v", i, got.Files[i], want[i])
		}
	}
}

func TestDiscovery_ListRevisionFiles_DefaultBranch(t *testing.T) {
	const otherSHA = "607a30d783dfa663caf39e06633721c8d4cfcd7e"
	tests := []struct {
		name string
		// refs answers the git ref advertisement; empty means 404.
		refs string
		want string
	}{
		{"named by the HEAD symref", advertisement(testSHA, "symref=HEAD:refs/heads/main"), "main"},
		{"another branch name", advertisement(testSHA, "symref=HEAD:refs/heads/master"), "master"},
		{"HEAD at another commit", advertisement(otherSHA, "symref=HEAD:refs/heads/main"), ""},
		{"no symref", advertisement(testSHA, ""), ""},
		{"no git endpoint", "", ""},
		{"not an advertisement", "<html></html>", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/models/Qwen/Qwen3-0.6B":
					fmt.Fprintf(w, `{"sha":%q}`, testSHA)
				case "/api/models/Qwen/Qwen3-0.6B/tree/" + testSHA:
					fmt.Fprint(w, `[{"type":"file","path":"config.json","size":726}]`)
				case "/Qwen/Qwen3-0.6B/info/refs":
					if r.URL.Query().Get("service") != "git-upload-pack" {
						t.Errorf("unexpected ref request %s", r.URL)
					}
					if tt.refs == "" {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
					fmt.Fprint(w, tt.refs)
				default:
					t.Errorf("unexpected request %s", r.URL)
				}
			}))
			defer ts.Close()

			got, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "Qwen3-0.6B", "")
			// The name only labels the listing: without it, the files are still listed.
			if err != nil || got.Commit != testSHA || len(got.Files) != 1 {
				t.Fatalf("ListRevisionFiles() = %+v, %v", got, err)
			}
			if got.Revision != tt.want {
				t.Errorf("Revision = %q, want %q", got.Revision, tt.want)
			}
		})
	}
}

// advertisement is a smart HTTP ref advertisement whose first ref is HEAD at
// head, with the given capabilities.
func advertisement(head, capabilities string) string {
	pkt := func(s string) string { return fmt.Sprintf("%04x%s", len(s)+4, s) }
	return pkt("# service=git-upload-pack\n") + "0000" +
		pkt(head+" HEAD\x00multi_ack side-band-64k ofs-delta "+capabilities+"\n") +
		pkt(head+" refs/heads/main\n") + "0000"
}

func TestDiscovery_ListRevisionFiles_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    string
		message string
		want    error
	}{
		{"404", http.StatusNotFound, "", "", registrydiscovery.ErrRemoteNotFound},
		{"revision not found", http.StatusNotFound, "RevisionNotFound", "", registrydiscovery.ErrRemoteNotFound},
		// huggingface.co answers 401 for a missing repository.
		{"anonymous 401", http.StatusUnauthorized, "", "Invalid username or password.", registrydiscovery.ErrRemoteNotFound},
		{"repo not found", http.StatusUnauthorized, "RepoNotFound", "", registrydiscovery.ErrRemoteNotFound},
		{"bad credential", http.StatusUnauthorized, "", "Invalid credentials in Authorization header", registrydiscovery.ErrRemoteAccessDenied},
		{"gated", http.StatusForbidden, "GatedRepo", "", registrydiscovery.ErrRemoteGated},
		{"403", http.StatusForbidden, "", "", registrydiscovery.ErrRemoteAccessDenied},
		{"502", http.StatusBadGateway, "", "", registrydiscovery.ErrRemoteUnavailable},
		{"rate limited", http.StatusTooManyRequests, "", "", registrydiscovery.ErrRemoteUnavailable},
		// Statuses the Hub API does not use for a listing.
		{"400", http.StatusBadRequest, "", "", registrydiscovery.ErrRemoteUnavailable},
		{"410", http.StatusGone, "", "", registrydiscovery.ErrRemoteUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.code != "" {
					w.Header().Set("X-Error-Code", tt.code)
				}
				if tt.message != "" {
					w.Header().Set("X-Error-Message", tt.message)
				}
				w.WriteHeader(tt.status)
			}))
			defer ts.Close()

			_, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "nope", "")
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDiscovery_ListRevisionFiles_RateLimited(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	_, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "Qwen3-0.6B", "")
	if !errors.Is(err, registrydiscovery.ErrRemoteUnavailable) || !errors.Is(err, registrydiscovery.ErrRemoteRateLimited) ||
		!strings.Contains(err.Error(), "retry after 30s") {
		t.Fatalf("error = %v, want a rate limit with its retry delay", err)
	}
}

func TestDiscovery_ListRevisionFiles_Unreachable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := ts.URL
	ts.Close()

	_, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: url}, "Qwen", "x", "")
	if !errors.Is(err, registrydiscovery.ErrRemoteUnavailable) {
		t.Fatalf("error = %v, want ErrRemoteUnavailable", err)
	}
}

func TestDiscovery_ListRevisionFiles_RejectsCrossOriginPagination(t *testing.T) {
	var otherRequests atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		otherRequests.Add(1)
		fmt.Fprint(w, `[{"type":"file","path":"from-another-origin","size":1}]`)
	}))
	defer other.Close()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/models/Qwen/x" {
			fmt.Fprintf(w, `{"sha":%q}`, testSHA)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/api/next>; rel="next"`, other.URL))
		fmt.Fprint(w, `[]`)
	}))
	defer ts.Close()

	_, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "x", "")
	if !errors.Is(err, registrydiscovery.ErrRemoteUnavailable) {
		t.Fatalf("error = %v, want ErrRemoteUnavailable", err)
	}
	if n := otherRequests.Load(); n != 0 {
		t.Fatalf("the other origin received %d requests", n)
	}
}

// hf-mirror.com may redirect to huggingface.co, whose pagination links then
// name huggingface.co rather than the registry.
func TestDiscovery_ListRevisionFiles_FollowsPaginationAfterRedirect(t *testing.T) {
	var target *httptest.Server
	target = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/models/Qwen/big/revision/main":
			fmt.Fprintf(w, `{"sha":%q}`, testSHA)
		case r.URL.Query().Get("cursor") == "":
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/models/Qwen/big/tree/%s?recursive=true&cursor=p2>; rel="next"`, target.URL, testSHA))
			fmt.Fprint(w, `[{"type":"file","path":"a","size":1}]`)
		case r.URL.Query().Get("cursor") == "p2":
			fmt.Fprint(w, `[{"type":"file","path":"b","size":2}]`)
		default:
			t.Errorf("unexpected request %s", r.URL)
		}
	}))
	defer target.Close()
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.RequestURI(), http.StatusPermanentRedirect)
	}))
	defer mirror.Close()

	got, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: mirror.URL}, "Qwen", "big", "main")
	if err != nil {
		t.Fatalf("ListRevisionFiles() error = %v", err)
	}
	if len(got.Files) != 2 || got.Files[0].Path != "a" || got.Files[1].Path != "b" {
		t.Fatalf("files = %+v, want a and b", got.Files)
	}
}

func TestDiscovery_ListRevisionFiles_TooLarge(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		// Every page links to another one.
		"too many pages": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Link", fmt.Sprintf(`<%s?recursive=true&cursor=%s>; rel="next"`, r.URL.Path, r.URL.Query().Get("cursor")+"x"))
			fmt.Fprint(w, `[{"type":"file","path":"a","size":1}]`)
		},
		"too many files": func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "[")
			for i := range maxRevisionFiles + 1 {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"type":"file","path":"f%d"}`, i)
			}
			fmt.Fprint(w, "]")
		},
	}
	for name, tree := range tests {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/models/Qwen/x" {
					fmt.Fprintf(w, `{"sha":%q}`, testSHA)
					return
				}
				tree(w, r)
			}))
			defer ts.Close()

			_, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "x", "")
			if !errors.Is(err, registrydiscovery.ErrRemoteTooLarge) {
				t.Fatalf("error = %v, want ErrRemoteTooLarge", err)
			}
		})
	}
}

// For example the HTML error page of a proxy in front of the registry.
func TestDiscovery_ListRevisionFiles_NotAListing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>Service Unavailable</body></html>")
	}))
	defer ts.Close()

	_, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "x", "")
	if !errors.Is(err, registrydiscovery.ErrRemoteUnavailable) {
		t.Fatalf("error = %v, want ErrRemoteUnavailable", err)
	}
}

func TestNextLink(t *testing.T) {
	tests := map[string]string{
		``:                                       ``,
		`<https://hf.co/a?cursor=1>; rel="next"`: `https://hf.co/a?cursor=1`,
		`<https://hf.co/p>; rel="prev", <https://hf.co/n>; rel="next"`: `https://hf.co/n`,
		`<https://hf.co/p>; rel="prev"`:                                ``,
		`<https://hf.co/n>; foo="bar"; rel = "next"`:                   `https://hf.co/n`,
	}
	for header, want := range tests {
		if got := nextLink(header); got != want {
			t.Errorf("nextLink(%q) = %q, want %q", header, got, want)
		}
	}
}
