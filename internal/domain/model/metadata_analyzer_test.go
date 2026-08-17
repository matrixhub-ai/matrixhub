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
