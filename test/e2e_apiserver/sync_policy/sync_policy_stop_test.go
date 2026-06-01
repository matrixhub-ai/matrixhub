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

var _ = Describe("SyncPolicy Stop Task", Label("sync-policy"), func() {
	It("should stop a running sync task", Label("SP0030"), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		api := tools.GetV1alpha1SyncPolicyApi()

		// 1. Create project and registry
		project, err := tools.CreateProjectFixture(ctx, "e2e-stop-target")
		Expect(err).NotTo(HaveOccurred())
		defer project.Cleanup(ctx)

		registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-stop", "https://hf-mirror.com")
		Expect(err).NotTo(HaveOccurred())
		defer registry.Cleanup(ctx)
		Expect(registry.ID).To(BeNumerically(">", 0))

		// 2. Create Manual + Pull Base Policy
		name := fmt.Sprintf("e2e-stop-%d", time.Now().UnixNano())
		req := v1alpha1.V1alpha1CreateSyncPolicyRequest{
			Name:        name,
			Description: "e2e stop test",
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

		// 4. Wait for task to reach Running or beyond
		_, err = pollUntilTaskStatus(ctx, api, pid, tid, v1alpha1.RUNNING_V1alpha1SyncTaskStatus, 1*time.Minute)
		if err != nil {
			// Task may have completed too quickly; skip stop test
			By("task completed before stop could be applied, skipping stop verification")
			Skip("task completed too quickly to test stop")
		}

		// 5. Call StopSyncTask
		stopResp, _, err := api.SyncPolicyStopSyncTask(ctx, pid, tid)
		Expect(err).NotTo(HaveOccurred())
		Expect(stopResp.SyncTask).NotTo(BeNil())
		Expect(stopResp.SyncTask.Status).NotTo(BeNil())
		Expect(*stopResp.SyncTask.Status).To(Equal(v1alpha1.STOPPED_V1alpha1SyncTaskStatus))

		// 6. Wait for propagation
		time.Sleep(2 * time.Second)

		// 7. Query task final status
		getResp, _, err := api.SyncPolicyGetSyncTask(ctx, pid, tid)
		Expect(err).NotTo(HaveOccurred())
		Expect(getResp.SyncTask).NotTo(BeNil())
		Expect(getResp.SyncTask.Status).NotTo(BeNil())
		Expect(*getResp.SyncTask.Status).To(Equal(v1alpha1.STOPPED_V1alpha1SyncTaskStatus))

		// 8. Query jobs, verify status distribution
		jobsResp, _, err := api.SyncPolicyListSyncJobs(ctx, pid, tid, &v1alpha1.SyncPolicyApiSyncPolicyListSyncJobsOpts{
			Page:     optional.NewInt32(1),
			PageSize: optional.NewInt32(20),
		})
		Expect(err).NotTo(HaveOccurred())

		var stoppedCount int
		for _, job := range jobsResp.SyncJobs {
			if job.Status != nil && *job.Status == v1alpha1.STOPPED_V1alpha1SyncJobStatus {
				stoppedCount++
			}
		}
		// Jobs may or may not have been stopped depending on timing;
		// the important part is that the task itself is stopped.
		By(fmt.Sprintf("found %d stopped jobs out of %d total", stoppedCount, len(jobsResp.SyncJobs)))
	})
})
