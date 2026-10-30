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

package model_test

import (
	"context"
	"net/http"
	"slices"
	"time"

	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Model cache", Label("model", "model-cache"), func() {
	var (
		ctx       context.Context
		modelsApi *v1alpha1model.ModelsApiService
	)

	BeforeEach(func() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		DeferCleanup(cancel)
		modelsApi = tools.GetV1alpha1ModelsApi()
	})

	Context("in a local project", func() {
		var projectName string

		BeforeEach(func() {
			project, err := tools.CreateProjectFixture(ctx, "m-cache-local")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(project.Cleanup, context.Background())
			projectName = project.Name
		})

		It("should reject a pull, because only proxy projects have an upstream", Label("M00062"), func() {
			_, resp, err := modelsApi.ModelsCacheModel(ctx, projectName, "model", v1alpha1model.ModelsCacheModelBody{DryRun: true})
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should report the cache of a missing model as not found", Label("M00063"), func() {
			_, resp, err := modelsApi.ModelsGetModelCache(ctx, projectName, "missing", nil)
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Context("in a proxy project", func() {
		var (
			projectName  string
			viewerCookie string
		)

		BeforeEach(func() {
			registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-cache", "https://hf-mirror.com")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(registry.Cleanup, context.Background())

			projectsApi := tools.GetV1alpha1ProjectsApi()
			projectName = tools.GenerateTestProjectName("m-cache-proxy")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			_, _, err = projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:         projectName,
				Type_:        &projectType,
				RegistryId:   int32(registry.ID),
				Organization: "openai-community",
			})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(context.Background(), projectName)
			})

			var viewerID int32
			viewerID, viewerCookie, err = tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("m-cache-viewer"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = tools.DeleteUser(int64(viewerID))
			})
			memberType := v1alpha1project.USER_V1alpha1MemberType
			viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   viewerID,
				MemberType: &memberType,
				Role:       &viewerRole,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		// In a proxy project the same dry run would list upstream, so only
		// the limit rejects it.
		It("should reject more than 16 patterns", Label("M00064"), func() {
			_, resp, err := modelsApi.ModelsCacheModel(ctx, projectName, "gpt2", v1alpha1model.ModelsCacheModelBody{
				DryRun:         true,
				IgnorePatterns: slices.Repeat([]string{"*.bin"}, 17),
			})
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should not let a viewer preview or start a pull", Label("M00065"), func() {
			viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
			// A model that upstream does not have: should the permission check
			// ever let the viewer through, no pull of gigabytes starts.
			for _, dryRun := range []bool{true, false} {
				_, resp, err := viewerModelsApi.ModelsCacheModel(ctx, projectName, "no-such-model-e2e", v1alpha1model.ModelsCacheModelBody{DryRun: dryRun})
				Expect(err).To(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			}
		})

		It("should preview a pull from upstream without creating the model", Label("M00066"), func() {
			st, err := previewRetryingUnavailable(ctx, modelsApi, projectName, "gpt2", v1alpha1model.ModelsCacheModelBody{
				DryRun:        true,
				AllowPatterns: []string{"*.json"},
			})
			Expect(err).NotTo(HaveOccurred())
			// The name of the default branch is best effort: it is left out
			// when the registry does not tell it in time.
			Expect(st.Revision).To(Or(Equal("main"), BeEmpty()))
			Expect(st.Commit).To(HaveLen(40))
			Expect(st.Files).NotTo(BeEmpty())
			for _, f := range st.Files {
				Expect(f.Path).To(HaveSuffix(".json"))
			}
			Expect(st.Task).To(BeNil())

			_, resp, err := modelsApi.ModelsGetModel(ctx, projectName, "gpt2")
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should report a model that upstream does not have as not found", Label("M00067"), func() {
			_, resp, err := modelsApi.ModelsCacheModel(ctx, projectName, "no-such-model-e2e", v1alpha1model.ModelsCacheModelBody{DryRun: true})
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})

// previewRetryingUnavailable runs a dry run, retrying for a minute while
// upstream is unavailable: hf-mirror.com, and huggingface.co, which it
// redirects to, are sometimes out of reach of a CI runner for a few seconds.
func previewRetryingUnavailable(ctx context.Context, api *v1alpha1model.ModelsApiService, project, name string, body v1alpha1model.ModelsCacheModelBody) (v1alpha1model.V1alpha1ModelCacheStatus, error) {
	deadline := time.Now().Add(time.Minute)
	for {
		st, resp, err := api.ModelsCacheModel(ctx, project, name, body)
		if err == nil || resp == nil || resp.StatusCode != http.StatusServiceUnavailable || time.Now().After(deadline) {
			return st, err
		}
		GinkgoWriter.Printf("upstream unavailable, retrying: %v\n", err)
		time.Sleep(5 * time.Second)
	}
}
