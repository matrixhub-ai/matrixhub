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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/preset"
)

func TestDefaultManifest(t *testing.T) {
	manifest := preset.DefaultManifest()
	require.NoError(t, manifest.Validate())

	registryNames := make([]string, 0, len(manifest.Registries))
	for _, spec := range manifest.Registries {
		registryNames = append(registryNames, spec.Name)
	}
	// Registry IDs follow manifest order, and existing installations rely on these IDs.
	require.Equal(t, []string{"huggingface", "hf-mirror", "hf.m.daocloud"}, registryNames)
	require.Len(t, manifest.Projects, 4)
	require.Len(t, manifest.Models, 13)
	for _, spec := range manifest.Models {
		require.True(t, spec.IsRecommended(), spec.Name)
		require.Positive(t, spec.Size, spec.Name)
		require.Positive(t, spec.ParameterCount, spec.Name)
		require.NotEmpty(t, spec.Labels, spec.Name)
	}

	manifest.Models[0].Labels[0].Name = "changed"
	require.NotEqual(t, "changed", preset.DefaultManifest().Models[0].Labels[0].Name)
}

func TestSpecDefaults(t *testing.T) {
	require.Equal(t, "Qwen", preset.ProjectSpec{Name: "Qwen"}.UpstreamOrganization())
	require.Equal(t, "upstream", preset.ProjectSpec{Name: "Qwen", Organization: "upstream"}.UpstreamOrganization())

	enabled, disabled := true, false
	require.True(t, preset.ModelSpec{}.IsRecommended())
	require.True(t, preset.ModelSpec{Recommended: &enabled}.IsRecommended())
	require.False(t, preset.ModelSpec{Recommended: &disabled}.IsRecommended())
}

func TestManifestValidate(t *testing.T) {
	validManifest := func() *preset.Manifest {
		return &preset.Manifest{
			Registries: []preset.RegistrySpec{{Name: "mirror", URL: "https://mirror.example.com"}},
			Projects:   []preset.ProjectSpec{{Name: "team", Organization: "upstream_org.v2", Registry: "mirror"}},
			Models: []preset.ModelSpec{{
				Project: "team", Name: "Model-1.5B_v2", Size: 1, ParameterCount: 1,
				Labels: []preset.LabelSpec{{Category: "library", Name: "transformers"}, {Category: "other", Name: "transformers"}},
			}},
		}
	}

	for _, test := range []struct {
		name    string
		mutate  func(*preset.Manifest)
		wantErr []string
	}{
		{name: "valid manifest", mutate: func(*preset.Manifest) {}},
		{name: "empty manifest", mutate: func(m *preset.Manifest) { *m = preset.Manifest{} }},
		{
			name: "registry without name or URL",
			mutate: func(m *preset.Manifest) {
				m.Registries = append(m.Registries, preset.RegistrySpec{Name: " "})
			},
			wantErr: []string{"registries[1].name: is required", "registries[1].url: is required"},
		},
		{
			name: "duplicate registry",
			mutate: func(m *preset.Manifest) {
				m.Registries = append(m.Registries, m.Registries[0])
			},
			wantErr: []string{`registries[1].name: duplicate registry "mirror"`},
		},
		{
			name: "long registry name",
			mutate: func(m *preset.Manifest) {
				m.Registries[0].Name = strings.Repeat("r", 65)
				m.Projects[0].Registry = m.Registries[0].Name
			},
			wantErr: []string{"registries[0].name: must be at most 64 characters"},
		},
		{
			name:    "registry URL with an unsupported scheme",
			mutate:  func(m *preset.Manifest) { m.Registries[0].URL = "ftp://mirror.example.com" },
			wantErr: []string{"registries[0].url: must be an absolute http or https URL"},
		},
		{
			name:    "relative registry URL",
			mutate:  func(m *preset.Manifest) { m.Registries[0].URL = "mirror.example.com" },
			wantErr: []string{"registries[0].url: must be an absolute http or https URL"},
		},
		{
			name:    "long registry URL",
			mutate:  func(m *preset.Manifest) { m.Registries[0].URL = "https://example.com/" + strings.Repeat("a", 236) },
			wantErr: []string{"registries[0].url: must be at most 255 characters"},
		},
		{
			name:    "project name outside the project API rules",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Name, m.Models[0].Project = "team_models", "team_models" },
			wantErr: []string{"projects[0].name: must match"},
		},
		{
			name:    "single-character project name",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Name, m.Models[0].Project = "t", "t" },
			wantErr: []string{"projects[0].name: must match"},
		},
		{
			name:    "duplicate project",
			mutate:  func(m *preset.Manifest) { m.Projects = append(m.Projects, m.Projects[0]) },
			wantErr: []string{`projects[1].name: duplicate project "team"`},
		},
		{
			name:    "organization with a path separator",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Organization = "org/team" },
			wantErr: []string{"projects[0].organization: must match"},
		},
		{
			name:    "organization that is a relative path",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Organization = ".." },
			wantErr: []string{"projects[0].organization: must match"},
		},
		{
			name:    "long organization",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Organization = strings.Repeat("o", 65) },
			wantErr: []string{"projects[0].organization: must be at most 64 characters"},
		},
		{
			name:    "project without registry",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Registry = "" },
			wantErr: []string{"projects[0].registry: is required"},
		},
		{
			name:    "project with an unknown registry",
			mutate:  func(m *preset.Manifest) { m.Projects[0].Registry = "Mirror" },
			wantErr: []string{`projects[0].registry: "Mirror" is not defined in the manifest`},
		},
		{
			name:    "model with an unknown project",
			mutate:  func(m *preset.Manifest) { m.Models[0].Project = "other" },
			wantErr: []string{`models[0].project: "other" is not defined in the manifest`},
		},
		{
			name:    "model without project or name",
			mutate:  func(m *preset.Manifest) { m.Models[0].Project, m.Models[0].Name = "", "" },
			wantErr: []string{"models[0].project: is required", "models[0].name: is required"},
		},
		{
			name:    "model name outside the model API rules",
			mutate:  func(m *preset.Manifest) { m.Models[0].Name = "-model" },
			wantErr: []string{"models[0].name: must match"},
		},
		{
			name:    "reserved model name",
			mutate:  func(m *preset.Manifest) { m.Models[0].Name = "resolve" },
			wantErr: []string{`models[0].name: "resolve" is reserved`},
		},
		{
			name:    "long model name",
			mutate:  func(m *preset.Manifest) { m.Models[0].Name = strings.Repeat("m", 252) },
			wantErr: []string{"models[0].name: must be at most 251 characters"},
		},
		{
			name: "model names differing only in case",
			mutate: func(m *preset.Manifest) {
				m.Models = append(m.Models, preset.ModelSpec{Project: "team", Name: "model-1.5b_V2"})
			},
			wantErr: []string{`models[1].name: duplicate model "model-1.5b_V2" in project "team"`},
		},
		{
			name: "negative statistics",
			mutate: func(m *preset.Manifest) {
				m.Models[0].Size, m.Models[0].ParameterCount = -1, -1
			},
			wantErr: []string{"models[0].size: must not be negative", "models[0].parameterCount: must not be negative"},
		},
		{
			name:    "label category in another case",
			mutate:  func(m *preset.Manifest) { m.Models[0].Labels[0].Category = "Library" },
			wantErr: []string{"models[0].labels[0].category: must be one of: library, license, task, language, other"},
		},
		{
			name:    "label without name",
			mutate:  func(m *preset.Manifest) { m.Models[0].Labels[0].Name = "" },
			wantErr: []string{"models[0].labels[0].name: is required"},
		},
		{
			name:    "long label name",
			mutate:  func(m *preset.Manifest) { m.Models[0].Labels[0].Name = strings.Repeat("l", 65) },
			wantErr: []string{"models[0].labels[0].name: must be at most 64 characters"},
		},
		{
			name: "label names differing only in case",
			mutate: func(m *preset.Manifest) {
				m.Models[0].Labels = append(m.Models[0].Labels, preset.LabelSpec{Category: "library", Name: "Transformers"})
			},
			wantErr: []string{"models[0].labels[2].name: duplicate label library:Transformers"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := validManifest()
			test.mutate(manifest)
			err := manifest.Validate()
			if len(test.wantErr) == 0 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, want := range test.wantErr {
				require.ErrorContains(t, err, want)
			}
			require.Len(t, strings.Split(err.Error(), "\n"), len(test.wantErr))
		})
	}
}
