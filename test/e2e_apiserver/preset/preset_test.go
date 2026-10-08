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

package preset_test

import (
	"context"
	"strconv"
	"strings"

	"github.com/antihax/optional"

	"github.com/matrixhub-ai/matrixhub/internal/domain/preset"
	v1alpha1model "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/model"
	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	v1alpha1registry "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/registry"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Preset initialization", Label("preset"), func() {
	It("should expose preset registries, proxy projects, admin memberships and recommended models", Label("PR00001", "smoke"), func() {
		ctx := context.Background()
		registries, _, err := tools.GetV1alpha1RegistriesApi().RegistriesListRegistries(ctx,
			&v1alpha1registry.RegistriesApiRegistriesListRegistriesOpts{Page: optional.NewInt32(1), PageSize: optional.NewInt32(100)})
		Expect(err).NotTo(HaveOccurred())
		for _, expected := range []struct {
			id   int32
			name string
			url  string
		}{
			{1, "huggingface", "https://huggingface.co"},
			{2, "hf-mirror", "https://hf-mirror.com"},
			{3, "hf.m.daocloud", "https://hf.m.daocloud.io"},
		} {
			Expect(registries.Registries).To(ContainElement(And(
				HaveField("Id", expected.id),
				HaveField("Name", expected.name),
				HaveField("Url", expected.url),
				HaveField("Type_", HaveValue(Equal(v1alpha1registry.HUGGINGFACE_V1alpha1RegistryType))),
			)))
		}

		manifest := preset.DefaultManifest()
		Expect(manifest.Projects).To(HaveLen(4))
		Expect(manifest.Models).To(HaveLen(13))
		projectModels := make(map[string][]preset.ModelSpec)
		for _, spec := range manifest.Models {
			projectModels[spec.Project] = append(projectModels[spec.Project], spec)
		}
		projectsAPI := tools.GetV1alpha1ProjectsApi()
		for projectName, modelSpecs := range projectModels {
			project, _, err := projectsAPI.ProjectsGetProject(ctx, projectName)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.Type_).To(HaveValue(Equal(v1alpha1project.PUBLIC_V1alpha1ProjectType)))
			Expect(project.RegistryUrl).To(Equal("https://hf.m.daocloud.io"))
			Expect(project.Organization).To(Equal(projectName))
			Expect(project.ModelCount).To(BeNumerically(">=", len(modelSpecs)))
			members, _, err := projectsAPI.ProjectsListProjectMembers(ctx, projectName,
				&v1alpha1project.ProjectsApiProjectsListProjectMembersOpts{MemberName: optional.NewString("admin")})
			Expect(err).NotTo(HaveOccurred())
			Expect(members.Members).To(ContainElement(And(
				HaveField("MemberName", "admin"),
				HaveField("MemberType", HaveValue(Equal(v1alpha1project.USER_V1alpha1MemberType))),
				HaveField("Role", HaveValue(Equal(v1alpha1project.ADMIN_V1alpha1ProjectRoleType))),
			)))

			models, _, err := tools.GetV1alpha1ModelsApi().ModelsListModels(ctx, &v1alpha1model.ModelsApiModelsListModelsOpts{
				Project: optional.NewString(projectName), Popular: optional.NewBool(true),
				Page: optional.NewInt32(1), PageSize: optional.NewInt32(100),
			})
			Expect(err).NotTo(HaveOccurred())
			for _, spec := range modelSpecs {
				labels := make([]any, 0, len(spec.Labels))
				for _, label := range spec.Labels {
					labels = append(labels, And(
						HaveField("Name", label.Name),
						HaveField("Category", HaveValue(Equal(v1alpha1model.V1alpha1Category(strings.ToUpper(label.Category))))),
					))
				}
				Expect(models.Items).To(ContainElement(And(
					HaveField("Name", spec.Name),
					HaveField("Project", projectName),
					HaveField("Popular", true),
					HaveField("DefaultBranch", "main"),
					HaveField("Size", strconv.FormatInt(spec.Size, 10)),
					HaveField("ParameterCount", strconv.FormatInt(spec.ParameterCount, 10)),
					HaveField("Labels", ConsistOf(labels)),
				)))
				detail, _, err := tools.GetV1alpha1ModelsApi().ModelsGetModel(ctx, projectName, spec.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(detail.ReadmeContent).To(BeEmpty())
			}
		}
	})
})
