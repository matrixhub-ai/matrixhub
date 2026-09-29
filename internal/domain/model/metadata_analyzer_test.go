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
	"fmt"
	"math"
	"testing"

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

func TestAnalyzeRepoMetadataIndexedHeaderPrecedence(t *testing.T) {
	index := []byte(`{"metadata":{"total_size":100,"total_parameters":1},"weight_map":{"a":"a.safetensors","b":"b.safetensors"}}`)
	valid := buildSafetensorsFile(t, map[string][]int64{"weight": {2, 3}})
	invalid := rawParameterHeader(`{"weight":{"dtype":"BF16","shape":[2,3],"data_offsets":[0,9999]}}`)
	for _, tt := range []struct {
		name    string
		headers map[string][]byte
		sizes   map[string]int64
		want    int64
	}{
		{"all shards", map[string][]byte{"a.safetensors": valid, "b.safetensors": valid}, nil, 12},
		{"sizes do not double count", map[string][]byte{"a.safetensors": valid, "b.safetensors": valid}, map[string]int64{"a.safetensors": 500, "b.safetensors": 500}, 12},
		{"extra header ignored", map[string][]byte{"a.safetensors": valid, "b.safetensors": valid, "extra.safetensors": valid}, nil, 12},
		{"one missing shard", map[string][]byte{"a.safetensors": valid}, nil, 50},
		{"invalid shard", map[string][]byte{"a.safetensors": valid, "b.safetensors": []byte("invalid")}, nil, 50},
		{"invalid offsets", map[string][]byte{"a.safetensors": valid, "b.safetensors": invalid}, map[string]int64{"b.safetensors": 500}, 50},
		{"unknown size skips offset check", map[string][]byte{"a.safetensors": valid, "b.safetensors": invalid}, nil, 12},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON: []byte(`{"torch_dtype":"bfloat16"}`), SafetensorsIndexJSON: index,
				SafetensorsFiles: tt.headers, SafetensorsSizes: tt.sizes,
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
	for _, weightMap := range []string{`{}`, `null`, `{"a":1}`, `{"a":"../a.safetensors"}`, `{"a":"/a.safetensors"}`} {
		files := &git.RepoMetadataFiles{
			SafetensorsIndexJSON: []byte(fmt.Sprintf(`{"metadata":{"total_size":100},"weight_map":%s}`, weightMap)),
			SafetensorsFiles:     map[string][]byte{"a.safetensors": valid},
		}
		if got := analyzeParametersForTest(t, files); got != 50 {
			t.Fatalf("weight_map %s: count = %d, want 50", weightMap, got)
		}
	}
}

func TestAnalyzeRepoMetadataUnsupportedQuantizationGuard(t *testing.T) {
	index := []byte(`{"metadata":{"total_size":1000},"weight_map":{"weight":"model.safetensors"}}`)
	for _, dtype := range []string{"U8", "I8", "U16", "I16", "U32", "I32", "I64", "BF16"} {
		t.Run(dtype, func(t *testing.T) {
			header := parameterHeader(t, map[string]parameterTestTensor{"layer.weight": parameterTensor(dtype, 8)})
			files := &git.RepoMetadataFiles{
				ConfigJSON:           []byte(`{"torch_dtype":"bfloat16","quantization_config":{"quant_method":"modelopt"}}`),
				SafetensorsIndexJSON: index,
				SafetensorsFiles:     map[string][]byte{"model.safetensors": header},
			}
			want := int64(500)
			if dtype == "I64" || dtype == "BF16" {
				want = 8
			}
			if got := analyzeParametersForTest(t, files); got != want {
				t.Fatalf("indexed count = %d, want %d", got, want)
			}
			files.SafetensorsIndexJSON = nil
			if got := analyzeParametersForTest(t, files); got != 8 {
				t.Fatalf("unindexed count = %d, want 8", got)
			}
		})
	}
	for _, tt := range []struct {
		name    string
		config  string
		tensors map[string]parameterTestTensor
		want    int64
	}{
		{
			"supported fp8 unexpanded integer", `{"quantization_config":{"quant_method":"fp8"}}`,
			map[string]parameterTestTensor{"layer.weight": parameterTensor("I8", 8)}, 8,
		},
		{
			"generic supported expansion", `{"quantization_config":{"quant_method":"custom","bits":4}}`,
			map[string]parameterTestTensor{"layer.weight": parameterTensor("U8", 8)}, 16,
		},
		{
			"skipped integer is not guarded", `{"quantization_config":{"quant_method":"custom"}}`,
			map[string]parameterTestTensor{"norm.weight": parameterTensor("BF16", 8), "layer.weight_shape": parameterTensor("I32", 2)}, 8,
		},
		{
			"scalar integer is not guarded", `{"quantization_config":{"quant_method":"custom"}}`,
			map[string]parameterTestTensor{"norm.weight": parameterTensor("BF16", 8), "scalar": parameterTensor("I32")}, 8,
		},
		{
			"no declared method", `{}`,
			map[string]parameterTestTensor{"layer.weight": parameterTensor("I8", 8)}, 8,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON: []byte(tt.config), SafetensorsIndexJSON: index,
				SafetensorsFiles: map[string][]byte{"model.safetensors": parameterHeader(t, tt.tensors)},
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAnalyzeRepoMetadataEstimatesOnlyUnreadableHeaders(t *testing.T) {
	header := buildSafetensorsFile(t, map[string][]int64{"weight": {2, 3}})
	for _, tt := range []struct {
		name   string
		header []byte
		want   int64
	}{
		{"readable", header, 506},
		{"invalid JSON", []byte("invalid"), 1000},
		{"invalid shape", rawParameterHeader(`{"weight":{"dtype":"F32","shape":[-1],"data_offsets":[0,4]}}`), 1000},
		{"invalid offsets", rawParameterHeader(`{"weight":{"dtype":"F32","shape":[2],"data_offsets":[0,1001]}}`), 1000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON:       []byte(`{"torch_dtype":"bfloat16"}`),
				SafetensorsFiles: map[string][]byte{"local.safetensors": tt.header},
				SafetensorsSizes: map[string]int64{"local.safetensors": 1000, "remote.safetensors": 1000},
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
	files := &git.RepoMetadataFiles{SafetensorsSizes: map[string]int64{"a": 1, "b": 1}}
	if got := analyzeParametersForTest(t, files); got != 1 {
		t.Fatalf("combined size estimate = %d, want 1", got)
	}
}

func TestAnalyzeRepoMetadataParameterEstimates(t *testing.T) {
	for _, tt := range []struct {
		name   string
		config string
		size   int64
		want   int64
	}{
		{"V4 Pro", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, 864704792696, 1627679609780},
		{"V4 Flash", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, 159609485896, 300441385216},
		{"V4 Flash Base", `{"expert_dtype":"fp8","quantization_config":{"quant_method":"fp8"}}`, 294700000000, 294700000000},
		{"quantizer expert", `{"quantization_config":{"quant_method":"fp8","expert_dtype":"nvfp4"}}`, 17, 32},
		{"text expert", `{"text_config":{"expert_dtype":"mxfp4","quantization_config":{"quant_method":"fp8"}}}`, 17, 32},
		{"store dtype", `{"quantization_config":{"quant_method":"fp8","store_dtype":"mxfp4"}}`, 17, 32},
		{"expert precision precedence", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8","expert_dtype":"fp8"}}`, 17, 17},
		{"store precision precedence", `{"expert_dtype":"fp8","quantization_config":{"quant_method":"fp8","store_dtype":"fp4"}}`, 17, 17},
		{"fp8 without expert", `{"quantization_config":{"quant_method":"fp8"}}`, 17, 17},
		{"fp6 is not four bit", `{"expert_dtype":"fp6","quantization_config":{"quant_method":"fp8"}}`, 17, 17},
		{"GPTQ remains one byte", `{"quantization_config":{"quant_method":"gptq","bits":4}}`, 101, 101},
		{"AWQ remains one byte", `{"quantization_config":{"quant_method":"awq","w_bit":4}}`, 101, 101},
		{"bit width alias", `{"quantization_config":{"weight_bit_width":12}}`, 101, 50},
		{"fractional legacy width", `{"quantization_config":{"bits":12.5,"w_bit":4}}`, 101, 50},
		{"legacy first width wins", `{"quantization_config":{"quant_method":"gptq","bits":0.5,"w_bit":16}}`, 101, 101},
		{"bits precede method", `{"quantization_config":{"quant_method":"fp8","bits":16}}`, 101, 50},
		{"FP32", `{"torch_dtype":"float32"}`, 101, 25},
		{"BF16", `{"torch_dtype":"bfloat16"}`, 101, 50},
		{"dtype fallback", `{"dtype":"uint8"}`, 101, 101},
		{"missing config", "", 101, 50},
		{"invalid config", "{", 101, 50},
		{"mixed floor division", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, 1, 1},
		{"overflow", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, math.MaxInt64, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON:           []byte(tt.config),
				SafetensorsIndexJSON: []byte(fmt.Sprintf(`{"metadata":{"total_size":%d}}`, tt.size)),
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("index estimate = %d, want %d", got, tt.want)
			}
			files.SafetensorsIndexJSON = nil
			files.SafetensorsSizes = map[string]int64{"model.safetensors": tt.size}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("size estimate = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAnalyzeRepoMetadataRejectsHeaderCountOverflow(t *testing.T) {
	files := &git.RepoMetadataFiles{
		ConfigJSON:           []byte(`{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`),
		SafetensorsIndexJSON: []byte(`{"metadata":{"total_size":100},"weight_map":{"a":"model.safetensors"}}`),
		SafetensorsFiles: map[string][]byte{
			"model.safetensors": parameterHeader(t, map[string]parameterTestTensor{
				"model.experts.0.weight": parameterTensor("I8", 1<<31, 1<<31),
			}),
		},
	}
	if got := analyzeParametersForTest(t, files); got != 188 {
		t.Fatalf("ParameterCount = %d, want index fallback 188", got)
	}
	files.SafetensorsIndexJSON = nil
	if got := analyzeParametersForTest(t, files); got != 0 {
		t.Fatalf("ParameterCount = %d, want unknown 0", got)
	}
}

func TestAnalyzeRepoMetadataRejectsAggregateOverflow(t *testing.T) {
	large := parameterTensor("BF16", 1<<31, 1<<31)
	for _, headers := range []map[string][]byte{
		{"a.safetensors": parameterHeader(t, map[string]parameterTestTensor{"a": large, "b": large})},
		{
			"a.safetensors": parameterHeader(t, map[string]parameterTestTensor{"a": large}),
			"b.safetensors": parameterHeader(t, map[string]parameterTestTensor{"b": large}),
		},
	} {
		files := &git.RepoMetadataFiles{SafetensorsFiles: headers}
		if got := analyzeParametersForTest(t, files); got != 0 {
			t.Fatalf("overflowing header total = %d, want 0", got)
		}
		files.SafetensorsIndexJSON = []byte(`{"metadata":{"total_size":100},"weight_map":{"a":"a.safetensors","b":"b.safetensors"}}`)
		if got := analyzeParametersForTest(t, files); got != 50 {
			t.Fatalf("overflowing indexed total = %d, want estimate 50", got)
		}
	}
	files := &git.RepoMetadataFiles{
		SafetensorsFiles: map[string][]byte{"local.safetensors": buildSafetensorsFile(t, map[string][]int64{"weight": {2}})},
		SafetensorsSizes: map[string]int64{"missing-a.safetensors": math.MaxInt64, "missing-b.safetensors": 1},
	}
	if got := analyzeParametersForTest(t, files); got != 0 {
		t.Fatalf("overflowing missing size total = %d, want 0", got)
	}
}

func TestAnalyzeRepoMetadataCombinesPackedHeadersAndMixedPrecisionEstimate(t *testing.T) {
	files := &git.RepoMetadataFiles{
		ConfigJSON: []byte(`{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`),
		SafetensorsFiles: map[string][]byte{
			"local.safetensors": parameterHeader(t, map[string]parameterTestTensor{"model.experts.0.weight": parameterTensor("I8", 8)}),
		},
		SafetensorsSizes: map[string]int64{"local.safetensors": 1000, "missing.safetensors": 17},
	}
	if got := analyzeParametersForTest(t, files); got != 48 {
		t.Fatalf("combined count = %d, want 16 exact + 32 estimated", got)
	}
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
