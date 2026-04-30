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

package syncjob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type ISyncJobService interface {
	GetSyncJob(ctx context.Context, id int) (*SyncJob, error)
	CreateSyncJob(ctx context.Context, param *SyncJob) error
	UpdateSyncJob(ctx context.Context, param *SyncJob) error
	ExecuteSyncJob(ctx context.Context, param *SyncJob) error
	ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status SyncJobStatus, resourceType string) ([]*SyncJob, int64, error)

	ClaimPendingSyncJobs(ctx context.Context, nowMs int64) ([]job.DueJob, error)
	ExecuteSyncJobWithLog(ctx context.Context, jobID int) error

	SetOnJobDone(fn func(ctx context.Context, taskID int) error)
}

type SyncJobService struct {
	syncJobRepo  ISyncJobRepo
	registryRepo registry.IRegistryRepo
	projectRepo  project.IProjectRepo
	modelRepo    model.IModelRepo
	gitRepo      git.IGitRepo
	logStore     LogStore
	onJobDone    func(ctx context.Context, taskID int) error
}

type LogStore interface {
	Writer(jobID int) (io.WriteCloser, error)
	Reader(jobID int) (io.ReadCloser, error)
}

func NewSyncJobService(srepo ISyncJobRepo, rrepo registry.IRegistryRepo, prepo project.IProjectRepo, mrepo model.IModelRepo, grepo git.IGitRepo, logStore LogStore) ISyncJobService {
	return &SyncJobService{
		syncJobRepo:  srepo,
		registryRepo: rrepo,
		projectRepo:  prepo,
		modelRepo:    mrepo,
		gitRepo:      grepo,
		logStore:     logStore,
	}
}

// SetOnJobDone sets the callback invoked after a job finishes (used for task status aggregation).
func (sjs *SyncJobService) SetOnJobDone(fn func(ctx context.Context, taskID int) error) {
	sjs.onJobDone = fn
}

func (sjs *SyncJobService) GetSyncJob(ctx context.Context, id int) (*SyncJob, error) {
	return sjs.syncJobRepo.GetSyncJob(ctx, id)
}

func (sjs *SyncJobService) CreateSyncJob(ctx context.Context, syncJob *SyncJob) error {
	return sjs.syncJobRepo.CreateSyncJob(ctx, syncJob)
}

func (sjs *SyncJobService) UpdateSyncJob(ctx context.Context, syncJob *SyncJob) error {
	return sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob)
}

func (sjs *SyncJobService) ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status SyncJobStatus, resourceType string) ([]*SyncJob, int64, error) {
	return sjs.syncJobRepo.ListSyncJobsByTaskID(ctx, taskID, page, pageSize, status, resourceType)
}

func (sjs *SyncJobService) ExecuteSyncJob(ctx context.Context, syncJob *SyncJob) error {
	defer func() {
		syncJob.CompletedTimestamp = time.Now().Unix()
		if err := sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob); err != nil {
			fmt.Printf("update sync job failed: %v\n", err)
		}
	}()

	return sjs.executeGitJob(ctx, syncJob, nil)
}

const claimJobBatchLimit = 32

// ClaimPendingSyncJobs selects pending jobs and CAS-claims them for execution.
func (sjs *SyncJobService) ClaimPendingSyncJobs(ctx context.Context, _ int64) ([]job.DueJob, error) {
	candidates, err := sjs.syncJobRepo.SelectPendingJobs(ctx, claimJobBatchLimit)
	if err != nil {
		return nil, err
	}
	var out []job.DueJob
	for _, j := range candidates {
		claimed, err := sjs.syncJobRepo.UpdateJobStatusCAS(ctx, j.ID, SyncJobStatusPending, SyncJobStatusRunning)
		if err != nil {
			return nil, err
		}
		if !claimed {
			continue
		}
		out = append(out, job.DueJob{
			ID:          j.ID,
			PolicyID:    0,
			TriggerType: 0,
			FireAtMs:    time.Now().UnixMilli(),
		})
	}
	return out, nil
}

// ExecuteSyncJobWithLog executes a sync job with logging and status reporting.
func (sjs *SyncJobService) ExecuteSyncJobWithLog(ctx context.Context, jobID int) error {
	syncJob, err := sjs.syncJobRepo.GetSyncJob(ctx, jobID)
	if err != nil {
		return err
	}

	var logWriter io.WriteCloser
	if sjs.logStore != nil {
		logWriter, err = sjs.logStore.Writer(jobID)
		if err != nil {
			log.Errorw("open job log failed", "jobID", jobID, "error", err)
		}
	}
	if logWriter != nil {
		defer func() { _ = logWriter.Close() }()
	}

	// 如果 job 已被外部停止（例如用户手动停止 task），跳过执行
	if syncJob.Status == SyncJobStatusStopped {
		if logWriter != nil {
			_, _ = fmt.Fprintln(logWriter, "[INFO] Job was stopped before execution started, skipping")
		}
		syncJob.CompletedTimestamp = time.Now().Unix()
		if updateErr := sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob); updateErr != nil {
			log.Errorw("update sync job status failed", "jobID", jobID, "error", updateErr)
		}
		if sjs.onJobDone != nil {
			if repErr := sjs.onJobDone(ctx, syncJob.SyncTaskID); repErr != nil {
				log.Errorw("report task status failed", "taskID", syncJob.SyncTaskID, "error", repErr)
			}
		}
		return nil
	}

	err = sjs.executeGitJob(ctx, syncJob, logWriter)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			syncJob.Status = SyncJobStatusStopped
		} else {
			syncJob.Status = SyncJobStatusFailed
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "\n[ERROR] %v\n", err)
		}
	} else {
		syncJob.Status = SyncJobStatusSucceeded
	}
	syncJob.CompletedTimestamp = time.Now().Unix()

	if updateErr := sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob); updateErr != nil {
		log.Errorw("update sync job status failed", "jobID", jobID, "error", updateErr)
	}

	if sjs.onJobDone != nil {
		if repErr := sjs.onJobDone(ctx, syncJob.SyncTaskID); repErr != nil {
			log.Errorw("report task status failed", "taskID", syncJob.SyncTaskID, "error", repErr)
		}
	}

	return err
}

func (sjs *SyncJobService) executeGitJob(ctx context.Context, syncJob *SyncJob, logWriter io.Writer) error {
	reg, err := sjs.registryRepo.GetRegistry(ctx, syncJob.RemoteRegistryID)
	if err != nil {
		return fmt.Errorf("get registry(id=%d): %w", syncJob.RemoteRegistryID, err)
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] job start, id=%d, registry=%s (id=%d), resource_type=%s\n",
			syncJob.ID, reg.URL, syncJob.RemoteRegistryID, syncJob.ResourceType)
		_, _ = fmt.Fprintf(logWriter, "[INFO] source=%s/%s -> target=%s/%s, sync_type=%s\n",
			syncJob.RemoteProjectName, syncJob.RemoteResourceName,
			syncJob.ProjectName, syncJob.ResourceName, syncJob.SyncType)
	}

	switch syncJob.SyncType {
	case "pull":
		return sjs.executePullJob(ctx, syncJob, reg, logWriter)
	case "push":
		return sjs.executePushJob(ctx, syncJob, reg, logWriter)
	default:
		return fmt.Errorf("unknown sync type: %s", syncJob.SyncType)
	}
}

func (sjs *SyncJobService) executePullJob(ctx context.Context, syncJob *SyncJob, reg *registry.Registry, logWriter io.Writer) error {
	prj, err := sjs.projectRepo.GetProjectByName(ctx, syncJob.ProjectName)
	if err != nil {
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] project %s not found, creating new project\n", syncJob.ProjectName)
		}
		prj = &project.Project{
			Name: syncJob.ProjectName,
			Type: project.ProjectTypePublic,
		}
		prj, err = sjs.projectRepo.CreateProject(ctx, prj)
		if err != nil {
			return fmt.Errorf("create project(%s): %w", syncJob.ProjectName, err)
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] created project %s (id=%d)\n", prj.Name, prj.ID)
		}
	}
	gr := &git.GitRepository{
		RemoteRegistryURL:  reg.URL,
		RemoteProjectName:  syncJob.RemoteProjectName,
		RemoteResourceName: syncJob.RemoteResourceName,
		ProjectName:        syncJob.ProjectName,
		ResourceName:       syncJob.ResourceName,
		ResourceType:       syncJob.ResourceType,
		LogWriter:          logWriter,
	}
	mod, _ := sjs.modelRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
	if mod != nil {
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] model exists, pulling from remote\n")
		}
	} else {
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] model not found, creating model record and cloning from remote\n")
		}
		mod = &model.Model{
			Name:        syncJob.ResourceName,
			ProjectID:   prj.ID,
			ProjectName: syncJob.ProjectName,
		}

		if _, err = sjs.modelRepo.Create(ctx, mod); err != nil {
			return err
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] created model record (id=%d)\n", mod.ID)
		}
	}

	if err = sjs.gitRepo.PullFromRemote(ctx, gr); err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] pull completed\n")
	}

	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] job completed successfully\n")
	}
	return nil
}

func (sjs *SyncJobService) executePushJob(ctx context.Context, syncJob *SyncJob, reg *registry.Registry, logWriter io.Writer) error {
	mod, err := sjs.modelRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
	if err != nil || mod == nil {
		return fmt.Errorf("local model %s/%s not found: %w", syncJob.ProjectName, syncJob.ResourceName, err)
	}

	gr := &git.GitRepository{
		RemoteRegistryURL:  reg.URL,
		RemoteProjectName:  syncJob.RemoteProjectName,
		RemoteResourceName: syncJob.RemoteResourceName,
		ProjectName:        syncJob.ProjectName,
		ResourceName:       syncJob.ResourceName,
		ResourceType:       syncJob.ResourceType,
	}
	if reg.GetCredential() != nil {
		if bc := registry.AsBasic(reg.GetCredential()); bc != nil {
			gr.Credential = &git.BasicCredential{
				Username: bc.Username,
				Password: bc.Password,
			}
		}
	}

	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] pushing to remote registry\n")
	}
	if err = sjs.gitRepo.PushToRemote(ctx, gr); err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] push completed\n")
	}
	return nil
}
