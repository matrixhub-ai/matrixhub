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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	gitstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	xetclient "github.com/wzshiming/xet/client"
	xetstorage "github.com/wzshiming/xet/storage"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/domain/authz"
	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/robot"
	"github.com/matrixhub-ai/matrixhub/internal/domain/user"
	"github.com/matrixhub-ai/matrixhub/internal/infra/config"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

type gitAuth struct {
	basicAuthValidator authenticate.BasicAuthValidator
	publicKeyValidator authenticate.PublicKeyValidator
	tokenValidator     authenticate.TokenValidator
	tokenSignValidator authenticate.TokenSignValidator
}

// xetStore is the xet store together with its GC surface.
type xetStore interface {
	xetstorage.Storage
	xetstorage.GCStore
}

type gitStorage struct {
	storage      *gitstorage.Storage
	xetStorage   xetStore
	xetAuthFn    func(token string) bool
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

	// repos consumed by the HTTP auth middlewares in Handler.
	akRepo      user.IAccessTokenRepo
	sessionRepo user.ISessionRepo
	userRepo    user.IUserRepo
	robotRepo   robot.IRobotRepo
}

// New builds the storage/xet/mirror layer. Hooks and auth validators that need
// domain services attach later via Bind, preserving the original init order.
func New(cfg *config.Config) *Backend {
	b := &Backend{config: cfg}
	b.initGitStorage()
	return b
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
) {
	b.modelService = modelService
	b.datasetService = datasetService
	b.akRepo = akRepo
	b.sessionRepo = sessionRepo
	b.userRepo = userRepo
	b.robotRepo = robotRepo
	// Normalize wrapper: SSH sessions and signed LFS tokens re-enter with an
	// authcodec blob UserInfo and bypass the HTTP identityNormalizer middleware.
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

// XetGCStore exposes the xet store's GC surface for the repo layer.
func (b *Backend) XetGCStore() xetstorage.GCStore { return b.storage.xetStorage }

// preOpenHook ports the fork openRepo() semantics: model repos are ensured in
// the DB before any open; reads additionally sync from the remote registry
// when a mirror is configured.
func (b *Backend) preOpenHook(ctx context.Context, repoName string, write bool) error {
	repoType, project, name, ok := utils.ParseFromRepoName(repoName)
	if !ok || repoType != "models" {
		return nil
	}
	if write {
		// git-receive-pack path: only ensure the model record exists.
		if _, err := b.modelService.EnsureModel(ctx, project, name); err != nil {
			return fmt.Errorf("ensure model %s/%s: %w", project, name, err)
		}
		return nil
	}
	if b.storage.sharedMirror == nil {
		return nil
	}
	if _, err := b.modelService.EnsureModel(ctx, project, name); err != nil {
		log.Errorf("failed to ensure model for %s/%s: %v", project, name, err)
		return err
	}
	if err := b.modelService.CheckOrSyncFromRemote(ctx, project, name); err != nil {
		log.Errorf("failed to sync from remote for %s/%s: %v", project, name, err)
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
		return nil
	}
	if err != nil {
		log.Warnw("sync metadata after receive failed", "repo", repoName, "error", err)
	}
	return nil
}

func (b *Backend) mirrorSource(ctx context.Context, repoName string) (string, bool, error) {
	// return baseURL + "/" + repoName, true, nil
	return "", false, nil
}

func (b *Backend) mirrorDestination(ctx context.Context, repoName string) (string, bool, error) {
	return "", false, nil
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
	b.auth.basicAuthValidator = middleware.GitBasicAuthAuthn(akRepo, userRepo, robotRepo)
	b.auth.publicKeyValidator = middleware.GitPublicKeyAuthn(sshKeyRepo, userRepo)
	b.auth.tokenValidator = middleware.GitHTTPAuthn(akRepo, userRepo, robotRepo)
	// tokenSignValidator is created in initGitStorage: the mirror needs its
	// XET token mint at construction time, before repos/services exist.
}

func (b *Backend) initGitStorage() {
	storage := gitstorage.NewStorage(
		gitstorage.WithRootDir(b.config.DataDir),
	)

	// xet content storage holds all LFS bytes under DataDir/xet/storage; the
	// xet client keeps its chunk cache under DataDir/xet/chunks.
	xetStorage, err := xetstorage.NewFileStorage(
		xetstorage.WithBasePath(filepath.Join(b.config.DataDir, "xet", "storage")),
	)
	if err != nil {
		log.Fatalw("create xet storage failed", "error", err)
	}
	chunksDir := filepath.Join(b.config.DataDir, "xet", "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		log.Fatalw("create xet chunk cache dir failed", "error", err)
	}
	xetClient, err := xetclient.NewClient(
		xetclient.WithCacheDir(chunksDir),
	)
	if err != nil {
		log.Fatalw("create xet client failed", "error", err)
	}

	// Generating and validating temporary tokens: http lfs download in ssh
	// ports, and minting/validating xet CAS access tokens.
	tokenSignValidator := authenticate.NewTokenSignValidator([]byte(b.config.APIServer.TokenSigningSecret))
	mintToken, xetAuthFn, err := authenticate.NewXETTokenScheme(tokenSignValidator)
	if err != nil {
		log.Fatalw("create xet token scheme failed", "error", err)
	}

	sharedMirror, err := mirror.NewMirror(
		mirror.WithMirrorSourceFunc(b.mirrorSource),
		// Must stay non-nil: PushToRemote silently no-ops without a destination
		// func, even when the per-call DestinationURL is set (jobserver manual
		// push sync relies on per-call URLs).
		mirror.WithMirrorDestinationFunc(b.mirrorDestination),
		mirror.WithMirrorRefFilterFunc(b.mirrorRefFilter),
		// No receive hooks here (nil is load-bearing): mirror pulls carry
		// remote-namespace repo names (e.g. "Qwen/Qwen3-32B"), which
		// preReceiveHook would misparse as local project/name and reject or
		// phantom-create models. The git/hf backends attach the hooks with
		// local repo names instead.
		mirror.WithXETStorage(xetStorage),
		mirror.WithXETClient(xetClient),
		mirror.WithMintToken(mintToken),
		mirror.WithExternalURL(b.config.APIServer.HostURL),
		mirror.WithDataDir(filepath.Join(b.config.DataDir, "xet")),
		mirror.WithRepositoriesFS(storage.RepositoriesFS()),
		mirror.WithConcurrency(2),
		// No WithTTL: upstream removed sync throttling (46b09dd "fix download
		// model") so every open re-checks the remote.
		// No WithGitOutputFunc: it would override per-call Pull/PushOptions.Output
		// and swallow jobserver task logs.
	)
	if err != nil {
		log.Fatalw("create git mirror failed", "error", err)
	}

	b.storage.storage = storage
	b.storage.xetStorage = xetStorage
	b.storage.xetAuthFn = xetAuthFn
	b.storage.sharedMirror = sharedMirror
	b.auth.tokenSignValidator = tokenSignValidator
}
