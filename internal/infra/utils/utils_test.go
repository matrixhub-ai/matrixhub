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

package utils

import "testing"

func TestCalculatePagesDoesNotOverflowInt32Total(t *testing.T) {
	if got, want := CalculatePages(int64(1<<31-1), 20), int32(107374183); got != want {
		t.Fatalf("CalculatePages() = %d, want %d", got, want)
	}
}

func TestCalculatePagesCapsAtInt32Max(t *testing.T) {
	if got, want := CalculatePages(int64(1<<63-1), 1), int32(1<<31-1); got != want {
		t.Fatalf("CalculatePages() = %d, want %d", got, want)
	}
}

func TestClampInt32(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want int32
	}{
		{name: "within range", in: 42, want: 42},
		{name: "above max", in: 1<<31 + 1, want: 1<<31 - 1},
		{name: "below min", in: -(1 << 31) - 1, want: -1 << 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClampInt32(tt.in); got != tt.want {
				t.Fatalf("ClampInt32(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
