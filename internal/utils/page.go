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

import (
	"math"

	"github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
)

var (
	defaultPage     = int32(1)
	defaultPageSize = int32(10)
)

func NewPage(page, pageSize int32) *v1alpha1.Pagination {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	return &v1alpha1.Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    0,
		Pages:    0,
	}
}

func SetPageTotal(p *v1alpha1.Pagination, total int32) *v1alpha1.Pagination {
	p.Total = total
	p.Pages = int32(math.Ceil(float64(total) / float64(p.PageSize)))
	return p
}
