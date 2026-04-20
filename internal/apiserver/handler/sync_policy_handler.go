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
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncjob"
	"github.com/matrixhub-ai/matrixhub/internal/domain/syncpolicy"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

type SyncPolicyHandler struct {
	syncPolicyService syncpolicy.ISyncPolicyService
	syncJobService    syncjob.ISyncJobService
	registryRepo      registry.IRegistryRepo
}

func NewSyncPolicyHandler(syncPolicyService syncpolicy.ISyncPolicyService, syncJobService syncjob.ISyncJobService, registryRepo registry.IRegistryRepo) IHandler {
	return &SyncPolicyHandler{
		syncPolicyService: syncPolicyService,
		syncJobService:    syncJobService,
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

	policy := &syncpolicy.SyncPolicy{}
	h.applyCommonFields(policy, request.Name, request.Description, request.Bandwidth,
		request.TriggerTypeSchedule, syncpolicy.TriggerType(request.TriggerType),
		request.IsOverwrite, false)

	switch request.PolicyType {
	case v1alpha1.SyncPolicyType_SYNC_POLICY_TYPE_PULL_BASE:
		pullPolicy := request.GetPullBasePolicy()
		if pullPolicy == nil {
			return nil, status.Error(codes.InvalidArgument, "pull_base_policy is required")
		}
		h.applyPullBasePolicy(policy, pullPolicy)
	case v1alpha1.SyncPolicyType_SYNC_POLICY_TYPE_PUSH_BASE:
		pushPolicy := request.GetPushBasePolicy()
		if pushPolicy == nil {
			return nil, status.Error(codes.InvalidArgument, "push_base_policy is required")
		}
		h.applyPushBasePolicy(policy, pushPolicy)
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid policy_type")
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

	existingPolicy, err := h.syncPolicyService.GetSyncPolicy(ctx, int(request.SyncPolicyId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync policy not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync policy")
	}

	h.applyCommonFields(existingPolicy, request.Name, request.Description, request.Bandwidth,
		request.TriggerTypeSchedule, syncpolicy.TriggerType(request.TriggerType),
		request.IsOverwrite, request.IsDisabled)

	if pullPolicy := request.GetPullBasePolicy(); pullPolicy != nil {
		h.applyPullBasePolicy(existingPolicy, pullPolicy)
	} else if pushPolicy := request.GetPushBasePolicy(); pushPolicy != nil {
		h.applyPushBasePolicy(existingPolicy, pushPolicy)
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

	// Service creates the task synchronously and executes jobs asynchronously
	task, err := h.syncPolicyService.CreateExcecuteSyncTaskAndSyncJobs(ctx, policy)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create sync task")
	}

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

	tasks, total, err := h.syncPolicyService.ListSyncTasksByPolicyID(ctx, int(request.SyncPolicyId), page, pageSize, syncpolicy.SyncTaskStatus(request.Status))
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

// GetSyncTask gets a sync task by ID
func (h *SyncPolicyHandler) GetSyncTask(ctx context.Context, request *v1alpha1.GetSyncTaskRequest) (*v1alpha1.GetSyncTaskResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	task, err := h.syncPolicyService.GetSyncTask(ctx, int(request.SyncTaskId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync task not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync task")
	}

	if task.SyncPolicyID != int(request.SyncPolicyId) {
		return nil, status.Error(codes.InvalidArgument, "sync task does not belong to the specified policy")
	}

	return &v1alpha1.GetSyncTaskResponse{
		SyncTask: syncTaskToProto(task),
	}, nil
}

// UpdateSyncPolicySwitch toggles a sync policy's enable/disable state
func (h *SyncPolicyHandler) UpdateSyncPolicySwitch(ctx context.Context, request *v1alpha1.UpdateSyncPolicySwitchRequest) (*v1alpha1.UpdateSyncPolicySwitchResponse, error) {
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

	policy.IsDisabled = request.IsDisabled

	if err := h.syncPolicyService.UpdateSyncPolicy(ctx, policy); err != nil {
		return nil, status.Error(codes.Internal, "failed to update sync policy")
	}

	return &v1alpha1.UpdateSyncPolicySwitchResponse{
		SyncPolicy: h.syncPolicyToProto(ctx, policy),
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

	if err := h.syncPolicyService.UpdateSyncTask(ctx, task); err != nil {
		return nil, status.Error(codes.Internal, "failed to stop sync task")
	}

	return &v1alpha1.StopSyncTaskResponse{
		SyncTask: syncTaskToProto(task),
	}, nil
}

// ListSyncJobs lists sync jobs for a task
func (h *SyncPolicyHandler) ListSyncJobs(ctx context.Context, request *v1alpha1.ListSyncJobsRequest) (*v1alpha1.ListSyncJobsResponse, error) {
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

	statusFilter := syncjob.SyncJobStatusUnspecified
	if request.Status != v1alpha1.SyncJobStatus_SYNC_JOB_STATUS_UNSPECIFIED {
		statusFilter = syncjob.SyncJobStatus(request.Status)
	}

	resourceTypeFilter := ""
	if request.ResourceType != v1alpha1.ResourceType_RESOURCE_TYPE_UNSPECIFIED {
		resourceTypeFilter = resourceTypeProtoToString(request.ResourceType)
	}

	jobs, total, err := h.syncJobService.ListSyncJobsByTaskID(ctx, int(request.SyncTaskId), page, pageSize, statusFilter, resourceTypeFilter)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sync jobs")
	}

	items := make([]*v1alpha1.SyncJob, len(jobs))
	for i, j := range jobs {
		items[i] = syncJobToProto(j)
	}

	return &v1alpha1.ListSyncJobsResponse{
		SyncJobs: items,
		Pagination: &v1alpha1.Pagination{
			Total:    int32(total),
			Page:     request.Page,
			PageSize: request.PageSize,
		},
	}, nil
}

// GetSyncJobLog gets the log for a sync job
func (h *SyncPolicyHandler) GetSyncJobLog(ctx context.Context, request *v1alpha1.GetSyncJobLogRequest) (*v1alpha1.GetSyncJobLogResponse, error) {
	if err := request.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	_, err := h.syncJobService.GetSyncJob(ctx, int(request.SyncJobId))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, "sync job not found")
		}
		return nil, status.Error(codes.Internal, "failed to get sync job")
	}

	return &v1alpha1.GetSyncJobLogResponse{
		Log: "",
	}, nil
}

// applyCommonFields fills the common scalar fields of a SyncPolicy.
func (h *SyncPolicyHandler) applyCommonFields(p *syncpolicy.SyncPolicy, name, desc, bandwidth string, triggerSchedule *v1alpha1.TriggerTypeSchedule, triggerType syncpolicy.TriggerType, isOverwrite, isDisabled bool) {
	p.Name = name
	p.Description = desc
	p.TriggerType = triggerType
	p.Bandwidth = bandwidth
	p.TriggerSchedule = ""
	if triggerSchedule != nil {
		p.TriggerSchedule = triggerSchedule.Cron
	}
	p.IsOverwrite = isOverwrite
	p.IsDisabled = isDisabled
}

// applyPullBasePolicy fills pull-base-specific fields into a SyncPolicy.
func (h *SyncPolicyHandler) applyPullBasePolicy(p *syncpolicy.SyncPolicy, pull *v1alpha1.PullBasePolicy) {
	p.PolicyType = syncpolicy.SyncPolicyTypePull
	p.RegistryID = int(pull.SourceRegistryId)
	p.ResourceTypes = resourceTypesToString(pull.GetResourceTypes())
	remoteProject, remoteName := parseResourcePath(pull.ResourceName)
	p.RemoteProjectName = remoteProject
	p.RemoteResourceName = remoteName
	p.LocalProjectName = pull.TargetProjectName
	// LocalResourceName is intentionally left empty; generator auto-fills from remote
}

// applyPushBasePolicy fills push-base-specific fields into a SyncPolicy.
func (h *SyncPolicyHandler) applyPushBasePolicy(p *syncpolicy.SyncPolicy, push *v1alpha1.PushBasePolicy) {
	p.PolicyType = syncpolicy.SyncPolicyTypePush
	p.RegistryID = int(push.TargetRegistryId)
	p.ResourceTypes = resourceTypesToString(push.GetResourceTypes())
	localProject, localName := parseResourcePath(push.ResourceName)
	p.LocalProjectName = localProject
	p.LocalResourceName = localName
	p.RemoteProjectName = push.TargetProjectName
	// RemoteResourceName is intentionally left empty; generator auto-fills from local
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
	if p.TriggerSchedule != "" {
		item.TriggerTypeSchedule = &v1alpha1.TriggerTypeSchedule{
			Cron: p.TriggerSchedule,
		}
	}

	resourceTypes := parseResourceTypesString(p.ResourceTypes)

	switch p.PolicyType {
	case syncpolicy.SyncPolicyTypePull:
		pullPolicy := &v1alpha1.PullBasePolicy{
			SourceRegistryId:  uint32(p.RegistryID),
			ResourceName:      buildResourcePath(p.RemoteProjectName, p.RemoteResourceName),
			ResourceTypes:     resourceTypes,
			TargetProjectName: p.LocalProjectName,
		}
		if p.RegistryID > 0 && h.registryRepo != nil {
			if reg, err := h.registryRepo.GetRegistry(ctx, p.RegistryID); err == nil && reg != nil {
				pullPolicy.SourceRegistry = convertDomainRegistryToAPIRegistry(reg)
			}
		}
		item.Policy = &v1alpha1.SyncPolicyItem_PullBasePolicy{
			PullBasePolicy: pullPolicy,
		}

	case syncpolicy.SyncPolicyTypePush:
		pushPolicy := &v1alpha1.PushBasePolicy{
			ResourceName:      buildResourcePath(p.LocalProjectName, p.LocalResourceName),
			ResourceTypes:     resourceTypes,
			TargetRegistryId:  uint32(p.RegistryID),
			TargetProjectName: p.RemoteProjectName,
		}
		if p.RegistryID > 0 && h.registryRepo != nil {
			if reg, err := h.registryRepo.GetRegistry(ctx, p.RegistryID); err == nil && reg != nil {
				pushPolicy.TargetRegistry = convertDomainRegistryToAPIRegistry(reg)
			}
		}
		item.Policy = &v1alpha1.SyncPolicyItem_PushBasePolicy{
			PushBasePolicy: pushPolicy,
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
		StoppedItems:       int64(t.StoppedItems),
		FailedItems:        int64(t.FailedItems),
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

// parseResourcePath splits "project/name" into project and name.
func parseResourcePath(fullPath string) (project, name string) {
	parts := strings.SplitN(fullPath, "/", 2)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", fullPath
}

// buildResourcePath joins project and name into "project/name".
func buildResourcePath(project, name string) string {
	if project == "" {
		return name
	}
	return project + "/" + name
}

func syncJobToProto(j *syncjob.SyncJob) *v1alpha1.SyncJob {
	if j == nil {
		return nil
	}

	action := "push"
	if j.SyncType == "pull" {
		action = "clone"
	}

	return &v1alpha1.SyncJob{
		Id:                 int32(j.ID),
		SyncTaskId:         int32(j.SyncTaskID),
		ResourceType:       stringToResourceType(j.ResourceType),
		ResourceName:       j.ResourceName,
		TargetResourceName: j.RemoteResourceName,
		Action:             action,
		Status:             v1alpha1.SyncJobStatus(j.Status),
		CompletedTimestamp: j.CompletedTimestamp,
	}
}

func resourceTypeProtoToString(rt v1alpha1.ResourceType) string {
	switch rt {
	case v1alpha1.ResourceType_RESOURCE_TYPE_MODEL:
		return "model"
	case v1alpha1.ResourceType_RESOURCE_TYPE_DATASET:
		return "dataset"
	default:
		return ""
	}
}

func stringToResourceType(s string) v1alpha1.ResourceType {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "model":
		return v1alpha1.ResourceType_RESOURCE_TYPE_MODEL
	case "dataset":
		return v1alpha1.ResourceType_RESOURCE_TYPE_DATASET
	default:
		return v1alpha1.ResourceType_RESOURCE_TYPE_UNSPECIFIED
	}
}

// Ensure SyncPolicyHandler implements v1alpha1.SyncPolicyServer
var _ v1alpha1.SyncPolicyServer = (*SyncPolicyHandler)(nil)
