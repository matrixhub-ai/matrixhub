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
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
)

func TestSyncTaskDB_RecountTask(t *testing.T) {
	ctx := context.Background()

	t.Run("counts the task's jobs and saves only the status and counts", func(t *testing.T) {
		database, _ := newSQLiteRepositoryTestDatabase(t)
		repo := NewSyncTaskDB(database)
		policyID := createRecountTestPolicy(t, database)
		task, _ := createRecountTestTask(t, database, policyID,
			syncjob.SyncJobStatusSucceeded, syncjob.SyncJobStatusSucceeded,
			syncjob.SyncJobStatusFailed, syncjob.SyncJobStatusRunning)
		// The job of another task must not be counted.
		createRecountTestTask(t, database, policyID, syncjob.SyncJobStatusStopped)

		err := repo.RecountTask(ctx, task.ID, func(locked *syncpolicy.SyncTask, counts map[syncjob.SyncJobStatus]int) {
			require.Equal(t, task.ID, locked.ID)
			require.Equal(t, map[syncjob.SyncJobStatus]int{
				syncjob.SyncJobStatusSucceeded: 2,
				syncjob.SyncJobStatusFailed:    1,
				syncjob.SyncJobStatusRunning:   1,
			}, counts)
			locked.TotalItems = 4
			locked.SuccessfulItems = 2
			locked.FailedItems = 1
			locked.CompletePercents = 75
			locked.StartedTimestamp = 999 // not a field that a recount saves
		})
		require.NoError(t, err)

		saved, err := repo.GetSyncTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, syncpolicy.SyncTaskStatusRunning, saved.Status)
		require.Equal(t, 4, saved.TotalItems)
		require.Equal(t, 2, saved.SuccessfulItems)
		require.Equal(t, 1, saved.FailedItems)
		require.Equal(t, 75, saved.CompletePercents)
		require.Equal(t, task.StartedTimestamp, saved.StartedTimestamp)
	})

	t.Run("a missing task is not found", func(t *testing.T) {
		database, _ := newSQLiteRepositoryTestDatabase(t)

		err := NewSyncTaskDB(database).RecountTask(ctx, 404, func(*syncpolicy.SyncTask, map[syncjob.SyncJobStatus]int) {
			t.Error("update ran for a missing task")
		})
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("jobs that end and report at once leave the task succeeded", func(t *testing.T) {
		database, _ := newSQLiteRepositoryTestDatabase(t)
		tasks := NewSyncTaskDB(database)
		jobRepo := NewSyncJobDB(database)
		svc := syncpolicy.NewSyncPolicyService(nil, tasks, nil, nil, nil)
		policyID := createRecountTestPolicy(t, database)

		for range 20 {
			task, jobs := createRecountTestTask(t, database, policyID,
				syncjob.SyncJobStatusRunning, syncjob.SyncJobStatusRunning, syncjob.SyncJobStatusRunning,
				syncjob.SyncJobStatusRunning, syncjob.SyncJobStatusRunning)

			// As in the job server, each job saves its own status, then reports.
			errs := make(chan error, len(jobs))
			var wg sync.WaitGroup
			for _, job := range jobs {
				wg.Add(1)
				go func() {
					defer wg.Done()
					job.Status = syncjob.SyncJobStatusSucceeded
					if err := jobRepo.UpdateSyncJob(ctx, job); err != nil {
						errs <- err
						return
					}
					errs <- svc.ReportTaskStatus(ctx, task.ID)
				}()
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				require.NoError(t, err)
			}

			saved, err := tasks.GetSyncTask(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, syncpolicy.SyncTaskStatusSucceeded, saved.Status)
			require.Equal(t, 5, saved.SuccessfulItems)
			require.Equal(t, 100, saved.CompletePercents)
		}
	})

	t.Run("a job that ends during a recount is counted by the next one", func(t *testing.T) {
		database, _ := newSQLiteRepositoryTestDatabase(t)
		sqlDB, err := database.DB()
		require.NoError(t, err)
		tasks := NewSyncTaskDB(database)
		jobRepo := NewSyncJobDB(database)
		policyID := createRecountTestPolicy(t, database)
		task, jobs := createRecountTestTask(t, database, policyID,
			syncjob.SyncJobStatusSucceeded, syncjob.SyncJobStatusRunning)
		countSucceeded := func(locked *syncpolicy.SyncTask, counts map[syncjob.SyncJobStatus]int) {
			locked.SuccessfulItems = counts[syncjob.SyncJobStatusSucceeded]
		}

		waitsBefore := sqlDB.Stats().WaitCount
		reported := make(chan error, 1)
		require.NoError(t, tasks.RecountTask(ctx, task.ID, func(locked *syncpolicy.SyncTask, counts map[syncjob.SyncJobStatus]int) {
			// The second job ends and reports while this recount, which saw
			// it running, is still in its transaction.
			go func() {
				jobs[1].Status = syncjob.SyncJobStatusSucceeded
				if err := jobRepo.UpdateSyncJob(ctx, jobs[1]); err != nil {
					reported <- err
					return
				}
				reported <- tasks.RecountTask(ctx, task.ID, countSucceeded)
			}()
			// Go on once the job's save waits for the connection this recount
			// holds. Without the transaction nothing makes it wait: the report
			// finishes first, and this recount then saves its stale count.
			deadline := time.Now().Add(10 * time.Second)
			for sqlDB.Stats().WaitCount == waitsBefore && len(reported) == 0 {
				if time.Now().After(deadline) {
					t.Error("the job's report neither waited nor finished")
					break
				}
				time.Sleep(time.Millisecond)
			}
			countSucceeded(locked, counts)
		}))
		require.NoError(t, <-reported)

		saved, err := tasks.GetSyncTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, 2, saved.SuccessfulItems)
	})
}

// TestSyncTaskDB_RecountTaskSQL checks the statements a recount sends on
// MySQL, which the SQLite tests above cannot show.
func TestSyncTaskDB_RecountTaskSQL(t *testing.T) {
	ctx := context.Background()
	lockTask := regexp.QuoteMeta("SELECT * FROM `sync_tasks` WHERE `sync_tasks`.`id` = ? ORDER BY `sync_tasks`.`id` LIMIT ? FOR UPDATE")
	countJobs := regexp.QuoteMeta("SELECT status, COUNT(*) AS jobs FROM `sync_jobs` WHERE sync_task_id = ? GROUP BY `status`")
	taskRow := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "status", "started_timestamp", "total_items", "stopped_items"}).
			AddRow(7, int64(syncpolicy.SyncTaskStatusRunning), 123, 3, 2)
	}

	t.Run("locks the task row before it counts the jobs, and updates only the recounted columns", func(t *testing.T) {
		repo, mock := newSyncTaskDBForSQLTest(t)
		mock.ExpectBegin()
		mock.ExpectQuery(lockTask).WithArgs(7, 1).WillReturnRows(taskRow())
		mock.ExpectQuery(countJobs).WithArgs(7).
			WillReturnRows(sqlmock.NewRows([]string{"status", "jobs"}).AddRow(int64(syncjob.SyncJobStatusSucceeded), 3))
		mock.ExpectExec(regexp.QuoteMeta("UPDATE `sync_tasks` SET `status`=?,`completed_timestamp`=?,`total_items`=?,`successful_items`=?,`stopped_items`=?,`failed_items`=?,`complete_percents`=?,`updated_at`=? WHERE `id` = ?")).
			WithArgs(syncpolicy.SyncTaskStatusSucceeded, 456, 3, 3, 0, 0, 100, sqlmock.AnyArg(), 7).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.RecountTask(ctx, 7, func(task *syncpolicy.SyncTask, counts map[syncjob.SyncJobStatus]int) {
			require.Equal(t, map[syncjob.SyncJobStatus]int{syncjob.SyncJobStatusSucceeded: 3}, counts)
			task.Status = syncpolicy.SyncTaskStatusSucceeded
			task.CompletedTimestamp = 456
			task.SuccessfulItems = 3
			task.StoppedItems = 0 // a zero is written too
			task.CompletePercents = 100
			task.StartedTimestamp = 999 // not a recounted column, so not written
		})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a failed count rolls back without calling update", func(t *testing.T) {
		repo, mock := newSyncTaskDBForSQLTest(t)
		countErr := errors.New("count failed")
		mock.ExpectBegin()
		mock.ExpectQuery(lockTask).WithArgs(7, 1).WillReturnRows(taskRow())
		mock.ExpectQuery(countJobs).WithArgs(7).WillReturnError(countErr)
		mock.ExpectRollback()

		err := repo.RecountTask(ctx, 7, func(*syncpolicy.SyncTask, map[syncjob.SyncJobStatus]int) {
			t.Error("update ran after the count failed")
		})
		require.ErrorIs(t, err, countErr)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func newSyncTaskDBForSQLTest(t *testing.T) (syncpolicy.ISyncTaskRepo, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, sqlDB.Close())
	})

	database, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)

	return NewSyncTaskDB(database), mock
}

func createRecountTestPolicy(t *testing.T, database *gorm.DB) int {
	t.Helper()
	require.NoError(t, database.Exec(`INSERT INTO sync_policies
		(name, next_run_at, updated_at) VALUES (?, ?, ?)`, "recount", 0, time.Now()).Error)

	var policy struct{ ID int }
	require.NoError(t, database.Table("sync_policies").Select("id").Where("name = ?", "recount").Take(&policy).Error)
	return policy.ID
}

// createRecountTestTask creates a running task of the policy, with one job in
// each of the given statuses.
func createRecountTestTask(t *testing.T, database *gorm.DB, policyID int, statuses ...syncjob.SyncJobStatus) (*syncpolicy.SyncTask, []*syncjob.SyncJob) {
	t.Helper()
	task := &syncpolicy.SyncTask{
		SyncPolicyID:     policyID,
		TriggerType:      syncpolicy.TriggerTypeManual,
		Status:           syncpolicy.SyncTaskStatusRunning,
		StartedTimestamp: 123,
		TotalItems:       len(statuses),
	}
	require.NoError(t, database.Create(task).Error)

	jobs := make([]*syncjob.SyncJob, 0, len(statuses))
	for _, status := range statuses {
		job := &syncjob.SyncJob{
			RemoteProjectName:  "remote",
			RemoteResourceName: "resource",
			ResourceName:       "resource",
			ResourceType:       "model",
			SyncType:           "pull",
			Status:             status,
			SyncTaskID:         task.ID,
		}
		require.NoError(t, database.Create(job).Error)
		jobs = append(jobs, job)
	}
	return task, jobs
}
