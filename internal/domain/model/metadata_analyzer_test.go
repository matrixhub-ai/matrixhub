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

package model

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestAnalyzeRepoMetadataCountsSingleSafetensorsFile(t *testing.T) {
	files := &git.RepoMetadataFiles{
		SafetensorsFiles: map[string][]byte{
			"model.safetensors": buildSafetensorsFile(t, map[string][]int64{
				"model.embed_tokens.weight": {2, 3},
				"lm_head.weight":            {4, 5},
			}),
		},
	}

	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}

	if metadata.ParameterCount != 26 {
		t.Fatalf("ParameterCount = %d, want 26", metadata.ParameterCount)
	}
}

func TestAnalyzeRepoMetadataCountsShardedSafetensorsFilesWithoutTotalSize(t *testing.T) {
	files := &git.RepoMetadataFiles{
		SafetensorsIndexJSON: []byte(`{
			"weight_map": {
				"model.embed_tokens.weight": "model-00001-of-00002.safetensors",
				"lm_head.weight": "model-00002-of-00002.safetensors"
			}
		}`),
		SafetensorsFiles: map[string][]byte{
			"model-00001-of-00002.safetensors": buildSafetensorsFile(t, map[string][]int64{
				"model.embed_tokens.weight": {2, 3},
			}),
			"model-00002-of-00002.safetensors": buildSafetensorsFile(t, map[string][]int64{
				"lm_head.weight": {4, 5},
			}),
		},
	}

	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}

	if metadata.ParameterCount != 26 {
		t.Fatalf("ParameterCount = %d, want 26", metadata.ParameterCount)
	}
}

func TestAnalyzeRepoMetadataPrefersSafetensorsIndexTotalSize(t *testing.T) {
	files := &git.RepoMetadataFiles{
		ConfigJSON: []byte(`{"torch_dtype": "bfloat16"}`),
		SafetensorsIndexJSON: []byte(`{
			"metadata": {
				"total_size": 100
			}
		}`),
		SafetensorsFiles: map[string][]byte{
			"model-00001-of-00002.safetensors": buildSafetensorsFile(t, map[string][]int64{
				"model.embed_tokens.weight": {2, 3},
			}),
		},
	}

	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}

	if metadata.ParameterCount != 50 {
		t.Fatalf("ParameterCount = %d, want 50", metadata.ParameterCount)
	}
}

func TestAnalyzeRepoMetadataEstimatesFromSafetensorsSizes(t *testing.T) {
	// No index and no readable header: all that is left is the size recorded in
	// the LFS pointer. Sizes are Qwen2.5-0.5B's, whose real count is 494032768.
	files := &git.RepoMetadataFiles{
		ConfigJSON: []byte(`{"torch_dtype": "bfloat16"}`),
		SafetensorsSizes: map[string]int64{
			"model.safetensors": 988097824,
		},
	}

	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}

	if metadata.ParameterCount != 494048912 {
		t.Fatalf("ParameterCount = %d, want 494048912", metadata.ParameterCount)
	}
}

func TestAnalyzeRepoMetadataCombinesHeadersAndSafetensorsSizes(t *testing.T) {
	// A partly-fetched sharded model with no index total_size: one shard's header
	// was readable, the other only yielded its LFS pointer size. Counting just the
	// readable shard would report a fraction of the real parameter count.
	files := &git.RepoMetadataFiles{
		ConfigJSON: []byte(`{"torch_dtype": "bfloat16"}`),
		SafetensorsFiles: map[string][]byte{
			"model-00001-of-00002.safetensors": buildSafetensorsFile(t, map[string][]int64{
				"model.embed_tokens.weight": {2, 3},
				"lm_head.weight":            {4, 5},
			}),
		},
		SafetensorsSizes: map[string]int64{
			"model-00002-of-00002.safetensors": 1024,
		},
	}

	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatalf("AnalyzeRepoMetadata() error = %v", err)
	}

	// 26 exact from the header, plus 1024 bytes of bfloat16 weights.
	if metadata.ParameterCount != 26+512 {
		t.Fatalf("ParameterCount = %d, want %d", metadata.ParameterCount, 26+512)
	}
}

func TestAnalyzeRepoMetadataClassifiesTags(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files *git.RepoMetadataFiles
		want  []ClassifiedTag
	}{
		{
			name: "card fields then config fields without empty or duplicate tags",
			files: &git.RepoMetadataFiles{
				ReadmeContent: []byte(`---
pipeline_tag: text-generation
library_name: transformers
language:
- en
- zh
license: mit
tags:
- llama
- llama
- ""
---
# Model
`),
				ConfigJSON: []byte(`{"model_type":"llama","quantization_config":{"quant_method":"awq"}}`),
			},
			want: []ClassifiedTag{
				{Name: "text-generation", Category: "task"},
				{Name: "transformers", Category: "library"},
				{Name: "en", Category: "language"},
				{Name: "zh", Category: "language"},
				{Name: "mit", Category: "license"},
				{Name: "llama", Category: "other"},
				{Name: "awq", Category: "other"},
			},
		},
		{
			name: "same name in different categories",
			files: &git.RepoMetadataFiles{
				ReadmeContent: []byte("---\nlicense: other\n---\n"),
				ConfigJSON:    []byte(`{"model_type":"other"}`),
			},
			want: []ClassifiedTag{
				{Name: "other", Category: "license"},
				{Name: "other", Category: "other"},
			},
		},
		{
			name: "quantization config without method",
			files: &git.RepoMetadataFiles{
				ConfigJSON: []byte(`{"model_type":"bert","quantization_config":{"bits":8}}`),
			},
			want: []ClassifiedTag{{Name: "bert", Category: "other"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := AnalyzeRepoMetadata(tc.files)

			require.NoError(t, err)
			require.Equal(t, tc.want, metadata.Tags)
		})
	}
}

func TestAnalyzeRepoMetadataIgnoresUnparsableCardAndConfig(t *testing.T) {
	readme := "---\ntags: [unclosed\n---\n# Model\n"
	files := &git.RepoMetadataFiles{
		ReadmeContent:        []byte(readme),
		ConfigJSON:           []byte(`{"torch_dtype":`),
		SafetensorsIndexJSON: []byte(`{"metadata":{"total_size":100}}`),
		Size:                 300,
	}

	metadata, err := AnalyzeRepoMetadata(files)

	require.NoError(t, err)
	require.Equal(t, readme, metadata.ReadmeContent)
	require.EqualValues(t, 300, metadata.Size)
	// An unreadable config keeps the default two bytes per parameter.
	require.EqualValues(t, 50, metadata.ParameterCount)
	require.Empty(t, metadata.Tags)
}

func TestAnalyzeRepoMetadataParameterCount(t *testing.T) {
	sixParameters := safetensorsContent(`{"w":{"shape":[2,3]}}`)

	for _, tc := range []struct {
		name  string
		files *git.RepoMetadataFiles
		want  int64
	}{
		{
			name:  "no weights",
			files: &git.RepoMetadataFiles{ConfigJSON: []byte(`{"torch_dtype":"float16"}`)},
			want:  0,
		},
		{
			name: "quantization bits set bytes per parameter",
			files: &git.RepoMetadataFiles{
				ConfigJSON:           []byte(`{"quantization_config":{"quant_method":"gptq","bits":8}}`),
				SafetensorsIndexJSON: []byte(`{"metadata":{"total_size":1000}}`),
			},
			want: 1000,
		},
		{
			name: "float32 dtype sets four bytes per parameter",
			files: &git.RepoMetadataFiles{
				ConfigJSON:       []byte(`{"torch_dtype":"float32"}`),
				SafetensorsSizes: map[string]int64{"model.safetensors": 400},
			},
			want: 100,
		},
		{
			name: "unsupported dtype keeps two bytes per parameter",
			files: &git.RepoMetadataFiles{
				ConfigJSON:       []byte(`{"torch_dtype":"float64"}`),
				SafetensorsSizes: map[string]int64{"model.safetensors": 400},
			},
			want: 200,
		},
		{
			name: "unreadable index falls back to headers",
			files: &git.RepoMetadataFiles{
				SafetensorsIndexJSON: []byte(`{"metadata":`),
				SafetensorsFiles:     map[string][]byte{"model.safetensors": sixParameters},
			},
			want: 6,
		},
		{
			name: "zero index total size falls back to headers",
			files: &git.RepoMetadataFiles{
				SafetensorsIndexJSON: []byte(`{"metadata":{"total_size":0}}`),
				SafetensorsFiles:     map[string][]byte{"model.safetensors": sixParameters},
			},
			want: 6,
		},
		{
			name: "overflowing header and size estimates report unknown",
			files: &git.RepoMetadataFiles{
				SafetensorsFiles: map[string][]byte{
					"model-00001-of-00002.safetensors": safetensorsContent(`{"w":{"shape":[9223372036854775807]}}`),
				},
				SafetensorsSizes: map[string]int64{"model-00002-of-00002.safetensors": 4},
			},
			want: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := AnalyzeRepoMetadata(tc.files)

			require.NoError(t, err)
			require.Equal(t, tc.want, metadata.ParameterCount)
		})
	}
}

func TestDeduplicateClassifiedTags(t *testing.T) {
	got := deduplicateClassifiedTags([]ClassifiedTag{
		{Name: "llama", Category: "other"},
		{Name: "", Category: "task"},
		{Name: "mit", Category: "license"},
		{Name: "llama", Category: "other"},
		{Name: "mit", Category: "other"},
	})

	require.Equal(t, []ClassifiedTag{
		{Name: "llama", Category: "other"},
		{Name: "mit", Category: "license"},
		{Name: "mit", Category: "other"},
	}, got)
	require.Empty(t, deduplicateClassifiedTags(nil))
}

func TestInferParameterBytes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		want   int64
	}{
		{name: "invalid json", config: `{"torch_dtype":`, want: 0},
		{name: "no dtype information", config: `{}`, want: 0},
		{name: "sub-byte quantization bits round up", config: `{"quantization_config":{"bits":4}}`, want: 1},
		{name: "quantization bits win over torch_dtype", config: `{"quantization_config":{"bits":16},"torch_dtype":"float32"}`, want: 2},
		{name: "quantization w_bit", config: `{"quantization_config":{"w_bit":8}}`, want: 1},
		{name: "quantization weight_bit_width", config: `{"quantization_config":{"weight_bit_width":32}}`, want: 4},
		{name: "quant_method gptq", config: `{"quantization_config":{"quant_method":"gptq"}}`, want: 1},
		{name: "quant_method awq", config: `{"quantization_config":{"quant_method":"awq"}}`, want: 1},
		{name: "quant_method int4", config: `{"quantization_config":{"quant_method":"int4"}}`, want: 1},
		{name: "quant_method nf4", config: `{"quantization_config":{"quant_method":"nf4"}}`, want: 1},
		{name: "quant_method fp4", config: `{"quantization_config":{"quant_method":"fp4"}}`, want: 1},
		{name: "quant_method int8", config: `{"quantization_config":{"quant_method":"int8"}}`, want: 1},
		{name: "quant_method fp8", config: `{"quantization_config":{"quant_method":"fp8"}}`, want: 1},
		{name: "quant_method is case insensitive", config: `{"quantization_config":{"quant_method":"GPTQ"}}`, want: 1},
		{
			name:   "unknown quant_method falls back to torch_dtype",
			config: `{"quantization_config":{"quant_method":"bitsandbytes"},"torch_dtype":"bfloat16"}`,
			want:   2,
		},
		{name: "non-object quantization_config is ignored", config: `{"quantization_config":"fp8","torch_dtype":"float32"}`, want: 4},
		{name: "torch_dtype float32", config: `{"torch_dtype":"float32"}`, want: 4},
		{name: "torch_dtype fp32", config: `{"torch_dtype":"fp32"}`, want: 4},
		{name: "torch_dtype float16", config: `{"torch_dtype":"float16"}`, want: 2},
		{name: "torch_dtype fp16", config: `{"torch_dtype":"fp16"}`, want: 2},
		{name: "torch_dtype bfloat16", config: `{"torch_dtype":"bfloat16"}`, want: 2},
		{name: "torch_dtype bf16", config: `{"torch_dtype":"bf16"}`, want: 2},
		{name: "torch_dtype half", config: `{"torch_dtype":"half"}`, want: 2},
		{name: "torch_dtype int8", config: `{"torch_dtype":"int8"}`, want: 1},
		{name: "torch_dtype uint8", config: `{"torch_dtype":"uint8"}`, want: 1},
		{name: "torch_dtype fp8", config: `{"torch_dtype":"fp8"}`, want: 1},
		{name: "torch_dtype is case insensitive", config: `{"torch_dtype":"BFloat16"}`, want: 2},
		{name: "dtype when torch_dtype is missing", config: `{"dtype":"float32"}`, want: 4},
		{name: "unknown torch_dtype falls back to dtype", config: `{"torch_dtype":"auto","dtype":"int8"}`, want: 1},
		{name: "unsupported dtype", config: `{"torch_dtype":"float64"}`, want: 0},
		{name: "non-string dtype", config: `{"torch_dtype":16}`, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, inferParameterBytes([]byte(tc.config)))
		})
	}
}

func TestExtractBits(t *testing.T) {
	for _, tc := range []struct {
		name     string
		quantCfg map[string]any
		want     int
	}{
		{name: "no bit width", quantCfg: map[string]any{"quant_method": "gptq"}, want: 0},
		{name: "json number bits", quantCfg: map[string]any{"bits": float64(4)}, want: 4},
		{name: "int bits", quantCfg: map[string]any{"bits": 8}, want: 8},
		{name: "bits win over w_bit", quantCfg: map[string]any{"bits": float64(8), "w_bit": float64(4)}, want: 8},
		{name: "non-positive json number falls through", quantCfg: map[string]any{"bits": float64(0), "w_bit": float64(4)}, want: 4},
		{name: "non-positive int falls through", quantCfg: map[string]any{"bits": -1, "weight_bit_width": 16}, want: 16},
		{name: "non-numeric bits are ignored", quantCfg: map[string]any{"bits": "4"}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractBits(tc.quantCfg))
		})
	}
}

func TestAddParameterCounts(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b int64
		want int64
	}{
		{name: "sum", a: 26, b: 512, want: 538},
		{name: "largest representable sum", a: math.MaxInt64, b: 0, want: math.MaxInt64},
		{name: "negative first count", a: -1, b: 1, want: 0},
		{name: "negative second count", a: 1, b: -1, want: 0},
		{name: "overflow", a: math.MaxInt64, b: 1, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, addParameterCounts(tc.a, tc.b))
		})
	}
}

func TestSumSafetensorsSizes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sizes map[string]int64
		want  int64
	}{
		{name: "no sizes", want: 0},
		{name: "positive sizes", sizes: map[string]int64{"a.safetensors": 100, "b.safetensors": 24}, want: 124},
		{name: "zero size", sizes: map[string]int64{"a.safetensors": 100, "b.safetensors": 0}, want: 0},
		{name: "negative size", sizes: map[string]int64{"a.safetensors": -1}, want: 0},
		{name: "overflow", sizes: map[string]int64{"a.safetensors": math.MaxInt64, "b.safetensors": 1}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, sumSafetensorsSizes(tc.sizes))
		})
	}
}

func TestCountSafetensorsParameters(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string][]byte
		want  int64
	}{
		{name: "no files", want: 0},
		{
			name: "unreadable headers are skipped",
			files: map[string][]byte{
				"model-00001-of-00002.safetensors": safetensorsContent(`{"w":{"shape":[2,3]}}`),
				"model-00002-of-00002.safetensors": []byte("not safetensors"),
			},
			want: 6,
		},
		{
			name: "overflow",
			files: map[string][]byte{
				"model-00001-of-00002.safetensors": safetensorsContent(`{"w":{"shape":[9223372036854775807]}}`),
				"model-00002-of-00002.safetensors": safetensorsContent(`{"w":{"shape":[1]}}`),
			},
			want: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, countSafetensorsParameters(tc.files))
		})
	}
}

func TestCountSafetensorsHeaderParameters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content []byte
		want    int64
		wantOK  bool
	}{
		{name: "shorter than length prefix", content: []byte{1, 2, 3}},
		{name: "header length beyond content", content: append(binary.LittleEndian.AppendUint64(nil, 100), "{}"...)},
		{name: "header length beyond int64", content: binary.LittleEndian.AppendUint64(nil, math.MaxUint64)},
		{name: "invalid header json", content: safetensorsContent(`{"w":`)},
		{name: "negative dimension", content: safetensorsContent(`{"w":{"shape":[2,-3]}}`)},
		{name: "dimension product overflow", content: safetensorsContent(`{"w":{"shape":[9223372036854775807,2]}}`)},
		{name: "tensor sum overflow", content: safetensorsContent(`{"a":{"shape":[9223372036854775807]},"b":{"shape":[1]}}`)},
		{name: "empty header", content: safetensorsContent(`{}`), want: 0, wantOK: true},
		{
			name:    "metadata is skipped and scalars count once",
			content: safetensorsContent(`{"__metadata__":{"format":"pt"},"scalar":{"shape":[]},"w":{"shape":[2,3]}}`),
			want:    7,
			wantOK:  true,
		},
		{name: "empty tensor", content: safetensorsContent(`{"w":{"shape":[4,0]}}`), want: 0, wantOK: true},
		{
			name:    "bytes after the header are ignored",
			content: append(safetensorsContent(`{"w":{"shape":[5]}}`), 0, 0, 0, 0),
			want:    5,
			wantOK:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := countSafetensorsHeaderParameters(tc.content)

			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
		})
	}
}

// safetensorsContent prefixes a raw JSON header with its little-endian length,
// which lets tests build malformed headers that buildSafetensorsFile cannot.
func safetensorsContent(header string) []byte {
	return append(binary.LittleEndian.AppendUint64(nil, uint64(len(header))), header...)
}

func buildSafetensorsFile(t *testing.T, tensors map[string][]int64) []byte {
	t.Helper()

	header := map[string]any{
		"__metadata__": map[string]string{"format": "pt"},
	}
	offset := int64(0)
	for name, shape := range tensors {
		count := int64(1)
		for _, dim := range shape {
			count *= dim
		}
		size := count * 2
		header[name] = map[string]any{
			"dtype":        "BF16",
			"shape":        shape,
			"data_offsets": []int64{offset, offset + size},
		}
		offset += size
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	content := make([]byte, 8+len(headerBytes))
	binary.LittleEndian.PutUint64(content[:8], uint64(len(headerBytes)))
	copy(content[8:], headerBytes)
	return content
}
