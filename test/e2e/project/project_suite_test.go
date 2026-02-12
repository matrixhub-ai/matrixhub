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

package project_test

import (
	"context"
	"os"
	"testing"
	"time"

	mhe2e "github.com/matrixhub-ai/matrixhub/test/e2e"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Project Suite")
}

var (
	apiClient *mhe2e.MatrixHubClient
	ctx       context.Context
	cancel    context.CancelFunc
)

var _ = BeforeSuite(func() {
	defer GinkgoRecover()

	baseURL := os.Getenv("APISERVER_URL")
	if baseURL == "" {
		baseURL = mhe2e.GetBaseURL()
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)

	GinkgoWriter.Printf("Waiting for API server at %s...\n", baseURL)
	err := mhe2e.WaitForServiceReady(ctx, baseURL, mhe2e.DefaultServiceReadyTimeout)
	Expect(err).NotTo(HaveOccurred(), "API server should be ready")

	apiClient = mhe2e.NewMatrixHubClient(baseURL)
})

var _ = AfterSuite(func() {
	if cancel != nil {
		cancel()
	}
})
