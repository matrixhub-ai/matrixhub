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

package cleanup

import (
	"context"
	"reflect"
	"testing"
)

type fakeCleanupRepo struct {
	modelPaths   []string
	datasetPaths []string
}

func (f *fakeCleanupRepo) ListAllModelPaths(context.Context) ([]string, error) {
	return f.modelPaths, nil
}

func (f *fakeCleanupRepo) ListAllDatasetPaths(context.Context) ([]string, error) {
	return f.datasetPaths, nil
}

type fakeCleanupStorageRepo struct {
	gotModelPaths   []string
	gotDatasetPaths []string
	orphanedRepos   []*OrphanedRepo
	orphanedLFS     []*OrphanedLFS
	deletedRepos    []string
	deletedLFS      []string
	reposSize       int64
	lfsSize         int64
}

func (f *fakeCleanupStorageRepo) FindOrphanedRepos(_ context.Context, validModelPaths, validDatasetPaths []string) ([]*OrphanedRepo, error) {
	f.gotModelPaths = validModelPaths
	f.gotDatasetPaths = validDatasetPaths
	return f.orphanedRepos, nil
}

func (f *fakeCleanupStorageRepo) FindOrphanedLFS(context.Context) ([]*OrphanedLFS, error) {
	return f.orphanedLFS, nil
}

func (f *fakeCleanupStorageRepo) DeleteRepo(_ context.Context, path string) error {
	f.deletedRepos = append(f.deletedRepos, path)
	return nil
}

func (f *fakeCleanupStorageRepo) DeleteLFSObject(_ context.Context, object *OrphanedLFS) error {
	f.deletedLFS = append(f.deletedLFS, object.OID)
	return nil
}

func (f *fakeCleanupStorageRepo) RepositoriesSize(context.Context) int64 {
	return f.reposSize
}

func (f *fakeCleanupStorageRepo) LFSSize(context.Context) int64 {
	return f.lfsSize
}

func TestCleanupServicePreviewUsesDomainPorts(t *testing.T) {
	dbRepo := &fakeCleanupRepo{
		modelPaths:   []string{"project/model"},
		datasetPaths: []string{"project/dataset"},
	}
	storageRepo := &fakeCleanupStorageRepo{
		orphanedRepos: []*OrphanedRepo{{Path: "orphan/model.git", SizeBytes: 10}},
		orphanedLFS:   []*OrphanedLFS{{OID: "12345678", SizeBytes: 20}},
	}
	service := NewCleanupService(dbRepo, storageRepo)

	preview, err := service.PreviewCleanup(context.Background(), true, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if preview.TotalReclaimable != 30 {
		t.Fatalf("expected total reclaimable 30, got %d", preview.TotalReclaimable)
	}
	if !reflect.DeepEqual(storageRepo.gotModelPaths, dbRepo.modelPaths) {
		t.Fatalf("expected model paths %v, got %v", dbRepo.modelPaths, storageRepo.gotModelPaths)
	}
	if !reflect.DeepEqual(storageRepo.gotDatasetPaths, dbRepo.datasetPaths) {
		t.Fatalf("expected dataset paths %v, got %v", dbRepo.datasetPaths, storageRepo.gotDatasetPaths)
	}
}

func TestCleanupServiceExecuteDelegatesDeletionToStoragePort(t *testing.T) {
	storageRepo := &fakeCleanupStorageRepo{
		orphanedRepos: []*OrphanedRepo{{Path: "orphan/model.git", SizeBytes: 10}},
		orphanedLFS:   []*OrphanedLFS{{OID: "12345678", SizeBytes: 20}},
	}
	service := NewCleanupService(&fakeCleanupRepo{}, storageRepo)

	result, err := service.ExecuteCleanup(context.Background(), true, true, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ReposDeleted != 1 || result.LFSObjectsDeleted != 1 || result.SpaceReclaimed != 30 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !reflect.DeepEqual(storageRepo.deletedRepos, []string{"orphan/model.git"}) {
		t.Fatalf("expected repo deletion through storage port, got %v", storageRepo.deletedRepos)
	}
	if !reflect.DeepEqual(storageRepo.deletedLFS, []string{"12345678"}) {
		t.Fatalf("expected lfs deletion through storage port, got %v", storageRepo.deletedLFS)
	}
}
