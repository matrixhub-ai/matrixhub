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
	"fmt"
	"strings"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registrydiscovery"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
)

// SyncJobGenerator generates sync tasks and jobs from a sync policy.
// The abstraction decouples job generation logic from the database layer,
// making it testable and extensible for future policy types.
type SyncJobGenerator interface {
	Generate(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error)
}

type syncJobGenerator struct {
	registryRepo registry.IRegistryRepo
	discoveries  map[string]registrydiscovery.Discovery
}

// NewSyncJobGenerator creates a new SyncJobGenerator instance.
func NewSyncJobGenerator(registryRepo registry.IRegistryRepo, discoveries map[string]registrydiscovery.Discovery) SyncJobGenerator {
	return &syncJobGenerator{
		registryRepo: registryRepo,
		discoveries:  discoveries,
	}
}

func (g *syncJobGenerator) Generate(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error) {
	task := g.buildTask(policy)

	var jobs []*syncjob.SyncJob
	var err error

	if policy.IsPullBase() && policy.HasWildcardResourceName() {
		jobs, err = g.buildJobsFromDiscovery(ctx, policy)
	} else {
		jobs = g.buildJobsFromStatic(policy)
	}
	if err != nil {
		return nil, nil, err
	}

	task.TotalItems = len(jobs)
	return task, jobs, nil
}

func (g *syncJobGenerator) buildTask(policy *SyncPolicy) *SyncTask {
	return &SyncTask{
		SyncPolicyID:       policy.ID,
		TriggerType:        policy.TriggerType,
		Status:             SyncTaskStatusRunning,
		StartedTimestamp:   time.Now().Unix(),
		CompletedTimestamp: 0,
		SuccessfulItems:    0,
		StoppedItems:       0,
		FailedItems:        0,
		CompletePercents:   0,
	}
}

func (g *syncJobGenerator) buildJobsFromStatic(policy *SyncPolicy) []*syncjob.SyncJob {
	resourceTypes := g.parseResourceTypes(policy.ResourceTypes)
	var jobs []*syncjob.SyncJob
	for _, rt := range resourceTypes {
		jobs = append(jobs, g.buildJob(policy, rt))
	}
	return jobs
}

func (g *syncJobGenerator) buildJobsFromDiscovery(ctx context.Context, policy *SyncPolicy) ([]*syncjob.SyncJob, error) {
	reg, err := g.registryRepo.GetRegistry(ctx, policy.RegistryID)
	if err != nil {
		return nil, fmt.Errorf("get registry(id=%d): %w", policy.RegistryID, err)
	}

	providerKey := registrydiscovery.KeyFromRegistryType(reg.Type)
	disc, ok := g.discoveries[providerKey]
	if !ok {
		return nil, fmt.Errorf("no discovery for registry type: %s (key=%s)", reg.Type, providerKey)
	}

	resourceTypes := g.parseResourceTypes(policy.ResourceTypes)
	var allRepos []registrydiscovery.RemoteRepository
	for _, rt := range resourceTypes {
		repos, err := disc.ListRepositories(ctx, reg, registrydiscovery.Filter{
			Namespace:    policy.RemoteProjectName,
			ResourceType: rt,
		})
		if err != nil {
			return nil, fmt.Errorf("list repositories (type=%s): %w", rt, err)
		}
		allRepos = append(allRepos, repos...)
	}

	var jobs []*syncjob.SyncJob
	for _, repo := range allRepos {
		jobs = append(jobs, g.buildJobFromRepo(policy, repo))
	}
	return jobs, nil
}

func (g *syncJobGenerator) buildJob(policy *SyncPolicy, resourceType string) *syncjob.SyncJob {
	resourceName := policy.LocalResourceName
	if resourceName == "" {
		resourceName = policy.RemoteResourceName
	}

	remoteResourceName := policy.RemoteResourceName
	if remoteResourceName == "" {
		remoteResourceName = policy.LocalResourceName
	}

	job := &syncjob.SyncJob{
		RemoteRegistryID:   policy.RegistryID,
		RemoteProjectName:  policy.RemoteProjectName,
		RemoteResourceName: remoteResourceName,
		ProjectName:        policy.LocalProjectName,
		ResourceName:       resourceName,
		ResourceType:       resourceType,
		Status:             syncjob.SyncJobStatusRunning,
		CompletePercents:   0,
	}

	if policy.IsPullBase() {
		job.SyncType = "pull"
	} else {
		job.SyncType = "push"
	}

	return job
}

func (g *syncJobGenerator) buildJobFromRepo(policy *SyncPolicy, repo registrydiscovery.RemoteRepository) *syncjob.SyncJob {
	job := &syncjob.SyncJob{
		RemoteRegistryID:   policy.RegistryID,
		RemoteProjectName:  repo.Namespace,
		RemoteResourceName: repo.Name,
		ProjectName:        policy.LocalProjectName,
		ResourceName:       repo.Name,
		ResourceType:       repo.ResourceType,
		Status:             syncjob.SyncJobStatusRunning,
		CompletePercents:   0,
	}

	if policy.IsPullBase() {
		job.SyncType = "pull"
	} else {
		job.SyncType = "push"
	}

	return job
}

func (g *syncJobGenerator) parseResourceTypes(resourceTypes string) []string {
	if resourceTypes == "" {
		return []string{"model"}
	}

	var result []string
	types := strings.Split(resourceTypes, ",")
	for _, t := range types {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "all" {
			return []string{"model", "dataset"}
		}
		if t == "model" || t == "dataset" {
			result = append(result, t)
		}
	}

	if len(result) == 0 {
		return []string{"model"}
	}
	return result
}
