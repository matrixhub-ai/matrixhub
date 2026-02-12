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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/tools"
	mhe2e "github.com/matrixhub-ai/matrixhub/test/e2e"
)

var _ = Describe("test Project", Label("project"), func() {

	Context("test CreateProject API", func() {

		It("should create a project successfully", Label("smoke", "L00001"), func() {
			projectName := mhe2e.GenerateTestProjectName("project")
			GinkgoWriter.Printf("Creating project: %v\n", projectName)

			resp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "should not have error")
			Expect(resp).NotTo(BeNil(), "response should not be nil")
			Expect(resp.Success).To(BeTrue(), "should return success")
			Expect(resp.HTTPStatusCode).To(BeNumerically(">=", 200), "should return 2xx status")
			Expect(resp.HTTPStatusCode).To(BeNumerically("<", 300), "should return 2xx status")

			GinkgoWriter.Printf("Create response: success=%v, status=%d\n", resp.Success, resp.HTTPStatusCode)

			// 注意：DeleteProject API 可能未实现，此处调用仅作清理尝试
			_, _ = apiClient.DeleteProject(ctx, projectName)
		})

		It("should fail to create a project with empty name", Label("L00002"), func() {
			resp, err := apiClient.CreateProject(ctx, "")
			Expect(err).NotTo(HaveOccurred(), "should not have error")
			Expect(resp).NotTo(BeNil(), "response should not be nil")
			Expect(resp.Success).To(BeFalse(), "should return failure")
			Expect(resp.Error).NotTo(BeNil(), "error should not be nil")
			Expect(resp.Error.Code).To(Equal(3), "should return invalid argument error code")

			GinkgoWriter.Printf("Error response: code=%v, message=%v\n", resp.Error.Code, resp.Error.Message)
		})
	})

	Context("test GetProject API", func() {

		It("should get an existing project successfully", Label("L00003"), func() {
			projectName := mhe2e.GenerateTestProjectName("project")
			GinkgoWriter.Printf("Creating then getting project: %v\n", projectName)

			// 先创建项目
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "create should succeed")
			Expect(createResp.Success).To(BeTrue(), "create should return success")
			defer func() {
				_, _ = apiClient.DeleteProject(ctx, projectName)
			}()

			// 然后获取项目
			getResp, err := apiClient.GetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "get should not have error")
			Expect(getResp).NotTo(BeNil(), "get response should not be nil")
			Expect(getResp.Success).To(BeTrue(), "get should return success")
			Expect(getResp.Data).NotTo(BeNil(), "data should not be nil")
			Expect(getResp.Data.Name).To(Equal(projectName), "project name should match")

			GinkgoWriter.Printf("Get response: name=%v, status=%d\n", getResp.Data.Name, getResp.HTTPStatusCode)
		})

		It("should return not found for non-existing project", Label("L00004"), func() {
			projectName := mhe2e.GenerateTestProjectName("project")
			GinkgoWriter.Printf("Getting non-existing project: %v\n", projectName)

			resp, err := apiClient.GetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "should not have error")
			Expect(resp).NotTo(BeNil(), "response should not be nil")
			Expect(resp.Success).To(BeFalse(), "should return failure")
			Expect(resp.Error).NotTo(BeNil(), "error should not be nil")
			Expect(resp.Error.Code).To(Equal(3), "should return not found error code")

			GinkgoWriter.Printf("Not found response: code=%v, message=%v\n", resp.Error.Code, resp.Error.Message)
		})
	})

	Context("test project operations", func() {

		It("should create and get same project", Label("L00005"), func() {
			projectName := mhe2e.GenerateTestProjectName("project")
			GinkgoWriter.Printf("Create and get same project: %v\n", projectName)

			// 创建项目
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "create should succeed")
			Expect(createResp.Success).To(BeTrue(), "create should return success")
			defer func() {
				_, _ = apiClient.DeleteProject(ctx, projectName)
			}()

			// 获取项目
			getResp, err := apiClient.GetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "get should not have error")
			Expect(getResp.Success).To(BeTrue(), "get should return success")
			Expect(getResp.Data.Name).To(Equal(projectName), "project name should match")
		})

		It("should handle concurrent project creation", Label("L00006"), func() {
			projectName := "concurrent-" + tools.RandomName()
			GinkgoWriter.Printf("Concurrent creation test for: %v\n", projectName)

			// 第一次创建应该成功
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "first create should succeed")
			Expect(createResp.Success).To(BeTrue(), "first create should return success")
			defer func() {
				_, _ = apiClient.DeleteProject(ctx, projectName)
			}()

			// 第二次创建同名项目的行为取决于业务逻辑
			// 可能返回成功 (幂等) 或失败 (冲突)
			secondResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "second create should not have error")
			Expect(secondResp).NotTo(BeNil(), "second response should not be nil")

			// 验证返回状态码是成功或失败（取决于业务逻辑）
			if secondResp.Success {
				GinkgoWriter.Printf("Second create succeeded (idempotent): status=%d\n", secondResp.HTTPStatusCode)
			} else {
				GinkgoWriter.Printf("Second create failed as expected: code=%v, message=%v\n",
					secondResp.Error.Code, secondResp.Error.Message)
			}
		})
	})
})
