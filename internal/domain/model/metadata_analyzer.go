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
	"encoding/json"
	"math"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/hf"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
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
	WeightMap json.RawMessage `json:"weight_map"`
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
	var config parameterConfig
	configValid := true
	if len(files.ConfigJSON) > 0 {
		if cfg, err := hf.ParseConfigData(bytes.NewReader(files.ConfigJSON)); err == nil {
			if cfg.ModelType != "" {
				tags = append(tags, ClassifiedTag{Name: cfg.ModelType, Category: "other"})
			}
			if cfg.QuantizationConfig != nil && cfg.QuantizationConfig.QuantMethod != "" {
				tags = append(tags, ClassifiedTag{Name: cfg.QuantizationConfig.QuantMethod, Category: "other"})
			}
		}
		if err := json.Unmarshal(files.ConfigJSON, &config); err != nil {
			log.Warnw("Cannot read parameter counting config", "error", err)
			configValid = false
		}
	}
	rules, err := resolveParameterRules(config)
	if err != nil {
		log.Warnw("Cannot resolve parameter counting quantization", "error", err)
		configValid = false
	}
	metadata.ParameterCount = inferSafetensorsParameters(files, rules, inferParameterBytes(config, rules), configValid)

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

func inferParameterBytes(config parameterConfig, rules parameterRules) parameterRatio {
	if rules.method == "fp8" && rules.expertBits == 4 {
		// Packed FP4 experts use half a byte plus one E8M0 scale per 32 values.
		return parameterRatio{17, 32}
	}

	if quantConfig := config.object("quantization_config"); quantConfig != nil {
		if bits := estimatedParameterBits(quantConfig); bits > 0 {
			return parameterRatio{uint64(bits/8 + min(1, bits%8)), 1}
		}
		switch strings.ToLower(quantConfig.text("quant_method")) {
		case "gptq", "awq", "int4", "nf4", "fp4", "int8", "fp8":
			return parameterRatio{1, 1}
		}
	}

	for _, key := range []string{"torch_dtype", "dtype"} {
		switch strings.ToLower(config.text(key)) {
		case "float32", "fp32":
			return parameterRatio{4, 1}
		case "float16", "fp16", "bfloat16", "bf16", "half":
			return parameterRatio{2, 1}
		case "int8", "uint8", "fp8":
			return parameterRatio{1, 1}
		}
	}
	return parameterRatio{2, 1}
}

func estimatedParameterBits(config parameterConfig) int64 {
	for _, key := range []string{"bits", "w_bit", "weight_bit_width"} {
		var value float64
		if json.Unmarshal(config[key], &value) == nil && value > 0 && value < math.MaxInt64 {
			return int64(value)
		}
	}
	return 0
}

func estimateParameterCount(totalSize int64, parameterBytes parameterRatio) (int64, error) {
	if totalSize == 0 {
		return 0, nil
	}
	return (parameterRatio{parameterBytes.denominator, parameterBytes.numerator}).apply(totalSize, false)
}

// addParameterCounts adds two parameter counts, returning 0 when either is
// invalid or the sum would overflow. 0 means "unknown" throughout this file.
func addParameterCounts(a, b int64) int64 {
	if a < 0 || b < 0 || a > math.MaxInt64-b {
		log.Warnw("Parameter count sum is invalid", "exact", a, "estimated", b)
		return 0
	}
	return a + b
}
