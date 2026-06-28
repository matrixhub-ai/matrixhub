package apiserver

import (
	"context"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	domainmodel "github.com/matrixhub-ai/matrixhub/internal/domain/model"
)

type fakeSSHModelService struct {
	ensureCalls []string
	syncCalls   []string
}

func (f *fakeSSHModelService) CreateModel(context.Context, string, string) (*domainmodel.Model, error) {
	return nil, nil
}
func (f *fakeSSHModelService) GetModel(context.Context, string, string) (*domainmodel.Model, error) {
	return nil, nil
}
func (f *fakeSSHModelService) ListModels(context.Context, *domainmodel.Filter) ([]*domainmodel.Model, int64, error) {
	return nil, 0, nil
}
func (f *fakeSSHModelService) DeleteModel(context.Context, string, string) error { return nil }
func (f *fakeSSHModelService) EnsureModel(ctx context.Context, project, name string) (*domainmodel.Model, error) {
	f.ensureCalls = append(f.ensureCalls, project+"/"+name)
	return &domainmodel.Model{Name: name, ProjectName: project}, nil
}
func (f *fakeSSHModelService) ListModelTaskLabels(context.Context) ([]*domainmodel.Label, error) {
	return nil, nil
}
func (f *fakeSSHModelService) ListModelFrameLabels(context.Context) ([]*domainmodel.Label, error) {
	return nil, nil
}
func (f *fakeSSHModelService) ListModelRevisions(context.Context, string, string) (*git.Revisions, error) {
	return nil, nil
}
func (f *fakeSSHModelService) ListModelCommits(context.Context, string, string, string, int, int) ([]*git.Commit, int64, error) {
	return nil, 0, nil
}
func (f *fakeSSHModelService) GetModelCommit(context.Context, string, string, string) (*git.Commit, error) {
	return nil, nil
}
func (f *fakeSSHModelService) CreateModelCommit(context.Context, string, string, string, *git.Commit, []git.CommitOperation) (string, error) {
	return "", nil
}
func (f *fakeSSHModelService) GetModelTree(context.Context, string, string, string, string) ([]*git.TreeEntry, error) {
	return nil, nil
}
func (f *fakeSSHModelService) GetModelBlob(context.Context, string, string, string, string) (*git.TreeEntry, error) {
	return nil, nil
}
func (f *fakeSSHModelService) UpdateModelSetting(context.Context, string, string, *domainmodel.SettingUpdate) error {
	return nil
}
func (f *fakeSSHModelService) SyncMetadata(context.Context, string, string) error { return nil }
func (f *fakeSSHModelService) CheckOrSyncFromRemote(ctx context.Context, project, name string) error {
	f.syncCalls = append(f.syncCalls, project+"/"+name)
	return nil
}

func TestSSHPreOpenHookSyncsOnRead(t *testing.T) {
	service := &fakeSSHModelService{}
	hook := newSSHPreOpenHook(service)

	if err := hook(context.Background(), "proj/model.git", false); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}
	if got := len(service.syncCalls); got != 1 {
		t.Fatalf("expected one sync call, got %d", got)
	}
	if service.syncCalls[0] != "proj/model" {
		t.Fatalf("expected sync for proj/model, got %s", service.syncCalls[0])
	}
	if got := len(service.ensureCalls); got != 0 {
		t.Fatalf("expected no ensure calls, got %d", got)
	}
}

func TestSSHPreOpenHookEnsuresOnWrite(t *testing.T) {
	service := &fakeSSHModelService{}
	hook := newSSHPreOpenHook(service)

	if err := hook(context.Background(), "proj/model.git", true); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}
	if got := len(service.ensureCalls); got != 1 {
		t.Fatalf("expected one ensure call, got %d", got)
	}
	if service.ensureCalls[0] != "proj/model" {
		t.Fatalf("expected ensure for proj/model, got %s", service.ensureCalls[0])
	}
	if got := len(service.syncCalls); got != 0 {
		t.Fatalf("expected no sync calls, got %d", got)
	}
}

func TestSSHPreOpenHookIgnoresNonModels(t *testing.T) {
	service := &fakeSSHModelService{}
	hook := newSSHPreOpenHook(service)

	if err := hook(context.Background(), "datasets/proj/dataset.git", true); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}
	if len(service.ensureCalls) != 0 || len(service.syncCalls) != 0 {
		t.Fatalf("expected no model service calls, got ensure=%v sync=%v", service.ensureCalls, service.syncCalls)
	}
}
