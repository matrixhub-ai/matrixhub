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

package sync_policy_test

import (
	"context"
	"fmt"
	"time"

	"github.com/antihax/optional"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/sync_policy"
)

func ptrPolicyType(t v1alpha1.V1alpha1SyncPolicyType) *v1alpha1.V1alpha1SyncPolicyType {
	return &t
}

func ptrTriggerType(t v1alpha1.V1alpha1TriggerType) *v1alpha1.V1alpha1TriggerType {
	return &t
}

func ptrResourceType(t v1alpha1.V1alpha1ResourceType) *v1alpha1.V1alpha1ResourceType {
	return &t
}

// pollUntilTaskStatus polls the sync task status until it reaches the target status or timeout.
func pollUntilTaskStatus(ctx context.Context, api *v1alpha1.SyncPolicyApiService, policyID int32, taskID int32, targetStatus v1alpha1.V1alpha1SyncTaskStatus, timeout time.Duration) (*v1alpha1.V1alpha1SyncTask, error) {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for task %d to reach status %s", taskID, targetStatus)
		}

		resp, _, err := api.SyncPolicyGetSyncTask(ctx, policyID, taskID)
		if err != nil {
			return nil, err
		}
		if resp.SyncTask != nil && resp.SyncTask.Status != nil && *resp.SyncTask.Status == targetStatus {
			return resp.SyncTask, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		time.Sleep(5 * time.Second)
	}
}

// waitForTaskCompletion polls until the sync task reaches a terminal status (Succeeded, Failed, or Stopped).
func waitForTaskCompletion(ctx context.Context, api *v1alpha1.SyncPolicyApiService, policyID int32, taskID int32, timeout time.Duration) (*v1alpha1.V1alpha1SyncTask, error) {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for task %d to complete", taskID)
		}

		resp, _, err := api.SyncPolicyGetSyncTask(ctx, policyID, taskID)
		if err != nil {
			return nil, err
		}
		if resp.SyncTask != nil && resp.SyncTask.Status != nil {
			switch *resp.SyncTask.Status {
			case v1alpha1.SUCCEEDED_V1alpha1SyncTaskStatus,
				v1alpha1.FAILED_V1alpha1SyncTaskStatus,
				v1alpha1.STOPPED_V1alpha1SyncTaskStatus:
				return resp.SyncTask, nil
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		time.Sleep(5 * time.Second)
	}
}

// waitForJobs polls until at least one sync job exists for the given task.
func waitForJobs(ctx context.Context, api *v1alpha1.SyncPolicyApiService, policyID int32, taskID int32, timeout time.Duration) ([]v1alpha1.V1alpha1SyncJob, error) {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for jobs of task %d", taskID)
		}

		resp, _, err := api.SyncPolicyListSyncJobs(ctx, policyID, taskID, &v1alpha1.SyncPolicyApiSyncPolicyListSyncJobsOpts{
			Page:     optional.NewInt32(1),
			PageSize: optional.NewInt32(20),
		})
		if err != nil {
			return nil, err
		}
		if len(resp.SyncJobs) > 0 {
			return resp.SyncJobs, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		time.Sleep(5 * time.Second)
	}
}
