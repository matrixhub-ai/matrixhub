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
	mhe2e "github.com/matrixhub-ai/matrixhub/test/e2e/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

		It("should get an existing project", Label("L00003"), func() {
			projectName := mhe2e.GenerateTestProjectName("project")
			GinkgoWriter.Printf("Get project test: %v\n", projectName)

			// Create project first
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "create should not have error")
			Expect(createResp.Success).To(BeTrue(), "create should succeed")
			defer func() {
				_, _ = apiClient.DeleteProject(ctx, projectName)
			}()

			// Get project
			getResp, err := apiClient.GetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "get should not have error")
			Expect(getResp).NotTo(BeNil(), "response should not be nil")
			Expect(getResp.Success).To(BeTrue(), "get should succeed")
			Expect(getResp.Data).NotTo(BeNil(), "project data should not be nil")
			Expect(getResp.Data.Name).To(Equal(projectName), "project name should match")

			GinkgoWriter.Printf("Get response: name=%v\n", getResp.Data.Name)
		})

		It("should fail to get a non-existent project", Label("L00004"), func() {
			resp, err := apiClient.GetProject(ctx, "non-existent-project-xyz")
			Expect(err).NotTo(HaveOccurred(), "should not have error")
			Expect(resp).NotTo(BeNil(), "response should not be nil")
			Expect(resp.Success).To(BeFalse(), "should return failure")
			Expect(resp.Error).NotTo(BeNil(), "error should not be nil")

			GinkgoWriter.Printf("Error response: code=%v, message=%v\n", resp.Error.Code, resp.Error.Message)
		})
	})

	Context("test ListProjects API", func() {

		It("should list projects successfully", Label("L00005"), func() {
			projectName := mhe2e.GenerateTestProjectName("project")
			GinkgoWriter.Printf("List projects test: %v\n", projectName)

			// Create a project first
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "create should not have error")
			Expect(createResp.Success).To(BeTrue(), "create should succeed")
			defer func() {
				_, _ = apiClient.DeleteProject(ctx, projectName)
			}()

			// List projects
			listResp, err := apiClient.ListProjects(ctx, "", 1, 10)
			Expect(err).NotTo(HaveOccurred(), "list should not have error")
			Expect(listResp).NotTo(BeNil(), "response should not be nil")
			Expect(listResp.Success).To(BeTrue(), "list should succeed")
			Expect(listResp.Projects).NotTo(BeNil(), "projects should not be nil")

			// Find our project in the list
			var found bool
			for _, p := range listResp.Projects {
				if p.Name == projectName {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "created project should be in the list")

			GinkgoWriter.Printf("List response: found %d projects\n", len(listResp.Projects))
		})

		It("should filter projects by name", Label("L00006"), func() {
			projectName := mhe2e.GenerateTestProjectName("filter-test")

			// Create a project
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(createResp.Success).To(BeTrue())
			defer func() {
				_, _ = apiClient.DeleteProject(ctx, projectName)
			}()

			// Search with name filter
			listResp, err := apiClient.ListProjects(ctx, "filter-test", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Success).To(BeTrue())
			Expect(len(listResp.Projects)).To(BeNumerically(">=", 1), "should find at least one project")

			GinkgoWriter.Printf("Filter response: found %d projects with 'filter-test'\n", len(listResp.Projects))
		})
	})

	Context("test DeleteProject API", func() {

		It("should delete an existing project", Label("L00007"), func() {
			projectName := mhe2e.GenerateTestProjectName("delete-test")
			GinkgoWriter.Printf("Delete project test: %v\n", projectName)

			// Create project
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(createResp.Success).To(BeTrue())

			// Delete project
			deleteResp, err := apiClient.DeleteProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "delete should not have error")
			Expect(deleteResp).NotTo(BeNil(), "response should not be nil")
			Expect(deleteResp.Success).To(BeTrue(), "delete should succeed")

			// Verify project is deleted
			getResp, err := apiClient.GetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Success).To(BeFalse(), "get should fail after delete")

			GinkgoWriter.Printf("Delete response: success=%v\n", deleteResp.Success)
		})

		It("should fail to delete a non-existent project", Label("L00008"), func() {
			resp, err := apiClient.DeleteProject(ctx, "non-existent-project-xyz")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
			Expect(resp.Success).To(BeFalse(), "delete non-existent should fail")

			GinkgoWriter.Printf("Delete non-existent response: success=%v\n", resp.Success)
		})
	})

	Context("test ProjectMember API", func() {

		var projectName string

		BeforeEach(func() {
			projectName = mhe2e.GenerateTestProjectName("member-test")
			createResp, err := apiClient.CreateProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(createResp.Success).To(BeTrue())
		})

		AfterEach(func() {
			_, _ = apiClient.DeleteProject(ctx, projectName)
		})

		It("should list project members", Label("L00009"), func() {
			listResp, err := apiClient.ListProjectMembers(ctx, projectName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp).NotTo(BeNil())
			Expect(listResp.Success).To(BeTrue())

			GinkgoWriter.Printf("List members response: found %d members\n", len(listResp.Members))
		})

		It("should add a member to project", Label("L00010"), func() {
			memberID := "test-user-001"

			addResp, err := apiClient.AddProjectMemberWithRole(ctx, projectName, memberID, "MEMBER_TYPE_USER", "ROLE_TYPE_PROJECT_VIEWER")
			Expect(err).NotTo(HaveOccurred())
			Expect(addResp).NotTo(BeNil())
			Expect(addResp.Success).To(BeTrue())

			// Verify member is added
			listResp, err := apiClient.ListProjectMembers(ctx, projectName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Success).To(BeTrue())

			var found bool
			for _, m := range listResp.Members {
				if m.MemberID == memberID {
					found = true
					Expect(m.Role).To(Equal("ROLE_TYPE_PROJECT_VIEWER"))
					break
				}
			}
			Expect(found).To(BeTrue(), "member should be added")

			GinkgoWriter.Printf("Add member response: success=%v\n", addResp.Success)

			// Cleanup
			_, _ = apiClient.RemoveProjectMembers(ctx, projectName, []map[string]string{
				{"member_id": memberID, "member_type": "MEMBER_TYPE_USER"},
			})
		})

		It("should update member role", Label("L00011"), func() {
			memberID := "test-user-002"

			// Add member first
			addResp, err := apiClient.AddProjectMemberWithRole(ctx, projectName, memberID, "MEMBER_TYPE_USER", "ROLE_TYPE_PROJECT_VIEWER")
			Expect(err).NotTo(HaveOccurred())
			Expect(addResp.Success).To(BeTrue())

			// Update role
			updateResp, err := apiClient.UpdateProjectMemberRole(ctx, projectName, memberID, "MEMBER_TYPE_USER", "ROLE_TYPE_PROJECT_EDITOR")
			Expect(err).NotTo(HaveOccurred())
			Expect(updateResp).NotTo(BeNil())
			Expect(updateResp.Success).To(BeTrue())

			// Verify role is updated
			listResp, err := apiClient.ListProjectMembers(ctx, projectName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Success).To(BeTrue())

			var found bool
			for _, m := range listResp.Members {
				if m.MemberID == memberID {
					found = true
					Expect(m.Role).To(Equal("ROLE_TYPE_PROJECT_EDITOR"))
					break
				}
			}
			Expect(found).To(BeTrue(), "member should have updated role")

			GinkgoWriter.Printf("Update member role response: success=%v\n", updateResp.Success)

			// Cleanup
			_, _ = apiClient.RemoveProjectMembers(ctx, projectName, []map[string]string{
				{"member_id": memberID, "member_type": "MEMBER_TYPE_USER"},
			})
		})

		It("should remove member from project", Label("L00012"), func() {
			memberID := "test-user-003"

			// Add member first
			addResp, err := apiClient.AddProjectMemberWithRole(ctx, projectName, memberID, "MEMBER_TYPE_USER", "ROLE_TYPE_PROJECT_VIEWER")
			Expect(err).NotTo(HaveOccurred())
			Expect(addResp.Success).To(BeTrue())

			// Remove member
			removeResp, err := apiClient.RemoveProjectMembers(ctx, projectName, []map[string]string{
				{"member_id": memberID, "member_type": "MEMBER_TYPE_USER"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(removeResp).NotTo(BeNil())
			Expect(removeResp.Success).To(BeTrue())

			// Verify member is removed
			listResp, err := apiClient.ListProjectMembers(ctx, projectName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Success).To(BeTrue())

			for _, m := range listResp.Members {
				Expect(m.MemberID).NotTo(Equal(memberID), "member should be removed")
			}

			GinkgoWriter.Printf("Remove member response: success=%v\n", removeResp.Success)
		})
	})
})
