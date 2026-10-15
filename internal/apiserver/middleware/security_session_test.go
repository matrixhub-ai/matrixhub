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
package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestSecuritySameOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin, fetchSite string
		allowed           bool
	}{
		{"http://localhost:13871", "same-origin", true},
		{"https://localhost:13871", "same-origin", true},
		{"", "", false},
		{"null", "", false},
		{"http://evil.invalid", "cross-site", false},
		{"http://localhost:13871", "cross-site", false},
		{"http://localhost:13871/path", "", false},
		{"http://evil@localhost:13871", "", false},
		{"http://localhost", "", false},
	} {
		r := httptest.NewRequest("POST", "http://localhost:13871/api/security/v1alpha1/models/p/m/rescan", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Sec-Fetch-Site", tc.fetchSite)
		if got := securitySameOrigin(r); got != tc.allowed {
			t.Errorf("origin=%q fetchSite=%q got=%v want=%v", tc.origin, tc.fetchSite, got, tc.allowed)
		}
	}
}
