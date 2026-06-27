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

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

type fakeModelRepo struct {
	modelPaths []string
}

type fakeDatasetRepo struct {
	datasetPaths []string
}

func (f *fakeModelRepo) Create(context.Context, *model.Model) (*model.Model, error) { return nil, nil }

func (f *fakeModelRepo) GetByProjectAndName(context.Context, string, string) (*model.Model, error) {
	return nil, nil
}

func (f *fakeModelRepo) List(context.Context, *model.Filter) ([]*model.Model, int64, error) {
	return nil, 0, nil
}

func (f *fakeModelRepo) Delete(context.Context, string, string) error { return nil }

func (f *fakeModelRepo) UpdateMetadata(context.Context, int64, *model.MetadataUpdate) error {
	return nil
}

func (f *fakeModelRepo) UpdateSetting(context.Context, int64, *model.SettingUpdate) error {
	return nil
}

func (f *fakeModelRepo) ListAllPaths(context.Context) ([]string, error) {
	return f.modelPaths, nil
}

func (f *fakeDatasetRepo) Create(context.Context, *dataset.Dataset) (*dataset.Dataset, error) {
	return nil, nil
}

func (f *fakeDatasetRepo) GetByProjectAndName(context.Context, string, string) (*dataset.Dataset, error) {
	return nil, nil
}

func (f *fakeDatasetRepo) List(context.Context, *model.Filter) ([]*dataset.Dataset, int64, error) {
	return nil, 0, nil
}

func (f *fakeDatasetRepo) Delete(context.Context, string, string) error { return nil }

func (f *fakeDatasetRepo) ListAllPaths(context.Context) ([]string, error) {
	return f.datasetPaths, nil
}

type fakeGitRepo struct {
	gotModelPaths   []string
	gotDatasetPaths []string
	orphanedRepos   []*git.OrphanedRepo
	orphanedLFS     []*git.OrphanedLFS
	deletedRepos    []string
	deletedLFS      []string
	reposSize       int64
	lfsSize         int64
}

func (f *fakeGitRepo) CreateRepository(context.Context, string, string, string) error { return nil }

func (f *fakeGitRepo) RepositoryExists(context.Context, string, string, string) (bool, error) {
	return false, nil
}

func (f *fakeGitRepo) DeleteRepository(context.Context, string, string, string) error { return nil }

func (f *fakeGitRepo) ListRevisions(context.Context, string, string, string) (*git.Revisions, error) {
	return nil, nil
}

func (f *fakeGitRepo) ListCommits(context.Context, string, string, string, string, int, int) ([]*git.Commit, int64, error) {
	return nil, 0, nil
}

func (f *fakeGitRepo) GetCommit(context.Context, string, string, string, string) (*git.Commit, error) {
	return nil, nil
}

func (f *fakeGitRepo) CreateCommit(context.Context, string, string, string, string, *git.Commit, []git.CommitOperation) (string, error) {
	return "", nil
}

func (f *fakeGitRepo) GetTree(context.Context, string, string, string, string, string) ([]*git.TreeEntry, error) {
	return nil, nil
}

func (f *fakeGitRepo) GetBlob(context.Context, string, string, string, string, string) (*git.TreeEntry, error) {
	return nil, nil
}

func (f *fakeGitRepo) PullFromRemote(context.Context, *git.GitRepository) error { return nil }

func (f *fakeGitRepo) PushToRemote(context.Context, *git.GitRepository) error { return nil }

func (f *fakeGitRepo) ExtractMetadata(context.Context, string, string, string) (*git.RepoMetadataFiles, error) {
	return nil, nil
}

func (f *fakeGitRepo) FindOrphanedRepos(_ context.Context, validModelPaths, validDatasetPaths []string) ([]*git.OrphanedRepo, error) {
	f.gotModelPaths = validModelPaths
	f.gotDatasetPaths = validDatasetPaths
	return f.orphanedRepos, nil
}

func (f *fakeGitRepo) FindOrphanedLFS(context.Context) ([]*git.OrphanedLFS, error) {
	return f.orphanedLFS, nil
}

func (f *fakeGitRepo) DeleteRepositoryAtRelPath(_ context.Context, path string) error {
	f.deletedRepos = append(f.deletedRepos, path)
	return nil
}

func (f *fakeGitRepo) DeleteLFSObject(_ context.Context, object *git.OrphanedLFS) error {
	f.deletedLFS = append(f.deletedLFS, object.OID)
	return nil
}

func (f *fakeGitRepo) RepositoriesSize(context.Context) int64 {
	return f.reposSize
}

func (f *fakeGitRepo) LFSSize(context.Context) int64 {
	return f.lfsSize
}

func TestCleanupServicePreviewUsesDomainPorts(t *testing.T) {
	modelRepo := &fakeModelRepo{
		modelPaths: []string{"project/model"},
	}
	datasetRepo := &fakeDatasetRepo{
		datasetPaths: []string{"project/dataset"},
	}
	gitRepo := &fakeGitRepo{
		orphanedRepos: []*git.OrphanedRepo{{Path: "orphan/model.git", SizeBytes: 10}},
		orphanedLFS:   []*git.OrphanedLFS{{OID: "12345678", SizeBytes: 20}},
	}
	service := NewCleanupService(modelRepo, datasetRepo, gitRepo)

	preview, err := service.PreviewCleanup(context.Background(), true, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if preview.TotalReclaimable != 30 {
		t.Fatalf("expected total reclaimable 30, got %d", preview.TotalReclaimable)
	}
	if !reflect.DeepEqual(gitRepo.gotModelPaths, modelRepo.modelPaths) {
		t.Fatalf("expected model paths %v, got %v", modelRepo.modelPaths, gitRepo.gotModelPaths)
	}
	if !reflect.DeepEqual(gitRepo.gotDatasetPaths, datasetRepo.datasetPaths) {
		t.Fatalf("expected dataset paths %v, got %v", datasetRepo.datasetPaths, gitRepo.gotDatasetPaths)
	}
}

func TestCleanupServiceExecuteDelegatesDeletionToStoragePort(t *testing.T) {
	gitRepo := &fakeGitRepo{
		orphanedRepos: []*git.OrphanedRepo{{Path: "orphan/model.git", SizeBytes: 10}},
		orphanedLFS:   []*git.OrphanedLFS{{OID: "12345678", SizeBytes: 20}},
	}
	service := NewCleanupService(&fakeModelRepo{}, &fakeDatasetRepo{}, gitRepo)

	result, err := service.ExecuteCleanup(context.Background(), true, true, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ReposDeleted != 1 || result.LFSObjectsDeleted != 1 || result.SpaceReclaimed != 30 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !reflect.DeepEqual(gitRepo.deletedRepos, []string{"orphan/model.git"}) {
		t.Fatalf("expected repo deletion through git port, got %v", gitRepo.deletedRepos)
	}
	if !reflect.DeepEqual(gitRepo.deletedLFS, []string{"12345678"}) {
		t.Fatalf("expected lfs deletion through git port, got %v", gitRepo.deletedLFS)
	}
}
