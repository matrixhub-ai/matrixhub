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
//
//go:generate go tool mockgen -source=generator.go -destination=mocks/generator_mock.go -package=mocks
type SyncJobGenerator interface {
	Generate(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error)
}

// LocalResourceRepository lists resources already stored in MatrixHub.
type LocalResourceRepository interface {
	ListAllPaths(ctx context.Context) ([]string, error)
}

// LocalResourceSource associates a local resource repository with its type.
type LocalResourceSource struct {
	ResourceType string
	Repo         LocalResourceRepository
}

type syncJobGenerator struct {
	registryRepo  registry.IRegistryRepo
	discoveries   map[string]registrydiscovery.Discovery
	localResource map[string]LocalResourceRepository
}

// NewSyncJobGenerator creates a new SyncJobGenerator instance.
func NewSyncJobGenerator(registryRepo registry.IRegistryRepo, discoveries map[string]registrydiscovery.Discovery, localSources ...LocalResourceSource) SyncJobGenerator {
	localResource := make(map[string]LocalResourceRepository, len(localSources))
	for _, source := range localSources {
		if source.Repo != nil {
			localResource[source.ResourceType] = source.Repo
		}
	}

	return &syncJobGenerator{
		registryRepo:  registryRepo,
		discoveries:   discoveries,
		localResource: localResource,
	}
}

func (g *syncJobGenerator) Generate(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error) {
	task := g.buildTask(policy)

	var jobs []*syncjob.SyncJob
	var err error

	switch {
	case policy.IsPullBase() && policy.HasWildcardResourceName():
		jobs, err = g.buildJobsFromDiscovery(ctx, policy)
	case policy.IsPushBase() && policy.HasWildcardResourceName():
		jobs, err = g.buildJobsFromLocalDiscovery(ctx, policy)
	default:
		jobs = g.buildJobsFromStatic(policy)
	}
	if err != nil {
		return nil, nil, err
	}

	task.TotalItems = len(jobs)
	return task, jobs, nil
}

func (g *syncJobGenerator) buildJobsFromLocalDiscovery(ctx context.Context, policy *SyncPolicy) ([]*syncjob.SyncJob, error) {
	if len(g.localResource) == 0 {
		return nil, fmt.Errorf("local resource discovery is not configured")
	}

	resourceTypes := g.parseResourceTypes(policy.ResourceTypes)
	var jobs []*syncjob.SyncJob
	for _, rt := range resourceTypes {
		repo, ok := g.localResource[rt]
		if !ok {
			return nil, fmt.Errorf("no local resource repository for type: %s", rt)
		}

		paths, err := repo.ListAllPaths(ctx)
		if err != nil {
			return nil, fmt.Errorf("list local resources (type=%s): %w", rt, err)
		}
		for _, path := range paths {
			project, name := splitResourcePath(path)
			if !matchesWildcardResource(project, name, policy.LocalProjectName, policy.LocalResourceName) {
				continue
			}
			jobs = append(jobs, g.buildPushJobFromPath(policy, project, name, rt))
		}
	}
	return jobs, nil
}

func (g *syncJobGenerator) buildPushJobFromPath(policy *SyncPolicy, project, name, resourceType string) *syncjob.SyncJob {
	return &syncjob.SyncJob{
		RemoteRegistryID:   policy.RegistryID,
		RemoteProjectName:  policy.RemoteProjectName,
		RemoteResourceName: name,
		ProjectName:        project,
		ResourceName:       name,
		ResourceType:       resourceType,
		SyncType:           "push",
		Status:             syncjob.SyncJobStatusRunning,
		CompletePercents:   0,
	}
}

func splitResourcePath(path string) (project, name string) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", path
	}
	return parts[0], parts[1]
}

func matchesWildcardResource(project, name, projectPattern, namePattern string) bool {
	if projectPattern != "" && project != projectPattern {
		return false
	}
	return namePattern == "*" || namePattern == "**"
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
