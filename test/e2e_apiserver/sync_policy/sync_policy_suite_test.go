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

package sync_policy_test

import (
	"testing"

	testenv "github.com/matrixhub-ai/matrixhub/test/e2e_apiserver/init"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSyncPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SyncPolicy Suite")
}

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	testenv.InitTestEnvironment()
})

var _ = AfterSuite(func() {
	testenv.CleanupTestEnvironment()
})
