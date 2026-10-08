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

package preset

import "context"

type RegistrySpec struct {
	ID          int
	Name        string
	Description string
	URL         string
}

type ProjectSpec struct {
	Name         string
	Organization string
	Registry     string
}

type ModelSpec struct {
	Project        string
	Name           string
	Recommended    bool
	Size           int64
	ParameterCount int64
	Labels         []LabelSpec
}

type LabelSpec struct {
	Category string
	Name     string
}

type Manifest struct {
	Registries []RegistrySpec
	Projects   []ProjectSpec
	Models     []ModelSpec
}

// DefaultManifest returns independent seed data; upgrades do not backfill it.
// Sizes include all upstream files; parameter counts come from safetensors.total.
func DefaultManifest() *Manifest {
	return &Manifest{
		Registries: []RegistrySpec{
			{ID: 1, Name: "huggingface", Description: "Official Hugging Face Hub", URL: "https://huggingface.co"},
			{ID: 2, Name: "hf-mirror", Description: "Hugging Face community mirror", URL: "https://hf-mirror.com"},
			{ID: 3, Name: "hf.m.daocloud", Description: "DaoCloud Hugging Face mirror", URL: "https://hf.m.daocloud.io"},
		},
		Projects: []ProjectSpec{
			{Name: "deepseek-ai", Organization: "deepseek-ai", Registry: "hf.m.daocloud"},
			{Name: "Qwen", Organization: "Qwen", Registry: "hf.m.daocloud"},
			{Name: "MiniMaxAI", Organization: "MiniMaxAI", Registry: "hf.m.daocloud"},
			{Name: "zai-org", Organization: "zai-org", Registry: "hf.m.daocloud"},
		},
		Models: []ModelSpec{
			{
				Project: "deepseek-ai", Name: "DeepSeek-V4-Pro", Recommended: true,
				Size: 864739856856, ParameterCount: 1598839674782,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "license", Name: "mit"},
					{Category: "other", Name: "deepseek_v4"},
					{Category: "other", Name: "fp8"},
				},
			},
			{
				Project: "deepseek-ai", Name: "DeepSeek-V4-Flash", Recommended: true,
				Size: 159630041626, ParameterCount: 290944616402,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "license", Name: "mit"},
					{Category: "other", Name: "deepseek_v4"},
					{Category: "other", Name: "fp8"},
				},
			},
			{
				Project: "deepseek-ai", Name: "DeepSeek-V4-Flash-Base", Recommended: true,
				Size: 294692678949, ParameterCount: 292021347282,
				Labels: []LabelSpec{
					{Category: "other", Name: "deepseek_v4"},
					{Category: "other", Name: "fp8"},
				},
			},
			{
				Project: "deepseek-ai", Name: "DeepSeek-OCR", Recommended: true,
				Size: 6684381812, ParameterCount: 3336106240,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "license", Name: "mit"},
					{Category: "task", Name: "image-text-to-text"},
					{Category: "language", Name: "multilingual"},
					{Category: "other", Name: "deepseek"},
					{Category: "other", Name: "vision-language"},
					{Category: "other", Name: "ocr"},
					{Category: "other", Name: "custom_code"},
					{Category: "other", Name: "deepseek_vl_v2"},
				},
			},
			{
				Project: "Qwen", Name: "Qwen3.8-27B", Recommended: true,
				Size: 55586114863, ParameterCount: 27781427952,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "task", Name: "image-text-to-text"},
					{Category: "license", Name: "apache-2.0"},
					{Category: "other", Name: "qwen3_5"},
				},
			},
			{
				Project: "Qwen", Name: "Qwen3.8-2.4T-A95B", Recommended: true,
				Size: 4892388741252, ParameterCount: 2446182725504,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "task", Name: "text-generation"},
					{Category: "license", Name: "other"},
					{Category: "other", Name: "qwen3_5_moe_text"},
				},
			},
			{
				Project: "Qwen", Name: "Qwen-Image-Bench", Recommended: true,
				Size: 54733735650, ParameterCount: 27356728560,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "task", Name: "image-text-to-text"},
					{Category: "license", Name: "apache-2.0"},
					{Category: "other", Name: "qwen3_5"},
					{Category: "language", Name: "en"},
					{Category: "language", Name: "zh"},
					{Category: "other", Name: "judge-model"},
					{Category: "other", Name: "text-to-image"},
					{Category: "other", Name: "evaluation"},
					{Category: "other", Name: "benchmark"},
					{Category: "other", Name: "qwen"},
				},
			},
			{
				Project: "Qwen", Name: "Qwen-AgentWorld-35B-A3B", Recommended: true,
				Size: 69344322030, ParameterCount: 34660610688,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "license", Name: "apache-2.0"},
					{Category: "task", Name: "text-generation"},
					{Category: "other", Name: "qwen"},
					{Category: "other", Name: "world-model"},
					{Category: "other", Name: "agent"},
					{Category: "other", Name: "environment-simulation"},
					{Category: "other", Name: "qwen3_5_moe"},
				},
			},
			{
				Project: "MiniMaxAI", Name: "MiniMax-H3", Recommended: true,
				Size: 498474749480, ParameterCount: 33122992896,
				Labels: []LabelSpec{
					{Category: "license", Name: "other"},
					{Category: "task", Name: "image-text-to-video"},
					{Category: "library", Name: "minimax-h3"},
					{Category: "other", Name: "text-to-video"},
					{Category: "other", Name: "image-to-video"},
					{Category: "other", Name: "image-text-to-video"},
					{Category: "other", Name: "video-to-video"},
					{Category: "other", Name: "text-to-audio-video"},
					{Category: "other", Name: "image-to-audio-video"},
					{Category: "other", Name: "image-text-to-audio-video"},
					{Category: "other", Name: "video-to-audio-video"},
					{Category: "other", Name: "audio-to-audio-video"},
					{Category: "other", Name: "audio-video-generation"},
					{Category: "other", Name: "multimodal"},
					{Category: "other", Name: "synchronized-audio-video"},
					{Category: "other", Name: "reference-to-audio-video"},
					{Category: "other", Name: "diffusers"},
				},
			},
			{
				Project: "MiniMaxAI", Name: "MiniMax-M3", Recommended: true,
				Size: 854200504173, ParameterCount: 427040140160,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "task", Name: "image-text-to-text"},
					{Category: "license", Name: "other"},
					{Category: "other", Name: "multimodal"},
					{Category: "other", Name: "agent"},
					{Category: "other", Name: "moe"},
					{Category: "other", Name: "coding"},
					{Category: "other", Name: "video"},
					{Category: "other", Name: "minimax_m3_vl"},
				},
			},
			{
				Project: "MiniMaxAI", Name: "MiniMax-M3-MXFP8", Recommended: true,
				Size: 443776005285, ParameterCount: 440279845760,
				Labels: []LabelSpec{
					{Category: "library", Name: "transformers"},
					{Category: "task", Name: "image-text-to-text"},
					{Category: "license", Name: "other"},
					{Category: "other", Name: "multimodal"},
					{Category: "other", Name: "agent"},
					{Category: "other", Name: "moe"},
					{Category: "other", Name: "coding"},
					{Category: "other", Name: "video"},
					{Category: "other", Name: "minimax_m3_vl"},
					{Category: "other", Name: "mxfp8"},
				},
			},
			{
				Project: "zai-org", Name: "GLM-5.3", Recommended: true,
				Size: 755663689206, ParameterCount: 753329940480,
				Labels: []LabelSpec{
					{Category: "task", Name: "text-generation"},
					{Category: "library", Name: "transformers"},
					{Category: "language", Name: "en"},
					{Category: "language", Name: "zh"},
					{Category: "license", Name: "other"},
					{Category: "other", Name: "glm_moe_dsa"},
					{Category: "other", Name: "fp8"},
				},
			},
			{
				Project: "zai-org", Name: "GLM-5.3-Flash", Recommended: true,
				Size: 328366173469, ParameterCount: 321323031390,
				Labels: []LabelSpec{
					{Category: "task", Name: "image-text-to-text"},
					{Category: "library", Name: "transformers"},
					{Category: "language", Name: "en"},
					{Category: "language", Name: "zh"},
					{Category: "license", Name: "mit"},
					{Category: "other", Name: "glm5_next"},
					{Category: "other", Name: "fp8"},
				},
			},
		},
	}
}

//go:generate go tool mockgen -source=preset.go -destination=mocks/preset_repo_mock.go -package=mocks
type IPresetRepo interface {
	// SeedIfEmpty atomically inserts all presets only if registries and projects
	// are both empty. A concurrent seed's primary-key conflict is a no-op.
	SeedIfEmpty(ctx context.Context, manifest *Manifest) (applied bool, err error)
}
