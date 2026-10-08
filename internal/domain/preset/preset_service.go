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

package preset

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

const reconcileConcurrency = 2

type Service struct {
	enabled     bool
	manifest    *Manifest
	presetRepo  IPresetRepo
	projectRepo project.IProjectRepo
	modelRepo   model.IModelRepo
	gitRepo     git.IGitRepo
}

func NewService(enabled bool, manifest *Manifest, presetRepo IPresetRepo, projectRepo project.IProjectRepo,
	modelRepo model.IModelRepo, gitRepo git.IGitRepo,
) *Service {
	return &Service{
		enabled: enabled, manifest: manifest, presetRepo: presetRepo, projectRepo: projectRepo,
		modelRepo: modelRepo, gitRepo: gitRepo,
	}
}

// Apply is database-only and best effort; a failed seed must not prevent startup.
func (s *Service) Apply(ctx context.Context) {
	if !s.enabled {
		return
	}
	applied, err := s.presetRepo.SeedIfEmpty(ctx, s.manifest)
	if err != nil {
		log.Warnw("preset initialization failed", "error", err)
		return
	}
	log.Infow("preset initialization completed", "applied", applied)
}

// Reconcile repairs local repositories in one bounded pass, without upstream
// requests or writes to model records, metadata, or labels.
func (s *Service) Reconcile(ctx context.Context) {
	if !s.enabled {
		return
	}
	jobs := make(chan ModelSpec)
	var workers sync.WaitGroup
	for range reconcileConcurrency {
		workers.Go(func() {
			for spec := range jobs {
				if ctx.Err() != nil {
					return
				}
				if err := s.reconcileModel(ctx, spec); err != nil {
					log.Warnw("preset model reconciliation failed", "project", spec.Project, "model", spec.Name, "error", err)
				}
			}
		})
	}
	for _, spec := range s.manifest.Models {
		select {
		case jobs <- spec:
		case <-ctx.Done():
		}
	}
	close(jobs)
	workers.Wait()
	log.Info("preset model reconciliation completed")
}

func (s *Service) reconcileModel(ctx context.Context, spec ModelSpec) error {
	_, err := s.modelRepo.GetByProjectAndName(ctx, spec.Project, spec.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	prj, err := s.projectRepo.GetProjectByName(ctx, spec.Project)
	if err != nil {
		return err
	}
	if !prj.HasProxy() {
		return nil
	}
	exists, err := s.gitRepo.RepositoryExists(ctx, "models", spec.Project, spec.Name)
	if err != nil {
		return fmt.Errorf("check preset repository: %w", err)
	}
	if !exists {
		if err := s.gitRepo.CreateRepository(ctx, "models", spec.Project, spec.Name); err != nil {
			return fmt.Errorf("repair preset repository: %w", err)
		}
	}
	return nil
}
