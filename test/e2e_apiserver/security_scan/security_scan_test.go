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

package security_scan_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	testtools "github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// SECURITY-SCAN E2E (issue #1066). Requires a server built with the scan
// subsystem and a reachable clamd; when clamd is absent the tests degrade to
// the "scanner unavailable" expectations (task failed, never clean).

// EICAR is the industry-standard harmless antivirus test string.
const eicarString = `X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`

// evilPickle is a STATIC sample: pickle opcodes referencing os.system. It is
// only ever walked as bytes — nothing deserializes or executes it.
var evilPickle = append([]byte{0x80, 0x02, 'c', 'o', 's', '\n', 's', 'y', 's', 't', 'e', 'm', '\n', '.', 0x59, 0x59}, 0x59)

// safePickle references a torch rebuild entrypoint (allowlisted).
var safePickle = append([]byte{0x80, 0x02, 'c', 't', 'o', 'r', 'c', 'h', '.', '_', 'u', 't', 'i', 'l', 's', '\n', '_', 'r', 'e', 'b', 'u', 'i', 'l', 'd', '_', 't', 'e', 'n', 's', 'o', 'r', '_', 'v', '2', '\n', '.'}, 0x58)

type scanStatusResp struct {
	Revision string `json:"revision"`
	Status   string `json:"status"`
	HFStatus string `json:"hfStatus"`
}

type reportResp struct {
	Status  string `json:"status"`
	Verdict string `json:"verdict"`
	Files   []struct {
		Path     string   `json:"path"`
		Severity string   `json:"severity"`
		Rules    []string `json:"rules"`
	} `json:"files"`
	Counts map[string]int `json:"counts"`
}

type modelInfoResp struct {
	SHA                string `json:"sha"`
	SecurityRepoStatus *struct {
		Kind    string `json:"kind"`
		Status  string `json:"status"`
		Details struct {
			Status  string         `json:"status"`
			Verdict string         `json:"verdict"`
			Counts  map[string]int `json:"counts"`
		} `json:"details"`
	} `json:"securityRepoStatus"`
}

func runCmd(dir string, env map[string]string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	e := os.Environ()
	for k, v := range env {
		e = append(e, k+"="+v)
	}
	cmd.Env = e
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func hfEnv(token string) map[string]string {
	return map[string]string{
		"HF_ENDPOINT":         testtools.GetBaseURL(),
		"HF_HUB_DISABLE_XET":  "1",
		"HF_TOKEN":            token,
		"HF_HUB_ETAG_TIMEOUT": "5",
	}
}

func httpJSON(method, url, user, token string, body io.Reader) (int, []byte) {
	req, err := http.NewRequest(method, url, body)
	Expect(err).NotTo(HaveOccurred())
	if token != "" {
		if user != "" {
			req.Header.Set("Authorization", "Basic "+
				base64.StdEncoding.EncodeToString([]byte(user+":"+token)))
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

// waitForScan polls the scan status endpoint until the revision leaves
// scanning/unscanned or the timeout hits.
func waitForScan(user, token, repoID, rev string) scanStatusResp {
	var last scanStatusResp
	Eventually(func() string {
		url := fmt.Sprintf("%s/api/scan/v1alpha1/status/models/%s/revision/%s",
			testtools.GetBaseURL(), repoID, rev)
		code, data := httpJSON(http.MethodGet, url, user, token, nil)
		if code != http.StatusOK {
			return fmt.Sprintf("http %d", code)
		}
		_ = json.Unmarshal(data, &last)
		return last.Status
	}, 90*time.Second, 2*time.Second).ShouldNot(BeElementOf("scanning", "unscanned"),
		"scan must finish; last status: %+v", last)
	return last
}

var _ = Describe("Security scanning and admission", Label("security-scan"), func() {
	var (
		ctx      context.Context
		root     string
		fixture  *testtools.ProjectUserFixture
		token    string
		username string
	)

	BeforeEach(func() {
		ctx = context.Background()
		root = GinkgoT().TempDir()
		var err error
		fixture, err = testtools.CreateProjectUserFixture(ctx, "secscan", v1alpha1project.EDITOR_V1alpha1ProjectRoleType)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			if err := fixture.Cleanup(context.Background()); err != nil {
				GinkgoWriter.Printf("secscan fixture cleanup failed: %v\n", err)
			}
		})
		token, err = fixture.CreateAccessToken(ctx)
		Expect(err).NotTo(HaveOccurred())
		username = fixture.Username
	})

	uploadFiles := func(model string, files map[string][]byte) string {
		dir := filepath.Join(root, model)
		Expect(os.MkdirAll(dir, 0o755)).To(Succeed())
		for name, content := range files {
			Expect(os.WriteFile(filepath.Join(dir, name), content, 0o600)).To(Succeed())
		}
		repoID := fixture.Project.Name + "/" + model
		_, err := runCmd(root, hfEnv(token), "hf", "repos", "create", repoID, "--repo-type", "model")
		Expect(err).NotTo(HaveOccurred())
		_, err = runCmd(root, hfEnv(token), "hf", "upload", repoID, dir, ".", "--commit-message", "secscan upload")
		Expect(err).NotTo(HaveOccurred())
		return repoID
	}

	modelInfo := func(repoID string, query string) modelInfoResp {
		code, data := httpJSON(http.MethodGet,
			testtools.GetBaseURL()+"/api/models/"+repoID+query, "", token, nil)
		Expect(code).To(Equal(http.StatusOK), string(data))
		var m modelInfoResp
		Expect(json.Unmarshal(data, &m)).To(Succeed())
		return m
	}

	It("scans clean models and allows download", Label("SEC00001", "smoke"), func() {
		model := testtools.GenerateTestModelName("sec-clean")
		repoID := uploadFiles(model, map[string][]byte{
			"README.md": []byte("# clean model\n"),
			"safe.pkl":  safePickle,
			"model.bin": []byte("weights-weights-weights"),
		})
		st := waitForScan(username, token, repoID, "main")
		Expect(st.Status).To(Equal("pass"), "clean model must scan as pass, got %+v", st)
		Expect(st.HFStatus).To(Equal("clean"))

		info := modelInfo(repoID, "?securityStatus=true&files_metadata=true")
		Expect(info.SecurityRepoStatus).NotTo(BeNil())
		Expect(info.SecurityRepoStatus.Status).To(Equal("clean"))
		Expect(info.SHA).NotTo(BeEmpty())

		// download must be allowed now
		code, _ := httpJSON(http.MethodGet,
			testtools.GetBaseURL()+"/"+repoID+"/resolve/main/README.md", "", token, nil)
		Expect(code).To(Equal(http.StatusOK))
	})

	It("blocks EICAR and malicious pickles end to end", Label("SEC00002", "smoke"), func() {
		model := testtools.GenerateTestModelName("sec-evil")
		repoID := uploadFiles(model, map[string][]byte{
			"README.md": []byte("# evil model\n"),
			"eicar.com": []byte(eicarString),
			"evil.pkl":  evilPickle,
		})
		st := waitForScan(username, token, repoID, "main")
		Expect(st.Status).To(Equal("blocked"), "EICAR + malicious pickle must block, got %+v", st)
		Expect(st.HFStatus).To(Equal("infected"))

		info := modelInfo(repoID, "?securityStatus=true")
		Expect(info.SecurityRepoStatus.Status).To(Equal("infected"))
		Expect(info.SecurityRepoStatus.Details.Counts["critical"]).To(BeNumerically(">=", 1))

		// HF-compatible download gets a stable structured 403.
		code, body := httpJSON(http.MethodGet,
			testtools.GetBaseURL()+"/"+repoID+"/resolve/main/eicar.com", "", token, nil)
		Expect(code).To(Equal(http.StatusForbidden), string(body))
		var e struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		Expect(json.Unmarshal(body, &e)).To(Succeed())
		Expect(e.Error).To(Equal("RevisionBlocked"))
		Expect(e.Message).To(ContainSubstring("security policy"))

		// report API: per-file evidence, no payload bytes
		rcode, rdata := httpJSON(http.MethodGet,
			fmt.Sprintf("%s/api/scan/v1alpha1/reports/models/%s/revision/main", testtools.GetBaseURL(), repoID),
			username, token, nil)
		Expect(rcode).To(Equal(http.StatusOK), string(rdata))
		var rep reportResp
		Expect(json.Unmarshal(rdata, &rep)).To(Succeed())
		Expect(rep.Status).To(Equal("blocked"))
		byPath := map[string]string{}
		for _, f := range rep.Files {
			byPath[f.Path] = f.Severity
		}
		Expect(byPath["eicar.com"]).To(Equal("critical"))
		Expect(byPath["evil.pkl"]).To(Equal("critical"))
		Expect(byPath["README.md"]).To(Equal("clean"))
		Expect(string(rdata)).NotTo(ContainSubstring(eicarString), "report must not echo payloads")
	})

	It("re-scans on content change and never reuses stale verdicts", Label("SEC00003"), func() {
		model := testtools.GenerateTestModelName("sec-evolve")
		repoID := uploadFiles(model, map[string][]byte{
			"README.md": []byte("# v1 clean\n"),
		})
		st1 := waitForScan(username, token, repoID, "main")
		Expect(st1.Status).To(Equal("pass"))

		// Same repo, new dangerous content → new revision, new verdict.
		_, err := runCmd(root, hfEnv(token), "hf", "upload", repoID,
			filepath.Join(root, model, "README.md"), "README.md", "--commit-message", "v2")
		Expect(err).NotTo(HaveOccurred())
		evilModel := testtools.GenerateTestModelName("sec-evolve2")
		_ = evilModel
		dir := filepath.Join(root, model)
		Expect(os.WriteFile(filepath.Join(dir, "drop.pkl"), evilPickle, 0o600)).To(Succeed())
		_, err = runCmd(root, hfEnv(token), "hf", "upload", repoID, dir, ".", "--commit-message", "v2 with payload")
		Expect(err).NotTo(HaveOccurred())
		st2 := waitForScan(username, token, repoID, "main")
		Expect(st2.Status).To(Equal("blocked"), "content change must produce a fresh verdict")
		Expect(st2.Revision).NotTo(Equal(st1.Revision))
	})

	It("supports manual rescan and records audit events", Label("SEC00004"), func() {
		model := testtools.GenerateTestModelName("sec-rescan")
		repoID := uploadFiles(model, map[string][]byte{"README.md": []byte("# rescan\n")})
		waitForScan(username, token, repoID, "main")

		code, body := httpJSON(http.MethodPost,
			fmt.Sprintf("%s/api/scan/v1alpha1/reports/models/%s/revision/main/rescan",
				testtools.GetBaseURL(), repoID), username, token, nil)
		Expect(code).To(Equal(http.StatusAccepted), string(body))
		st := waitForScan(username, token, repoID, "main")
		Expect(st.Status).To(Equal("pass"))

		acode, adata := httpJSON(http.MethodGet,
			testtools.GetBaseURL()+"/api/scan/v1alpha1/audit?name="+model, username, token, nil)
		Expect(acode).To(Equal(http.StatusOK), string(adata))
		var audit struct {
			Events []struct {
				Action string `json:"action"`
				Actor  string `json:"actor"`
			} `json:"events"`
		}
		Expect(json.Unmarshal(adata, &audit)).To(Succeed())
		actions := map[string]bool{}
		for _, e := range audit.Events {
			actions[e.Action] = true
		}
		Expect(actions["task_created"]).To(BeTrue())
		Expect(actions["task_completed"]).To(BeTrue())
	})

	It("rejects anonymous access to scan evidence", Label("SEC00005"), func() {
		code, _ := httpJSON(http.MethodGet,
			testtools.GetBaseURL()+"/api/scan/v1alpha1/audit", "", "", nil)
		Expect(code).To(Equal(http.StatusUnauthorized), "anonymous must not read scan evidence")
	})
})
