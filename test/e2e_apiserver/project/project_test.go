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
	"fmt"

	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	v1alpha1user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/user"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	"github.com/antihax/optional"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Project", Label("project"), func() {
	var (
		ctx         context.Context
		projectsApi *v1alpha1project.ProjectsApiService
	)

	BeforeEach(func() {
		ctx = context.Background()
		projectsApi = tools.GetV1alpha1ProjectsApi()
	})

	Context("CreateProject API", func() {
		It("should create a project successfully", Label("L00001"), func() {
			projectName := tools.GenerateTestProjectName("project")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			GinkgoWriter.Printf("Creating project: %v\n", projectName)

			resp, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("Create response: project=%v\n", resp)

			_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
		})

		It("should fail to create a project with empty name", Label("L00002"), func() {
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name: "",
			})
			Expect(err).To(HaveOccurred())

			GinkgoWriter.Printf("Error creating project with empty name: %v\n", err)
		})

		It("should create project with valid name - alphanumeric and hyphen", Label("L00018"), func() {
			validNames := []string{
				tools.GenerateTestProjectName("test-project"),
				tools.GenerateTestProjectName("t1"),
				tools.GenerateTestProjectName("1test2"),
				tools.GenerateTestProjectName("test"),
				tools.GenerateTestProjectName("12"),
			}
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			for _, name := range validNames {
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  name,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred(), "project %s should be created successfully", name)
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, name)
			}
		})

		It("should fail to create project with invalid name", Label("L00019"), func() {
			invalidNames := []string{
				"t",        // too short (less than 2 chars)
				"-test",    // starts with hyphen
				"test 01",  // contains space
				"test%123", // contains special char
				"test*123", // contains special char
				"test~01",  // contains special char
				"1test-",   // ends with hyphen
				"Test",     // uppercase not allowed
			}

			for _, name := range invalidNames {
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name: name,
				})
				Expect(err).To(HaveOccurred(), "project %s should fail to create", name)
				GinkgoWriter.Printf("Invalid name '%s': error=%v\n", name, err)
			}
		})

		It("should create public project successfully", Label("L00020"), func() {
			projectName := tools.GenerateTestProjectName("public-project")
			projectType := v1alpha1project.PUBLIC_V1alpha1ProjectType

			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify project is created
			getResp, _, err := projectsApi.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*getResp.Type_).To(Equal(v1alpha1project.PUBLIC_V1alpha1ProjectType))

			_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
		})

		It("should create private project successfully", Label("L00021"), func() {
			projectName := tools.GenerateTestProjectName("private-project")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify project is created
			getResp, _, err := projectsApi.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*getResp.Type_).To(Equal(v1alpha1project.PRIVATE_V1alpha1ProjectType))

			_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
		})

		It("should fail to create duplicate project name", Label("L00022"), func() {
			projectName := tools.GenerateTestProjectName("duplicate-test")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			// Create first project
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Try to create duplicate
			_, _, err = projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).To(HaveOccurred(), "duplicate project should fail")

			GinkgoWriter.Printf("Duplicate project error: %v\n", err)
		})

		It("should fail to create project with unspecified type", Label("L00045"), func() {
			projectName := tools.GenerateTestProjectName("unspecified-type")
			unspecifiedType := v1alpha1project.UNSPECIFIED_V1alpha1ProjectType

			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &unspecifiedType,
			})
			Expect(err).To(HaveOccurred(), "project with unspecified type should fail")

			GinkgoWriter.Printf("Unspecified type error: %v\n", err)
		})
	})

	Context("GetProject API", func() {
		It("should get an existing project", Label("L00003"), func() {
			projectName := tools.GenerateTestProjectName("project")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			GinkgoWriter.Printf("Get project test: %v\n", projectName)

			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Get project
			getResp, _, err := projectsApi.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Name).To(Equal(projectName))

			GinkgoWriter.Printf("Get response: name=%v\n", getResp.Name)
		})

		It("should fail to get a non-existent project", Label("L00004"), func() {
			_, _, err := projectsApi.ProjectsGetProject(ctx, "non-existent-project-xyz")
			Expect(err).To(HaveOccurred())

			GinkgoWriter.Printf("Error getting non-existent project: %v\n", err)
		})
	})

	Context("ListProjects API", func() {
		It("should list projects successfully", Label("L00005"), func() {
			projectName := tools.GenerateTestProjectName("project")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			GinkgoWriter.Printf("List projects test: %v\n", projectName)

			// Create a project first
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// List projects
			listResp, _, err := projectsApi.ProjectsListProjects(ctx, &v1alpha1project.ProjectsApiProjectsListProjectsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Projects).NotTo(BeNil())

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
			projectName := tools.GenerateTestProjectName("filter-test")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			// Create a project
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Search with name filter
			listResp, _, err := projectsApi.ProjectsListProjects(ctx, &v1alpha1project.ProjectsApiProjectsListProjectsOpts{
				Name:     optional.NewString("filter-test"),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResp.Projects)).To(BeNumerically(">=", 1), "should find at least one project")

			GinkgoWriter.Printf("Filter response: found %d projects with 'filter-test'\n", len(listResp.Projects))
		})
	})

	Context("DeleteProject API", func() {
		It("should delete an existing project", Label("L00007"), func() {
			projectName := tools.GenerateTestProjectName("delete-test")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			GinkgoWriter.Printf("Delete project test: %v\n", projectName)

			// Create project
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())

			// Delete project
			_, _, err = projectsApi.ProjectsDeleteProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())

			// Verify project is deleted
			_, _, err = projectsApi.ProjectsGetProject(ctx, projectName)
			Expect(err).To(HaveOccurred(), "get should fail after delete")

			GinkgoWriter.Printf("Delete response: success\n")
		})

		It("should fail to delete a non-existent project", Label("L00008"), func() {
			_, _, err := projectsApi.ProjectsDeleteProject(ctx, "non-existent-project-xyz")
			Expect(err).To(HaveOccurred())

			GinkgoWriter.Printf("Delete non-existent error: %v\n", err)
		})
	})

	Context("UpdateProject API", func() {
		It("should update project type from public to private", Label("L00023"), func() {
			projectName := tools.GenerateTestProjectName("update-test")
			publicType := v1alpha1project.PUBLIC_V1alpha1ProjectType
			privateType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			// Create public project
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &publicType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Update to private
			_, _, err = projectsApi.ProjectsUpdateProject(ctx, projectName, v1alpha1project.ProjectsUpdateProjectBody{
				Type_: &privateType,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify update
			getResp, _, err := projectsApi.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*getResp.Type_).To(Equal(v1alpha1project.PRIVATE_V1alpha1ProjectType))

			GinkgoWriter.Printf("Update project type response: success\n")
		})

		It("should update project type from private to public", Label("L00024"), func() {
			projectName := tools.GenerateTestProjectName("update-test-private")
			privateType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			publicType := v1alpha1project.PUBLIC_V1alpha1ProjectType

			// Create private project
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &privateType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Update to public
			_, _, err = projectsApi.ProjectsUpdateProject(ctx, projectName, v1alpha1project.ProjectsUpdateProjectBody{
				Type_: &publicType,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify update
			getResp, _, err := projectsApi.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*getResp.Type_).To(Equal(v1alpha1project.PUBLIC_V1alpha1ProjectType))

			GinkgoWriter.Printf("Update project type to public response: success\n")
		})
	})

	Context("ProjectMember API", func() {
		var projectName string

		BeforeEach(func() {
			projectName = tools.GenerateTestProjectName("member-test")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
		})

		It("should list project members", Label("L00009"), func() {
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Members).NotTo(BeNil())

			GinkgoWriter.Printf("List members response: found %d members\n", len(listResp.Members))
		})

		It("should add a member with viewer role", Label("L00010"), func() {
			memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("viewer"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.VIEWER_V1alpha1ProjectRoleType

			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &role,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify member is added with viewer role
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())

			var found bool
			for _, m := range listResp.Members {
				if m.MemberId == memberID {
					found = true
					Expect(*m.Role).To(Equal(v1alpha1project.VIEWER_V1alpha1ProjectRoleType))
					break
				}
			}
			Expect(found).To(BeTrue(), "member should be added with viewer role")

			GinkgoWriter.Printf("Add member with viewer role response: success\n")

			// Cleanup
			memberToRemove := v1alpha1project.USER_V1alpha1MemberType
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: memberID, MemberType: &memberToRemove},
				},
			})
		})

		It("should add a member with editor role", Label("L00025"), func() {
			memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("editor"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.EDITOR_V1alpha1ProjectRoleType

			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &role,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify member is added with editor role
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())

			var found bool
			for _, m := range listResp.Members {
				if m.MemberId == memberID {
					found = true
					Expect(*m.Role).To(Equal(v1alpha1project.EDITOR_V1alpha1ProjectRoleType))
					break
				}
			}
			Expect(found).To(BeTrue(), "member should be added with editor role")

			GinkgoWriter.Printf("Add member with editor role response: success\n")

			// Cleanup
			memberToRemove := v1alpha1project.USER_V1alpha1MemberType
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: memberID, MemberType: &memberToRemove},
				},
			})
		})

		It("should add a member with admin role", Label("L00026"), func() {
			memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("admin"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.ADMIN_V1alpha1ProjectRoleType

			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &role,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify member is added with admin role
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())

			var found bool
			for _, m := range listResp.Members {
				if m.MemberId == memberID {
					found = true
					Expect(*m.Role).To(Equal(v1alpha1project.ADMIN_V1alpha1ProjectRoleType))
					break
				}
			}
			Expect(found).To(BeTrue(), "member should be added with admin role")

			GinkgoWriter.Printf("Add member with admin role response: success\n")

			// Cleanup
			memberToRemove := v1alpha1project.USER_V1alpha1MemberType
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: memberID, MemberType: &memberToRemove},
				},
			})
		})

		It("should fail to add duplicate member to project", Label("L00027"), func() {
			memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("duplicate"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.VIEWER_V1alpha1ProjectRoleType

			// Add member first time
			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &role,
			})
			Expect(err).NotTo(HaveOccurred())

			// Try to add same member again
			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &role,
			})
			Expect(err).To(HaveOccurred(), "adding duplicate member should fail")

			GinkgoWriter.Printf("Duplicate member error: %v\n", err)

			// Cleanup
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: memberID, MemberType: &memberType},
				},
			})
		})

		It("should update member role from viewer to editor", Label("L00011"), func() {
			memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("role-update"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
			editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType

			// Add member first with viewer role
			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &viewerRole,
			})
			Expect(err).NotTo(HaveOccurred())

			// Update role to editor
			_, _, err = projectsApi.ProjectsUpdateProjectMemberRole(ctx, projectName, memberID, v1alpha1project.ProjectsUpdateProjectMemberRoleBody{
				MemberType: &memberType,
				Role:       &editorRole,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify role is updated
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())

			var found bool
			for _, m := range listResp.Members {
				if m.MemberId == memberID {
					found = true
					Expect(*m.Role).To(Equal(v1alpha1project.EDITOR_V1alpha1ProjectRoleType))
					break
				}
			}
			Expect(found).To(BeTrue(), "member should have updated role")

			GinkgoWriter.Printf("Update member role response: success\n")

			// Cleanup
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: memberID, MemberType: &memberType},
				},
			})
		})

		It("should remove single member from project", Label("L00012"), func() {
			memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("remove"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.VIEWER_V1alpha1ProjectRoleType

			// Add member first
			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   memberID,
				MemberType: &memberType,
				Role:       &role,
			})
			Expect(err).NotTo(HaveOccurred())

			// Remove member
			_, _, err = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: memberID, MemberType: &memberType},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify member is removed
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())

			for _, m := range listResp.Members {
				Expect(m.MemberId).NotTo(Equal(memberID), "member should be removed")
			}

			GinkgoWriter.Printf("Remove member response: success\n")
		})

		It("should batch remove members from project", Label("L00028"), func() {
			member1, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("batch1"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			member2, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("batch2"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			member3, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername("batch3"), "Test@123456", false)
			Expect(err).NotTo(HaveOccurred())
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.VIEWER_V1alpha1ProjectRoleType

			// Add members first
			for _, memberID := range []int32{member1, member2, member3} {
				_, _, err := projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   memberID,
					MemberType: &memberType,
					Role:       &role,
				})
				Expect(err).NotTo(HaveOccurred())
			}

			// Batch remove members
			_, _, err = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: member1, MemberType: &memberType},
					{MemberId: member2, MemberType: &memberType},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify members are removed
			listResp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())

			for _, m := range listResp.Members {
				Expect(m.MemberId).NotTo(Equal(member1), "member1 should be removed")
				Expect(m.MemberId).NotTo(Equal(member2), "member2 should be removed")
			}

			// member3 should still exist
			var member3Found bool
			for _, m := range listResp.Members {
				if m.MemberId == member3 {
					member3Found = true
					break
				}
			}
			Expect(member3Found).To(BeTrue(), "member3 should still exist")

			GinkgoWriter.Printf("Batch remove members response: success\n")

			// Cleanup remaining member
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: member3, MemberType: &memberType},
				},
			})
		})

		It("should list members with pagination", Label("L00030"), func() {
			// Add multiple members
			memberType := v1alpha1project.USER_V1alpha1MemberType
			role := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
			var memberIDs []int32
			for i := 0; i < 15; i++ {
				memberID, _, err := tools.CreateUserAndLoginWithID(tools.GenerateTestUsername(fmt.Sprintf("page-%d", i)), "Test@123456", false)
				Expect(err).NotTo(HaveOccurred())
				memberIDs = append(memberIDs, memberID)
				_, _, _ = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   memberID,
					MemberType: &memberType,
					Role:       &role,
				})
			}

			// Get first page
			page1Resp, _, err := projectsApi.ProjectsListProjectMembers(ctx, projectName, &v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(page1Resp.Members)).To(BeNumerically("<=", 10))

			GinkgoWriter.Printf("Page 1: found %d members, total=%d\n", len(page1Resp.Members), page1Resp.Pagination.Total)

			// Cleanup
			for _, memberID := range memberIDs {
				_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
					Members: []v1alpha1project.V1alpha1MemberToRemove{
						{MemberId: memberID, MemberType: &memberType},
					},
				})
			}
		})
	})

	Context("Multi-user permission tests", func() {
		var (
			viewerUsername    string
			viewerPassword    string
			viewerCookie      string
			viewerID          int32
			editorUsername    string
			editorPassword    string
			editorCookie      string
			editorID          int32
			projAdminUsername string
			projAdminPassword string
			projAdminCookie   string
			projAdminID       int32
		)

		BeforeEach(func() {
			viewerPassword = "Test@123456"
			editorPassword = "Test@123456"
			projAdminPassword = "Test@123456"

			// Create viewer user
			viewerUsername = tools.GenerateTestUsername("viewer")
			var err error
			viewerID, viewerCookie, err = tools.CreateUserAndLoginWithID(viewerUsername, viewerPassword, false)
			Expect(err).NotTo(HaveOccurred(), "Failed to create and login viewer user")

			// Create editor user
			editorUsername = tools.GenerateTestUsername("editor")
			editorID, editorCookie, err = tools.CreateUserAndLoginWithID(editorUsername, editorPassword, false)
			Expect(err).NotTo(HaveOccurred(), "Failed to create and login editor user")

			// Create project admin user
			projAdminUsername = tools.GenerateTestUsername("projadmin")
			projAdminID, projAdminCookie, err = tools.CreateUserAndLoginWithID(projAdminUsername, projAdminPassword, false)
			Expect(err).NotTo(HaveOccurred(), "Failed to create and login projAdmin user")
		})

		Context("Platform-level permission", func() {
			It("should deny normal user from creating users", Label("L00035"), func() {
				viewerUserApi := tools.CreateUserClientWithCookie(viewerCookie)
				_, _, err := viewerUserApi.UsersCreateUser(ctx, v1alpha1user.V1alpha1CreateUserRequest{
					Username: "should-not-be-created",
					Password: "Test@123456",
				})
				Expect(err).To(HaveOccurred(), "Normal user should not be able to create users")
				GinkgoWriter.Printf("Normal user correctly denied creating user: %v\n", err)
			})
		})

		Context("Project viewer permissions", func() {
			var projectName string

			BeforeEach(func() {
				projectName = tools.GenerateTestProjectName("viewer-perm")
				projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  projectName,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred())

				// Add viewer as viewer role member
				memberType := v1alpha1project.USER_V1alpha1MemberType
				viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
				_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   viewerID,
					MemberType: &memberType,
					Role:       &viewerRole,
				})
				Expect(err).NotTo(HaveOccurred(), "Failed to add viewer as member")
			})

			AfterEach(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			})

			It("should allow viewer to get project details", Label("L00036"), func() {
				viewerProjectApi := tools.CreateProjectClientWithCookie(viewerCookie)
				getResp, _, err := viewerProjectApi.ProjectsGetProject(ctx, projectName)
				Expect(err).NotTo(HaveOccurred(), "Viewer member should be able to get project")
				Expect(getResp.Name).To(Equal(projectName))
			})

			It("should deny viewer from updating project", Label("L00037"), func() {
				viewerProjectApi := tools.CreateProjectClientWithCookie(viewerCookie)
				privateType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := viewerProjectApi.ProjectsUpdateProject(ctx, projectName, v1alpha1project.ProjectsUpdateProjectBody{
					Type_: &privateType,
				})
				Expect(err).To(HaveOccurred(), "Viewer should not be able to update project")
				GinkgoWriter.Printf("Viewer correctly denied updating project: %v\n", err)
			})

			It("should deny viewer from deleting project", Label("L00038"), func() {
				viewerProjectApi := tools.CreateProjectClientWithCookie(viewerCookie)
				_, _, err := viewerProjectApi.ProjectsDeleteProject(ctx, projectName)
				Expect(err).To(HaveOccurred(), "Viewer should not be able to delete project")
				GinkgoWriter.Printf("Viewer correctly denied deleting project: %v\n", err)
			})

			It("should deny viewer from managing members", Label("L00039"), func() {
				viewerProjectApi := tools.CreateProjectClientWithCookie(viewerCookie)
				memberType := v1alpha1project.USER_V1alpha1MemberType
				editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType
				_, _, err := viewerProjectApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   editorID,
					MemberType: &memberType,
					Role:       &editorRole,
				})
				Expect(err).To(HaveOccurred(), "Viewer should not be able to add members")
				GinkgoWriter.Printf("Viewer correctly denied managing members: %v\n", err)
			})
		})

		Context("Project editor permissions", func() {
			var projectName string

			BeforeEach(func() {
				projectName = tools.GenerateTestProjectName("editor-perm")
				projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  projectName,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred())

				// Add editor as editor role member
				memberType := v1alpha1project.USER_V1alpha1MemberType
				editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType
				_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   editorID,
					MemberType: &memberType,
					Role:       &editorRole,
				})
				Expect(err).NotTo(HaveOccurred(), "Failed to add editor as member")
			})

			AfterEach(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			})

			It("should deny editor from deleting project", Label("L00040"), func() {
				editorProjectApi := tools.CreateProjectClientWithCookie(editorCookie)
				_, _, err := editorProjectApi.ProjectsDeleteProject(ctx, projectName)
				Expect(err).To(HaveOccurred(), "Editor should not be able to delete project")
				GinkgoWriter.Printf("Editor correctly denied deleting project: %v\n", err)
			})

			It("should deny editor from managing members", Label("L00041"), func() {
				editorProjectApi := tools.CreateProjectClientWithCookie(editorCookie)
				memberType := v1alpha1project.USER_V1alpha1MemberType
				viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
				_, _, err := editorProjectApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   viewerID,
					MemberType: &memberType,
					Role:       &viewerRole,
				})
				Expect(err).To(HaveOccurred(), "Editor should not be able to add members")
				GinkgoWriter.Printf("Editor correctly denied managing members: %v\n", err)
			})
		})

		Context("Project admin permissions", func() {
			var projectName string

			BeforeEach(func() {
				projectName = tools.GenerateTestProjectName("padmin-perm")
				projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  projectName,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred())

				// Add projAdmin as admin role member
				memberType := v1alpha1project.USER_V1alpha1MemberType
				adminRole := v1alpha1project.ADMIN_V1alpha1ProjectRoleType
				_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   projAdminID,
					MemberType: &memberType,
					Role:       &adminRole,
				})
				Expect(err).NotTo(HaveOccurred(), "Failed to add projAdmin as member")
			})

			AfterEach(func() {
				// Project might already be deleted by test, ignore error
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			})

			It("should allow project admin to manage members", Label("L00042"), func() {
				projAdminProjectApi := tools.CreateProjectClientWithCookie(projAdminCookie)

				// Add viewer as member
				memberType := v1alpha1project.USER_V1alpha1MemberType
				viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
				_, _, err := projAdminProjectApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   viewerID,
					MemberType: &memberType,
					Role:       &viewerRole,
				})
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to add members")

				// Remove member
				_, _, err = projAdminProjectApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
					Members: []v1alpha1project.V1alpha1MemberToRemove{
						{MemberId: viewerID, MemberType: &memberType},
					},
				})
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to remove members")
				GinkgoWriter.Printf("Project admin successfully managed members\n")
			})

			It("should allow project admin to update project", Label("L00043"), func() {
				projAdminProjectApi := tools.CreateProjectClientWithCookie(projAdminCookie)
				privateType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projAdminProjectApi.ProjectsUpdateProject(ctx, projectName, v1alpha1project.ProjectsUpdateProjectBody{
					Type_: &privateType,
				})
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to update project")
				GinkgoWriter.Printf("Project admin successfully updated project\n")
			})

			It("should allow project admin to delete project", Label("L00044"), func() {
				projAdminProjectApi := tools.CreateProjectClientWithCookie(projAdminCookie)
				_, _, err := projAdminProjectApi.ProjectsDeleteProject(ctx, projectName)
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to delete project")
				GinkgoWriter.Printf("Project admin successfully deleted project\n")
			})
		})

		It("should allow any logged-in user to list and create projects", Label("L00031"), func() {
			viewerProjectApi := tools.CreateProjectClientWithCookie(viewerCookie)

			// Any logged-in user should be able to list projects
			_, _, err := viewerProjectApi.ProjectsListProjects(ctx, &v1alpha1project.ProjectsApiProjectsListProjectsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred(), "Logged-in user should be able to list projects")

			// Any logged-in user should be able to create projects (project.create in all roles)
			projectName := tools.GenerateTestProjectName("viewer-project")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			_, _, err = viewerProjectApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred(), "Logged-in user should be able to create project")
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			GinkgoWriter.Printf("Normal user successfully listed and created projects\n")
		})

		It("should allow project admin to add members", Label("L00032"), func() {
			projectName := tools.GenerateTestProjectName("perm-test")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			// Create project as admin
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Add editor as a member with editor role
			memberType := v1alpha1project.USER_V1alpha1MemberType
			editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType
			_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
				MemberId:   editorID,
				MemberType: &memberType,
				Role:       &editorRole,
			})
			Expect(err).NotTo(HaveOccurred())

			// Editor should now be able to get the project
			editorProjectApi := tools.CreateProjectClientWithCookie(editorCookie)
			getResp, _, err := editorProjectApi.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred(), "Editor member should be able to get project")
			Expect(getResp.Name).To(Equal(projectName))

			GinkgoWriter.Printf("Editor successfully accessed project after being added as member\n")

			// Cleanup
			_, _, _ = projectsApi.ProjectsRemoveProjectMembers(ctx, projectName, v1alpha1project.ProjectsRemoveProjectMembersBody{
				Members: []v1alpha1project.V1alpha1MemberToRemove{
					{MemberId: editorID, MemberType: &memberType},
				},
			})
		})

		It("should deny non-member access to private project", Label("L00033"), func() {
			projectName := tools.GenerateTestProjectName("private-test")
			privateType := v1alpha1project.PRIVATE_V1alpha1ProjectType

			// Create private project as admin
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &privateType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			// Viewer (non-member) should not be able to get private project
			viewerProjectApi := tools.CreateProjectClientWithCookie(viewerCookie)
			_, _, err = viewerProjectApi.ProjectsGetProject(ctx, projectName)
			Expect(err).To(HaveOccurred(), "Non-member should not be able to access private project")

			GinkgoWriter.Printf("Non-member correctly denied access to private project: %v\n", err)
		})
	})
})
