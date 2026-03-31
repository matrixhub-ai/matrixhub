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

	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	"github.com/antihax/optional"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func boolPtr(b bool) *bool { return &b }

var _ = Describe("Model", Label("model"), func() {
	var (
		ctx         context.Context
		modelsApi   *v1alpha1model.ModelsApiService
		projectsApi *v1alpha1project.ProjectsApiService
	)

	BeforeEach(func() {
		ctx = context.Background()
		modelsApi = tools.GetV1alpha1ModelsApi()
		projectsApi = tools.GetV1alpha1ProjectsApi()
	})

	// ═══════════════════════════════════════════════════════════
	// 1. Label APIs
	// ═══════════════════════════════════════════════════════════
	Context("ListModelTaskLabels API", func() {
		It("should list task labels successfully", Label("M00001"), func() {
			resp, _, err := modelsApi.ModelsListModelTaskLabels(ctx)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("Task labels: found %d labels\n", len(resp.Items))
		})
	})

	Context("ListModelFrameLabels API", func() {
		It("should list framework/library labels successfully", Label("M00002"), func() {
			resp, _, err := modelsApi.ModelsListModelFrameLabels(ctx)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("Frame labels: found %d labels\n", len(resp.Items))
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 2. CRUD APIs (CreateModel / GetModel / ListModels / DeleteModel)
	// ═══════════════════════════════════════════════════════════
	Context("CRUD APIs", func() {
		var projectName string

		BeforeEach(func() {
			projectName = tools.GenerateTestProjectName("model-crud")
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

		// --- CreateModel ---
		It("should create a model successfully", Label("M00003"), func() {
			modelName := tools.GenerateTestModelName("model")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify model is created via GetModel
			getResp, _, err := modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Name).To(Equal(modelName))
			Expect(getResp.Project).To(Equal(projectName))

			GinkgoWriter.Printf("Created model: name=%v, project=%v\n", getResp.Name, getResp.Project)
		})

		It("should fail to create duplicate model", Label("M00004"), func() {
			modelName := tools.GenerateTestModelName("dup-model")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Try to create duplicate
			_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).To(HaveOccurred(), "duplicate model should fail")

			GinkgoWriter.Printf("Duplicate model error: %v\n", err)
		})

		It("should fail to create a model with empty project", Label("M00005"), func() {
			modelName := tools.GenerateTestModelName("model")
			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: "",
				Name:    modelName,
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail to create a model with empty name", Label("M00006"), func() {
			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    "",
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail to create a model in non-existent project", Label("M00007"), func() {
			modelName := tools.GenerateTestModelName("model")
			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: "non-existent-project-xyz",
				Name:    modelName,
			})
			Expect(err).To(HaveOccurred())
		})

		// --- GetModel ---
		It("should get an existing model with all expected fields", Label("M00008"), func() {
			modelName := tools.GenerateTestModelName("get-model")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			getResp, _, err := modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Name).To(Equal(modelName))
			Expect(getResp.Project).To(Equal(projectName))
			Expect(getResp.Id).NotTo(BeZero())
			Expect(getResp.CreatedAt).NotTo(BeEmpty())

			GinkgoWriter.Printf("GetModel: id=%v, name=%v, project=%v\n", getResp.Id, getResp.Name, getResp.Project)
		})

		It("should fail to get a non-existent model", Label("M00009"), func() {
			_, _, err := modelsApi.ModelsGetModel(ctx, projectName, "non-existent-model-xyz")
			Expect(err).To(HaveOccurred())
		})

		It("should fail to get a model from non-existent project", Label("M00010"), func() {
			_, _, err := modelsApi.ModelsGetModel(ctx, "nonexistent", "nonexistent-model")
			Expect(err).To(HaveOccurred())
		})

		// --- ListModels ---
		It("should list models with pagination", Label("M00011"), func() {
			modelName := tools.GenerateTestModelName("list-model")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(50),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listResp.Items).NotTo(BeNil())
			Expect(listResp.Pagination).NotTo(BeNil())

			GinkgoWriter.Printf("ListModels: total=%v, page=%v\n",
				listResp.Pagination.Total, listResp.Pagination.Page)
		})

		It("should filter models by project", Label("M00012"), func() {
			modelName := tools.GenerateTestModelName("filter-model")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project:  optional.NewString(projectName),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(20),
			})
			Expect(err).NotTo(HaveOccurred())

			var found bool
			for _, m := range listResp.Items {
				if m.Name == modelName {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "created model should be in the project-filtered list")
		})

		It("should search models by keyword", Label("M00013"), func() {
			modelName := tools.GenerateTestModelName("search-target")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project:  optional.NewString(projectName),
				Search:   optional.NewString("search-target"),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResp.Items)).To(BeNumerically(">=", 1))
		})

		It("should respect page_size=1", Label("M00014"), func() {
			// Create 2 models
			for i := 0; i < 2; i++ {
				name := tools.GenerateTestModelName("paging")
				_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    name,
				})
				Expect(err).NotTo(HaveOccurred())
			}

			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project:  optional.NewString(projectName),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(1),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResp.Items)).To(Equal(1))
			Expect(listResp.Pagination.Total).To(BeNumerically(">=", 2))
		})

		It("should return empty list for non-existent project filter", Label("M00015"), func() {
			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project:  optional.NewString("nonexistent-project-xyz"),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResp.Items)).To(Equal(0))
		})

		// --- DeleteModel ---
		It("should delete an existing model and verify", Label("M00016"), func() {
			modelName := tools.GenerateTestModelName("delete-model")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Delete
			_, _, err = modelsApi.ModelsDeleteModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())

			// Verify deleted via GetModel
			_, _, err = modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).To(HaveOccurred(), "get should fail after delete")
		})

		It("should fail to delete a non-existent model", Label("M00017"), func() {
			_, _, err := modelsApi.ModelsDeleteModel(ctx, projectName, "non-existent-model-xyz")
			Expect(err).To(HaveOccurred())
		})

		It("should fail to delete an already deleted model", Label("M00018"), func() {
			modelName := tools.GenerateTestModelName("double-delete")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// First delete
			_, _, err = modelsApi.ModelsDeleteModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())

			// Second delete should fail
			_, _, err = modelsApi.ModelsDeleteModel(ctx, projectName, modelName)
			Expect(err).To(HaveOccurred(), "double delete should fail")
		})

		// --- Full lifecycle ---
		It("should handle full create-get-list-delete lifecycle", Label("M00019"), func() {
			modelName := tools.GenerateTestModelName("lifecycle")

			// Create
			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Get
			getResp, _, err := modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Name).To(Equal(modelName))
			Expect(getResp.Project).To(Equal(projectName))

			// List
			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project:  optional.NewString(projectName),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(100),
			})
			Expect(err).NotTo(HaveOccurred())
			var found bool
			for _, m := range listResp.Items {
				if m.Name == modelName {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())

			// Delete
			_, _, err = modelsApi.ModelsDeleteModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())

			// Verify deleted
			_, _, err = modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).To(HaveOccurred())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 3. ListModelRevisions (requires pre-existing model with git data)
	// ═══════════════════════════════════════════════════════════
	Context("ListModelRevisions API", Label("git"), func() {
		var (
			gitProject string
			gitModel   string
		)

		BeforeEach(func() {
			gitProject = tools.GetGitModelProject()
			gitModel = tools.GetGitModelName()

			// Verify the git model exists, skip if not available
			_, _, err := modelsApi.ModelsGetModel(ctx, gitProject, gitModel)
			if err != nil {
				Skip("GIT_MODEL not available: " + gitProject + "/" + gitModel + " — set MATRIXHUB_GIT_PROJECT and MATRIXHUB_GIT_MODEL env vars")
			}
		})

		It("should list revisions successfully", Label("M00020"), func() {
			resp, _, err := modelsApi.ModelsListModelRevisions(ctx, gitProject, gitModel)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).NotTo(BeNil())

			GinkgoWriter.Printf("Revisions: branches=%d, tags=%d\n",
				len(resp.Items.Branches), len(resp.Items.Tags))
		})

		It("should fail for non-existent model", Label("M00021"), func() {
			_, _, err := modelsApi.ModelsListModelRevisions(ctx, "nonexistent", "nonexistent")
			Expect(err).To(HaveOccurred())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 4. ListModelCommits + GetModelCommit (requires git data)
	// ═══════════════════════════════════════════════════════════
	Context("Commits APIs", Label("git"), func() {
		var (
			gitProject string
			gitModel   string
		)

		BeforeEach(func() {
			gitProject = tools.GetGitModelProject()
			gitModel = tools.GetGitModelName()

			_, _, err := modelsApi.ModelsGetModel(ctx, gitProject, gitModel)
			if err != nil {
				Skip("GIT_MODEL not available: " + gitProject + "/" + gitModel)
			}
		})

		It("should list commits with default branch", Label("M00022"), func() {
			resp, _, err := modelsApi.ModelsListModelCommits(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsListModelCommitsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).NotTo(BeNil())
			Expect(resp.Pagination).NotTo(BeNil())

			GinkgoWriter.Printf("Commits: count=%d, total=%d\n", len(resp.Items), resp.Pagination.Total)
		})

		It("should list commits with revision=main", Label("M00023"), func() {
			resp, _, err := modelsApi.ModelsListModelCommits(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsListModelCommitsOpts{
				Revision: optional.NewString("main"),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).NotTo(BeNil())
		})

		It("should list commits with pagination", Label("M00024"), func() {
			resp, _, err := modelsApi.ModelsListModelCommits(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsListModelCommitsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(2),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resp.Items)).To(BeNumerically("<=", 2))
		})

		It("should fail to list commits for non-existent model", Label("M00025"), func() {
			_, _, err := modelsApi.ModelsListModelCommits(ctx, "nonexistent", "nonexistent", &v1alpha1model.ModelsApiModelsListModelCommitsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(10),
			})
			Expect(err).To(HaveOccurred())
		})

		It("should get a specific commit with valid fields", Label("M00026"), func() {
			// Get first commit ID
			listResp, _, err := modelsApi.ModelsListModelCommits(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsListModelCommitsOpts{
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(1),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResp.Items)).To(BeNumerically(">=", 1), "need at least one commit")

			commitID := listResp.Items[0].Id
			Expect(commitID).NotTo(BeEmpty())

			// Get specific commit
			commit, _, err := modelsApi.ModelsGetModelCommit(ctx, gitProject, gitModel, commitID)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit.Id).To(Equal(commitID))
			Expect(commit.Message).NotTo(BeEmpty())
			Expect(commit.AuthorName).NotTo(BeEmpty())
			Expect(commit.AuthorDate).NotTo(BeEmpty())
			// Commit ID should be 40-char SHA-1
			Expect(len(commit.Id)).To(Equal(40))

			GinkgoWriter.Printf("GetModelCommit: id=%v, author=%v, message=%v\n",
				commit.Id[:8], commit.AuthorName, commit.Message)
		})

		It("should fail to get non-existent commit ID", Label("M00027"), func() {
			_, _, err := modelsApi.ModelsGetModelCommit(ctx, gitProject, gitModel,
				"0000000000000000000000000000000000000000")
			Expect(err).To(HaveOccurred())
		})

		It("should fail to get commit from non-existent model", Label("M00028"), func() {
			_, _, err := modelsApi.ModelsGetModelCommit(ctx, "nonexistent", "nonexistent", "abc123")
			Expect(err).To(HaveOccurred())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 5. GetModelTree (requires git data)
	// ═══════════════════════════════════════════════════════════
	Context("GetModelTree API", Label("git"), func() {
		var (
			gitProject string
			gitModel   string
		)

		BeforeEach(func() {
			gitProject = tools.GetGitModelProject()
			gitModel = tools.GetGitModelName()

			_, _, err := modelsApi.ModelsGetModel(ctx, gitProject, gitModel)
			if err != nil {
				Skip("GIT_MODEL not available: " + gitProject + "/" + gitModel)
			}
		})

		It("should get root tree successfully", Label("M00029"), func() {
			resp, _, err := modelsApi.ModelsGetModelTree(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelTreeOpts{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).NotTo(BeNil())

			GinkgoWriter.Printf("GetModelTree root: found %d entries\n", len(resp.Items))

			// Validate first item structure if available
			if len(resp.Items) > 0 {
				item := resp.Items[0]
				Expect(item.Name).NotTo(BeEmpty())
				Expect(item.Path).NotTo(BeEmpty())
				Expect(item.Type_).NotTo(BeNil())
				GinkgoWriter.Printf("  first item: name=%v, type=%v, path=%v\n", item.Name, *item.Type_, item.Path)
			}
		})

		It("should get tree with revision=main", Label("M00030"), func() {
			resp, _, err := modelsApi.ModelsGetModelTree(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelTreeOpts{
				Revision: optional.NewString("main"),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Items).NotTo(BeNil())
		})

		It("should handle non-existent path gracefully", Label("M00031"), func() {
			resp, _, err := modelsApi.ModelsGetModelTree(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelTreeOpts{
				Path: optional.NewString("nonexistent-path-xyz"),
			})
			// Either returns error or empty items
			if err == nil {
				Expect(len(resp.Items)).To(Equal(0))
			}
		})

		It("should fail for non-existent model", Label("M00032"), func() {
			_, _, err := modelsApi.ModelsGetModelTree(ctx, "nonexistent", "nonexistent", &v1alpha1model.ModelsApiModelsGetModelTreeOpts{})
			Expect(err).To(HaveOccurred())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 6. GetModelBlob (requires git data)
	// ═══════════════════════════════════════════════════════════
	Context("GetModelBlob API", Label("git"), func() {
		var (
			gitProject    string
			gitModel      string
			firstFilePath string
			firstDirPath  string
		)

		BeforeEach(func() {
			gitProject = tools.GetGitModelProject()
			gitModel = tools.GetGitModelName()

			_, _, err := modelsApi.ModelsGetModel(ctx, gitProject, gitModel)
			if err != nil {
				Skip("GIT_MODEL not available: " + gitProject + "/" + gitModel)
			}

			// Discover file and directory paths from tree
			treeResp, _, err := modelsApi.ModelsGetModelTree(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelTreeOpts{})
			if err != nil {
				Skip("cannot get tree for GIT_MODEL")
			}

			for _, item := range treeResp.Items {
				if item.Type_ == nil {
					continue
				}
				if *item.Type_ == v1alpha1model.FILE_V1alpha1FileType && firstFilePath == "" {
					firstFilePath = item.Path
				}
				if *item.Type_ == v1alpha1model.DIR_V1alpha1FileType && firstDirPath == "" {
					firstDirPath = item.Path
				}
			}
		})

		It("should get blob for a file with valid fields", Label("M00033"), func() {
			if firstFilePath == "" {
				Skip("no file found in tree")
			}

			resp, _, err := modelsApi.ModelsGetModelBlob(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelBlobOpts{
				Path: optional.NewString(firstFilePath),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Name).NotTo(BeEmpty())
			Expect(resp.Path).NotTo(BeEmpty())
			Expect(resp.Type_).NotTo(BeNil())
			Expect(*resp.Type_).To(Equal(v1alpha1model.FILE_V1alpha1FileType))

			GinkgoWriter.Printf("GetModelBlob: name=%v, path=%v, size=%v, sha256=%v\n",
				resp.Name, resp.Path, resp.Size, resp.Sha256)
		})

		It("should get blob with revision=main", Label("M00034"), func() {
			if firstFilePath == "" {
				Skip("no file found in tree")
			}

			resp, _, err := modelsApi.ModelsGetModelBlob(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelBlobOpts{
				Revision: optional.NewString("main"),
				Path:     optional.NewString(firstFilePath),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Name).NotTo(BeEmpty())
		})

		It("should fail for non-existent file path", Label("M00035"), func() {
			_, _, err := modelsApi.ModelsGetModelBlob(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelBlobOpts{
				Path: optional.NewString("nonexistent-file-xyz.txt"),
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail for directory path", Label("M00036"), func() {
			if firstDirPath == "" {
				Skip("no directory found in tree")
			}

			_, _, err := modelsApi.ModelsGetModelBlob(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelBlobOpts{
				Path: optional.NewString(firstDirPath),
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail for non-existent model", Label("M00037"), func() {
			_, _, err := modelsApi.ModelsGetModelBlob(ctx, "nonexistent", "nonexistent", &v1alpha1model.ModelsApiModelsGetModelBlobOpts{
				Path: optional.NewString("README.md"),
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail for empty path", Label("M00038"), func() {
			_, _, err := modelsApi.ModelsGetModelBlob(ctx, gitProject, gitModel, &v1alpha1model.ModelsApiModelsGetModelBlobOpts{})
			Expect(err).To(HaveOccurred())
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 7. Multi-user permission tests
	// ═══════════════════════════════════════════════════════════
	Context("Multi-user permission tests", func() {
		var (
			viewerCookie    string
			viewerID        int32
			editorCookie    string
			editorID        int32
			projAdminCookie string
			projAdminID     int32
		)

		BeforeEach(func() {
			password := "Test@123456"

			viewerUsername := tools.GenerateTestUsername("m-viewer")
			var err error
			viewerID, viewerCookie, err = tools.CreateUserAndLoginWithID(viewerUsername, password, false)
			Expect(err).NotTo(HaveOccurred())

			editorUsername := tools.GenerateTestUsername("m-editor")
			editorID, editorCookie, err = tools.CreateUserAndLoginWithID(editorUsername, password, false)
			Expect(err).NotTo(HaveOccurred())

			projAdminUsername := tools.GenerateTestUsername("m-padmin")
			projAdminID, projAdminCookie, err = tools.CreateUserAndLoginWithID(projAdminUsername, password, false)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Project viewer model permissions", func() {
			var projectName string
			var modelName string

			BeforeEach(func() {
				projectName = tools.GenerateTestProjectName("m-viewer-perm")
				projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  projectName,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred())

				memberType := v1alpha1project.USER_V1alpha1MemberType
				viewerRole := v1alpha1project.VIEWER_V1alpha1ProjectRoleType
				_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   viewerID,
					MemberType: &memberType,
					Role:       &viewerRole,
				})
				Expect(err).NotTo(HaveOccurred())

				modelName = tools.GenerateTestModelName("viewer-model")
				_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    modelName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			})

			It("should allow viewer to get model", Label("M00039"), func() {
				viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
				getResp, _, err := viewerModelsApi.ModelsGetModel(ctx, projectName, modelName)
				Expect(err).NotTo(HaveOccurred(), "Viewer should be able to get model")
				Expect(getResp.Name).To(Equal(modelName))
			})

			It("should deny viewer from creating model", Label("M00040"), func() {
				viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
				_, _, err := viewerModelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    tools.GenerateTestModelName("viewer-create"),
				})
				Expect(err).To(HaveOccurred(), "Viewer should not be able to create model")
			})

			It("should deny viewer from deleting model", Label("M00041"), func() {
				viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
				_, _, err := viewerModelsApi.ModelsDeleteModel(ctx, projectName, modelName)
				Expect(err).To(HaveOccurred(), "Viewer should not be able to delete model")
			})
		})

		Context("Project editor model permissions", func() {
			var projectName string
			var modelName string

			BeforeEach(func() {
				projectName = tools.GenerateTestProjectName("m-editor-perm")
				projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  projectName,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred())

				memberType := v1alpha1project.USER_V1alpha1MemberType
				editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType
				_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   editorID,
					MemberType: &memberType,
					Role:       &editorRole,
				})
				Expect(err).NotTo(HaveOccurred())

				modelName = tools.GenerateTestModelName("editor-model")
				_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    modelName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			})

			It("should allow editor to get model", Label("M00042"), func() {
				editorModelsApi := tools.CreateModelClientWithCookie(editorCookie)
				getResp, _, err := editorModelsApi.ModelsGetModel(ctx, projectName, modelName)
				Expect(err).NotTo(HaveOccurred(), "Editor should be able to get model")
				Expect(getResp.Name).To(Equal(modelName))
			})

			It("should allow editor to create model", Label("M00043"), func() {
				editorModelsApi := tools.CreateModelClientWithCookie(editorCookie)
				_, _, err := editorModelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    tools.GenerateTestModelName("editor-create"),
				})
				Expect(err).NotTo(HaveOccurred(), "Editor should be able to create model")
			})

			It("should deny editor from deleting model", Label("M00044"), func() {
				editorModelsApi := tools.CreateModelClientWithCookie(editorCookie)
				_, _, err := editorModelsApi.ModelsDeleteModel(ctx, projectName, modelName)
				Expect(err).To(HaveOccurred(), "Editor should not be able to delete model")
			})
		})

		Context("Project admin model permissions", func() {
			var projectName string
			var modelName string

			BeforeEach(func() {
				projectName = tools.GenerateTestProjectName("m-padmin-perm")
				projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  projectName,
					Type_: &projectType,
				})
				Expect(err).NotTo(HaveOccurred())

				memberType := v1alpha1project.USER_V1alpha1MemberType
				adminRole := v1alpha1project.ADMIN_V1alpha1ProjectRoleType
				_, _, err = projectsApi.ProjectsAddProjectMemberWithRole(ctx, projectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   projAdminID,
					MemberType: &memberType,
					Role:       &adminRole,
				})
				Expect(err).NotTo(HaveOccurred())

				modelName = tools.GenerateTestModelName("padmin-model")
				_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    modelName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			})

			It("should allow project admin to get model", Label("M00045"), func() {
				projAdminModelsApi := tools.CreateModelClientWithCookie(projAdminCookie)
				getResp, _, err := projAdminModelsApi.ModelsGetModel(ctx, projectName, modelName)
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to get model")
				Expect(getResp.Name).To(Equal(modelName))
			})

			It("should allow project admin to create model", Label("M00046"), func() {
				projAdminModelsApi := tools.CreateModelClientWithCookie(projAdminCookie)
				_, _, err := projAdminModelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: projectName,
					Name:    tools.GenerateTestModelName("padmin-create"),
				})
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to create model")
			})

			It("should allow project admin to delete model", Label("M00047"), func() {
				projAdminModelsApi := tools.CreateModelClientWithCookie(projAdminCookie)
				_, _, err := projAdminModelsApi.ModelsDeleteModel(ctx, projectName, modelName)
				Expect(err).NotTo(HaveOccurred(), "Project admin should be able to delete model")
			})
		})

		It("should deny non-member from accessing model in project", Label("M00048"), func() {
			projectName := tools.GenerateTestProjectName("m-nonmember")
			projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
			_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
				Name:  projectName,
				Type_: &projectType,
			})
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, projectName)
			}()

			modelName := tools.GenerateTestModelName("nonmember")
			_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
			_, _, err = viewerModelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).To(HaveOccurred(), "Non-member should not be able to access model")
		})

		Context("ListModels permission tests", func() {
			var publicProjectName string
			var privateProjectName string
			var publicModelName string
			var privateModelName string

			BeforeEach(func() {
				publicProjectName = tools.GenerateTestProjectName("m-list-public")
				publicType := v1alpha1project.PUBLIC_V1alpha1ProjectType
				_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  publicProjectName,
					Type_: &publicType,
				})
				Expect(err).NotTo(HaveOccurred())

				publicModelName = tools.GenerateTestModelName("pub-model")
				_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: publicProjectName,
					Name:    publicModelName,
				})
				Expect(err).NotTo(HaveOccurred())

				privateProjectName = tools.GenerateTestProjectName("m-list-private")
				privateType := v1alpha1project.PRIVATE_V1alpha1ProjectType
				_, _, err = projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
					Name:  privateProjectName,
					Type_: &privateType,
				})
				Expect(err).NotTo(HaveOccurred())

				privateModelName = tools.GenerateTestModelName("priv-model")
				_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
					Project: privateProjectName,
					Name:    privateModelName,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, publicProjectName)
				_, _, _ = projectsApi.ProjectsDeleteProject(ctx, privateProjectName)
			})

			It("should show public project models to member", Label("M00049"), func() {
				memberType := v1alpha1project.USER_V1alpha1MemberType
				editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType
				_, _, err := projectsApi.ProjectsAddProjectMemberWithRole(ctx, publicProjectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   editorID,
					MemberType: &memberType,
					Role:       &editorRole,
				})
				Expect(err).NotTo(HaveOccurred())

				editorModelsApi := tools.CreateModelClientWithCookie(editorCookie)
				listResp, _, err := editorModelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
					Project:  optional.NewString(publicProjectName),
					Page:     optional.NewInt32(1),
					PageSize: optional.NewInt32(50),
				})
				Expect(err).NotTo(HaveOccurred())

				var found bool
				for _, m := range listResp.Items {
					if m.Name == publicModelName {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Member should see public project models")
			})

			It("should show public project models to non-member", Label("M00050"), func() {
				viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
				listResp, _, err := viewerModelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
					Project:  optional.NewString(publicProjectName),
					Page:     optional.NewInt32(1),
					PageSize: optional.NewInt32(50),
				})
				Expect(err).NotTo(HaveOccurred())

				var found bool
				for _, m := range listResp.Items {
					if m.Name == publicModelName {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Non-member should see public project models")
			})

			It("should show private project models to member", Label("M00051"), func() {
				memberType := v1alpha1project.USER_V1alpha1MemberType
				editorRole := v1alpha1project.EDITOR_V1alpha1ProjectRoleType
				_, _, err := projectsApi.ProjectsAddProjectMemberWithRole(ctx, privateProjectName, v1alpha1project.ProjectsAddProjectMemberWithRoleBody{
					MemberId:   editorID,
					MemberType: &memberType,
					Role:       &editorRole,
				})
				Expect(err).NotTo(HaveOccurred())

				editorModelsApi := tools.CreateModelClientWithCookie(editorCookie)
				listResp, _, err := editorModelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
					Project:  optional.NewString(privateProjectName),
					Page:     optional.NewInt32(1),
					PageSize: optional.NewInt32(50),
				})
				Expect(err).NotTo(HaveOccurred())

				var found bool
				for _, m := range listResp.Items {
					if m.Name == privateModelName {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Member should see private project models")
			})

			It("should not show private project models to non-member", Label("M00052"), func() {
				viewerModelsApi := tools.CreateModelClientWithCookie(viewerCookie)
				listResp, _, err := viewerModelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
					Project:  optional.NewString(privateProjectName),
					Page:     optional.NewInt32(1),
					PageSize: optional.NewInt32(50),
				})
				Expect(err).NotTo(HaveOccurred())

				var found bool
				for _, m := range listResp.Items {
					if m.Name == privateModelName {
						found = true
						break
					}
				}
				Expect(found).To(BeFalse(), "Non-member should not see private project models")
			})
		})
	})

	// ═══════════════════════════════════════════════════════════
	// 8. UpdateModelSetting API
	// ═══════════════════════════════════════════════════════════
	Context("UpdateModelSetting API", func() {
		var projectName string

		BeforeEach(func() {
			projectName = tools.GenerateTestProjectName("model-setting")
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

		It("should set model as popular", Label("M00053"), func() {
			modelName := tools.GenerateTestModelName("setting")
			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Set popular = true
			_, _, err = modelsApi.ModelsUpdateModelSetting(ctx, projectName, modelName, v1alpha1model.ModelsUpdateModelSettingBody{
				Popular: boolPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify via GetModel
			getResp, _, err := modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Popular).To(BeTrue())

			GinkgoWriter.Printf("UpdateModelSetting: model=%v, popular=%v\n", getResp.Name, getResp.Popular)
		})

		It("should unset model as popular", Label("M00054"), func() {
			modelName := tools.GenerateTestModelName("setting")
			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Set popular = true first
			_, _, err = modelsApi.ModelsUpdateModelSetting(ctx, projectName, modelName, v1alpha1model.ModelsUpdateModelSettingBody{
				Popular: boolPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())

			// Set popular = false
			_, _, err = modelsApi.ModelsUpdateModelSetting(ctx, projectName, modelName, v1alpha1model.ModelsUpdateModelSettingBody{
				Popular: boolPtr(false),
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify via GetModel
			getResp, _, err := modelsApi.ModelsGetModel(ctx, projectName, modelName)
			Expect(err).NotTo(HaveOccurred())
			Expect(getResp.Popular).To(BeFalse())
		})

		It("should filter popular models in list", Label("M00055"), func() {
			modelA := tools.GenerateTestModelName("popular")
			modelB := tools.GenerateTestModelName("normal")

			_, _, err := modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelA,
			})
			Expect(err).NotTo(HaveOccurred())

			_, _, err = modelsApi.ModelsCreateModel(ctx, v1alpha1model.V1alpha1CreateModelRequest{
				Project: projectName,
				Name:    modelB,
			})
			Expect(err).NotTo(HaveOccurred())

			// Set modelA as popular
			_, _, err = modelsApi.ModelsUpdateModelSetting(ctx, projectName, modelA, v1alpha1model.ModelsUpdateModelSettingBody{
				Popular: boolPtr(true),
			})
			Expect(err).NotTo(HaveOccurred())

			// List with popular filter
			listResp, _, err := modelsApi.ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project:  optional.NewString(projectName),
				Popular:  optional.NewBool(true),
				Page:     optional.NewInt32(1),
				PageSize: optional.NewInt32(50),
			})
			Expect(err).NotTo(HaveOccurred())

			var foundA, foundB bool
			for _, m := range listResp.Items {
				if m.Name == modelA {
					foundA = true
				}
				if m.Name == modelB {
					foundB = true
				}
			}
			Expect(foundA).To(BeTrue(), "Popular model should appear in popular list")
			Expect(foundB).To(BeFalse(), "Non-popular model should not appear in popular list")
		})

		It("should fail to update setting for non-existent model", Label("M00056"), func() {
			_, _, err := modelsApi.ModelsUpdateModelSetting(ctx, projectName, "nonexistent-model", v1alpha1model.ModelsUpdateModelSettingBody{
				Popular: boolPtr(true),
			})
			Expect(err).To(HaveOccurred())
		})

		It("should fail to update setting for non-existent project", Label("M00057"), func() {
			_, _, err := modelsApi.ModelsUpdateModelSetting(ctx, "nonexistent-project", "nonexistent-model", v1alpha1model.ModelsUpdateModelSettingBody{
				Popular: boolPtr(true),
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
