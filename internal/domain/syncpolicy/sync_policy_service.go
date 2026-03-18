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

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type ISyncPolicyService interface {
	GetSyncPolicy(ctx context.Context, param *SyncPolicy) (*SyncPolicy, error)
	CreateSyncPolicy(ctx context.Context, param *SyncPolicy) error
	UpdateSyncPolicy(ctx context.Context, param *SyncPolicy) error
	DeleteSyncPolicy(ctx context.Context, param *SyncPolicy) error
	GetSyncTask(ctx context.Context, param *SyncTask) (*SyncTask, error)
	CreateSyncTask(ctx context.Context, param *SyncTask) (*SyncTask, error)
	CreateSyncTaskAndSyncJobs(ctx context.Context, param *SyncPolicy) error
	CreateExcecuteSyncTaskAndSyncJobs(ctx context.Context, param *SyncPolicy) error
}

type SyncPolicyService struct {
	syncPolicyRepo ISyncPolicyRepo
	syncTaskRepo   ISyncTaskRepo
	syncJobService syncjob.ISyncJobService
}

func NewSyncPolicyService(sprepo ISyncPolicyRepo, strepo ISyncTaskRepo, sjservice syncjob.ISyncJobService) ISyncPolicyService {
	return &SyncPolicyService{
		syncPolicyRepo: sprepo,
		syncTaskRepo:   strepo,
		syncJobService: sjservice,
	}
}

func (sps *SyncPolicyService) GetSyncPolicy(ctx context.Context, param *SyncPolicy) (*SyncPolicy, error) {
	return sps.syncPolicyRepo.GetSyncPolicy(ctx, param)
}

func (sps *SyncPolicyService) CreateSyncPolicy(ctx context.Context, param *SyncPolicy) error {
	return sps.syncPolicyRepo.CreateSyncPolicy(ctx, param)
}
func (sps *SyncPolicyService) UpdateSyncPolicy(ctx context.Context, param *SyncPolicy) error {
	return sps.syncPolicyRepo.UpdateSyncPolicy(ctx, param)
}

func (sps *SyncPolicyService) DeleteSyncPolicy(ctx context.Context, param *SyncPolicy) error {
	return sps.syncPolicyRepo.DeleteSyncPolicy(ctx, param)
}
func (sps *SyncPolicyService) GetSyncTask(ctx context.Context, syncTask *SyncTask) (*SyncTask, error) {
	return sps.syncTaskRepo.GetSyncTask(ctx, syncTask)
}

func (sps *SyncPolicyService) CreateSyncTask(ctx context.Context, syncTask *SyncTask) (*SyncTask, error) {
	return sps.syncTaskRepo.CreateSyncTask(ctx, syncTask)
}

func (sps *SyncPolicyService) CreateSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) error {
	task, jobs, err := sps.syncPolicyRepo.GenerateSyncTaskAndSyncJobs(ctx, policy)
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

func (sps *SyncPolicyService) CreateExcecuteSyncTaskAndSyncJobs(ctx context.Context, policy *SyncPolicy) error {
	task, jobs, err := sps.syncPolicyRepo.GenerateSyncTaskAndSyncJobs(ctx, policy)
	if err != nil {
		return err
	}
	task, err = sps.syncTaskRepo.CreateSyncTask(ctx, task)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		job.SyncTaskID = task.ID
		if err := sps.syncJobService.CreateAndExcecuteSyncJob(ctx, job); err != nil {
			log.Infow("CreateAndExcecuteSyncJob failed", "error", err)
		}
	}
	return nil
}
