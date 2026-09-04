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

package hfd_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var hfHTTPClient = &http.Client{Timeout: 60 * time.Second}

// hfRequest performs an HTTP request against the HF-compatible API. An empty
// token sends no Authorization header.
func hfRequest(ctx context.Context, method, url, token string, body []byte) (int, []byte, http.Header) {
	GinkgoHelper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	Expect(err).NotTo(HaveOccurred())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := hfHTTPClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		_ = resp.Body.Close()
	}()
	respBody, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("%s %s -> %d\n%s\n", method, url, resp.StatusCode, truncateForLog(respBody))
	return resp.StatusCode, respBody, resp.Header
}

func truncateForLog(b []byte) []byte {
	const limit = 512
	if len(b) > limit {
		return append(b[:limit:limit], []byte("...")...)
	}
	return b
}

// commitNDJSON builds the NDJSON body of the HF commit API: a header line
// followed by one base64-encoded file line.
func commitNDJSON(summary, path string, content []byte) []byte {
	GinkgoHelper()
	header, err := json.Marshal(map[string]any{
		"key":   "header",
		"value": map[string]any{"summary": summary},
	})
	Expect(err).NotTo(HaveOccurred())
	file, err := json.Marshal(map[string]any{
		"key": "file",
		"value": map[string]any{
			"path":     path,
			"content":  base64.StdEncoding.EncodeToString(content),
			"encoding": "base64",
		},
	})
	Expect(err).NotTo(HaveOccurred())
	return append(append(header, '\n'), append(file, '\n')...)
}

var _ = Describe("GitProto HF API", Label("gitproto"), func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should upload via the commit API and download identical bytes via resolve", Label("GP00201", "smoke", "hf"), func() {
		f := setupFixture(ctx)

		// Binary-safe payload: base64 file encoding and the resolve download
		// must round-trip arbitrary bytes.
		payload := []byte(fmt.Sprintf("hf e2e payload %s seed=%d \x00\x01\xfe\xffbinary\n", f.project, GinkgoRandomSeed()))
		const filePath = "data/blob.bin"

		commitURL := fmt.Sprintf("%s/api/models/%s/%s/commit/main", tools.GetBaseURL(), f.project, modelName)
		status, body, _ := hfRequest(ctx, http.MethodPost, commitURL, f.token, commitNDJSON("gitproto e2e upload", filePath, payload))
		Expect(status).To(Equal(http.StatusOK), "commit API must succeed: %s", body)

		var commitResp struct {
			CommitOid string `json:"commitOid"`
		}
		Expect(json.Unmarshal(body, &commitResp)).To(Succeed())
		Expect(commitResp.CommitOid).NotTo(BeEmpty(), "commit response must carry the new commit oid")

		resolveURL := fmt.Sprintf("%s/%s/%s/resolve/main/%s", tools.GetBaseURL(), f.project, modelName, filePath)
		status, got, headers := hfRequest(ctx, http.MethodGet, resolveURL, f.token, nil)
		Expect(status).To(Equal(http.StatusOK), "resolve download must succeed: %s", got)
		Expect(got).To(Equal(payload), "downloaded bytes must match uploaded bytes")
		Expect(headers.Get("X-Repo-Commit")).To(Equal(commitResp.CommitOid), "resolve must report the commit just created")
	})

	It("should return the plain username from whoami-v2 for a bearer token", Label("GP00202", "smoke", "hf"), func() {
		f := setupFixture(ctx)

		status, body, _ := hfRequest(ctx, http.MethodGet, tools.GetBaseURL()+"/api/whoami-v2", f.token, nil)
		Expect(status).To(Equal(http.StatusOK), "whoami-v2 with a valid token: %s", body)

		var whoami struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		Expect(json.Unmarshal(body, &whoami)).To(Succeed())
		Expect(whoami.Type).To(Equal("user"))
		// Identity normalization: the exposed name must be exactly the plain
		// username, never the internal encoded identity blob.
		Expect(whoami.Name).To(Equal(f.username))
		Expect(whoami.ID).To(Equal(f.username))
	})

	It("should reject forged and missing bearer tokens", Label("GP00203", "hf"), func() {
		f := setupFixture(ctx)
		whoamiURL := tools.GetBaseURL() + "/api/whoami-v2"

		// No credentials at all: anonymous whoami is 401.
		status, _, _ := hfRequest(ctx, http.MethodGet, whoamiURL, "", nil)
		Expect(status).To(Equal(http.StatusUnauthorized), "anonymous whoami must be 401")

		// Forged tokens must never authenticate. The server currently answers
		// 500 (the git token validator surfaces the lookup failure as an
		// internal error instead of 401); accept the proper 401 as well so a
		// future server-side fix does not break this spec.
		rejected := []int{http.StatusUnauthorized, http.StatusInternalServerError}
		for _, forged := range []string{"hf_forged_token_e2e", "mh_forged_token_e2e"} {
			status, body, _ := hfRequest(ctx, http.MethodGet, whoamiURL, forged, nil)
			Expect(status).To(BeElementOf(rejected), "forged token %q on whoami: %s", forged, body)
			Expect(string(body)).NotTo(ContainSubstring(f.username), "forged token must not leak an identity")
		}

		// A forged token must not allow writes either.
		commitURL := fmt.Sprintf("%s/api/models/%s/%s/commit/main", tools.GetBaseURL(), f.project, modelName)
		status, body, _ := hfRequest(ctx, http.MethodPost, commitURL, "mh_forged_token_e2e", commitNDJSON("forged", "x.txt", []byte("nope")))
		Expect(status).To(BeElementOf(rejected), "forged token on commit: %s", body)
	})
})
