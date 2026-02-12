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

package common

import (
	"os"
	"time"
)

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seed := time.Now().UnixNano()
	result := make([]byte, length)

	for i := range length {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		result[i] = charset[seed%int64(len(charset))]
	}

	return string(result)
}

// Default timeouts to be used in context.WithTimeout
const (
	PodStartTimeout        = time.Minute * 5
	PodReStartTimeout      = time.Minute * 5
	ExecCommandTimeout     = time.Minute * 5
	EventOccurTimeout      = time.Second * 30
	ResourceDeleteTimeout  = time.Minute * 5
	BatchCreateTimeout     = time.Minute * 5
	InformerSyncStatusTime = time.Second * 30
	ServiceReadyTimeout    = time.Minute * 3
)

// ForcedWaitingTime is a default waiting time between operations
var ForcedWaitingTime = time.Second

// MatrixHub configurations
var (
	MatrixHubNamespace     = "matrixhub"
	MatrixHubServiceName   = "matrixhub"
	MatrixHubPort          = 9527
	MatrixHubDeployment    = "matrixhub"
	MatrixHubConfigMapName = "matrixhub-config"
)

// Environment variables
var (
	E2EClusterName    = os.Getenv("E2E_CLUSTER_NAME")
	E2EKubeConfig     = os.Getenv("KUBECONFIG")
	E2EMatrixHubImage = os.Getenv("E2E_MATRIXHUB_IMAGE")
)

func init() {
	if E2EClusterName == "" {
		E2EClusterName = "matrixhub-e2e"
	}
	if E2EMatrixHubImage == "" {
		E2EMatrixHubImage = "ghcr.io/matrixhub-ai/matrixhub:latest"
	}
}
