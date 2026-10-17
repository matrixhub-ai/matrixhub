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
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

const testSHA = "c1899de289a04d12100db370d81485cdf75e47ca"

func TestDiscovery_ListRevisionFiles(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models/Qwen/Qwen3-0.6B":
			fmt.Fprintf(w, `{"sha":%q}`, testSHA)
		case "/api/models/Qwen/Qwen3-0.6B/tree/" + testSHA:
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected request %s", r.URL)
		}
	}))
	defer ts.Close()

	got, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "Qwen3-0.6B", "")
	if err != nil || got.Commit != testSHA {
		t.Fatalf("ListRevisionFiles() = %+v, %v", got, err)
	}
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
		{"gated", http.StatusForbidden, "GatedRepo", "", registrydiscovery.ErrRemoteAccessDenied},
		{"403", http.StatusForbidden, "", "", registrydiscovery.ErrRemoteAccessDenied},
		{"502", http.StatusBadGateway, "", "", registrydiscovery.ErrRemoteUnavailable},
		{"rate limited", http.StatusTooManyRequests, "", "", registrydiscovery.ErrRemoteUnavailable},
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
	if !errors.Is(err, registrydiscovery.ErrRemoteUnavailable) || !strings.Contains(err.Error(), "retry after 30s") {
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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/models/Qwen/x" {
			fmt.Fprintf(w, `{"sha":%q}`, testSHA)
			return
		}
		w.Header().Set("Link", `<https://evil.example/api/next>; rel="next"`)
		fmt.Fprint(w, `[]`)
	}))
	defer ts.Close()

	if _, err := New().ListRevisionFiles(context.Background(), &registry.Registry{URL: ts.URL}, "Qwen", "x", ""); err == nil {
		t.Fatal("ListRevisionFiles() followed a pagination link to another origin")
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
