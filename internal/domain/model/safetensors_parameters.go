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
	"fmt"
	"math"
	"math/bits"
	"path"
	"strings"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type parameterRatio struct {
	numerator   uint64
	denominator uint64
}

func (r parameterRatio) apply(value int64, round bool) (int64, error) {
	if value < 0 || r.numerator == 0 || r.denominator == 0 {
		return 0, fmt.Errorf("invalid parameter count ratio or value")
	}
	high, low := bits.Mul64(uint64(value), r.numerator)
	if high >= r.denominator {
		return 0, fmt.Errorf("parameter count overflows int64")
	}
	quotient, remainder := bits.Div64(high, low, r.denominator)
	if round && remainder >= r.denominator/2+r.denominator%2 {
		if quotient >= math.MaxInt64 {
			return 0, fmt.Errorf("rounded parameter count overflows int64")
		}
		quotient++
	}
	if quotient > math.MaxInt64 {
		return 0, fmt.Errorf("parameter count overflows int64")
	}
	return int64(quotient), nil
}

type safetensorsTensor struct {
	dtype    string
	elements int64
}

type safetensorsHeader map[string]safetensorsTensor

type safetensorsInteger int64

func (value *safetensorsInteger) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("safetensors dimensions and offsets must be integers")
	}
	return json.Unmarshal(raw, (*int64)(value))
}

func parseSafetensorsHeader(content []byte, fileSize int64, sizeKnown bool) (safetensorsHeader, error) {
	if len(content) < 8 {
		return nil, fmt.Errorf("missing safetensors header length")
	}
	length := binary.LittleEndian.Uint64(content[:8])
	if length > 64*1024*1024 || length > uint64(len(content)-8) {
		return nil, fmt.Errorf("invalid safetensors header length: %d", length)
	}
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(content[8:8+int(length)], &entries); err != nil {
		return nil, fmt.Errorf("decode safetensors header: %w", err)
	}
	if entries == nil {
		return nil, fmt.Errorf("safetensors header must be an object")
	}
	header := make(safetensorsHeader, len(entries))
	for name, raw := range entries {
		if name == "__metadata__" {
			continue
		}
		var tensor struct {
			Dtype   string               `json:"dtype"`
			Shape   []safetensorsInteger `json:"shape"`
			Offsets []safetensorsInteger `json:"data_offsets"`
		}
		if err := json.Unmarshal(raw, &tensor); err != nil {
			return nil, fmt.Errorf("decode tensor %q: %w", name, err)
		}
		if tensor.Shape == nil || len(tensor.Offsets) != 2 {
			return nil, fmt.Errorf("tensor %q has missing shape or data_offsets", name)
		}
		if tensor.Offsets[0] < 0 || tensor.Offsets[1] < tensor.Offsets[0] ||
			sizeKnown && (fileSize < 0 || int64(tensor.Offsets[1]) > fileSize) {
			return nil, fmt.Errorf("tensor %q has invalid data_offsets for its file size", name)
		}
		empty := len(tensor.Shape) == 0
		for _, dim := range tensor.Shape {
			if dim < 0 || dim > 1<<32 {
				return nil, fmt.Errorf("tensor %q has invalid dimension %d", name, dim)
			}
			empty = empty || dim == 0
		}
		var elements int64
		if !empty {
			elements = 1
			for _, dim := range tensor.Shape {
				if elements > math.MaxInt64/int64(dim) {
					return nil, fmt.Errorf("tensor %q element count overflows int64", name)
				}
				elements *= int64(dim)
			}
		}
		header[name] = safetensorsTensor{dtype: tensor.Dtype, elements: elements}
	}
	return header, nil
}

// countSafetensorsHeaders follows the Hub's counting rules in huggingface.js:
// https://github.com/huggingface/huggingface.js/blob/210be68c40a0f57b4681fe73330405d19a80d439/packages/hub/src/lib/parse-safetensors-metadata.ts
// Keep quantization handling in sync so logical parameter counts match the Hub.
func countSafetensorsHeaders(headers map[string]safetensorsHeader, rules parameterRules) (map[string]int64, bool) {
	counts := make(map[string]int64, len(headers))
	mlxModules := rules.mlxModules(headers)
	unexpandedInteger := false
	for filename, header := range headers {
		total, unexpanded, err := countSafetensorsHeader(header, rules, mlxModules)
		if err != nil {
			log.Warnw("Cannot count safetensors header", "file", filename, "error", err)
			continue
		}
		counts[filename] = total
		unexpandedInteger = unexpandedInteger || unexpanded
	}
	return counts, unexpandedInteger
}

func countSafetensorsHeader(header safetensorsHeader, rules parameterRules, mlxModules map[string]bool) (int64, bool, error) {
	var total int64
	unexpandedInteger := false
	for name, tensor := range header {
		if tensor.elements == 0 || rules.skip(name, tensor.dtype, mlxModules) {
			continue
		}
		multiplier := rules.multiplier(name, tensor.dtype, mlxModules)
		count, err := multiplier.apply(tensor.elements, true)
		if err != nil {
			return 0, false, fmt.Errorf("count tensor %q: %w", name, err)
		}
		if count > math.MaxInt64-total {
			return 0, false, fmt.Errorf("safetensors header parameter count overflows int64")
		}
		total += count
		if integerContainerBits(tensor.dtype) != 0 && multiplier.numerator == multiplier.denominator {
			unexpandedInteger = true
		}
	}
	return total, unexpandedInteger, nil
}

func readableSafetensorsHeaders(files map[string][]byte, sizes map[string]int64) map[string]safetensorsHeader {
	headers := make(map[string]safetensorsHeader, len(files))
	for name, content := range files {
		size, known := sizes[name]
		header, err := parseSafetensorsHeader(content, size, known)
		if err != nil {
			log.Warnw("Cannot read safetensors header", "file", name, "error", err)
			continue
		}
		headers[name] = header
	}
	return headers
}

func indexedSafetensorsFiles(weightMapJSON json.RawMessage, files map[string][]byte) (map[string][]byte, bool) {
	var weightMap map[string]string
	if json.Unmarshal(weightMapJSON, &weightMap) != nil || len(weightMap) == 0 {
		return nil, false
	}
	shards := make(map[string][]byte)
	for _, name := range weightMap {
		name = path.Clean(name)
		if strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") || !strings.HasSuffix(name, ".safetensors") {
			return nil, false
		}
		content, ok := files[name]
		if !ok {
			return nil, false
		}
		shards[name] = content
	}
	return shards, true
}

func inferSafetensorsParameters(files *git.RepoMetadataFiles, rules parameterRules, parameterBytes parameterRatio, configValid bool) int64 {
	var index safetensorsIndex
	if len(files.SafetensorsIndexJSON) > 0 {
		if err := json.Unmarshal(files.SafetensorsIndexJSON, &index); err != nil {
			log.Warnw("Cannot read safetensors index", "error", err)
		}
	}
	if index.Metadata.TotalSize > 0 {
		if shards, complete := indexedSafetensorsFiles(index.WeightMap, files.SafetensorsFiles); complete {
			headers := readableSafetensorsHeaders(shards, files.SafetensorsSizes)
			counts, unexpanded := countSafetensorsHeaders(headers, rules)
			if len(counts) == len(shards) && configValid {
				if rules.supportedMethod() || !unexpanded {
					if total, err := sumParameterCounts(counts); err == nil {
						return total
					} else {
						log.Warnw("Cannot sum safetensors parameter counts", "error", err)
					}
				} else {
					log.Warnw("Estimating parameters for unsupported packed quantization", "method", rules.method)
				}
			}
		}
		total, err := estimateParameterCount(index.Metadata.TotalSize, parameterBytes)
		if err != nil {
			log.Warnw("Cannot estimate indexed parameter count", "error", err)
		}
		return total
	}

	headers := readableSafetensorsHeaders(files.SafetensorsFiles, files.SafetensorsSizes)
	counts, _ := countSafetensorsHeaders(headers, rules)
	unreadableSizes := make(map[string]int64)
	for name, size := range files.SafetensorsSizes {
		if _, counted := counts[name]; !counted {
			unreadableSizes[name] = size
		}
	}
	exact, err := sumParameterCounts(counts)
	if err != nil {
		log.Warnw("Cannot sum safetensors parameter counts", "error", err)
		return 0
	}
	unreadableSize, err := sumParameterCounts(unreadableSizes)
	if err != nil {
		log.Warnw("Cannot sum unreadable safetensors sizes", "error", err)
		return 0
	}
	estimated, err := estimateParameterCount(unreadableSize, parameterBytes)
	if err != nil {
		log.Warnw("Cannot estimate unreadable safetensors parameters", "error", err)
		return 0
	}
	return addParameterCounts(exact, estimated)
}

func sumParameterCounts(counts map[string]int64) (int64, error) {
	var total int64
	for _, count := range counts {
		if count < 0 || count > math.MaxInt64-total {
			return 0, fmt.Errorf("parameter count or safetensors size sum is invalid: %d + %d", total, count)
		}
		total += count
	}
	return total, nil
}
