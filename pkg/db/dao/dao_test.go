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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/matrixhub-ai/matrixhub/pkg/db"
)

type Data struct {
	db.Model
	ID   uint
	Name string
}

func (d Data) GetID() uint {
	return d.ID
}

func TestDAO(t *testing.T) {
	ctx := context.Background()
	database, err := db.New("sqlite3", "file::memory:?cache=shared", true)
	assert.NoError(t, err)

	dao := NewDAO(db.G[Data](database))

	err = dao.MigrateTable(ctx)
	assert.NoError(t, err)

	data := Data{
		Name: "test",
	}

	t.Run("Create and Get Data", func(t *testing.T) {
		err := dao.Create(ctx, data)
		assert.NoError(t, err)

		result, err := dao.GetByName(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, "test", result.Name)
	})

	t.Run("Update Data Name", func(t *testing.T) {
		result, err := dao.GetByName(ctx, "test")
		assert.NoError(t, err)

		rowsAffected, err := dao.UpdateName(ctx, result.ID, "updated-test")
		assert.NoError(t, err)
		assert.Equal(t, 1, rowsAffected)

		updatedResult, err := dao.GetByName(ctx, "updated-test")
		assert.NoError(t, err)
		assert.Equal(t, "updated-test", updatedResult.Name)
	})

	t.Run("Delete Data", func(t *testing.T) {
		err := dao.Create(ctx, Data{
			Name: "to-be-deleted",
		})
		assert.NoError(t, err)

		result, err := dao.GetByName(ctx, "updated-test")
		assert.NoError(t, err)

		rowsAffected, err := dao.Delete(ctx, result.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, rowsAffected)

		_, err = dao.GetByName(ctx, "updated-test")
		assert.Error(t, err)
	})

	t.Run("Clear Deleted Data", func(t *testing.T) {
		result, err := dao.GetByName(ctx, "to-be-deleted")
		assert.NoError(t, err)

		rowsAffected, err := dao.Delete(ctx, result.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1, rowsAffected)

		rowsCleared, err := dao.ClearDeleted(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 2, rowsCleared)
	})

	t.Run("List All Data", func(t *testing.T) {
		// Re-create a data to test ClearDeleted
		err := dao.Create(ctx, Data{
			Name: "to-be-deleted",
		})
		assert.NoError(t, err)

		err = dao.Create(ctx, Data{
			Name: "to-be-deleted-2",
		})
		assert.NoError(t, err)

		results, _, err := dao.List(ctx, "", 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(results))
	})
}
