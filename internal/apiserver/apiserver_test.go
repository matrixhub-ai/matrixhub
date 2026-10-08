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

package apiserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIFirstDispatchesOnlyAPIServerPathsToGin(t *testing.T) {
	tests := []struct {
		path     string
		staticUI bool
		want     string
	}{
		{"/healthz", true, "api"},
		{"/healthzx", true, "protocol"},
		{"/api/v1alpha1/models", true, "api"},
		{"/apis/v1alpha1/x", true, "api"},
		{"/api/models/a/b", true, "protocol"},
		{"/api/datasets/a/b/revision/main", true, "protocol"},
		{"/api/whoami-v2", true, "protocol"},
		{"/api/repos/create", true, "protocol"},
		{"/api/settings/x", true, "protocol"},
		{"/assets/app.js", true, "api"},
		{"/assets/app.js", false, "protocol"},
		{"/favicon.ico", true, "api"},
		{"/favicon.ico", false, "protocol"},
		{"/proj/repo.git/info/refs", true, "protocol"},
		{"/", true, "protocol"},
		// Projects named api, apis or assets are shadowed under these prefixes, like v1/v2 are by the CAS routes.
		{"/api/v1alpha1/resolve/main/README.md", true, "api"},
		{"/apis/v1alpha1/info/refs", true, "api"},
		{"/assets/repo/resolve/main/README.md", true, "api"},
		{"/api/v1alpha1.git/info/refs", true, "protocol"},
	}
	for _, test := range tests {
		name := test.path
		if !test.staticUI {
			name += " without static dir"
		}
		t.Run(name, func(t *testing.T) {
			var served string
			record := func(name string) http.Handler {
				return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { served = name })
			}
			handler := apiFirst(record("api"), record("protocol"), apiServerPath(test.staticUI))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, test.path, nil))

			if served != test.want {
				t.Fatalf("%s served by %q, want %q", test.path, served, test.want)
			}
		})
	}
}
