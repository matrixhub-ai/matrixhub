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
	"fmt"
	"net/http"
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
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		DeferCleanup(cancel)
		modelsApi = tools.GetV1alpha1ModelsApi()
	})

	// createProxyProject creates a proxy project of organization on a
	// Hugging Face registry at url.
	createProxyProject := func(url, organization string) string {
		registry, err := tools.CreateHuggingFaceRegistryFixture(ctx, "e2e-cache", url)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(registry.Cleanup, context.Background())

		projectsApi := tools.GetV1alpha1ProjectsApi()
		projectName := tools.GenerateTestProjectName("m-cache-proxy")
		projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
		_, _, err = projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
			Name:         projectName,
			Type_:        &projectType,
			RegistryId:   int32(registry.ID),
			Organization: organization,
		})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_, _, _ = projectsApi.ProjectsDeleteProject(context.Background(), projectName)
		})
		return projectName
	}

	It("should reject a model of a local project, which has no upstream", Label("M00062"), func() {
		project, err := tools.CreateProjectFixture(ctx, "m-cache-local")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(project.Cleanup, context.Background())

		_, resp, err := modelsApi.ModelsCacheModel(ctx, project.Name, "model", v1alpha1model.ModelsCacheModelBody{})
		Expect(err).To(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	})

	Context("in a proxy project whose upstream is unreachable", func() {
		var projectName string

		BeforeEach(func() {
			// Nothing listens there: the jobs that the specs create fail
			// without contacting any host.
			projectName = createProxyProject("http://127.0.0.1:1", "upstream-org")
		})

		It("should create a pending job", Label("M00063"), func() {
			job, _, err := modelsApi.ModelsCacheModel(ctx, projectName, "gpt2", v1alpha1model.ModelsCacheModelBody{Revision: "main"})
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Id).To(BeNumerically(">", 0))
			Expect(job.Status).NotTo(BeNil())
			Expect(*job.Status).To(Equal(v1alpha1model.PENDING_V1alpha1SyncJobStatus))
			Expect(job.Revision).To(Equal("main"))
			Expect(job.CreatedTimestamp).NotTo(BeEmpty())
		})

		It("should reject a ref that MatrixHub does not mirror", Label("M00064"), func() {
			_, resp, err := modelsApi.ModelsCacheModel(ctx, projectName, "gpt2", v1alpha1model.ModelsCacheModelBody{Revision: "refs/pr/1"})
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should not let a viewer create a job", Label("M00065"), func() {
			viewerID, viewerCookie, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("m-cache-viewer"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = tools.DeleteUser(int64(viewerID))
			})
			memberType := v1alpha1project.USER_V1alpha1MemberType
			viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
			_, _, err = tools.GetV1alpha1ProjectsApi().ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   viewerID,
				MemberType: &memberType,
				Role:       &viewerRole,
			})
			Expect(err).NotTo(HaveOccurred())

			_, resp, err := tools.CreateModelClientWithCookie(viewerCookie).
				ModelsCacheModel(ctx, projectName, "gpt2", v1alpha1model.ModelsCacheModelBody{})
			Expect(err).To(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})

	It("should pull the model when the job server runs the job", Label("M00066"), func() {
		// The registry URL is dereferenced by the job server, not by this
		// test process, so it must be an address that the deployment can
		// reach.
		selfURL := tools.GetSelfURL()
		if selfURL == "" {
			Skip(fmt.Sprintf("%s not set: no server-reachable MatrixHub URL to use as an upstream", tools.EnvMatrixHubSelfURL))
		}

		// A public project of this deployment stands in for the upstream
		// organization, so no public hub is contacted.
		source, err := tools.CreatePublicProjectFixture(ctx, "m-cache-src")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(source.Cleanup, context.Background())
		_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
			Project: source.Name,
			Name:    "tiny",
		})
		Expect(err).NotTo(HaveOccurred())

		projectName := createProxyProject(selfURL, source.Name)
		_, resp, err := modelsApi.ModelsGetModel(ctx, projectName, "tiny")
		Expect(err).To(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		job, _, err := modelsApi.ModelsCacheModel(ctx, projectName, "tiny", v1alpha1model.ModelsCacheModelBody{})
		Expect(err).NotTo(HaveOccurred())
		Expect(job.Id).To(BeNumerically(">", 0))

		// The job creates the model once it has pulled it.
		Eventually(func() error {
			_, _, err := modelsApi.ModelsGetModel(ctx, projectName, "tiny")
			return err
		}, 2*time.Minute, 2*time.Second).Should(Succeed())
	})
})
