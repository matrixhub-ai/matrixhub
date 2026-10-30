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
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1 "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/sync_policy"
	"github.com/matrixhub-ai/matrixhub/test/tools"
)

// The task and job APIs address a task by its policy and a job by its task.
// A path whose IDs do not belong together must find nothing, rather than the
// task or job of its last ID.
var _ = Describe("SyncPolicy API paths", Label("sync-policy"), func() {
	It("should reject a task or job under a policy or task it does not belong to", Label("SP0050"), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// The registry URL is dereferenced by the job server, so it must be
		// reachable from the deployment; see SP0011.
		selfURL := tools.GetSelfURL()
		if selfURL == "" {
			Skip(fmt.Sprintf("%s not set: no server-reachable MatrixHub URL to use as a stand-in registry", tools.EnvMatrixHubSelfURL))
		}

		api := tools.GetV1alpha1SyncPolicyApi()

		source, err := tools.CreatePublicProjectFixture(ctx, "e2e-path-src")
		Expect(err).NotTo(HaveOccurred())
		defer source.Cleanup(ctx)
		_, _, err = tools.GetV1alpha1ModelsApi().ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
			Project: source.Name,
			Name:    "path-model",
		})
		Expect(err).NotTo(HaveOccurred())

		target, err := tools.CreateProjectFixture(ctx, "e2e-path-dst")
		Expect(err).NotTo(HaveOccurred())
		defer target.Cleanup(ctx)

		registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-path", selfURL)
		Expect(err).NotTo(HaveOccurred())
		defer registry.Cleanup(ctx)

		resp, _, err := api.SyncPolicyCreateSyncPolicy(ctx, v1alpha1.V1alpha1CreateSyncPolicyRequest{
			Name:        fmt.Sprintf("e2e-path-%d", time.Now().UnixNano()),
			Description: "e2e sync API paths",
			PolicyType:  ptrPolicyType(v1alpha1.PULL_BASE_V1alpha1SyncPolicyType),
			TriggerType: ptrTriggerType(v1alpha1.MANUAL_V1alpha1TriggerType),
			PullBasePolicy: &v1alpha1.V1alpha1PullBasePolicy{
				SourceRegistryId:  registry.ID,
				ResourceName:      source.Name + "/path-model",
				ResourceTypes:     []v1alpha1.V1alpha1ResourceType{v1alpha1.MODEL_V1alpha1ResourceType},
				TargetProjectName: target.Name,
			},
		})
		Expect(err).NotTo(HaveOccurred())
		pid := resp.SyncPolicy.Id
		defer func() {
			_, _, _ = api.SyncPolicyDeleteSyncPolicy(ctx, pid)
		}()

		taskResp, _, err := api.SyncPolicyCreateSyncTask(ctx, pid)
		Expect(err).NotTo(HaveOccurred())
		tid := taskResp.Id
		jobs, err := waitForJobs(ctx, api, pid, tid, 2*time.Minute)
		Expect(err).NotTo(HaveOccurred())
		jid := jobs[0].Id
		_, err = waitForTaskCompletion(ctx, api, pid, tid, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred())

		// The right path finds the log.
		_, _, err = api.SyncPolicyGetSyncJobLog(ctx, pid, tid, jid)
		Expect(err).NotTo(HaveOccurred())

		// The task does not belong to policy pid+1, whether or not that
		// policy exists.
		_, httpResp, err := api.SyncPolicyGetSyncTask(ctx, pid+1, tid)
		Expect(err).To(HaveOccurred())
		Expect(httpResp.StatusCode).To(Equal(http.StatusBadRequest))
		_, httpResp, err = api.SyncPolicyListSyncJobs(ctx, pid+1, tid, nil)
		Expect(err).To(HaveOccurred())
		Expect(httpResp.StatusCode).To(Equal(http.StatusBadRequest))
		_, httpResp, err = api.SyncPolicyGetSyncJobLog(ctx, pid+1, tid, jid)
		Expect(err).To(HaveOccurred())
		Expect(httpResp.StatusCode).To(Equal(http.StatusBadRequest))

		// Policy and task 0 exist for no job.
		_, httpResp, err = api.SyncPolicyGetSyncJobLog(ctx, 0, 0, jid)
		Expect(err).To(HaveOccurred())
		Expect(httpResp.StatusCode).To(Equal(http.StatusNotFound))
	})
})
