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

package tools

import (
	"context"
	"fmt"

	v1alpha1project "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/project"
	v1alpha1registry "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/registry"
)

type ProjectFixture struct {
	Name    string
	cleanup func(context.Context)
}

func (f ProjectFixture) Cleanup(ctx context.Context) {
	if f.cleanup != nil {
		f.cleanup(ctx)
	}
}

func CreateProjectFixture(ctx context.Context, prefix string) (ProjectFixture, error) {
	projectsApi := GetV1alpha1ProjectsApi()
	name := GenerateTestProjectName(prefix)
	// projects.name is varchar(64) in DB; keep room for suffixes.
	if len(name) > 60 {
		name = name[:60]
	}

	projectType := v1alpha1project.PRIVATE_V1alpha1ProjectType
	_, _, err := projectsApi.ProjectsCreateProject(ctx, v1alpha1project.V1alpha1CreateProjectRequest{
		Name:  name,
		Type_: &projectType,
	})
	if err != nil {
		return ProjectFixture{}, err
	}

	return ProjectFixture{
		Name: name,
		cleanup: func(ctx context.Context) {
			_, _, _ = projectsApi.ProjectsDeleteProject(ctx, name)
		},
	}, nil
}

type RegistryFixture struct {
	ID      int64
	Name    string
	cleanup func(context.Context)
}

func (f RegistryFixture) Cleanup(ctx context.Context) {
	if f.cleanup != nil {
		f.cleanup(ctx)
	}
}

func CreateHuggingFaceRegistryFixture(ctx context.Context, prefix, url string) (RegistryFixture, error) {
	registriesApi := GetV1alpha1RegistriesApi()
	name := fmt.Sprintf("%s-%s", prefix, GenerateTestProjectName("registry"))
	// registries.name is varchar(64) in DB; keep room for suffixes.
	if len(name) > 60 {
		name = name[:60]
	}

	registryType := v1alpha1registry.HUGGINGFACE_V1alpha1RegistryType
	resp, _, err := registriesApi.RegistriesCreateRegistry(ctx, v1alpha1registry.V1alpha1CreateRegistryRequest{
		Name:        name,
		Description: "e2e test registry",
		Type_:       &registryType,
		Url:         url,
		Insecure:    false,
	})
	if err != nil {
		return RegistryFixture{}, err
	}
	if resp.Registry == nil {
		return RegistryFixture{}, fmt.Errorf("create registry: empty response")
	}

	id := int64(resp.Registry.Id)
	return RegistryFixture{
		ID:   id,
		Name: name,
		cleanup: func(ctx context.Context) {
			_, _, _ = registriesApi.RegistriesDeleteRegistry(ctx, int32(id))
		},
	}, nil
}
