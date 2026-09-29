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
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

type parameterConfig map[string]json.RawMessage

func (c parameterConfig) object(key string) parameterConfig {
	var value parameterConfig
	if len(c[key]) == 0 || json.Unmarshal(c[key], &value) != nil {
		return nil
	}
	return value
}

func (c parameterConfig) text(key string) string {
	var value string
	if len(c[key]) == 0 || json.Unmarshal(c[key], &value) != nil {
		return ""
	}
	return value
}

func (c parameterConfig) number(key string) (int64, bool) {
	var value float64
	if !hasConfigValue(c[key]) || json.Unmarshal(c[key], &value) != nil ||
		value < math.MinInt64 || value >= math.MaxInt64 || math.Trunc(value) != value {
		return 0, false
	}
	return int64(value), true
}

func (c parameterConfig) flag(key string) bool {
	var value bool
	return len(c[key]) > 0 && json.Unmarshal(c[key], &value) == nil && value
}

func (c parameterConfig) strings(key string) []string {
	var value []string
	if len(c[key]) == 0 || json.Unmarshal(c[key], &value) != nil {
		return nil
	}
	return value
}

func hasConfigValue(value json.RawMessage) bool {
	return len(value) > 0 && !bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

func subByteBits(value json.RawMessage) int64 {
	var format string
	if json.Unmarshal(value, &format) != nil {
		return 0
	}
	format = strings.ToLower(format)
	for _, token := range []string{"nvfp4", "mxfp4", "fp4", "int4", "uint4", "nf4"} {
		if strings.Contains(format, token) {
			return 4
		}
	}
	if strings.Contains(format, "int2") {
		return 2
	}
	return 0
}

type quantizationGroup struct {
	name   string
	config parameterConfig
}

type parameterRules struct {
	config     parameterConfig
	method     string
	expertBits int64
	groups     []quantizationGroup
}

func resolveParameterRules(modelConfig parameterConfig) (parameterRules, error) {
	textConfig := modelConfig.object("text_config")
	quantConfig := modelQuantizationConfig(modelConfig)
	if quantConfig == nil {
		quantConfig = modelQuantizationConfig(textConfig)
	}
	rules := parameterRules{
		config: quantConfig,
		method: strings.ToLower(quantConfig.text("quant_method")),
	}
	for _, value := range []json.RawMessage{
		quantConfig["expert_dtype"], modelConfig["expert_dtype"],
		textConfig["expert_dtype"], quantConfig["store_dtype"],
	} {
		if hasConfigValue(value) {
			rules.expertBits = subByteBits(value)
			break
		}
	}
	if rules.method == "compressed-tensors" {
		var err error
		rules.groups, err = parseQuantizationGroups(quantConfig["config_groups"])
		if err != nil {
			return rules, err
		}
	}
	return rules, nil
}

func modelQuantizationConfig(config parameterConfig) parameterConfig {
	for _, key := range []string{"quantization", "quantization_config"} {
		candidate := config.object(key)
		method := strings.ToLower(candidate.text("quant_method"))
		_, declaresMethod := candidate["quant_method"]
		if candidate != nil && (!declaresMethod || method == "mlx") && validMLXConfig(candidate) {
			candidate["quant_method"] = json.RawMessage(`"mlx"`)
			if !hasConfigValue(candidate["mode"]) {
				candidate["mode"] = json.RawMessage(`"affine"`)
			}
			return candidate
		}
		if key == "quantization_config" {
			return candidate
		}
	}
	return nil
}

func parseQuantizationGroups(raw json.RawMessage) ([]quantizationGroup, error) {
	if !hasConfigValue(raw) {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("quantization config_groups must be an object")
	}
	var groups []quantizationGroup
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("read quantization group name: %w", err)
		}
		name, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("invalid quantization group name")
		}
		var config parameterConfig
		if err := decoder.Decode(&config); err != nil || config == nil {
			return nil, fmt.Errorf("invalid quantization group %q", name)
		}
		groups = append(groups, quantizationGroup{name: name, config: config})
	}
	// JSON object iteration in the Hub puts integer keys before other keys,
	// whose original order decides the first matching quantization group.
	slices.SortStableFunc(groups, func(a, b quantizationGroup) int {
		index := func(key string) uint64 {
			n, err := strconv.ParseUint(key, 10, 32)
			if err != nil || strconv.FormatUint(n, 10) != key {
				return math.MaxUint32
			}
			return n
		}
		aIndex, bIndex := index(a.name), index(b.name)
		switch {
		case aIndex < bIndex:
			return -1
		case aIndex > bIndex:
			return 1
		default:
			return 0
		}
	})
	return groups, nil
}

func (r parameterRules) supportedMethod() bool {
	switch r.method {
	case "", "fp8", "mxfp4", "gptq", "awq", "compressed-tensors", "bitsandbytes", "mlx":
		return true
	default:
		return false
	}
}

type mlxParameters struct {
	bits int64
	mode string
}

func parseMLXParameters(config parameterConfig) (mlxParameters, bool) {
	mode := "affine"
	if _, present := config["mode"]; present {
		mode = strings.ToLower(config.text("mode"))
	}
	var bits, groupSize int64
	switch mode {
	case "affine":
		bits, groupSize = 4, 64
	case "mxfp4":
		bits, groupSize = 4, 32
	case "nvfp4":
		bits, groupSize = 4, 16
	case "mxfp8":
		bits, groupSize = 8, 32
	default:
		return mlxParameters{}, false
	}
	for key, target := range map[string]*int64{"bits": &bits, "group_size": &groupSize} {
		if hasConfigValue(config[key]) {
			value, ok := config.number(key)
			if !ok {
				return mlxParameters{}, false
			}
			*target = value
		}
	}
	valid := false
	switch mode {
	case "affine":
		valid = slices.Contains([]int64{2, 3, 4, 5, 6, 8}, bits) &&
			slices.Contains([]int64{32, 64, 128}, groupSize)
	case "mxfp4":
		valid = bits == 4 && groupSize == 32
	case "nvfp4":
		valid = bits == 4 && groupSize == 16
	case "mxfp8":
		valid = bits == 8 && groupSize == 32
	}
	return mlxParameters{bits: bits, mode: mode}, valid
}

func validMLXConfig(config parameterConfig) bool {
	hasParameters := func(value parameterConfig) bool {
		for _, key := range []string{"bits", "group_size", "mode"} {
			if _, ok := value[key]; ok {
				return true
			}
		}
		return false
	}
	if _, valid := parseMLXParameters(config); hasParameters(config) && valid {
		return true
	}
	for key := range config {
		override := config.object(key)
		if _, valid := parseMLXParameters(override); hasParameters(override) && valid {
			return true
		}
	}
	return false
}

func (r parameterRules) mlxParameters(name string) (mlxParameters, bool) {
	module := tensorModule(name)
	if bytes.Equal(bytes.TrimSpace(r.config[module]), []byte("false")) {
		return mlxParameters{}, false
	}
	if override := r.config.object(module); override != nil {
		return parseMLXParameters(override)
	}
	return parseMLXParameters(r.config)
}

func (r parameterRules) mlxModules(headers map[string]safetensorsHeader) map[string]bool {
	if r.method != "mlx" {
		return nil
	}
	type moduleState struct{ weight, scales, biases bool }
	states := make(map[string]moduleState)
	for _, header := range headers {
		for name, tensor := range header {
			module := tensorModule(name)
			state := states[module]
			switch tensorSuffix(name) {
			case "weight":
				state.weight = state.weight || tensor.dtype == "U32"
			case "scales":
				state.scales = true
			case "biases":
				state.biases = true
			}
			states[module] = state
		}
	}
	modules := make(map[string]bool)
	for module, state := range states {
		params, ok := r.mlxParameters(module + ".weight")
		if ok && state.weight && state.scales && (params.mode != "affine" || state.biases) {
			modules[module] = true
		}
	}
	return modules
}

func tensorSuffix(name string) string {
	return name[strings.LastIndexByte(name, '.')+1:]
}

func tensorModule(name string) string {
	if index := strings.LastIndexByte(name, '.'); index >= 0 {
		return name[:index]
	}
	return name
}

func packedTensorModule(name string) string {
	if index := strings.LastIndex(name, ".weight"); index >= 0 {
		return name[:index]
	}
	return name
}

func parameterGlobMatch(pattern, name string) bool {
	first, rest, wildcard := strings.Cut(pattern, "*")
	if !wildcard {
		return pattern == name
	}
	if !strings.HasPrefix(name, first) {
		return false
	}
	name = strings.TrimPrefix(name, first)
	for {
		part, remaining, more := strings.Cut(rest, "*")
		if !more {
			return strings.HasSuffix(name, part)
		}
		index := strings.Index(name, part)
		if index < 0 {
			return false
		}
		name = name[index+len(part):]
		rest = remaining
	}
}

func compressedTargetMatches(target, module string) bool {
	pattern, regex := strings.CutPrefix(target, "re:")
	if !regex {
		return target == module
	}
	pattern = strings.TrimPrefix(pattern, "^")
	if strings.HasSuffix(pattern, "$") {
		pattern = strings.TrimSuffix(pattern, "$")
	} else {
		pattern += ".*"
	}
	pattern = strings.ReplaceAll(strings.ReplaceAll(pattern, ".*", "*"), `\.`, ".")
	if strings.ContainsAny(pattern, `\+?()[]{}|^$`) {
		return false
	}
	return parameterGlobMatch(pattern, module)
}

func (r parameterRules) quantized(name string, mlxModules map[string]bool) bool {
	if r.config == nil {
		return false
	}
	if r.method == "mlx" {
		_, valid := r.mlxParameters(name)
		return valid && (mlxModules == nil || mlxModules[tensorModule(name)])
	}
	for _, target := range r.config.strings("ignore") {
		if compressedTargetMatches(target, packedTensorModule(name)) {
			return false
		}
	}
	for _, pattern := range r.config.strings("modules_to_not_convert") {
		if strings.Contains(pattern, "*") {
			if parameterGlobMatch(pattern, name) {
				return false
			}
		} else if strings.Contains(name, pattern) {
			return false
		}
	}
	return true
}

func (r parameterRules) skip(name, dtype string, mlxModules map[string]bool) bool {
	switch dtype {
	case "F8_E8M0", "E8M0", "UE8":
		return true
	}
	if r.config == nil {
		return false
	}
	suffix := tensorSuffix(name)
	switch suffix {
	case "weight_scale", "weight_scale_inv", "weight_scale_2", "weight_global_scale",
		"weight_zero_point", "input_scale", "input_global_scale", "input_zero_point",
		"zero_point", "weight_shape", "weight_g_idx":
		return true
	}
	switch r.method {
	case "mlx":
		params, _ := r.mlxParameters(name)
		return r.quantized(name, mlxModules) && (suffix == "scales" || suffix == "biases" && params.mode == "affine")
	case "mxfp4":
		return dtype == "U8" && strings.HasSuffix(suffix, "_scales")
	case "bitsandbytes":
		return slices.Contains([]string{"absmax", "quant_map", "nested_absmax", "nested_quant_map"}, suffix) ||
			strings.HasPrefix(suffix, "bitsandbytes__")
	case "gptq", "awq":
		return r.quantized(name, nil) && slices.Contains([]string{"qzeros", "g_idx", "scales"}, suffix)
	default:
		return false
	}
}

func integerContainerBits(dtype string) int64 {
	switch dtype {
	case "I8", "U8":
		return 8
	case "I16", "U16":
		return 16
	case "I32", "U32":
		return 32
	default:
		return 0
	}
}

func packingRatio(dtype string, width int64) parameterRatio {
	container := integerContainerBits(dtype)
	if width <= 0 || width >= container {
		return parameterRatio{1, 1}
	}
	return parameterRatio{uint64(container), uint64(width)}
}

func (r parameterRules) multiplier(name, dtype string, mlxModules map[string]bool) parameterRatio {
	identity := parameterRatio{1, 1}
	if !r.quantized(name, mlxModules) {
		return identity
	}
	bits, _ := r.config.number("bits")
	switch r.method {
	case "fp8":
		if strings.Contains(name, ".experts.") && !strings.Contains(name, "shared_experts") {
			return packingRatio(dtype, r.expertBits)
		}
	case "mlx":
		if dtype == "U32" && tensorSuffix(name) == "weight" {
			params, _ := r.mlxParameters(name)
			return packingRatio(dtype, params.bits)
		}
	case "mxfp4":
		if dtype == "U8" && strings.Contains(name, "_blocks") {
			return parameterRatio{2, 1}
		}
	case "gptq", "awq":
		if tensorSuffix(name) == "qweight" {
			if bits <= 0 {
				bits = 4
			}
			return parameterRatio{uint64(max(1, 32/bits)), 1}
		}
		if dtype == "U8" && (bits == 2 || bits == 4) {
			return packingRatio(dtype, bits)
		}
	case "compressed-tensors":
		return r.compressedMultiplier(name, dtype)
	case "bitsandbytes":
		if dtype == "U8" && r.config.flag("load_in_4bit") {
			return parameterRatio{2, 1}
		}
	default:
		if dtype == "U8" && (bits == 4 || r.config.flag("load_in_4bit")) {
			return parameterRatio{2, 1}
		}
	}
	return identity
}

func (r parameterRules) compressedMultiplier(name, dtype string) parameterRatio {
	var matched parameterConfig
	for _, group := range r.groups {
		if slices.ContainsFunc(group.config.strings("targets"), func(target string) bool {
			return compressedTargetMatches(target, packedTensorModule(name))
		}) {
			matched = group.config
			break
		}
	}
	formatConfig := r.config
	if hasConfigValue(matched["format"]) {
		formatConfig = matched
	}
	format := strings.ToLower(formatConfig.text("format"))
	if !strings.HasSuffix(format, "pack-quantized") {
		return parameterRatio{1, 1}
	}
	if weights := matched.object("weights"); hasConfigValue(weights["num_bits"]) {
		width, _ := weights.number("num_bits")
		return packingRatio(dtype, width)
	}
	for _, group := range r.groups {
		if width, ok := group.config.object("weights").number("num_bits"); ok && width != 0 {
			return packingRatio(dtype, width)
		}
	}
	width := subByteBits(formatConfig["format"])
	if width == 0 {
		width = 4
	}
	return packingRatio(dtype, width)
}
