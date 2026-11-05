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

package repo

import (
	"context"
	"database/sql"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
)

func TestSyncJobDB_JobTypes(t *testing.T) {
	database, _ := newSQLiteRepositoryTestDatabase(t)
	ctx := context.Background()
	repo := NewSyncJobDB(database)

	type row struct {
		JobType    string
		Params     sql.NullString
		SyncTaskID sql.NullInt64
	}
	readRow := func(id int) row {
		t.Helper()
		var r row
		require.NoError(t, database.Table("sync_jobs").Select("job_type, params, sync_task_id").
			Where("id = ?", id).Row().Scan(&r.JobType, &r.Params, &r.SyncTaskID))
		return r
	}

	// A job created without a type, as a sync task creates its jobs, is a
	// sync job without parameters.
	syncJob := &syncjob.SyncJob{
		RemoteProjectName:  "remote",
		RemoteResourceName: "resource",
		ResourceName:       "resource",
		ResourceType:       "model",
		SyncType:           "pull",
		Status:             syncjob.SyncJobStatusPending,
		SyncTaskID:         lo.ToPtr(5),
	}
	require.NoError(t, repo.CreateSyncJobs(ctx, []*syncjob.SyncJob{syncJob}))
	require.Equal(t, row{JobType: syncjob.JobTypeSync, SyncTaskID: sql.NullInt64{Int64: 5, Valid: true}}, readRow(syncJob.ID))

	// A model cache job keeps its parameters and belongs to no sync task,
	// also after the job server saves its status.
	cacheJob := &syncjob.SyncJob{
		JobType:            syncjob.JobTypeModelCache,
		RemoteRegistryID:   3,
		RemoteProjectName:  "upstream-org",
		RemoteResourceName: "gpt2",
		ProjectName:        "proxy",
		ResourceName:       "gpt2",
		ResourceType:       "model",
		SyncType:           "pull",
		Status:             syncjob.SyncJobStatusPending,
		Params:             []byte(`{"revision":"v1.0"}`),
	}
	require.NoError(t, repo.CreateSyncJob(ctx, cacheJob))
	want := row{JobType: syncjob.JobTypeModelCache, Params: sql.NullString{String: `{"revision":"v1.0"}`, Valid: true}}
	require.Equal(t, want, readRow(cacheJob.ID))

	stored, err := repo.GetSyncJob(ctx, cacheJob.ID)
	require.NoError(t, err)
	require.Equal(t, syncjob.JobTypeModelCache, stored.JobType)
	require.Nil(t, stored.SyncTaskID)
	require.False(t, stored.HasSyncTask())
	params, err := stored.ModelCacheParams()
	require.NoError(t, err)
	require.Equal(t, "v1.0", params.Revision)

	stored.Status = syncjob.SyncJobStatusSucceeded
	require.NoError(t, repo.UpdateSyncJob(ctx, stored))
	require.Equal(t, want, readRow(cacheJob.ID))

	// The jobs of a sync task never include it.
	jobs, total, err := repo.ListSyncJobsByTaskID(ctx, 0, 1, 10, syncjob.SyncJobStatusUnspecified, "")
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, jobs)
}
