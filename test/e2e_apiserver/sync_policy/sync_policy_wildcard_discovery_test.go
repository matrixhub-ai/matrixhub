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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1 "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/sync_policy"
	"github.com/matrixhub-ai/matrixhub/test/tools"
)

// A wildcard pull policy resolves the concrete resource list through registry
// discovery. Discovery must query the registry's configured URL — the same host
// the resulting sync jobs pull from. When it queries a hardcoded upstream
// instead, an operator who points a registry at a mirror or an air-gapped proxy
// gets an empty or wrong match set while the pull half of the pipeline still
// targets the mirror.
//
// MatrixHub serves a HuggingFace-compatible /api/models, so this suite points a
// HUGGINGFACE-typed registry back at the instance under test and uses it as the
// mirror. That keeps the assertion hermetic: the repositories that must be
// discovered are created by the test itself, and no public Hub is contacted.
var _ = Describe("SyncPolicy Wildcard Discovery", Label("sync-policy", "git"), func() {
	It("should discover repositories from the registry URL rather than a hardcoded hub", Label("SP0011"), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// The registry URL is dereferenced by the job server, not by this test
		// process, so it must be an address that is routable from inside the
		// deployment. tools.GetBaseURL() is not: under KIND it is a NodePort on
		// the host. Skip rather than guess.
		selfURL := tools.GetSelfURL()
		if selfURL == "" {
			Skip(fmt.Sprintf("%s not set: no server-reachable MatrixHub URL to use as a stand-in registry", tools.EnvMatrixHubSelfURL))
		}

		api := tools.GetV1alpha1SyncPolicyApi()
		modelsApi := tools.GetV1alpha1ModelsApi()

		// The source project doubles as the remote namespace ("author") on the
		// stand-in mirror. Public so discovery works without credentials.
		source, err := tools.CreatePublicProjectFixture(ctx, "e2e-wc-src")
		Expect(err).NotTo(HaveOccurred())
		defer source.Cleanup(ctx)

		wantModels := []string{"wildcard-alpha", "wildcard-beta"}
		for _, name := range wantModels {
			_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: source.Name,
				Name:    name,
			})
			Expect(err).NotTo(HaveOccurred(), "create source model %s/%s", source.Name, name)
		}

		target, err := tools.CreateProjectFixture(ctx, "e2e-wc-dst")
		Expect(err).NotTo(HaveOccurred())
		defer target.Cleanup(ctx)

		// The registry URL is this very deployment. If discovery ignored it and
		// went to huggingface.co, the randomly-named namespace would not exist
		// there and no jobs would be generated.
		registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-wc", selfURL)
		Expect(err).NotTo(HaveOccurred())
		defer registry.Cleanup(ctx)
		Expect(registry.ID).To(BeNumerically(">", 0))

		name := fmt.Sprintf("e2e-wildcard-discovery-%d", time.Now().UnixNano())
		req := v1alpha1.V1alpha1CreateSyncPolicyRequest{
			Name:        name,
			Description: "e2e wildcard discovery honours the registry URL",
			PolicyType:  ptrPolicyType(v1alpha1.PULL_BASE_V1alpha1SyncPolicyType),
			TriggerType: ptrTriggerType(v1alpha1.MANUAL_V1alpha1TriggerType),
			PullBasePolicy: &v1alpha1.V1alpha1PullBasePolicy{
				SourceRegistryId: registry.ID,
				// "<namespace>/*" is what turns this into a discovery-driven policy.
				ResourceName:      fmt.Sprintf("%s/*", source.Name),
				ResourceTypes:     []v1alpha1.V1alpha1ResourceType{v1alpha1.MODEL_V1alpha1ResourceType},
				TargetProjectName: target.Name,
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

		taskResp, _, err := api.SyncPolicyCreateSyncTask(ctx, pid)
		Expect(err).NotTo(HaveOccurred())
		tid := taskResp.Id
		Expect(tid).To(BeNumerically(">", 0))

		// One job per discovered repository. An empty job list is exactly the
		// symptom of discovery hitting the wrong host, so this is the assertion
		// that guards the regression.
		jobs, err := waitForJobs(ctx, api, pid, tid, 2*time.Minute)
		Expect(err).NotTo(HaveOccurred(), "no sync jobs generated — discovery did not reach the registry URL")

		// For a pull job the API reports the source as "<registry>:<namespace>/<resource>".
		discovered := make([]string, 0, len(jobs))
		for _, job := range jobs {
			Expect(job.Action).To(Equal("clone"))
			Expect(job.ResourceName).To(HavePrefix(fmt.Sprintf("%s:%s/", registry.Name, source.Name)),
				"job source should reference the namespace on the configured registry")
			discovered = append(discovered, job.ResourceName)
		}
		GinkgoWriter.Printf("discovered resources: %v\n", discovered)

		for _, want := range wantModels {
			Expect(discovered).To(ContainElement(fmt.Sprintf("%s:%s/%s", registry.Name, source.Name, want)),
				"wildcard discovery should have matched %s/%s from the registry URL", source.Name, want)
		}

		// The task itself is allowed to end in any terminal state: the pull half
		// may fail in a sandboxed environment. Discovery is what is under test.
		task, err := waitForTaskCompletion(ctx, api, pid, tid, 3*time.Minute)
		Expect(err).NotTo(HaveOccurred())
		Expect(task).NotTo(BeNil())
		Expect(task.TotalItems).NotTo(Equal("0"))
	})
})
