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

package syncpolicy

import (
	"context"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type ISyncPolicyService interface {
	GetSyncPolicy(ctx context.Context, id int) (*SyncPolicy, error)
	CreateSyncPolicy(ctx context.Context, param *SyncPolicy) error
	UpdateSyncPolicy(ctx context.Context, param *SyncPolicy) error
	DeleteSyncPolicy(ctx context.Context, id int) error
	ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]*SyncPolicy, int64, error)
	GetSyncTask(ctx context.Context, id int) (*SyncTask, error)
	CreateSyncTask(ctx context.Context, param *SyncTask) (*SyncTask, error)
	UpdateSyncTask(ctx context.Context, param *SyncTask) error
	ListSyncTasksByPolicyID(ctx context.Context, policyID int, page, pageSize int, status SyncTaskStatus) ([]*SyncTask, int64, error)
	CreateSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) error
	CreateSyncTaskAsync(ctx context.Context, policy *SyncPolicy) (*SyncTask, error)

	ClaimDueSyncPolicies(ctx context.Context, nowMs int64) ([]job.DueJob, error)
	// CreatePendingSyncTask inserts a sync_tasks row only; sync_task_processor runs git work later.
	CreatePendingSyncTask(ctx context.Context, policyID int, triggerType int) error

	// ClaimPendingSyncTasks selects pending tasks and CAS-claims them for execution.
	ClaimPendingSyncTasks(ctx context.Context, nowMs int64) ([]job.DueJob, error)
	// ExecuteSyncTask generates jobs from a claimed task and creates pending sync_jobs.
	ExecuteSyncTask(ctx context.Context, taskID int) error

	ReportTaskStatus(ctx context.Context, taskID int) error
}

type SyncPolicyService struct {
	syncPolicyRepo ISyncPolicyRepo
	syncTaskRepo   ISyncTaskRepo
	syncJobService syncjob.ISyncJobService
	projectRepo    project.IProjectRepo
	jobGenerator   SyncJobGenerator
}

func NewSyncPolicyService(sprepo ISyncPolicyRepo, strepo ISyncTaskRepo, sjservice syncjob.ISyncJobService, prepo project.IProjectRepo, jobGenerator SyncJobGenerator) ISyncPolicyService {
	return &SyncPolicyService{
		syncPolicyRepo: sprepo,
		syncTaskRepo:   strepo,
		syncJobService: sjservice,
		projectRepo:    prepo,
		jobGenerator:   jobGenerator,
	}
}

func (sps *SyncPolicyService) GetSyncPolicy(ctx context.Context, id int) (*SyncPolicy, error) {
	return sps.syncPolicyRepo.GetSyncPolicy(ctx, id)
}

func (sps *SyncPolicyService) CreateSyncPolicy(ctx context.Context, param *SyncPolicy) error {
	if err := param.ApplyScheduleNextRun(time.Now()); err != nil {
		return err
	}
	return sps.syncPolicyRepo.CreateSyncPolicy(ctx, param)
}

func (sps *SyncPolicyService) UpdateSyncPolicy(ctx context.Context, param *SyncPolicy) error {
	if err := param.ApplyScheduleNextRun(time.Now()); err != nil {
		return err
	}
	return sps.syncPolicyRepo.UpdateSyncPolicy(ctx, param)
}

func (sps *SyncPolicyService) DeleteSyncPolicy(ctx context.Context, id int) error {
	return sps.syncPolicyRepo.DeleteSyncPolicy(ctx, id)
}

func (sps *SyncPolicyService) ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]*SyncPolicy, int64, error) {
	return sps.syncPolicyRepo.ListSyncPolicies(ctx, page, pageSize, search)
}

func (sps *SyncPolicyService) GetSyncTask(ctx context.Context, id int) (*SyncTask, error) {
	return sps.syncTaskRepo.GetSyncTask(ctx, id)
}

func (sps *SyncPolicyService) CreateSyncTask(ctx context.Context, syncTask *SyncTask) (*SyncTask, error) {
	return sps.syncTaskRepo.CreateSyncTask(ctx, syncTask)
}

func (sps *SyncPolicyService) UpdateSyncTask(ctx context.Context, syncTask *SyncTask) error {
	return sps.syncTaskRepo.UpdateSyncTask(ctx, syncTask)
}

func (sps *SyncPolicyService) ListSyncTasksByPolicyID(ctx context.Context, policyID int, page, pageSize int, status SyncTaskStatus) ([]*SyncTask, int64, error) {
	return sps.syncTaskRepo.ListSyncTasksByPolicyID(ctx, policyID, page, pageSize, status)
}

func (sps *SyncPolicyService) CreateSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) error {
	task, jobs, err := sps.jobGenerator.Generate(ctx, policy)
	if err != nil {
		return err
	}
	task, err = sps.syncTaskRepo.CreateSyncTask(ctx, task)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		job.SyncTaskID = task.ID
		if err := sps.syncJobService.CreateSyncJob(ctx, job); err != nil {
			log.Infow("CreateSyncJob failed", "error", err)
		}
	}
	return nil
}

const claimBatchLimit = 32

// ClaimDueSyncPolicies selects due policies and CAS-advances next_run_at for each successfully claimed row.
func (sps *SyncPolicyService) ClaimDueSyncPolicies(ctx context.Context, nowMs int64) ([]job.DueJob, error) {
	candidates, err := sps.syncPolicyRepo.SelectDuePolicies(ctx, nowMs, claimBatchLimit)
	if err != nil {
		return nil, err
	}
	var out []job.DueJob
	for _, p := range candidates {
		snapshot := p.NextRunAt
		nextNext, ok := p.nextRunAtAfterClaim(nowMs)
		if !ok {
			continue
		}
		claimed, err := sps.syncPolicyRepo.AdvanceNextRunAtCAS(ctx, p.ID, snapshot, nextNext, nowMs)
		if err != nil {
			return nil, err
		}
		if !claimed {
			continue
		}
		out = append(out, job.DueJob{
			ID:          p.ID,
			PolicyID:    p.ID,
			TriggerType: int(p.TriggerType),
			FireAtMs:    nowMs,
		})
	}
	return out, nil
}

// CreatePendingSyncTask inserts a sync_tasks row for the policy; sync_task_processor will generate jobs and execute.
func (sps *SyncPolicyService) CreatePendingSyncTask(ctx context.Context, policyID int, triggerType int) error {
	if _, err := sps.syncPolicyRepo.GetSyncPolicy(ctx, policyID); err != nil {
		return err
	}
	task := &SyncTask{
		SyncPolicyID:       policyID,
		TriggerType:        TriggerType(triggerType),
		Status:             SyncTaskStatusPending,
		StartedTimestamp:   0,
		CompletedTimestamp: 0,
		TotalItems:         0,
		SuccessfulItems:    0,
		CompletePercents:   0,
	}
	_, err := sps.syncTaskRepo.CreateSyncTask(ctx, task)
	return err
}

// CreateSyncTaskAsync creates a pending sync task for async processing by syncTaskProcessor.
func (sps *SyncPolicyService) CreateSyncTaskAsync(ctx context.Context, policy *SyncPolicy) (*SyncTask, error) {
	task := &SyncTask{
		SyncPolicyID: policy.ID,
		TriggerType:  policy.TriggerType,
		Status:       SyncTaskStatusPending,
	}
	return sps.syncTaskRepo.CreateSyncTask(ctx, task)
}

// ClaimPendingSyncTasks selects pending tasks and CAS-claims them for execution.
func (sps *SyncPolicyService) ClaimPendingSyncTasks(ctx context.Context, _ int64) ([]job.DueJob, error) {
	candidates, err := sps.syncTaskRepo.SelectPendingTasks(ctx, claimBatchLimit)
	if err != nil {
		return nil, err
	}
	var out []job.DueJob
	for _, t := range candidates {
		claimed, err := sps.syncTaskRepo.UpdateTaskStatusCAS(ctx, t.ID, SyncTaskStatusPending, SyncTaskStatusRunning)
		if err != nil {
			return nil, err
		}
		if !claimed {
			continue
		}
		out = append(out, job.DueJob{
			ID:          t.ID,
			PolicyID:    t.SyncPolicyID,
			TriggerType: int(t.TriggerType),
			FireAtMs:    time.Now().UnixMilli(),
		})
	}
	return out, nil
}

// ExecuteSyncTask generates sync jobs from a claimed task and creates pending sync_jobs.
func (sps *SyncPolicyService) ExecuteSyncTask(ctx context.Context, taskID int) error {
	task, err := sps.syncTaskRepo.GetSyncTask(ctx, taskID)
	if err != nil {
		return err
	}
	policy, err := sps.syncPolicyRepo.GetSyncPolicy(ctx, task.SyncPolicyID)
	if err != nil {
		return err
	}

	// Ensure the target project exists before generating jobs so that concurrent
	// job execution does not race on project creation.
	if policy.LocalProjectName != "" {
		if _, err := sps.projectRepo.GetProjectByName(ctx, policy.LocalProjectName); err != nil {
			if _, createErr := sps.projectRepo.CreateProject(ctx, &project.Project{
				Name: policy.LocalProjectName,
				Type: project.ProjectTypePublic,
			}); createErr != nil {
				log.Errorw("pre-create project failed", "project", policy.LocalProjectName, "error", createErr)
			}
		}
	}

	_, jobs, err := sps.jobGenerator.Generate(ctx, policy)
	if err != nil {
		task.Status = SyncTaskStatusFailed
		task.CompletedTimestamp = time.Now().Unix()
		if updateErr := sps.syncTaskRepo.UpdateSyncTask(ctx, task); updateErr != nil {
			log.Errorw("failed to mark task as failed after generate error", "taskID", taskID, "error", updateErr)
		}
		return err
	}

	task.StartedTimestamp = time.Now().Unix()
	task.TotalItems = len(jobs)
	if err := sps.syncTaskRepo.UpdateSyncTask(ctx, task); err != nil {
		return err
	}

	for _, j := range jobs {
		j.SyncTaskID = task.ID
		j.Status = syncjob.SyncJobStatusPending
		if err := sps.syncJobService.CreateSyncJob(ctx, j); err != nil {
			log.Errorw("create sync job failed", "error", err, "taskID", task.ID)
		}
	}
	return nil
}

// ReportTaskStatus aggregates job statuses for a task and updates the task record.
func (sps *SyncPolicyService) ReportTaskStatus(ctx context.Context, taskID int) error {
	jobs, _, err := sps.syncJobService.ListSyncJobsByTaskID(ctx, taskID, 1, 10000, syncjob.SyncJobStatusUnspecified, "")
	if err != nil {
		return err
	}

	var total, succeeded, failed, stopped, running, pending int
	for _, j := range jobs {
		total++
		switch j.Status {
		case syncjob.SyncJobStatusSucceeded:
			succeeded++
		case syncjob.SyncJobStatusFailed:
			failed++
		case syncjob.SyncJobStatusStopped:
			stopped++
		case syncjob.SyncJobStatusRunning:
			running++
		case syncjob.SyncJobStatusPending:
			pending++
		}
	}

	task, err := sps.syncTaskRepo.GetSyncTask(ctx, taskID)
	if err != nil {
		return err
	}

	task.TotalItems = total
	task.SuccessfulItems = succeeded
	task.FailedItems = failed
	task.StoppedItems = stopped
	if total > 0 {
		task.CompletePercents = (succeeded + failed + stopped) * 100 / total
	}

	if running == 0 && pending == 0 {
		task.CompletedTimestamp = time.Now().Unix()
		switch {
		case failed > 0:
			task.Status = SyncTaskStatusFailed
		case stopped > 0 && succeeded == 0:
			task.Status = SyncTaskStatusStopped
		default:
			task.Status = SyncTaskStatusSucceeded
		}
	}

	return sps.syncTaskRepo.UpdateSyncTask(ctx, task)
}
