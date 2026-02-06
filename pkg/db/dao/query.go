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

package dao

import (
	"fmt"
	"slices"

	"github.com/matrixhub-ai/matrixhub/pkg/db"
)

// BuildOrderByQueries builds order by queries from the given orderBy strings.
func BuildOrderByQueries[T Model](orderBy []string, allowFields ...string) ([]QueryFunc[T], error) {
	query := []QueryFunc[T]{}
	for _, order := range orderBy {
		if len(order) < 2 {
			return nil, fmt.Errorf("invalid order_by format: %s", order)
		}

		field := order[1:]
		direction := "asc"
		if order[0] == '-' {
			direction = "desc"
		} else if order[0] != '+' {
			return nil, fmt.Errorf("invalid order_by prefix: %c", order[0])
		}

		if !slices.Contains(allowFields, field) {
			return nil, fmt.Errorf("ordering by field %s is not allowed", field)
		}

		query = append(query, func(q db.ChainInterface[T]) db.ChainInterface[T] {
			return q.Order(fmt.Sprintf("%s %s", field, direction))
		})
	}
	return query, nil
}
