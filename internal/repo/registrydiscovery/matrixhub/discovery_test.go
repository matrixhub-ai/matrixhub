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

package matrixhub

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
)

func writeJSON(w http.ResponseWriter, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := protojson.Marshal(msg)
	_, _ = w.Write(b)
}

func TestDiscovery_ListRepositories_Models(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1alpha1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("project") != "google" {
			t.Errorf("unexpected project: %s", r.URL.Query().Get("project"))
		}
		writeJSON(w, &v1alpha1.ListModelsResponse{
			Items: []*v1alpha1.Model{
				{Project: "google", Name: "bert-base"},
				{Project: "google", Name: "t5-base"},
			},
			Pagination: &v1alpha1.Pagination{Total: 2, Page: 1, PageSize: defaultPageSize},
		})
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
		t.Errorf("repos[0] = %+v", repos[0])
	}
	if repos[1].Namespace != "google" || repos[1].Name != "t5-base" || repos[1].ResourceType != "model" {
		t.Errorf("repos[1] = %+v", repos[1])
	}
}

func TestDiscovery_ListRepositories_Datasets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1alpha1/datasets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, &v1alpha1.ListDatasetsResponse{
			Items: []*v1alpha1.Dataset{
				{Project: "google", Name: "wiki"},
			},
			Pagination: &v1alpha1.Pagination{Total: 1, Page: 1, PageSize: defaultPageSize},
		})
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
	if repos[0].Namespace != "google" || repos[0].Name != "wiki" || repos[0].ResourceType != "dataset" {
		t.Errorf("repos[0] = %+v", repos[0])
	}
}

func TestDiscovery_ListRepositories_Pagination(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			fmt.Sscanf(p, "%d", &page)
		}
		var items []*v1alpha1.Model
		for i := (page - 1) * defaultPageSize; i < page*defaultPageSize && i < 250; i++ {
			items = append(items, &v1alpha1.Model{Project: "org", Name: fmt.Sprintf("model-%d", i)})
		}
		writeJSON(w, &v1alpha1.ListModelsResponse{
			Items:      items,
			Pagination: &v1alpha1.Pagination{Total: 250, Page: int32(page), PageSize: defaultPageSize},
		})
	}))
	defer ts.Close()

	d := New()
	reg := &registry.Registry{URL: ts.URL}
	repos, err := d.ListRepositories(context.Background(), reg, registrydiscovery.Filter{
		Namespace:    "org",
		ResourceType: "model",
	})
	if err != nil {
		t.Fatalf("ListRepositories error = %v", err)
	}
	if len(repos) != 250 {
		t.Fatalf("got %d repos, want 250", len(repos))
	}
	if callCount != 3 {
		t.Fatalf("expected 3 http calls, got %d", callCount)
	}
}

func TestDiscovery_ListRepositories_WithCredential(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		writeJSON(w, &v1alpha1.ListModelsResponse{
			Items:      []*v1alpha1.Model{{Project: "google", Name: "bert-base"}},
			Pagination: &v1alpha1.Pagination{Total: 1, Page: 1, PageSize: defaultPageSize},
		})
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
		writeJSON(w, &v1alpha1.ListModelsResponse{
			Items:      nil,
			Pagination: &v1alpha1.Pagination{Total: 0, Page: 1, PageSize: defaultPageSize},
		})
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
