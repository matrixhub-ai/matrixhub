package model

import (
	"context"
	"errors"
	"testing"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/role"
)

type modelRepoFake struct {
	models  map[string]*Model
	created []*Model
	getErr  error
}

func (f *modelRepoFake) Create(ctx context.Context, m *Model) (*Model, error) {
	if f.models == nil {
		f.models = make(map[string]*Model)
	}
	cp := *m
	f.created = append(f.created, &cp)
	f.models[m.ProjectName+"/"+m.Name] = &cp
	return &cp, nil
}

func (f *modelRepoFake) GetByProjectAndName(ctx context.Context, project, name string) (*Model, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if m := f.models[project+"/"+name]; m != nil {
		return m, nil
	}
	return nil, errors.New("model not found")
}

func (f *modelRepoFake) List(context.Context, *Filter) ([]*Model, int64, error) { return nil, 0, nil }
func (f *modelRepoFake) Delete(context.Context, string, string) error           { return nil }
func (f *modelRepoFake) UpdateMetadata(context.Context, int64, *MetadataUpdate) error {
	return nil
}
func (f *modelRepoFake) UpdateSetting(context.Context, int64, *SettingUpdate) error { return nil }
func (f *modelRepoFake) ListAllPaths(context.Context) ([]string, error)             { return nil, nil }

type gitRepoFake struct {
	repoExists        bool
	repositoryChecked bool
	createRepository  bool
}

func (f *gitRepoFake) CreateRepository(context.Context, string, string, string) error {
	f.createRepository = true
	f.repoExists = true
	return nil
}

func (f *gitRepoFake) RepositoryExists(context.Context, string, string, string) (bool, error) {
	f.repositoryChecked = true
	return f.repoExists, nil
}

func (f *gitRepoFake) DeleteRepository(context.Context, string, string, string) error { return nil }
func (f *gitRepoFake) ListRevisions(context.Context, string, string, string) (*git.Revisions, error) {
	return nil, nil
}
func (f *gitRepoFake) ListCommits(context.Context, string, string, string, string, int, int) ([]*git.Commit, int64, error) {
	return nil, 0, nil
}
func (f *gitRepoFake) GetCommit(context.Context, string, string, string, string) (*git.Commit, error) {
	return nil, nil
}
func (f *gitRepoFake) CreateCommit(context.Context, string, string, string, string, *git.Commit, []git.CommitOperation) (string, error) {
	return "", nil
}
func (f *gitRepoFake) GetTree(context.Context, string, string, string, string, string) ([]*git.TreeEntry, error) {
	return nil, nil
}
func (f *gitRepoFake) GetBlob(context.Context, string, string, string, string, string) (*git.TreeEntry, error) {
	return nil, nil
}
func (f *gitRepoFake) PullFromRemote(context.Context, *git.GitRepository) error { return nil }
func (f *gitRepoFake) PushToRemote(context.Context, *git.GitRepository) error   { return nil }
func (f *gitRepoFake) ExtractMetadata(context.Context, string, string, string) (*git.RepoMetadataFiles, error) {
	return nil, nil
}
func (f *gitRepoFake) FindOrphanedRepos(context.Context, []string, []string) ([]*git.OrphanedRepo, error) {
	return nil, nil
}
func (f *gitRepoFake) FindOrphanedLFS(context.Context) ([]*git.OrphanedLFS, error) {
	return nil, nil
}
func (f *gitRepoFake) DeleteRepositoryAtRelPath(context.Context, string) error { return nil }
func (f *gitRepoFake) DeleteLFSObject(context.Context, *git.OrphanedLFS) error { return nil }
func (f *gitRepoFake) RepositoriesSize(context.Context) int64                  { return 0 }
func (f *gitRepoFake) LFSSize(context.Context) int64                           { return 0 }

type labelRepoFake struct{}

func (f *labelRepoFake) ListByCategoryAndScope(context.Context, string, string) ([]*Label, error) {
	return nil, nil
}
func (f *labelRepoFake) GetByModelID(context.Context, int64) ([]*Label, error) { return nil, nil }
func (f *labelRepoFake) GetOrCreateByName(context.Context, string, string, string) (*Label, error) {
	return nil, nil
}
func (f *labelRepoFake) UpdateModelLabels(context.Context, int64, []int) error { return nil }

type projectRepoFake struct{}

func (f *projectRepoFake) CreateProject(context.Context, *project.Project) (*project.Project, error) {
	return nil, nil
}
func (f *projectRepoFake) GetProjectByID(context.Context, int) (*project.Project, error) {
	return nil, nil
}
func (f *projectRepoFake) GetProjectByName(context.Context, string) (*project.Project, error) {
	return &project.Project{ID: 1, Name: "proj"}, nil
}
func (f *projectRepoFake) GetProjectIDByName(context.Context, string) (int, error) { return 1, nil }
func (f *projectRepoFake) ListProjects(context.Context, string, project.ProjectType, project.PermissionFilter, bool, int, int) ([]*project.Project, int64, error) {
	return nil, 0, nil
}
func (f *projectRepoFake) UpdateProject(context.Context, *project.Project) error { return nil }
func (f *projectRepoFake) DeleteProject(context.Context, int) error              { return nil }
func (f *projectRepoFake) ListProjectInfoByNames(context.Context, []string) ([]*project.Project, error) {
	return nil, nil
}
func (f *projectRepoFake) ListProjectMembers(context.Context, int, string, int, int) ([]*project.ProjectMember, int64, error) {
	return nil, 0, nil
}
func (f *projectRepoFake) AddProjectMemberWithRole(context.Context, *project.ProjectMember) error {
	return nil
}
func (f *projectRepoFake) RemoveProjectMembers(context.Context, int, []*project.Member) error {
	return nil
}
func (f *projectRepoFake) UpdateProjectMemberRole(context.Context, int, project.Member, role.RoleType) error {
	return nil
}
func (f *projectRepoFake) GetUserProjectRole(context.Context, int, int) (int, error) { return 0, nil }

type registryRepoFake struct{}

func (f *registryRepoFake) ListRegistries(context.Context, int, int, string) ([]*registry.Registry, int64, error) {
	return nil, 0, nil
}
func (f *registryRepoFake) GetRegistry(context.Context, int) (*registry.Registry, error) {
	return nil, nil
}
func (f *registryRepoFake) CreateRegistry(context.Context, registry.Registry) (*registry.Registry, error) {
	return nil, nil
}
func (f *registryRepoFake) UpdateRegistry(context.Context, registry.Registry) error { return nil }
func (f *registryRepoFake) DeleteRegistry(context.Context, int) error               { return nil }
func (f *registryRepoFake) PingRegistry(context.Context, registry.Registry) (int, string, error) {
	return 0, "", nil
}

func newEnsureModelService(modelRepo *modelRepoFake, gitRepo *gitRepoFake) IModelService {
	return NewModelService(modelRepo, &labelRepoFake{}, gitRepo, &projectRepoFake{}, &registryRepoFake{})
}

func TestEnsureModelReturnsExistingModel(t *testing.T) {
	modelRepo := &modelRepoFake{models: map[string]*Model{"proj/model": {Name: "model", ProjectName: "proj"}}}
	gitRepo := &gitRepoFake{}
	service := newEnsureModelService(modelRepo, gitRepo)

	mod, err := service.EnsureModel(context.Background(), "proj", "model")
	if err != nil {
		t.Fatalf("EnsureModel returned error: %v", err)
	}
	if mod.Name != "model" {
		t.Fatalf("expected existing model, got %+v", mod)
	}
	if gitRepo.repositoryChecked || gitRepo.createRepository {
		t.Fatalf("expected no git repository operations for existing model")
	}
}

func TestEnsureModelCreatesRepoAndModelWhenBothMissing(t *testing.T) {
	modelRepo := &modelRepoFake{}
	gitRepo := &gitRepoFake{}
	service := newEnsureModelService(modelRepo, gitRepo)

	if _, err := service.EnsureModel(context.Background(), "proj", "model"); err != nil {
		t.Fatalf("EnsureModel returned error: %v", err)
	}
	if !gitRepo.repositoryChecked {
		t.Fatalf("expected repository existence check")
	}
	if !gitRepo.createRepository {
		t.Fatalf("expected repository creation")
	}
	if len(modelRepo.created) != 1 {
		t.Fatalf("expected one model record, got %d", len(modelRepo.created))
	}
}

func TestEnsureModelCreatesOnlyModelWhenRepoExists(t *testing.T) {
	modelRepo := &modelRepoFake{}
	gitRepo := &gitRepoFake{repoExists: true}
	service := newEnsureModelService(modelRepo, gitRepo)

	if _, err := service.EnsureModel(context.Background(), "proj", "model"); err != nil {
		t.Fatalf("EnsureModel returned error: %v", err)
	}
	if !gitRepo.repositoryChecked {
		t.Fatalf("expected repository existence check")
	}
	if gitRepo.createRepository {
		t.Fatalf("did not expect repository creation")
	}
	if len(modelRepo.created) != 1 {
		t.Fatalf("expected one model record, got %d", len(modelRepo.created))
	}
}
