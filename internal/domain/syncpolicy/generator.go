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
	"strings"
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
)

// SyncJobGenerator generates sync tasks and jobs from a sync policy.
// The abstraction decouples job generation logic from the database layer,
// making it testable and extensible for future policy types.
type SyncJobGenerator interface {
	Generate(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error)
}

type syncJobGenerator struct{}

// NewSyncJobGenerator creates a new SyncJobGenerator instance.
func NewSyncJobGenerator() SyncJobGenerator {
	return &syncJobGenerator{}
}

func (g *syncJobGenerator) Generate(ctx context.Context, policy *SyncPolicy) (*SyncTask, []*syncjob.SyncJob, error) {
	resourceTypes := g.parseResourceTypes(policy.ResourceTypes)

	task := &SyncTask{
		SyncPolicyID:       policy.ID,
		TriggerType:        policy.TriggerType,
		Status:             SyncTaskStatusRunning,
		StartedTimestamp:   time.Now().Unix(),
		CompletedTimestamp: 0,
		TotalItems:         len(resourceTypes),
		SuccessfulItems:    0,
		StoppedItems:       0,
		FailedItems:        0,
		CompletePercents:   0,
	}

	var jobs []*syncjob.SyncJob
	for _, resourceType := range resourceTypes {
		job := g.buildJob(policy, resourceType)
		jobs = append(jobs, job)
	}

	return task, jobs, nil
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
