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
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	modelv1alpha1 "github.com/matrixhub-ai/matrixhub/api/go/v1alpha1"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/modelcache"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

// fakeAuthz grants the permissions in allowed and records the checked ones.
type fakeAuthz struct {
	authz.IAuthzService
	allowed map[role.Permission]bool
	checked []role.Permission
}

func (a *fakeAuthz) VerifyProjectPermissionByName(_ context.Context, _ string, perm role.Permission) (bool, error) {
	a.checked = append(a.checked, perm)
	return a.allowed[perm], nil
}

type fakeModelCache struct {
	modelcache.IModelCacheService
	status *modelcache.CacheStatus
	err    error
	calls  int
	req    *modelcache.Request
}

func (c *fakeModelCache) CacheModel(_ context.Context, req *modelcache.Request) (*modelcache.CacheStatus, error) {
	c.calls++
	c.req = req
	return c.status, c.err
}

func (c *fakeModelCache) GetModelCache(context.Context, string, string, string) (*modelcache.CacheStatus, error) {
	c.calls++
	return c.status, c.err
}

func newCacheHandler(allowed ...role.Permission) (*ModelHandler, *fakeAuthz, *fakeModelCache) {
	az := &fakeAuthz{allowed: map[role.Permission]bool{}}
	for _, perm := range allowed {
		az.allowed[perm] = true
	}
	cache := &fakeModelCache{status: &modelcache.CacheStatus{Project: "proj", Name: "model", State: modelcache.CacheStateMissing}}
	return &ModelHandler{cache: cache, authzService: az}, az, cache
}

func requireCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if status.Code(err) != want {
		t.Fatalf("status code = %s, want %s (err: %v)", status.Code(err), want, err)
	}
}

func TestCacheModelNeedsPushPermission(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			h, az, cache := newCacheHandler(role.ModelGet, role.ModelPull)

			_, err := h.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{Project: "proj", Name: "model", DryRun: dryRun})

			requireCode(t, err, codes.PermissionDenied)
			if len(az.checked) != 1 || az.checked[0] != role.ModelPush {
				t.Fatalf("checked permissions = %v, want [%s]", az.checked, role.ModelPush)
			}
			if cache.calls != 0 {
				t.Fatalf("cache called %d times without permission", cache.calls)
			}
		})
	}
}

func TestCacheModelPassesTheRequest(t *testing.T) {
	h, _, cache := newCacheHandler(role.ModelPush)

	_, err := h.CacheModel(t.Context(), &modelv1alpha1.CacheModelRequest{
		Project:       "proj",
		Name:          "model",
		Revision:      "main",
		AllowPatterns: []string{"*.safetensors"},
		DryRun:        true,
	})

	if err != nil {
		t.Fatalf("CacheModel() error = %v", err)
	}
	if cache.req.Revision != "main" || !cache.req.DryRun || len(cache.req.Selection.AllowPatterns) != 1 {
		t.Fatalf("request = %+v", cache.req)
	}
}

func TestGetModelCacheNeedsGetPermission(t *testing.T) {
	h, az, cache := newCacheHandler()

	_, err := h.GetModelCache(t.Context(), &modelv1alpha1.GetModelCacheRequest{Project: "proj", Name: "model"})

	requireCode(t, err, codes.PermissionDenied)
	if len(az.checked) != 1 || az.checked[0] != role.ModelGet {
		t.Fatalf("checked permissions = %v, want [%s]", az.checked, role.ModelGet)
	}
	if cache.calls != 0 {
		t.Fatalf("cache called %d times without permission", cache.calls)
	}

	h, _, _ = newCacheHandler(role.ModelGet)
	if _, err := h.GetModelCache(t.Context(), &modelv1alpha1.GetModelCacheRequest{Project: "proj", Name: "model"}); err != nil {
		t.Fatalf("GetModelCache() error = %v", err)
	}
}

func TestCacheModelBoundsPatterns(t *testing.T) {
	tests := map[string]*modelv1alpha1.CacheModelRequest{
		"too many patterns": {Project: "proj", Name: "model", IgnorePatterns: slices.Repeat([]string{"*.bin"}, 17)},
		"pattern too long":  {Project: "proj", Name: "model", AllowPatterns: []string{strings.Repeat("a", 257)}},
		"invalid name":      {Project: "proj", Name: "../model"},
	}
	for name, req := range tests {
		t.Run(name, func(t *testing.T) {
			h, _, cache := newCacheHandler(role.ModelPush)

			_, err := h.CacheModel(t.Context(), req)

			requireCode(t, err, codes.InvalidArgument)
			if cache.calls != 0 {
				t.Fatalf("cache called for an invalid request")
			}
		})
	}
}

func TestModelCacheErrorToStatus(t *testing.T) {
	// Errors from local storage and syncs can name server paths.
	storageErr := errors.New("repository does not exist at /data/repositories/proj/model.git")
	tests := []struct {
		err  error
		want codes.Code
	}{
		{fmt.Errorf("%w: bad pattern", modelcache.ErrInvalidSelection), codes.InvalidArgument},
		{fmt.Errorf("%w: refs/pr/1", modelcache.ErrUnsupportedRevision), codes.InvalidArgument},
		{fmt.Errorf("%w: proj/model", modelcache.ErrModelNotFound), codes.NotFound},
		{fmt.Errorf("%w: %w", modelcache.ErrRevisionNotFound, storageErr), codes.NotFound},
		{fmt.Errorf("%w: %w", modelcache.ErrUpstreamSync, storageErr), codes.FailedPrecondition},
		{fmt.Errorf("%w: c9 resolves to c9 upstream", modelcache.ErrRevisionNotMirrored), codes.FailedPrecondition},
		{fmt.Errorf("%w: proj", modelcache.ErrNotProxyProject), codes.FailedPrecondition},
		{fmt.Errorf("%w: gated", modelcache.ErrUpstreamAccessDenied), codes.FailedPrecondition},
		{fmt.Errorf("%w: rate limited", modelcache.ErrUpstreamUnavailable), codes.Unavailable},
		{modelcache.ErrClosed, codes.Unavailable},
		{fmt.Errorf("%w: 8 running", modelcache.ErrTooManyTasks), codes.ResourceExhausted},
		{fmt.Errorf("list files: %w", context.Canceled), codes.Canceled},
		{fmt.Errorf("list files: %w", context.DeadlineExceeded), codes.DeadlineExceeded},
		{storageErr, codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			err := modelCacheErrorToStatus(tt.err)

			requireCode(t, err, tt.want)
			if msg := status.Convert(err).Message(); strings.Contains(msg, "/data/") {
				t.Fatalf("message %q leaks a server path", msg)
			}
		})
	}
}

func TestModelCacheStatusToProto(t *testing.T) {
	started := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	st := modelCacheStatusToProto(&modelcache.CacheStatus{
		Project: "proj",
		Name:    "model",
		State:   modelcache.CacheStatePartial,
		Task:    &modelcache.TaskStatus{ID: "run-1", State: modelcache.TaskRunning, StartedAt: started},
	})

	if st.State != modelv1alpha1.CacheState_CACHE_STATE_PARTIAL {
		t.Fatalf("state = %s", st.State)
	}
	if st.Task.GetId() != "run-1" || st.Task.GetState() != modelv1alpha1.CacheTaskState_CACHE_TASK_STATE_RUNNING ||
		st.Task.GetStartedAt() != "2026-10-15T08:00:00Z" || st.Task.GetFinishedAt() != "" {
		t.Fatalf("task = %+v", st.Task)
	}

	if st := modelCacheStatusToProto(&modelcache.CacheStatus{State: modelcache.CacheStateCached}); st.Task != nil {
		t.Fatalf("task = %+v, want unset without a run", st.Task)
	}
}
