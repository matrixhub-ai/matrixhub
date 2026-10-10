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
package repo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestArtifactScannerSafeFailureCodes(t *testing.T) {
	for _, tc := range []struct{ body, want string }{
		{`{"status":"failed","error":"pickle_metadata_limit","findings":[{"scanner":"clamav","rule":"MATCH"}]}`, "pickle_metadata_limit"},
		{`{"status":"failed","error":"private attacker payload"}`, "scanner_incomplete"},
		{`not json`, "scanner_incomplete"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503); _, _ = w.Write([]byte(tc.body)) }))
		scanner := ArtifactHTTPScanner{Endpoint: server.URL, Client: server.Client()}
		result, err := scanner.Scan(context.Background(), "weights.pt", []byte("test"))
		server.Close()
		var coded interface{ ScanFailureCode() string }
		if !errors.As(err, &coded) || coded.ScanFailureCode() != tc.want || strings.Contains(err.Error(), "private") {
			t.Fatal(result, err)
		}
		if tc.want == "pickle_metadata_limit" && len(result.Findings) != 1 {
			t.Fatal(result)
		}
	}
}
func TestArtifactScannerTimeoutCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(60 * time.Millisecond) }))
	defer server.Close()
	scanner := ArtifactHTTPScanner{Endpoint: server.URL, Client: &http.Client{Timeout: 30 * time.Millisecond}}
	_, err := scanner.Scan(context.Background(), "config.json", []byte("{}"))
	var coded interface{ ScanFailureCode() string }
	if !errors.As(err, &coded) || coded.ScanFailureCode() != "scanner_timeout" {
		t.Fatal(err)
	}
}
