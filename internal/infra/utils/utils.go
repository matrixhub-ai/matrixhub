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

const (
	maxInt32 = int64(1<<31 - 1)
	minInt32 = int64(-1 << 31)
)

func IsFullPageData(page, pageSize int) bool {
	return page == 1 && pageSize == -1
}

// ClampInt32 converts value to int32 without overflowing its bounds.
func ClampInt32(value int64) int32 {
	if value > maxInt32 {
		return int32(maxInt32)
	}
	if value < minInt32 {
		return int32(minInt32)
	}
	return int32(value)
}

// CalculatePages calculates the total number of pages based on total count and page size.
// It returns 0 if total or pageSize is 0 or negative.
func CalculatePages(total int64, pageSize int32) int32 {
	if total <= 0 || pageSize <= 0 {
		return 0
	}

	pageSize64 := int64(pageSize)
	pages := total / pageSize64
	if total%pageSize64 != 0 {
		pages++
	}

	return ClampInt32(pages)
}
