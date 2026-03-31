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

package handler

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type SyncPolicyHandler struct {
	syncPolicyService syncpolicy.ISyncPolicyService
	registryRepo      registry.IRegistryRepo
}

func NewSyncPolicyHandler(syncPolicyService syncpolicy.ISyncPolicyService, registryRepo registry.IRegistryRepo) IHandler {
	return &SyncPolicyHandler{
		syncPolicyService: syncPolicyService,
		registryRepo:      registryRepo,
	}
}

func (h *SyncPolicyHandler) RegisterToServer(options *ServerOptions) {
	// Register GRPC Handler
	v1alpha1.RegisterSyncPolicyServer(options.GRPCServer, h)
	if err := v1alpha1.RegisterSyncPolicyHandlerFromEndpoint(context.Background(), options.GatewayMux, options.GRPCAddr, options.GRPCDialOpt); err != nil {
		log.Errorf("register sync policy handler error: %s", err.Error())
	}
}

// ListSyncPolicies lists all sync policies with pagination and search
func (h *SyncPolicyHandler) ListSyncPolicies(ctx context.Context, request *v1alpha1.ListSyncPoliciesRequest) (*v1alpha1.ListSyncPoliciesResponse, error) {
	// Validate request
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page := int(request.Page)
	pageSize := int(request.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	// Call service
	policies, total, err := h.syncPolicyService.ListSyncPolicies(ctx, page, pageSize, request.Search)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sync policies")
	}

	// Convert to proto
	items := make([]*v1alpha1.SyncPolicyItem, len(policies))
	for i, p := range policies {
		items[i] = h.syncPolicyToProto(ctx, p)
	}

	return &v1alpha1.ListSyncPoliciesResponse{
		SyncPolicies: items,
		Pagination: &v1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
		},
	}, nil
}

// GetSyncPolicy gets a sync policy by ID
func (h *SyncPolicyHandler) GetSyncPolicy(ctx context.Context, request *v1alpha1.GetSyncPolicyRequest) (*v1alpha1.GetSyncPolicyResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	policy, err := h.syncPolicyService.GetSyncPolicy(ctx, int(request.SyncPolicyId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync policy not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync policy")
	}

	return &v1alpha1.GetSyncPolicyResponse{
		SyncPolicy: h.syncPolicyToProto(ctx, policy),
	}, nil
}

// CreateSyncPolicy creates a new sync policy
func (h *SyncPolicyHandler) CreateSyncPolicy(ctx context.Context, request *v1alpha1.CreateSyncPolicyRequest) (*v1alpha1.CreateSyncPolicyResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Only support PullBasePolicy for now
	if request.GetPushBasePolicy() != nil {
		return nil, status.Error(codes.Unimplemented, "push policy not implemented")
	}

	pullPolicy := request.GetPullBasePolicy()
	if pullPolicy == nil {
		return nil, status.Error(codes.InvalidArgument, "pull_base_policy is required")
	}

	// Convert resource types
	resourceTypes := resourceTypesToString(pullPolicy.GetResourceTypes())

	policy := &syncpolicy.SyncPolicy{
		Name:               request.Name,
		Description:        request.Description,
		PolicyType:         int(request.PolicyType),
		TriggerType:        int(request.TriggerType),
		SourceRegistryID:   int(pullPolicy.SourceRegistryId),
		ResourceName:       pullPolicy.ResourceName,
		ResourceTypes:      resourceTypes,
		TargetResourceName: pullPolicy.TargetResourceName,
		TargetProjectName:  pullPolicy.TargetProjectName,
		Bandwidth:          request.Bandwidth,
		IsOverwrite:        request.IsOverwrite,
		IsDisabled:         false,
	}

	if err := h.syncPolicyService.CreateSyncPolicy(ctx, policy); err != nil {
		return nil, status.Error(codes.Internal, "failed to create sync policy")
	}

	return &v1alpha1.CreateSyncPolicyResponse{
		SyncPolicy: h.syncPolicyToProto(ctx, policy),
	}, nil
}

// UpdateSyncPolicy updates a sync policy
func (h *SyncPolicyHandler) UpdateSyncPolicy(ctx context.Context, request *v1alpha1.UpdateSyncPolicyRequest) (*v1alpha1.UpdateSyncPolicyResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get existing policy
	existingPolicy, err := h.syncPolicyService.GetSyncPolicy(ctx, int(request.SyncPolicyId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync policy not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync policy")
	}

	// Update fields
	existingPolicy.Name = request.Name
	existingPolicy.Description = request.Description
	existingPolicy.TriggerType = int(request.TriggerType)
	existingPolicy.Bandwidth = request.Bandwidth
	existingPolicy.IsOverwrite = request.IsOverwrite
	existingPolicy.IsDisabled = request.IsDisabled

	// Update pull policy if provided
	if pullPolicy := request.GetPullBasePolicy(); pullPolicy != nil {
		existingPolicy.SourceRegistryID = int(pullPolicy.SourceRegistryId)
		existingPolicy.ResourceName = pullPolicy.ResourceName
		existingPolicy.ResourceTypes = resourceTypesToString(pullPolicy.GetResourceTypes())
		existingPolicy.TargetResourceName = pullPolicy.TargetResourceName
		existingPolicy.TargetProjectName = pullPolicy.TargetProjectName
	}

	if err := h.syncPolicyService.UpdateSyncPolicy(ctx, existingPolicy); err != nil {
		return nil, status.Error(codes.Internal, "failed to update sync policy")
	}

	return &v1alpha1.UpdateSyncPolicyResponse{
		SyncPolicy: h.syncPolicyToProto(ctx, existingPolicy),
	}, nil
}

// DeleteSyncPolicy deletes a sync policy
func (h *SyncPolicyHandler) DeleteSyncPolicy(ctx context.Context, request *v1alpha1.DeleteSyncPolicyRequest) (*v1alpha1.DeleteSyncPolicyResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get policy first for response
	policy, err := h.syncPolicyService.GetSyncPolicy(ctx, int(request.SyncPolicyId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync policy not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync policy")
	}

	if err := h.syncPolicyService.DeleteSyncPolicy(ctx, int(request.SyncPolicyId)); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete sync policy")
	}

	return &v1alpha1.DeleteSyncPolicyResponse{
		SyncPolicy: h.syncPolicyToProto(ctx, policy),
	}, nil
}

// CreateSyncTask creates a new sync task and executes it asynchronously
func (h *SyncPolicyHandler) CreateSyncTask(ctx context.Context, request *v1alpha1.CreateSyncTaskRequest) (*v1alpha1.CreateSyncTaskResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get the policy
	policy, err := h.syncPolicyService.GetSyncPolicy(ctx, int(request.SyncPolicyId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync policy not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync policy")
	}

	// Check if policy is disabled
	if policy.IsDisabled {
		return nil, status.Error(codes.FailedPrecondition, "sync policy is disabled")
	}

	// Create task synchronously
	task := &syncpolicy.SyncTask{
		SyncPolicyID: int(request.SyncPolicyId),
		TriggerType:  syncpolicy.TriggerTypeManual,
		Status:       syncpolicy.SyncTaskStatusRunning,
	}

	task, err = h.syncPolicyService.CreateSyncTask(ctx, task)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create sync task")
	}

	// Execute asynchronously
	go func() {
		log.Infow("sync task goroutine started", "task_id", task.ID, "policy_id", policy.ID)
		if err := h.syncPolicyService.CreateExcecuteSyncTaskAndSyncJobs(context.Background(), policy); err != nil {
			log.Errorw("failed to execute sync task", "error", err, "task_id", task.ID)
		} else {
			log.Infow("sync task goroutine finished", "task_id", task.ID, "policy_id", policy.ID)
		}
	}()

	return &v1alpha1.CreateSyncTaskResponse{
		Id: int32(task.ID),
	}, nil
}

// ListSyncTasks lists sync tasks for a policy
func (h *SyncPolicyHandler) ListSyncTasks(ctx context.Context, request *v1alpha1.ListSyncTasksRequest) (*v1alpha1.ListSyncTasksResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page := int(request.Page)
	pageSize := int(request.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	tasks, total, err := h.syncPolicyService.ListSyncTasksByPolicyID(ctx, int(request.SyncPolicyId), page, pageSize, request.Search)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sync tasks")
	}

	items := make([]*v1alpha1.SyncTask, len(tasks))
	for i, t := range tasks {
		items[i] = syncTaskToProto(t)
	}

	return &v1alpha1.ListSyncTasksResponse{
		SyncTasks: items,
		Pagination: &v1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
		},
	}, nil
}

// StopSyncTask stops a running sync task
func (h *SyncPolicyHandler) StopSyncTask(ctx context.Context, request *v1alpha1.StopSyncTaskRequest) (*v1alpha1.StopSyncTaskResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Get the task
	task, err := h.syncPolicyService.GetSyncTask(ctx, int(request.SyncTaskId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync task not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync task")
	}

	// Verify the task belongs to the policy
	if task.SyncPolicyID != int(request.SyncPolicyId) {
		return nil, status.Error(codes.InvalidArgument, "sync task does not belong to the specified policy")
	}

	// Update task status to stopped
	task.Status = syncpolicy.SyncTaskStatusStopped
	task.CompletedTimestamp = time.Now().Unix()

	if _, err := h.syncPolicyService.CreateSyncTask(ctx, task); err != nil {
		return nil, status.Error(codes.Internal, "failed to stop sync task")
	}

	return &v1alpha1.StopSyncTaskResponse{
		SyncTask: syncTaskToProto(task),
	}, nil
}

// Helper functions

func (h *SyncPolicyHandler) syncPolicyToProto(ctx context.Context, p *syncpolicy.SyncPolicy) *v1alpha1.SyncPolicyItem {
	if p == nil {
		return nil
	}

	item := &v1alpha1.SyncPolicyItem{
		Id:          int32(p.ID),
		Name:        p.Name,
		Description: p.Description,
		PolicyType:  v1alpha1.SyncPolicyType(p.PolicyType),
		TriggerType: v1alpha1.TriggerType(p.TriggerType),
		Bandwidth:   p.Bandwidth,
		IsOverwrite: p.IsOverwrite,
		IsDisabled:  p.IsDisabled,
	}

	// Add pull policy details if it's a pull type
	if p.PolicyType == syncpolicy.SyncPolicyTypePull {
		resourceTypes := parseResourceTypesString(p.ResourceTypes)
		pullPolicy := &v1alpha1.PullBasePolicy{
			SourceRegistryId:   uint32(p.SourceRegistryID),
			ResourceName:       p.ResourceName,
			ResourceTypes:      resourceTypes,
			TargetResourceName: p.TargetResourceName,
			TargetProjectName:  p.TargetProjectName,
		}

		// Fetch and populate source registry info if registry ID is set
		if p.SourceRegistryID > 0 && h.registryRepo != nil {
			if reg, err := h.registryRepo.GetRegistry(ctx, p.SourceRegistryID); err == nil && reg != nil {
				pullPolicy.SourceRegistry = convertDomainRegistryToAPIRegistry(reg)
			}
		}

		item.Policy = &v1alpha1.SyncPolicyItem_PullBasePolicy{
			PullBasePolicy: pullPolicy,
		}
	}

	return item
}

func syncTaskToProto(t *syncpolicy.SyncTask) *v1alpha1.SyncTask {
	if t == nil {
		return nil
	}

	return &v1alpha1.SyncTask{
		Id:                 int32(t.ID),
		SyncPolicyId:       int32(t.SyncPolicyID),
		TriggerType:        v1alpha1.TriggerType(t.TriggerType),
		Status:             v1alpha1.SyncTaskStatus(t.Status),
		StartedTimestamp:   t.StartedTimestamp,
		CompletedTimestamp: t.CompletedTimestamp,
		TotalItems:         int64(t.TotalItems),
		SuccessfulItems:    int64(t.SuccessfulItems),
	}
}

func resourceTypesToString(types []v1alpha1.ResourceType) string {
	var result []string
	for _, t := range types {
		switch t {
		case v1alpha1.ResourceType_RESOURCE_TYPE_MODEL:
			result = append(result, "model")
		case v1alpha1.ResourceType_RESOURCE_TYPE_DATASET:
			result = append(result, "dataset")
		}
	}
	if len(result) == 0 {
		return "model"
	}
	return strings.Join(result, ",")
}

func parseResourceTypesString(s string) []v1alpha1.ResourceType {
	if s == "" {
		return []v1alpha1.ResourceType{v1alpha1.ResourceType_RESOURCE_TYPE_MODEL}
	}

	parts := strings.Split(s, ",")
	var result []v1alpha1.ResourceType
	for _, p := range parts {
		switch strings.TrimSpace(strings.ToLower(p)) {
		case "model":
			result = append(result, v1alpha1.ResourceType_RESOURCE_TYPE_MODEL)
		case "dataset":
			result = append(result, v1alpha1.ResourceType_RESOURCE_TYPE_DATASET)
		case "all":
			return []v1alpha1.ResourceType{
				v1alpha1.ResourceType_RESOURCE_TYPE_MODEL,
				v1alpha1.ResourceType_RESOURCE_TYPE_DATASET,
			}
		}
	}

	if len(result) == 0 {
		return []v1alpha1.ResourceType{v1alpha1.ResourceType_RESOURCE_TYPE_MODEL}
	}
	return result
}

// Ensure SyncPolicyHandler implements v1alpha1.SyncPolicyServer
var _ v1alpha1.SyncPolicyServer = (*SyncPolicyHandler)(nil)
