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

package apiserver

import (
	"context"
	"net/http"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"

	"github.com/matrixhub-ai/matrixhub/internal/domain/artifactscan"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
	"github.com/matrixhub-ai/matrixhub/internal/jobserver"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
)

func (server *APIServer) initArtifactScan() {
	cfg := server.config.ArtifactScan
	if cfg == nil || !cfg.Enabled {
		return
	}
	store, err := repo.NewArtifactScanStore(server.repos.DB)
	if err != nil {
		panic(err)
	}
	source := &repo.ArtifactSource{Storage: server.gitStorage.storage, LFS: server.gitStorage.lfsStorage}
	scannerTimeout := cfg.ScannerTimeoutSeconds
	if scannerTimeout == 0 {
		scannerTimeout = 600
	}
	if scannerTimeout < 1 || scannerTimeout > 3600 {
		panic("invalid artifact scanner timeout")
	}
	scanner := &repo.ArtifactHTTPScanner{Endpoint: cfg.Endpoint, Client: &http.Client{Timeout: time.Duration(scannerTimeout) * time.Second}}
	server.artifactScan, err = artifactscan.NewWithLimits(store, source, scanner, artifactscan.Limits{
		MaxFileBytes: cfg.MaxFileBytes, MaxFiles: cfg.MaxFiles, TaskTimeout: time.Duration(cfg.TaskTimeoutSeconds) * time.Second,
	})
	if err != nil {
		panic(err)
	}
	server.artifactScanWorker = jobserver.NewArtifactScanWorker(server.artifactScan)
}

func (server *APIServer) artifactGitSnapshot(ctx context.Context, name, originalPath string) (string, func(), error) {
	canonical, repoType, valid := artifactRepoName(name)
	if !valid {
		return "", nil, git.ErrReadNotAdmitted
	}
	if repoType != "models" {
		return originalPath, func() {}, nil
	}
	snapshot, release, err := utils.GitReadSnapshot(ctx, originalPath, server.gitStorage.storage.TmpDir())
	if err != nil {
		return "", nil, err
	}
	revisions, err := repo.ArtifactRevisionsAtPath(ctx, snapshot, 256)
	if err == nil {
		user, _ := authenticate.GetUserInfo(ctx)
		checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		var decision artifactscan.Decision
		decision, err = server.artifactScan.AdmitRevisions(checkCtx, canonical, revisions, user.User, "git-snapshot")
		if err == nil && !decision.Allowed {
			err = git.ErrReadNotAdmitted
		}
	}
	if err != nil {
		release()
		return "", nil, err
	}
	return snapshot, release, nil
}

// SSH exec paths may have a leading slash and a .git suffix. Authorization,
// immutable reads and scan reports must all use the same logical repository.
func artifactRepoName(name string) (string, string, bool) {
	return utils.CanonicalRepoName(name)
}
