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

// Package hfd wires the hfd library (git/HF/LFS/xet protocol backends) into
// matrixhub: storage and mirror construction, matrixhub business hooks, and
// the auth validator adapters.
package hfd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	backendhf "github.com/matrixhub-ai/hfd/pkg/backend/hf"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	xetauth "github.com/wzshiming/xet/auth"
	xetclient "github.com/wzshiming/xet/client"
	xetmirror "github.com/wzshiming/xet/mirror"
	xetstorage "github.com/wzshiming/xet/storage"
	xetlocal "github.com/wzshiming/xet/storage/local"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type gitAuth struct {
	basicAuthValidator authenticate.BasicAuthValidator
	publicKeyValidator authenticate.PublicKeyValidator
	tokenValidator     authenticate.TokenValidator
	tokenSignValidator authenticate.TokenSignValidator
}

type gitStorage struct {
	storage      *gitstorage.Storage
	xetStorage   xetstorage.Storage
	casIssuer    *xetauth.Issuer
	sharedMirror *mirror.Mirror
}

// Backend bundles the hfd integration state shared by the HTTP protocol chain
// and the SSH server.
type Backend struct {
	config *config.Config

	// permissionHookFunc is composed at Bind time from the authz service; the
	// remaining protocol hooks are Backend methods.
	permissionHookFunc func(ctx context.Context, op permission.Operation, repoName string, opCtx permission.Context) (bool, error)

	auth    gitAuth
	storage gitStorage

	modelService   model.IModelService
	datasetService dataset.IDatasetService
	projectRepo    project.IProjectRepo
	registryRepo   registry.IRegistryRepo

	// repos consumed by the HTTP auth middlewares in Handler.
	akRepo      user.IAccessTokenRepo
	sessionRepo user.ISessionRepo
	userRepo    user.IUserRepo
	robotRepo   robot.IRobotRepo
}

// New builds the storage/xet/mirror layer. Hooks and auth validators that need
// domain services attach later via Bind, preserving the original init order.
func New(cfg *config.Config) (*Backend, error) {
	b := &Backend{config: cfg}
	err := b.init()
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Bind attaches domain services and repos: business hooks (permission,
// pre-open model sync, pre-receive) and the git auth validators.
func (b *Backend) Bind(
	modelService model.IModelService,
	datasetService dataset.IDatasetService,
	authzService authz.IAuthzService,
	akRepo user.IAccessTokenRepo,
	sessionRepo user.ISessionRepo,
	userRepo user.IUserRepo,
	robotRepo robot.IRobotRepo,
	sshKeyRepo user.ISSHKeyRepo,
	projectRepo project.IProjectRepo,
	registryRepo registry.IRegistryRepo,
) {
	b.modelService = modelService
	b.datasetService = datasetService
	b.akRepo = akRepo
	b.sessionRepo = sessionRepo
	b.userRepo = userRepo
	b.robotRepo = robotRepo
	b.projectRepo = projectRepo
	b.registryRepo = registryRepo
	b.permissionHookFunc = normalizePermissionHook(middleware.NewRepoEnforcer(authzService))
	b.initGitAuth(akRepo, userRepo, robotRepo, sshKeyRepo)
}

// Storage exposes the git repository storage for the repo layer.
func (b *Backend) Storage() *gitstorage.Storage {
	return b.storage.storage
}

// Mirror exposes the shared mirror for the repo layer.
func (b *Backend) Mirror() *mirror.Mirror {
	return b.storage.sharedMirror
}

// XetStorage exposes the xet store's GC surface for the repo layer.
func (b *Backend) XetStorage() xetstorage.Storage { return b.storage.xetStorage }

func (b *Backend) preOpenHook(ctx context.Context, repoName string, write bool) error {
	repoType, project, name, ok := utils.ParseFromRepoName(repoName)
	if !ok || repoType != "models" {
		return nil
	}
	if write {
		if _, err := b.modelService.EnsureModel(ctx, project, name); err != nil {
			return fmt.Errorf("ensure model %s/%s: %w", project, name, err)
		}
		return nil
	}
	if err := b.modelService.CheckOrSyncFromRemote(ctx, project, name); err != nil {
		return err
	}
	return nil
}

func (b *Backend) preReceiveHook(ctx context.Context, repoName string, updates []receive.RefUpdate) (bool, error) {
	repoType, project, name, ok := utils.ParseFromRepoName(repoName)
	if !ok {
		return false, nil
	}
	if repoType == "models" {
		_, err := b.modelService.EnsureModel(ctx, project, name)
		return err == nil, err
	}

	return false, nil
}

// postReceiveHook refreshes DB metadata (README, size, labels) after a push;
// failures must not fail the push, so they are only logged.
func (b *Backend) postReceiveHook(ctx context.Context, repoName string, updates []receive.RefUpdate) error {
	repoType, project, name, ok := utils.ParseFromRepoName(repoName)
	if !ok {
		return nil
	}
	var err error
	switch repoType {
	case "models":
		err = b.modelService.SyncMetadata(ctx, project, name)
	case "datasets":
		err = b.datasetService.SyncMetadata(ctx, project, name)
	default:
		return fmt.Errorf("unsupport type %q", repoType)
	}
	if err != nil {
		return fmt.Errorf("sync metadata after receive failed: %w", err)
	}
	return nil
}

type whoamiOrg struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Fullname string `json:"fullname"`
}

// whoami reads the DB rows behind the principal: session and token names may be stale.
func (b *Backend) whoami(ctx context.Context) (*backendhf.WhoamiResponse, error) {
	principal, ok := authenticate.IdentityFrom(ctx).(middleware.Principal)
	if !ok {
		return nil, errors.New("whoami: no matrixhub principal")
	}
	response := &backendhf.WhoamiResponse{
		Type: principal.TypeName(),
		Orgs: []any{},
		Auth: backendhf.WhoamiAuth{AccessToken: backendhf.WhoamiAccessToken{DisplayName: "token", Role: "write"}},
	}
	id := principal.GetID()
	var projects []*project.Project
	switch response.Type {
	case "user":
		account, err := b.userRepo.GetUser(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("whoami user %d: %w", id, err)
		}
		if account == nil {
			return nil, fmt.Errorf("whoami user %d: not found", id)
		}
		response.ID, response.Name, response.Fullname, response.Email = strconv.Itoa(account.ID), account.Username, account.Username, account.Email
		roles, err := b.userRepo.GetUserAllProjectRoles(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("whoami user %d projects: %w", id, err)
		}
		if len(roles) > 0 {
			if projects, err = b.projectRepo.ListProjectInfoByNames(ctx, slices.Sorted(maps.Keys(roles))); err != nil {
				return nil, fmt.Errorf("whoami user %d projects: %w", id, err)
			}
		}
	case "robot":
		account, err := b.robotRepo.GetRobot(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("whoami robot %d: %w", id, err)
		}
		if account == nil || !account.IsValid(time.Now()) {
			return nil, fmt.Errorf("whoami robot %d: not found or disabled", id)
		}
		response.ID, response.Name, response.Fullname = strconv.Itoa(account.ID), account.Name, account.Name
		projects = account.Projects
	default:
		return nil, fmt.Errorf("whoami: unsupported identity type %q", response.Type)
	}
	slices.SortFunc(projects, func(left, right *project.Project) int { return strings.Compare(left.Name, right.Name) })
	for _, p := range projects {
		response.Orgs = append(response.Orgs, whoamiOrg{Type: "org", ID: strconv.Itoa(p.ID), Name: p.Name, Fullname: p.Name})
	}
	return response, nil
}

const listPageSize = 100

// listRepos lists one project from the DB, cutting the cursor window out of the
// page-based Filter (two pages when the offset is unaligned).
func (b *Backend) listRepos(ctx context.Context, repoType string, query backendhf.ListQuery) ([]backendhf.RepoListItem, bool, error) {
	if query.Author == "" || strings.Contains(query.Author, "/") || (repoType != "models" && repoType != "datasets") {
		return nil, false, nil
	}
	if query.Offset < 0 || query.Offset >= math.MaxInt32 {
		return nil, false, nil
	}
	prj, err := b.projectRepo.GetProjectByName(ctx, query.Author)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	private := !prj.IsPublic()
	list := func(filter *model.Filter) ([]backendhf.RepoListItem, int64, error) {
		var items []backendhf.RepoListItem
		if repoType == "models" {
			models, total, err := b.modelService.ListModels(ctx, filter)
			for _, m := range models {
				item := repoListItem(m.ProjectName, m.Name, m.Labels, private)
				item.ModelID = item.RepoID
				items = append(items, item)
			}
			return items, total, err
		}
		datasets, total, err := b.datasetService.ListDatasets(ctx, filter)
		for _, d := range datasets {
			items = append(items, repoListItem(d.ProjectName, d.Name, d.Labels, private))
		}
		return items, total, err
	}

	limit := min(cmp.Or(query.Limit, listPageSize), math.MaxInt32-query.Offset)
	page := query.Offset/limit + 1
	skip := query.Offset - (page-1)*limit
	filter := &model.Filter{Project: query.Author, Search: query.Search, Label: query.FilterTags, Page: int32(page), PageSize: int32(limit)}
	items, total, err := list(filter)
	if err != nil {
		return nil, false, err
	}
	total = min(total, int64(math.MaxInt32))
	if skip > 0 && int64(page)*int64(limit) < total {
		filter.Page++
		next, _, err := list(filter)
		if err != nil {
			return nil, false, err
		}
		items = append(items, next...)
	}
	items = items[min(skip, len(items)):]
	if len(items) > limit {
		items = items[:limit]
	}
	return items, int64(query.Offset)+int64(len(items)) < total, nil
}

func repoListItem(project, name string, labels []model.Label, private bool) backendhf.RepoListItem {
	item := backendhf.RepoListItem{RepoID: project + "/" + name, Private: private}
	for _, label := range labels {
		if !slices.Contains(item.Tags, label.Name) {
			item.Tags = append(item.Tags, label.Name)
		}
		switch label.Category {
		case "task":
			item.PipelineTag = cmp.Or(item.PipelineTag, label.Name)
		case "library":
			item.LibraryName = cmp.Or(item.LibraryName, label.Name)
		}
	}
	return item
}

// createRepo returns no ref updates: the service owns Git and DB, so the hf
// handler must not run a post-receive sync on top.
func (b *Backend) createRepo(ctx context.Context, repoName string, _ backendhf.CreateRepoRequest) ([]receive.RefUpdate, error) {
	repoType, project, name, err := ownedRepo(repoName)
	if err != nil {
		return nil, err
	}

	switch repoType {
	case "models":
		_, err = b.modelService.CreateModel(ctx, project, name)
	case "datasets":
		_, err = b.datasetService.CreateDataset(ctx, project, name)
	default:
		return nil, fmt.Errorf("unsupport type %q", repoType)
	}

	// An existing repository answers 200 with its URL: hf upload re-creates with exist_ok and reads it.
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil, nil
	}
	return nil, catalogError(err)
}

func (b *Backend) deleteRepo(ctx context.Context, repoName string) error {
	repoType, project, name, err := ownedRepo(repoName)
	if err != nil {
		return err
	}

	switch repoType {
	case "models":
		return catalogError(b.modelService.DeleteModel(ctx, project, name))
	case "datasets":
		return catalogError(b.datasetService.DeleteDataset(ctx, project, name))
	default:
		return fmt.Errorf("unsupport type %q", repoType)
	}
}

// ownedRepo refuses spaces and kernels (which parse as a model named "proj/k").
func ownedRepo(repoName string) (repoType, project, name string, err error) {
	repoType, project, name, ok := utils.ParseFromRepoName(repoName)
	if !ok || strings.Contains(name, "/") || (repoType != "models" && repoType != "datasets") {
		return "", "", "", fmt.Errorf("unsupported repository %q", repoName)
	}
	return repoType, project, name, nil
}

// catalogError maps the services' textual errors onto hfd's 404 sentinel, as the gRPC handlers do.
func catalogError(err error) error {
	switch {
	case err == nil:
		return nil
	case strings.Contains(err.Error(), "not found"), strings.Contains(err.Error(), "does not exist"):
		return fmt.Errorf("%w: %v", repository.ErrRepositoryNotExists, err)
	}
	return err
}

func (b *Backend) mirrorRefFilter(ctx context.Context, repoName string, remoteRefs []string) ([]string, error) {
	filteredRefs := []string{}
	for _, ref := range remoteRefs {
		if strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/") {
			filteredRefs = append(filteredRefs, ref)
		}
	}
	return filteredRefs, nil
}

func (b *Backend) initGitAuth(
	akRepo user.IAccessTokenRepo,
	userRepo user.IUserRepo,
	robotRepo robot.IRobotRepo,
	sshKeyRepo user.ISSHKeyRepo,
) {
	b.auth.basicAuthValidator = authenticate.BasicAuthValidatorFunc(middleware.GitBasicAuthAuthn(akRepo, userRepo, robotRepo))
	b.auth.publicKeyValidator = authenticate.PublicKeyValidatorFunc(middleware.GitPublicKeyAuthn(sshKeyRepo, userRepo))
	b.auth.tokenValidator = authenticate.TokenValidatorFunc(middleware.GitHTTPAuthn(akRepo, userRepo, robotRepo))
}

func (b *Backend) init() error {
	xetStorage, err := xetlocal.NewStorage(
		xetlocal.WithBasePath(filepath.Join(b.config.DataDir, "xet", "storage")),
	)
	if err != nil {
		return fmt.Errorf("create xet storage failed: %w", err)
	}
	storage, err := gitstorage.NewStorage(
		gitstorage.WithRootDir(b.config.DataDir),
		gitstorage.WithXETStorage(xetStorage),
	)
	if err != nil {
		return fmt.Errorf("create git storage failed: %w", err)
	}
	chunksDir := filepath.Join(b.config.DataDir, "xet", "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("create xet chunk cache dir failed: %w", err)
	}
	xetClient, err := xetclient.NewClient(
		xetclient.WithCacheDir(chunksDir),
	)
	if err != nil {
		return fmt.Errorf("create xet client failed: %w", err)
	}

	// The signing key backs both signed LFS tokens and xet CAS grants.
	tokenSignValidator := newTokenSignValidator([]byte(b.config.APIServer.TokenSigningSecret))
	casIssuer, err := xetauth.NewIssuer([]byte(b.config.APIServer.TokenSigningSecret), time.Hour, nil)
	if err != nil {
		return fmt.Errorf("create xet token issuer failed: %w", err)
	}

	xetMirror, err := xetmirror.NewMirror(
		xetmirror.WithStorage(xetStorage),
		xetmirror.WithClient(xetClient),
		xetmirror.WithUpstream(b.mirrorUpstream),
		xetmirror.WithTransport(&mirrorTransport{backend: b}),
		xetmirror.WithCacheDir(filepath.Join(b.config.DataDir, "xet", "mirror")),
	)
	if err != nil {
		return fmt.Errorf("create xet mirror engine failed: %w", err)
	}

	sharedMirror, err := mirror.NewMirror(
		mirror.WithMirrorRefFilterFunc(b.mirrorRefFilter),
		// No receive hooks: the pull callers refresh metadata themselves.
		mirror.WithXETStorage(xetStorage),
		mirror.WithXETClient(xetClient),
		mirror.WithXETMirror(xetMirror),
		mirror.WithMintToken(casIssuer.Sign),
		mirror.WithExternalURL(b.config.APIServer.HostURL),
		mirror.WithDataDir(filepath.Join(b.config.DataDir, "xet")),
		mirror.WithRepositoriesFS(storage.RepositoriesFS()),
		mirror.WithConcurrency(4),
		// No WithGitOutputFunc: it would override per-call Output and swallow jobserver logs.
	)
	if err != nil {
		return fmt.Errorf("create git mirror failed: %w", err)
	}

	// Objects left by the pre-xet local LFS store are imported into xet; originals stay.
	if err := migrateLegacyLFS(context.Background(), filepath.Join(b.config.DataDir, "lfs"), sharedMirror); err != nil {
		return err
	}

	b.storage.storage = storage
	b.storage.xetStorage = xetStorage
	b.storage.casIssuer = casIssuer
	b.storage.sharedMirror = sharedMirror
	b.auth.tokenSignValidator = tokenSignValidator
	return nil
}
