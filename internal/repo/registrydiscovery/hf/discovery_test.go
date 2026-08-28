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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

func TestDiscovery_ListRepositories_Models(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("author") != "google" {
			t.Errorf("unexpected author: %s", r.URL.Query().Get("author"))
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("unexpected limit: %s", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"google/bert-base"},{"id":"google/t5-base"}]`)
	}))
	defer ts.Close()

	d := New()

	reg := &registry.Registry{URL: ts.URL}
	repos, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "model",
	})
	if err != nil {
		t.Fatalf("ListRepositories error = %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
	if repos[0].Namespace != "google" || repos[0].Name != "bert-base" || repos[0].ResourceType != "model" {
		t.Errorf("repos[0] = %+v, want google/bert-base model", repos[0])
	}
	if repos[1].Namespace != "google" || repos[1].Name != "t5-base" || repos[1].ResourceType != "model" {
		t.Errorf("repos[1] = %+v, want google/t5-base model", repos[1])
	}
}

func TestDiscovery_ListRepositories_Datasets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/datasets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"google/wiki"}]`)
	}))
	defer ts.Close()

	d := New()

	reg := &registry.Registry{URL: ts.URL}
	repos, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "dataset",
	})
	if err != nil {
		t.Fatalf("ListRepositories error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Name != "wiki" {
		t.Errorf("repos[0].Name = %s, want wiki", repos[0].Name)
	}
}

func TestDiscovery_ListRepositories_WithCredential(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"google/bert-base"}]`)
	}))
	defer ts.Close()

	d := New()

	reg := &registry.Registry{
		URL:            ts.URL,
		CredentialType: registry.CredentialTypeBasic,
		AuthInfo:       `{"username":"token","password":"secret"}`,
	}
	repos, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "model",
	})
	if err != nil {
		t.Fatalf("ListRepositories error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
}

func TestEndpointBase(t *testing.T) {
	tests := []struct {
		name    string
		reg     *registry.Registry
		want    string
		wantErr bool
	}{
		{
			name: "registry url is used as-is",
			reg:  &registry.Registry{URL: "https://hf-mirror.com"},
			want: "https://hf-mirror.com",
		},
		{
			name: "trailing slash is trimmed",
			reg:  &registry.Registry{URL: "https://hf-mirror.com/"},
			want: "https://hf-mirror.com",
		},
		{
			name: "path prefix is preserved",
			reg:  &registry.Registry{URL: "https://proxy.internal/hf/"},
			want: "https://proxy.internal/hf",
		},
		{
			name:    "empty registry url is an error, not a fallback",
			reg:     &registry.Registry{ID: 7, Name: "internal-mirror"},
			wantErr: true,
		},
		{
			name:    "url of only a slash is an error",
			reg:     &registry.Registry{ID: 8, Name: "broken", URL: "/"},
			wantErr: true,
		},
		{
			name:    "nil registry is an error",
			reg:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := endpointBase(tt.reg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("endpointBase() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("endpointBase() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("endpointBase() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A registry without a URL must fail loudly. Falling back to a default host
// would both match against the wrong catalogue and send the registry's
// credential to a host the operator never configured.
func TestDiscovery_ListRepositories_NoRegistryURL(t *testing.T) {
	d := New()

	reg := &registry.Registry{ID: 7, Name: "internal-mirror"}
	_, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "model",
	})
	if err == nil {
		t.Fatal("expected an error for a registry with no URL, got nil")
	}
	if !strings.Contains(err.Error(), "internal-mirror") {
		t.Errorf("error = %v, want it to name the offending registry", err)
	}
}

func TestDiscovery_ListRepositories_RegistryURLWithPathPrefix(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"google/bert-base"}]`)
	}))
	defer ts.Close()

	d := New()

	reg := &registry.Registry{URL: ts.URL + "/hf"}
	repos, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "model",
	})
	if err != nil {
		t.Fatalf("ListRepositories error = %v", err)
	}
	if gotPath != "/hf/api/models" {
		t.Errorf("request path = %q, want /hf/api/models", gotPath)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
}

func TestDiscovery_ListRepositories_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	d := New()

	reg := &registry.Registry{URL: ts.URL}
	_, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "model",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDiscovery_ListRepositories_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer ts.Close()

	d := New()

	reg := &registry.Registry{URL: ts.URL}
	repos, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "google",
		ResourceType: "model",
	})
	if err != nil {
		t.Fatalf("ListRepositories error = %v", err)
	}
	if len(repos) != 0 {
		t.Fatalf("got %d repos, want 0", len(repos))
	}
}
