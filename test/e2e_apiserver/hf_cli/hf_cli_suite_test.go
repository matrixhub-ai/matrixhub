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

package hf_cli_test

import (
	"os/exec"
	"testing"

	testenv "github.com/matrixhub-ai/matrixhub/test/e2e_apiserver/init"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHFCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HF CLI Suite")
}

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	_, err := exec.LookPath("hf")
	Expect(err).NotTo(HaveOccurred(), "hf CLI is required for the hf-cli e2e suite")
	testenv.InitTestEnvironment()
})

var _ = AfterSuite(func() {
	testenv.CleanupTestEnvironment()
})
