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
	"fmt"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
)

type ISyncJobService interface {
	GetSyncJob(ctx context.Context, id int) (*SyncJob, error)
	CreateSyncJob(ctx context.Context, param *SyncJob) error
	CreateAndExcecuteSyncJob(ctx context.Context, param *SyncJob) error
	ExecuteSyncJob(ctx context.Context, param *SyncJob) error
}

type SyncJobService struct {
	syncJobRepo  ISyncJobRepo
	registryRepo registry.IRegistryRepo
	projectRepo  project.IProjectRepo
	modelRepo    model.IModelRepo
	gitRepo      git.IGitRepo
}

func NewSyncJobService(srepo ISyncJobRepo, rrepo registry.IRegistryRepo, prepo project.IProjectRepo, mrepo model.IModelRepo, grepo git.IGitRepo) ISyncJobService {
	return &SyncJobService{
		syncJobRepo:  srepo,
		registryRepo: rrepo,
		projectRepo:  prepo,
		modelRepo:    mrepo,
		gitRepo:      grepo,
	}
}

func (sjs *SyncJobService) GetSyncJob(ctx context.Context, id int) (*SyncJob, error) {
	return sjs.syncJobRepo.GetSyncJob(ctx, id)
}

func (sjs *SyncJobService) CreateSyncJob(ctx context.Context, syncJob *SyncJob) error {
	return sjs.syncJobRepo.CreateSyncJob(ctx, syncJob)
}

func (sjs *SyncJobService) CreateAndExcecuteSyncJob(ctx context.Context, syncJob *SyncJob) error {
	if err := sjs.syncJobRepo.CreateSyncJob(ctx, syncJob); err != nil {
		return err
	}
	return sjs.ExecuteSyncJob(ctx, syncJob)
}

func (sjs *SyncJobService) ExecuteSyncJob(ctx context.Context, syncJob *SyncJob) error {
	reg, err := sjs.registryRepo.GetRegistry(ctx, syncJob.RemoteRegistryID)
	if err != nil {
		return fmt.Errorf("get registry(id=%d): %w", syncJob.RemoteRegistryID, err)
	}
	prj, err := sjs.projectRepo.GetProjectByName(ctx, syncJob.ProjectName)
	if err != nil {
		if syncJob.HasSyncTask() {
			prj = &project.Project{
				Name: syncJob.ProjectName,
			}
			prj, err = sjs.projectRepo.CreateProject(ctx, prj)
			if err != nil {
				return fmt.Errorf("create project(%s): %w", syncJob.ProjectName, err)
			}
		} else {
			return fmt.Errorf("get project(%s): %w", syncJob.ProjectName, err)
		}
	}
	gr := &git.GitRepository{
		RemoteRegistryURL:  reg.URL,
		RemoteProjectName:  syncJob.RemoteProjectName,
		RemoteResourceName: syncJob.RemoteResourceName,
		ProjectName:        syncJob.ProjectName,
		ResourceName:       syncJob.ResourceName,
		ResourceType:       syncJob.ResourceType,
	}
	mod, _ := sjs.modelRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
	if mod != nil {
		if err = sjs.gitRepo.PullFromRemote(ctx, gr); err != nil {
			return err
		}
	} else {
		mod = &model.Model{
			Name:        syncJob.ResourceName,
			ProjectID:   prj.ID,
			ProjectName: syncJob.ProjectName,
		}

		if _, err = sjs.modelRepo.Create(ctx, mod); err != nil {
			return err
		}
		if err = sjs.gitRepo.CloneFromRemote(ctx, gr); err != nil {
			return err
		}
	}

	return nil
}
