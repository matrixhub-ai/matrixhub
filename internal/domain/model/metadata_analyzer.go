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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/hf"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

// ClassifiedTag represents model metadata tags after applying model-domain rules.
type ClassifiedTag struct {
	Name     string
	Category string
}

// RepoMetadata is the structured model metadata extracted from raw repo files.
type RepoMetadata struct {
	ReadmeContent  string
	Size           int64
	ParameterCount int64
	Tags           []ClassifiedTag
}

type safetensorsIndex struct {
	Metadata struct {
		TotalSize int64 `json:"total_size"`
	} `json:"metadata"`
}

type safetensorsTensor struct {
	Shape []int64 `json:"shape"`
}

// AnalyzeRepoMetadata converts raw repo files into model-domain metadata.
// The git repo layer only loads bytes from tracked files; all model-specific
// interpretation (label classification, parameter_count inference) stays here.
func AnalyzeRepoMetadata(files *git.RepoMetadataFiles) (*RepoMetadata, error) {
	metadata := &RepoMetadata{Size: files.Size}
	var tags []ClassifiedTag

	if len(files.ReadmeContent) > 0 {
		metadata.ReadmeContent = string(files.ReadmeContent)
		if readme, err := hf.ParseReadme(bytes.NewReader(files.ReadmeContent)); err == nil {
			card := readme.Card
			if card.PipelineTag != "" {
				tags = append(tags, ClassifiedTag{Name: card.PipelineTag, Category: "task"})
			}
			if card.LibraryName != "" {
				tags = append(tags, ClassifiedTag{Name: card.LibraryName, Category: "library"})
			}
			for _, lang := range card.Language {
				tags = append(tags, ClassifiedTag{Name: lang, Category: "language"})
			}
			for _, lic := range card.License {
				tags = append(tags, ClassifiedTag{Name: lic, Category: "license"})
			}
			for _, tag := range card.Tags {
				tags = append(tags, ClassifiedTag{Name: tag, Category: "other"})
			}
		}
	}

	// parameter_count inference only uses config.json and safetensors metadata.
	// README.md is intentionally excluded here because it is descriptive text,
	// not structured machine metadata.
	//
	// Default to fp16/bf16-sized parameters when config.json does not expose
	// enough dtype/quantization information. This is a pragmatic default for
	// modern Hugging Face model repos.
	parameterBytes := int64(2)
	if len(files.ConfigJSON) > 0 {
		if cfg, err := hf.ParseConfigData(bytes.NewReader(files.ConfigJSON)); err == nil {
			if cfg.ModelType != "" {
				tags = append(tags, ClassifiedTag{Name: cfg.ModelType, Category: "other"})
			}
			if cfg.QuantizationConfig != nil && cfg.QuantizationConfig.QuantMethod != "" {
				tags = append(tags, ClassifiedTag{Name: cfg.QuantizationConfig.QuantMethod, Category: "other"})
			}
		}
		if inferred := inferParameterBytes(files.ConfigJSON); inferred > 0 {
			parameterBytes = inferred
		}
	}

	if len(files.SafetensorsIndexJSON) > 0 {
		// For most sharded HF models, model.safetensors.index.json metadata.total_size
		// gives total tensor bytes. Dividing by bytes-per-parameter gives a cheap,
		// usually-good parameter_count estimate without scanning all shards.
		var index safetensorsIndex
		if err := json.Unmarshal(files.SafetensorsIndexJSON, &index); err == nil && index.Metadata.TotalSize > 0 {
			metadata.ParameterCount = estimateParameterCount(index.Metadata.TotalSize, parameterBytes)
		}
	}
	if metadata.ParameterCount == 0 && len(files.SafetensorsFiles) > 0 {
		metadata.ParameterCount = countSafetensorsParameters(files.SafetensorsFiles)
	}

	metadata.Tags = deduplicateClassifiedTags(tags)
	return metadata, nil
}

func deduplicateClassifiedTags(tags []ClassifiedTag) []ClassifiedTag {
	type tagKey struct {
		Name     string
		Category string
	}

	seen := make(map[tagKey]struct{})
	result := make([]ClassifiedTag, 0, len(tags))

	for _, tag := range tags {
		if tag.Name == "" {
			continue
		}
		key := tagKey(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}

	return result
}

func inferParameterBytes(configBytes []byte) int64 {
	var raw map[string]any
	if err := json.Unmarshal(configBytes, &raw); err != nil {
		return 0
	}

	if quantCfg, ok := raw["quantization_config"].(map[string]any); ok {
		if bits := extractBits(quantCfg); bits > 0 {
			return int64(math.Ceil(float64(bits) / 8.0))
		}
		if method, ok := quantCfg["quant_method"].(string); ok {
			switch strings.ToLower(method) {
			case "gptq", "awq", "int4", "nf4", "fp4", "int8", "fp8":
				return 1
			}
		}
	}

	for _, key := range []string{"torch_dtype", "dtype"} {
		if val, ok := raw[key].(string); ok {
			switch strings.ToLower(val) {
			case "float32", "fp32":
				return 4
			case "float16", "fp16", "bfloat16", "bf16", "half":
				return 2
			case "int8", "uint8", "fp8":
				return 1
			}
		}
	}

	return 0
}

func extractBits(quantCfg map[string]any) int {
	for _, key := range []string{"bits", "w_bit", "weight_bit_width"} {
		switch v := quantCfg[key].(type) {
		case float64:
			if v > 0 {
				return int(v)
			}
		case int:
			if v > 0 {
				return v
			}
		}
	}
	return 0
}

func estimateParameterCount(totalSize, parameterBytes int64) int64 {
	if totalSize <= 0 || parameterBytes <= 0 {
		return 0
	}
	return totalSize / parameterBytes
}

func countSafetensorsParameters(files map[string][]byte) int64 {
	var total int64
	for _, content := range files {
		count, ok := countSafetensorsHeaderParameters(content)
		if !ok {
			continue
		}
		if count > math.MaxInt64-total {
			return 0
		}
		total += count
	}
	return total
}

func countSafetensorsHeaderParameters(content []byte) (int64, bool) {
	if len(content) < 8 {
		return 0, false
	}

	headerLength := binary.LittleEndian.Uint64(content[:8])
	if headerLength > uint64(len(content)-8) || headerLength > uint64(math.MaxInt64) {
		return 0, false
	}

	var tensors map[string]safetensorsTensor
	if err := json.Unmarshal(content[8:8+int(headerLength)], &tensors); err != nil {
		return 0, false
	}

	var total int64
	for name, tensor := range tensors {
		if name == "__metadata__" {
			continue
		}

		count := int64(1)
		for _, dim := range tensor.Shape {
			if dim < 0 || dim > math.MaxInt64/count {
				return 0, false
			}
			count *= dim
		}
		if count > math.MaxInt64-total {
			return 0, false
		}
		total += count
	}
	return total, true
}
