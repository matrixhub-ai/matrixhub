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
	"context"
	"fmt"
	"time"

	"github.com/antihax/optional"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/sync_policy"
	"github.com/matrixhub-ai/matrixhub/test/tools"
)

var _ = Describe("SyncPolicy Job Log", Label("sync-policy"), func() {
	It("should generate and retrieve sync job logs", Label("SP0040"), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		api := tools.GetV1alpha1SyncPolicyApi()

		// 1. Create project and registry
		project, err := tools.CreateProjectFixture(ctx, "e2e-log-target")
		Expect(err).NotTo(HaveOccurred())
		defer project.Cleanup(ctx)

		registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-log", "https://hf-mirror.com")
		Expect(err).NotTo(HaveOccurred())
		defer registry.Cleanup(ctx)
		Expect(registry.ID).To(BeNumerically(">", 0))

		// 2. Create Manual + Pull Base Policy
		name := fmt.Sprintf("e2e-log-%d", time.Now().UnixNano())
		req := v1alpha1.V1alpha1CreateSyncPolicyRequest{
			Name:        name,
			Description: "e2e log test",
			PolicyType:  ptrPolicyType(v1alpha1.PULL_BASE_V1alpha1SyncPolicyType),
			TriggerType: ptrTriggerType(v1alpha1.MANUAL_V1alpha1TriggerType),
			PullBasePolicy: &v1alpha1.V1alpha1PullBasePolicy{
				SourceRegistryId:  registry.ID,
				ResourceName:      "gpt2",
				ResourceTypes:     []v1alpha1.V1alpha1ResourceType{v1alpha1.MODEL_V1alpha1ResourceType},
				TargetProjectName: project.Name,
			},
			IsOverwrite: false,
		}

		resp, _, err := api.SyncPolicyCreateSyncPolicy(ctx, req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.SyncPolicy).NotTo(BeNil())
		pid := resp.SyncPolicy.Id
		Expect(pid).To(BeNumerically(">", 0))
		defer func() {
			_, _, _ = api.SyncPolicyDeleteSyncPolicy(ctx, pid)
		}()

		// 3. Manually create SyncTask
		taskResp, _, err := api.SyncPolicyCreateSyncTask(ctx, pid)
		Expect(err).NotTo(HaveOccurred())
		tid := taskResp.Id
		Expect(tid).To(BeNumerically(">", 0))

		// 4. Wait for task completion
		task, err := waitForTaskCompletion(ctx, api, pid, tid, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred())
		Expect(task).NotTo(BeNil())

		// 5. Get job list
		jobsResp, _, err := api.SyncPolicyListSyncJobs(ctx, pid, tid, &v1alpha1.SyncPolicyApiSyncPolicyListSyncJobsOpts{
			Page:     optional.NewInt32(1),
			PageSize: optional.NewInt32(20),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(jobsResp.SyncJobs)).To(BeNumerically(">", 0))

		// 6. Query and verify log for each job
		for _, job := range jobsResp.SyncJobs {
			logResp, _, err := api.SyncPolicyGetSyncJobLog(ctx, pid, tid, job.Id)
			Expect(err).NotTo(HaveOccurred())

			// Log should contain basic job information
			Expect(logResp.Log).To(ContainSubstring("job start"))
			Expect(logResp.Log).To(ContainSubstring("registry="))

			// Verify log content based on job status
			if job.Status != nil {
				switch *job.Status {
				case v1alpha1.SUCCEEDED_V1alpha1SyncJobStatus:
					Expect(logResp.Log).To(ContainSubstring("job completed successfully"))
				case v1alpha1.FAILED_V1alpha1SyncJobStatus:
					Expect(logResp.Log).To(ContainSubstring("[ERROR]"))
				}
			}
		}
	})
})
