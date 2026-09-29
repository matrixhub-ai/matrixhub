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

type parameterTestTensor struct {
	Dtype   string  `json:"dtype"`
	Shape   []int64 `json:"shape"`
	Offsets []int64 `json:"data_offsets"`
}

func parameterTensor(dtype string, shape ...int64) parameterTestTensor {
	if shape == nil {
		shape = []int64{}
	}
	return parameterTestTensor{Dtype: dtype, Shape: shape, Offsets: []int64{0, 0}}
}

func parameterHeader(t *testing.T, tensors map[string]parameterTestTensor) []byte {
	t.Helper()
	header, err := json.Marshal(tensors)
	if err != nil {
		t.Fatal(err)
	}
	return rawParameterHeader(string(header))
}

func rawParameterHeader(header string) []byte {
	content := make([]byte, 8+len(header))
	binary.LittleEndian.PutUint64(content, uint64(len(header)))
	copy(content[8:], header)
	return content
}

func parameterRulesForTest(t *testing.T, config string) parameterRules {
	t.Helper()
	var raw parameterConfig
	if config != "" {
		if err := json.Unmarshal([]byte(config), &raw); err != nil {
			t.Fatal(err)
		}
	}
	rules, err := resolveParameterRules(raw)
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func analyzeParametersForTest(t *testing.T, files *git.RepoMetadataFiles) int64 {
	t.Helper()
	metadata, err := AnalyzeRepoMetadata(files)
	if err != nil {
		t.Fatal(err)
	}
	return metadata.ParameterCount
}

func TestSafetensorsParameterCounting(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		tensors map[string]parameterTestTensor
		want    int64
	}{
		{
			name: "native sub-byte dtypes",
			tensors: map[string]parameterTestTensor{
				"f4": parameterTensor("F4", 10, 20), "fp4": parameterTensor("FP4", 100, 200),
				"f6a": parameterTensor("F6_E2M3", 5, 10), "f6b": parameterTensor("F6_E3M2", 8, 12),
			},
			want: 20346,
		},
		{
			name: "exponent-only dtypes without config",
			tensors: map[string]parameterTestTensor{
				"weight": parameterTensor("F4", 10, 20), "a": parameterTensor("E8M0", 4, 6),
				"b": parameterTensor("UE8", 50, 100), "c": parameterTensor("F8_E8M0", 7, 8),
			},
			want: 200,
		},
		{
			name: "scalars and zero dimensions",
			tensors: map[string]parameterTestTensor{
				"scalar": parameterTensor("F32"), "empty": parameterTensor("F32", 0, 5),
				"empty_last": parameterTensor("F32", 1<<32, 1<<32, 0), "weight": parameterTensor("F32", 3, 4),
			},
			want: 12,
		},
		{
			name:   "bare scale names remain learned parameters",
			config: `{"quantization_config":{"quant_method":"fp8"}}`,
			tensors: map[string]parameterTestTensor{
				"weight": parameterTensor("F8_E4M3", 8), "layer.scale": parameterTensor("F32", 3),
				"layer.scales": parameterTensor("F32", 4), "layer.weight_scale_inv": parameterTensor("F32", 5),
			},
			want: 15,
		},
		{
			name:   "mxfp4 blocks and scales",
			config: `{"quantization_config":{"quant_method":"mxfp4"}}`,
			tensors: map[string]parameterTestTensor{
				"mlp.weight_blocks": parameterTensor("U8", 16), "mlp.weight_scales": parameterTensor("U8", 2),
				"learned.weight_scales": parameterTensor("F32", 3), "other.weight": parameterTensor("U8", 5),
			},
			want: 40,
		},
		{
			name:   "bitsandbytes quantization state",
			config: `{"quantization_config":{"quant_method":"bitsandbytes","load_in_4bit":true}}`,
			tensors: map[string]parameterTestTensor{
				"layer.weight": parameterTensor("U8", 8), "norm.weight": parameterTensor("BF16", 2),
				"layer.weight.absmax": parameterTensor("U8", 3), "layer.weight.quant_map": parameterTensor("F32", 4),
				"layer.weight.nested_absmax": parameterTensor("U8", 5), "layer.weight.nested_quant_map": parameterTensor("F32", 6),
				"layer.weight.quant_state.bitsandbytes__nf4": parameterTensor("U8", 7),
				"layer.weight.quant_state.bitsandbytes__fp4": parameterTensor("U8", 8),
			},
			want: 18,
		},
		{
			name:   "GPTQ qweights and excluded auxiliaries",
			config: `{"quantization_config":{"quant_method":"gptq","bits":4,"modules_to_not_convert":["dense"]}}`,
			tensors: map[string]parameterTestTensor{
				"quantized.qweight": parameterTensor("I32", 12), "quantized.qzeros": parameterTensor("I32", 3),
				"quantized.g_idx": parameterTensor("I32", 4), "quantized.scales": parameterTensor("F16", 5),
				"dense.scales": parameterTensor("F16", 2), "dense.qweight": parameterTensor("I32", 4),
			},
			want: 102,
		},
		{
			name:   "AWQ defaults to four-bit weights",
			config: `{"quantization_config":{"quant_method":"awq"}}`,
			tensors: map[string]parameterTestTensor{
				"layer.qweight": parameterTensor("I32", 12), "layer.scales": parameterTensor("F16", 3),
				"norm.weight": parameterTensor("F16", 2),
			},
			want: 98,
		},
		{
			name:    "GPTQ retains whole values per container",
			config:  `{"quantization_config":{"quant_method":"gptq","bits":3}}`,
			tensors: map[string]parameterTestTensor{"layer.qweight": parameterTensor("I32", 3)},
			want:    30,
		},
		{
			name:   "compressed tensors exact three-bit ratio",
			config: `{"quantization_config":{"quant_method":"compressed-tensors","format":"pack-quantized","config_groups":{"g":{"weights":{"num_bits":3}}}}}`,
			tensors: map[string]parameterTestTensor{
				"one.weight_packed": parameterTensor("I32", 3), "two.weight_packed": parameterTensor("I32", 1),
				"one.weight_shape": parameterTensor("I32", 2), "one.weight_g_idx": parameterTensor("I32", 4),
				"one.weight_scale": parameterTensor("BF16", 2),
			},
			want: 43,
		},
		{
			name:    "compressed tensors format encodes width",
			config:  `{"quantization_config":{"quant_method":"compressed-tensors","format":"NVFP4-pack-quantized"}}`,
			tensors: map[string]parameterTestTensor{"layer.weight_packed": parameterTensor("U8", 7)},
			want:    14,
		},
		{
			name: "compressed tensors targeted widths and ignore",
			config: `{"quantization_config":{"quant_method":"compressed-tensors","format":"pack-quantized","ignore":["re:.*self_attn.*"],
				"config_groups":{"z":{"targets":["re:.*lm_head$"],"weights":{"num_bits":8}},"a":{"targets":["model.embed_tokens"],"weights":{"num_bits":4}}}}}`,
			tensors: map[string]parameterTestTensor{
				"model.lm_head.weight_packed":                   parameterTensor("I32", 2),
				"model.embed_tokens.weight_packed":              parameterTensor("I32", 2),
				"model.layers.0.self_attn.q_proj.weight_packed": parameterTensor("I32", 2),
				"model.other.weight_packed":                     parameterTensor("I32", 2),
			},
			want: 34,
		},
		{
			name: "targeted group format overrides global format",
			config: `{"quantization_config":{"quant_method":"compressed-tensors","format":"float-quantized",
				"config_groups":{"g":{"targets":["layer"],"format":"mxfp4-pack-quantized"}}}}`,
			tensors: map[string]parameterTestTensor{
				"layer.weight_packed": parameterTensor("I8", 8), "other.weight": parameterTensor("I8", 8),
			},
			want: 24,
		},
		{
			name:    "quantization method names are case insensitive",
			config:  `{"expert_dtype":"FP4","quantization_config":{"quant_method":"FP8"}}`,
			tensors: map[string]parameterTestTensor{"model.experts.0.weight": parameterTensor("I8", 8)},
			want:    16,
		},
		{
			name:   "default U8 packing",
			config: `{"quantization_config":{"quant_method":"custom","bits":4}}`,
			tensors: map[string]parameterTestTensor{
				"weight": parameterTensor("U8", 8), "other": parameterTensor("I8", 8),
			},
			want: 24,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzeParametersForTest(t, &git.RepoMetadataFiles{
				ConfigJSON:       []byte(tt.config),
				SafetensorsFiles: map[string][]byte{"model.safetensors": parameterHeader(t, tt.tensors)},
			})
			if got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSafetensorsQuantizationAuxiliarySuffixes(t *testing.T) {
	suffixes := []string{
		"weight_scale", "weight_scale_inv", "weight_scale_2", "weight_global_scale",
		"weight_zero_point", "input_scale", "input_global_scale", "input_zero_point",
		"zero_point", "weight_shape", "weight_g_idx",
	}
	for _, suffix := range suffixes {
		t.Run(suffix, func(t *testing.T) {
			header := parameterHeader(t, map[string]parameterTestTensor{
				"layer.weight": parameterTensor("I32", 2), "layer." + suffix: parameterTensor("I32", 5),
			})
			for _, config := range []string{"", `{"quantization_config":{}}`} {
				want := int64(7)
				if config != "" {
					want = 2
				}
				got := analyzeParametersForTest(t, &git.RepoMetadataFiles{
					ConfigJSON: []byte(config), SafetensorsFiles: map[string][]byte{"model.safetensors": header},
				})
				if got != want {
					t.Fatalf("config %s: count = %d, want %d", config, got, want)
				}
			}
		})
	}
}

func TestDeepSeekV4HeaderParameterCount(t *testing.T) {
	header := parameterHeader(t, map[string]parameterTestTensor{
		"layers.0.ffn.experts.0.w1.weight":      parameterTensor("I8", 3072, 3584),
		"layers.0.ffn.experts.0.w1.scale":       parameterTensor("F8_E8M0", 3072, 224),
		"layers.0.ffn.shared_experts.w1.weight": parameterTensor("F8_E4M3", 3072, 7168),
		"layers.0.ffn.shared_experts.w1.scale":  parameterTensor("F8_E8M0", 24, 56),
		"layers.0.attn.wkv.weight":              parameterTensor("F8_E4M3", 512, 7168),
		"layers.0.attn.wkv.scale":               parameterTensor("F8_E8M0", 4, 56),
		"layers.0.attn_norm.weight":             parameterTensor("BF16", 7168),
		"layers.0.attn.attn_sink":               parameterTensor("F32", 128),
		"layers.0.ffn.gate.tid2eid":             parameterTensor("I64", 129280, 6),
	})
	for _, tt := range []struct {
		expert string
		want   int64
	}{{"fp4", 48_493_184}, {"fp8", 37_483_136}} {
		t.Run(tt.expert, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON:       []byte(fmt.Sprintf(`{"expert_dtype":%q,"quantization_config":{"quant_method":"fp8","fmt":"e4m3","scale_fmt":"ue8m0","weight_block_size":[128,128]}}`, tt.expert)),
				SafetensorsFiles: map[string][]byte{"model.safetensors": header},
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMLXSafetensorsParameters(t *testing.T) {
	tests := []struct {
		name   string
		config string
		packed int64
		want   int64
	}{
		{"four bit", `{"quantization":{"mode":"affine","bits":4,"group_size":64}}`, 12, 103},
		{"six bit", `{"quantization":{"mode":"affine","bits":6,"group_size":64}}`, 18, 103},
		{"eight bit", `{"quantization":{"mode":"affine","bits":8,"group_size":64}}`, 24, 103},
		{"legacy", `{"quantization_config":{"bits":4,"group_size":64}}`, 12, 103},
		{"text config", `{"text_config":{"quantization":{"bits":4,"group_size":64}}}`, 12, 103},
		{"module override", `{"quantization":{"bits":8,"quantized":{"bits":4},"dense":false}}`, 12, 103},
		{"independent override defaults", `{"quantization":{"bits":8,"quantized":{"group_size":32}}}`, 12, 103},
		{"invalid global with valid module", `{"quantization":{"bits":4,"group_size":16,"quantized":{"mode":"nvfp4"}}}`, 12, 106},
		{"mxfp4", `{"quantization":{"mode":"mxfp4"}}`, 12, 106},
		{"nvfp4", `{"quantization":{"mode":"nvfp4"}}`, 12, 106},
		{"mxfp8", `{"quantization":{"mode":"mxfp8"}}`, 24, 106},
		{"disabled module", `{"quantization":{"bits":4,"quantized":false}}`, 12, 25},
		{"unsupported width", `{"quantization_config":{"quant_method":"mlx","bits":1}}`, 12, 25},
		{"malformed mode", `{"quantization_config":{"quant_method":"mlx","bits":4,"mode":1}}`, 12, 25},
		{"null mode", `{"quantization_config":{"quant_method":"mlx","bits":4,"mode":null}}`, 12, 25},
		{"invalid group size", `{"quantization_config":{"quant_method":"mlx","bits":4,"group_size":16}}`, 12, 25},
		{"fractional width", `{"quantization_config":{"quant_method":"mlx","bits":3.5}}`, 12, 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON: []byte(tt.config),
				SafetensorsFiles: map[string][]byte{"model.safetensors": parameterHeader(t, map[string]parameterTestTensor{
					"quantized.weight": parameterTensor("U32", tt.packed),
					"quantized.scales": parameterTensor("BF16", 3),
					"quantized.biases": parameterTensor("BF16", 3),
					"quantized.bias":   parameterTensor("BF16", 2),
					"dense.weight":     parameterTensor("BF16", 5),
				})},
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMLXRequiresSerializedStateAcrossShards(t *testing.T) {
	for _, tt := range []struct {
		name   string
		scales bool
		biases bool
		want   int64
	}{
		{"no state", false, false, 21},
		{"missing zero points", true, false, 24},
		{"missing scales", false, true, 24},
		{"complete state", true, true, 105},
	} {
		t.Run(tt.name, func(t *testing.T) {
			aux := map[string]parameterTestTensor{
				"learned.scales": parameterTensor("BF16", 4),
				"learned.biases": parameterTensor("BF16", 5),
			}
			if tt.scales {
				aux["quantized.scales"] = parameterTensor("BF16", 3)
			}
			if tt.biases {
				aux["quantized.biases"] = parameterTensor("BF16", 3)
			}
			files := &git.RepoMetadataFiles{
				ConfigJSON: []byte(`{"quantization":{"bits":4,"quantized":{"bits":4}}}`),
				SafetensorsFiles: map[string][]byte{
					"a.safetensors": parameterHeader(t, map[string]parameterTestTensor{"quantized.weight": parameterTensor("U32", 12)}),
					"b.safetensors": parameterHeader(t, aux),
				},
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestQuantizationConfigResolution(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   int64
	}{
		{"quantizer expert", `{"quantization_config":{"quant_method":"fp8","expert_dtype":"fp4"}}`, 16},
		{"model expert", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, 16},
		{"text quantizer", `{"text_config":{"quantization_config":{"quant_method":"fp8","expert_dtype":"fp4"}}}`, 16},
		{"text expert", `{"text_config":{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}}`, 16},
		{"quantizer precedence", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8","expert_dtype":"fp8"}}`, 8},
		{"root expert precedence", `{"expert_dtype":"fp8","text_config":{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}}`, 8},
		{"store dtype", `{"quantization_config":{"quant_method":"fp8","store_dtype":"mxfp4"}}`, 16},
		{"model before store", `{"expert_dtype":"fp8","quantization_config":{"quant_method":"fp8","store_dtype":"mxfp4"}}`, 8},
		{"quantizer before store", `{"quantization_config":{"quant_method":"fp8","expert_dtype":"fp8","store_dtype":"mxfp4"}}`, 8},
		{"malformed expert does not fall through", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8","expert_dtype":4}}`, 8},
		{"null expert falls through", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8","expert_dtype":null}}`, 16},
		{"malformed store", `{"quantization_config":{"quant_method":"fp8","store_dtype":{}}}`, 8},
		{"no expert", `{"quantization_config":{"quant_method":"fp8"}}`, 8},
		{"fp6 is native", `{"expert_dtype":"fp6","quantization_config":{"quant_method":"fp8"}}`, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := &git.RepoMetadataFiles{
				ConfigJSON: []byte(tt.config),
				SafetensorsFiles: map[string][]byte{"model.safetensors": parameterHeader(t, map[string]parameterTestTensor{
					"model.experts.0.weight": parameterTensor("I8", 8),
				})},
			}
			if got := analyzeParametersForTest(t, files); got != tt.want {
				t.Fatalf("ParameterCount = %d, want %d", got, tt.want)
			}
		})
	}

	files := &git.RepoMetadataFiles{
		ConfigJSON: []byte(`{"quantization":{"quant_method":"gptq","bits":2,"group_size":64,"mode":"affine"},"quantization_config":{"quant_method":"gptq","bits":4}}`),
		SafetensorsFiles: map[string][]byte{"model.safetensors": parameterHeader(t, map[string]parameterTestTensor{
			"quantized.qweight": parameterTensor("I32", 10), "quantized.scales": parameterTensor("F16", 2),
		})},
	}
	if got := analyzeParametersForTest(t, files); got != 80 {
		t.Fatalf("Non-MLX quantization config was relabeled: count = %d, want 80", got)
	}
}

func TestParameterMultipliers(t *testing.T) {
	const expert = "model.layers.0.ffn.experts.3.w1.weight"
	tests := []struct {
		name   string
		dtype  string
		config string
		want   parameterRatio
	}{
		{expert, "I8", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{8, 4}},
		{expert, "U8", `{"quantization_config":{"quant_method":"fp8","store_dtype":"mxfp4"}}`, parameterRatio{8, 4}},
		{expert, "I16", `{"expert_dtype":"nvfp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{16, 4}},
		{expert, "I32", `{"expert_dtype":"int2","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{32, 2}},
		{expert, "F8_E4M3", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{expert, "BF16", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{expert, "F6_E2M3", `{"expert_dtype":"fp6","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{"model.ffn.shared_experts.w1.weight", "I8", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{"model.ffn.shared_experts.experts.0.weight", "I8", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{"model.ffn.gate.tid2eid", "I32", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{"model.attn.wkv.weight", "I8", `{"expert_dtype":"fp4","quantization_config":{"quant_method":"fp8"}}`, parameterRatio{1, 1}},
		{"layer.weight", "U8", `{"quantization_config":{"quant_method":"gptq","bits":2}}`, parameterRatio{8, 2}},
		{"layer.weight", "U8", `{"quantization_config":{"quant_method":"awq","bits":4}}`, parameterRatio{8, 4}},
		{"layer.weight", "U8", `{"quantization_config":{"quant_method":"bitsandbytes","load_in_8bit":true}}`, parameterRatio{1, 1}},
		{"layer.weight", "U8", `{"quantization_config":{"quant_method":"other","load_in_4bit":true}}`, parameterRatio{2, 1}},
		{"layer.weight", "I8", `{"quantization_config":{"quant_method":"compressed-tensors","format":"pack-quantized","config_groups":{"g":{"weights":{"num_bits":8}}}}}`, parameterRatio{1, 1}},
		{"layer.weight", "I32", `{"quantization_config":{"quant_method":"compressed-tensors","format":"pack-quantized","config_groups":{"g":{"weights":{"num_bits":5}}}}}`, parameterRatio{32, 5}},
		{"layer.weight", "I32", `{"quantization_config":{"quant_method":"compressed-tensors","format":"pack-quantized","config_groups":{"g":{"weights":{"num_bits":-1}}}}}`, parameterRatio{1, 1}},
	}
	for index, tt := range tests {
		t.Run(fmt.Sprintf("%d/%s/%s", index, tt.name, tt.dtype), func(t *testing.T) {
			rules := parameterRulesForTest(t, tt.config)
			if got := rules.multiplier(tt.name, tt.dtype, nil); got != tt.want {
				t.Fatalf("multiplier = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParameterGlobMatch(t *testing.T) {
	for _, tt := range []struct {
		pattern, name string
		want          bool
	}{
		{"foo", "foo", true}, {"foo", "foobar", false}, {"foo", "xfoo", false}, {"foo", "xfoox", false},
		{"*.txt", "file.txt", true}, {"*.txt", ".txt", true}, {"*.txt", "file.txt.bak", false}, {"*.txt", "txt", false},
		{"model.*", "model.bin", true}, {"model.*", "model.", true}, {"model.*", "my_model.bin", false},
		{"*layer*", "model.layer.weight", true}, {"*layer*", "layer", true}, {"*layer*", "no_match", false},
		{"a*b*c", "abc", true}, {"a*b*c", "aXXbYYc", true}, {"a*b*c", "aXXbYY", false}, {"a*b*c", "XXbYYc", false},
		{"*", "anything", true}, {"*", "", true}, {"**", "", true}, {"a*a", "a", false}, {"*ab*b", "ab", false},
		{"lm_head", "lm_head", true}, {"lm_head", "model.lm_head", false}, {"*lm_head*", "model.lm_head.weight", true},
		{"a[0]*", "a[0].weight", true}, {"a?*", "ab", false},
	} {
		t.Run(tt.pattern+"/"+tt.name, func(t *testing.T) {
			if got := parameterGlobMatch(tt.pattern, tt.name); got != tt.want {
				t.Fatalf("match = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompressedTensorTargets(t *testing.T) {
	for _, tt := range []struct {
		target, module string
		want           bool
	}{
		{"model.embed_tokens", "model.embed_tokens", true}, {"model.embed_tokens", "model.embed_tokens_per_layer", false},
		{"Linear", "model.layers.0.mlp.down_proj", false},
		{"re:.*lm_head$", "model.lm_head", true}, {"re:.*lm_head$", "model.lm_head.weight", false},
		{"re:.*lm_head$", "lm_head", true}, {`re:model\.layers.*`, "model.layers.0.mlp.gate_proj", true},
		{`re:model\.layers.*`, "lm.model.layers.0", false}, {`re:^model\.layers.*`, "model.layers.0", true},
		{`re:.*mlp\.(gate|up)_proj.*`, "model.layers.0.mlp.gate_proj", false},
		{"re:(a+)+$", "aaaaaaaaaaaaaaaaaaaaab", false},
	} {
		t.Run(tt.target+"/"+tt.module, func(t *testing.T) {
			if got := compressedTargetMatches(tt.target, tt.module); got != tt.want {
				t.Fatalf("match = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameterQuantizationExclusions(t *testing.T) {
	for _, tt := range []struct {
		config string
		name   string
		want   bool
	}{
		{"", "model.layer.weight", false},
		{`{"quantization_config":{}}`, "model.layer.weight", true},
		{`{"quantization_config":{"modules_to_not_convert":["lm_head"]}}`, "model.lm_head.weight", false},
		{`{"quantization_config":{"modules_to_not_convert":["lm_head"]}}`, "lm_head", false},
		{`{"quantization_config":{"modules_to_not_convert":["lm_head"]}}`, "lm_head.weight", false},
		{`{"quantization_config":{"modules_to_not_convert":["lm_head"]}}`, "model.embed_tokens.weight", true},
		{`{"quantization_config":{"modules_to_not_convert":["*lm_head*"]}}`, "model.lm_head.weight", false},
		{`{"quantization_config":{"modules_to_not_convert":["*lm_head*"]}}`, "model.embed_tokens.weight", true},
		{`{"quantization_config":{"modules_to_not_convert":["lm_head","embed_tokens"]}}`, "model.embed_tokens.weight", false},
		{`{"quantization_config":{"ignore":["re:.*self_attn.*","re:.*lm_head.*"]}}`, "model.layers.0.self_attn.q_proj.weight", false},
		{`{"quantization_config":{"ignore":["re:.*self_attn.*","re:.*lm_head.*"]}}`, "model.layers.0.mlp.experts.0.weight", true},
	} {
		t.Run(tt.config+"/"+tt.name, func(t *testing.T) {
			rules := parameterRulesForTest(t, tt.config)
			if got := rules.quantized(tt.name, nil); got != tt.want {
				t.Fatalf("quantized = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameterRatioArithmetic(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value int64
		ratio parameterRatio
		round bool
		want  int64
		err   bool
	}{
		{"three bit round", 1, parameterRatio{32, 3}, true, 11, false},
		{"three bit floor", 1, parameterRatio{32, 3}, false, 10, false},
		{"five bit round down", 1, parameterRatio{32, 5}, true, 6, false},
		{"half round up", 1, parameterRatio{5, 2}, true, 3, false},
		{"wide intermediate", math.MaxInt64, parameterRatio{32, 32}, true, math.MaxInt64, false},
		{"large exact total", 2_779_931_837_184, parameterRatio{1, 1}, true, 2_779_931_837_184, false},
		{"overflow", math.MaxInt64, parameterRatio{2, 1}, false, 0, true},
		{"wide overflow", math.MaxInt64, parameterRatio{math.MaxUint64, 1}, false, 0, true},
		{"rounding overflow", 1, parameterRatio{math.MaxUint64, 2}, true, 0, true},
		{"negative", -1, parameterRatio{1, 1}, false, 0, true},
		{"zero denominator", 1, parameterRatio{1, 0}, false, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ratio.apply(tt.value, tt.round)
			if (err != nil) != tt.err || got != tt.want {
				t.Fatalf("apply() = %d, %v; want %d, error=%v", got, err, tt.want, tt.err)
			}
		})
	}
}

func TestCompressedTensorGroupOrder(t *testing.T) {
	for _, groups := range []string{
		`{"z":{"targets":["layer"],"weights":{"num_bits":8}},"a":{"targets":["layer"],"weights":{"num_bits":4}}}`,
		`{"2":{"targets":["layer"],"weights":{"num_bits":4}},"1":{"targets":["layer"],"weights":{"num_bits":8}}}`,
	} {
		config := `{"quantization_config":{"quant_method":"compressed-tensors","format":"pack-quantized","config_groups":` + groups + `}}`
		rules := parameterRulesForTest(t, config)
		if got := rules.multiplier("layer.weight_packed", "I32", nil); got != (parameterRatio{32, 8}) {
			t.Fatalf("groups %s: multiplier = %+v, want 32/8", groups, got)
		}
	}
}

func TestSafetensorsHeaderValidation(t *testing.T) {
	for _, tt := range []struct {
		name   string
		header string
		size   int64
		known  bool
		valid  bool
	}{
		{"valid", `{"weight":{"dtype":"F32","shape":[10,20],"data_offsets":[0,800]}}`, 1000, true, true},
		{"unknown file size", `{"weight":{"dtype":"F4","shape":[65536,65536],"data_offsets":[0,2147483648]}}`, 0, false, true},
		{"padding allowed", `{"weight":{"dtype":"F4","shape":[10,20],"data_offsets":[0,128]}}`, 128, true, true},
		{"empty dimension", `{"weight":{"dtype":"F32","shape":[0,8],"data_offsets":[0,0]}}`, 0, true, true},
		{"scalar", `{"weight":{"dtype":"F32","shape":[],"data_offsets":[0,4]}}`, 4, true, true},
		{"dimension boundary", `{"weight":{"dtype":"I8","shape":[4294967296],"data_offsets":[0,0]}}`, 0, false, true},
		{"absurd dimension", `{"weight":{"dtype":"F4","shape":[4611686018427387904,4294967296],"data_offsets":[0,4]}}`, 4, true, false},
		{"dimension beyond boundary", `{"weight":{"dtype":"I8","shape":[4294967297],"data_offsets":[0,0]}}`, 0, false, false},
		{"product overflow", `{"weight":{"dtype":"I8","shape":[4294967296,4294967296],"data_offsets":[0,0]}}`, 0, false, false},
		{"negative dimension", `{"weight":{"dtype":"F32","shape":[-1],"data_offsets":[0,0]}}`, 0, false, false},
		{"fractional dimension", `{"weight":{"dtype":"F32","shape":[1.5],"data_offsets":[0,0]}}`, 0, false, false},
		{"null dimension", `{"weight":{"dtype":"F32","shape":[null],"data_offsets":[0,0]}}`, 0, false, false},
		{"missing shape", `{"weight":{"dtype":"F32","data_offsets":[0,0]}}`, 0, false, false},
		{"null shape", `{"weight":{"dtype":"F32","shape":null,"data_offsets":[0,0]}}`, 0, false, false},
		{"non-array shape", `{"weight":{"dtype":"F32","shape":2,"data_offsets":[0,0]}}`, 0, false, false},
		{"missing offsets", `{"weight":{"dtype":"F32","shape":[1]}}`, 0, false, false},
		{"short offsets", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[0]}}`, 0, false, false},
		{"extra offsets", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[0,1,2]}}`, 0, false, false},
		{"negative offset", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[-1,0]}}`, 0, false, false},
		{"reversed offsets", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[4,0]}}`, 0, false, false},
		{"fractional offset", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[0,1.5]}}`, 0, false, false},
		{"null offset", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[null,0]}}`, 0, false, false},
		{"offset exceeds size", `{"weight":{"dtype":"F32","shape":[1],"data_offsets":[0,800]}}`, 100, true, false},
		{"invalid skipped tensor", `{"scale":{"dtype":"F8_E8M0","shape":[-1],"data_offsets":[0,1]}}`, 1, true, false},
		{"null header", `null`, 0, false, false},
		{"array header", `[]`, 0, false, false},
		{"null tensor", `{"weight":null}`, 0, false, false},
		{"empty header", `{}`, 0, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSafetensorsHeader(rawParameterHeader(tt.header), tt.size, tt.known)
			if (err == nil) != tt.valid {
				t.Fatalf("parse error = %v, want valid=%v", err, tt.valid)
			}
		})
	}
	for _, content := range [][]byte{
		nil, {1, 2}, {255, 255, 255, 255, 255, 255, 255, 255},
		rawParameterHeader("{"), append([]byte{10, 0, 0, 0, 0, 0, 0, 0}, []byte("{}")...),
	} {
		if _, err := parseSafetensorsHeader(content, 0, false); err == nil {
			t.Fatalf("accepted invalid header %v", content)
		}
	}
}

func TestSafetensorsIgnoresSelfReportedParameters(t *testing.T) {
	for _, declared := range []string{`"1"`, `"999000000000"`, `200`, `null`, `"invalid"`} {
		header := rawParameterHeader(fmt.Sprintf(`{"__metadata__":{"format":"pt","total_parameters":%s},"weight":{"dtype":"F32","shape":[10,20],"data_offsets":[0,800]}}`, declared))
		files := &git.RepoMetadataFiles{SafetensorsFiles: map[string][]byte{"model.safetensors": header}}
		if got := analyzeParametersForTest(t, files); got != 200 {
			t.Fatalf("declared %s: count = %d, want 200", declared, got)
		}
	}
}
