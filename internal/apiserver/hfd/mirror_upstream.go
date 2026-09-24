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

package hfd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	xetmirror "github.com/wzshiming/xet/mirror"
	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// mirrorTransport rewrites the engine's hub resolve requests from the local
// repository name to the DB source's base path and organization.
type mirrorTransport struct {
	backend *Backend
}

func (t *mirrorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	repoName, rest, ok := strings.Cut(strings.TrimPrefix(req.URL.Path, "/"), "/resolve/")
	if !ok {
		return http.DefaultTransport.RoundTrip(req)
	}
	sourceURL, ok, err := t.backend.mirrorSource(req.Context(), repoName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return http.DefaultTransport.RoundTrip(req)
	}
	origin, err := sourceOrigin(sourceURL)
	if err != nil {
		return nil, err
	}
	if req.URL.Scheme != origin.Scheme || req.URL.Host != origin.Host {
		return nil, fmt.Errorf("mirror transport: %s does not target the current source origin %s of %s", req.URL.Redacted(), origin, repoName)
	}
	req = req.Clone(req.Context())
	req.URL.Path, req.URL.RawPath = strings.TrimSuffix(sourceURL.Path, "/")+"/resolve/"+rest, ""
	return http.DefaultTransport.RoundTrip(req)
}

// mirrorUpstream is the engine's xetmirror.UpstreamFunc: the repository's
// registry origin with the registry's Basic password as the bearer token.
func (b *Backend) mirrorUpstream(ctx context.Context, escapedRepo string) (*url.URL, string, error) {
	repoName, err := url.PathUnescape(escapedRepo)
	if err != nil {
		return nil, "", fmt.Errorf("mirror upstream: repository %q: %w", escapedRepo, err)
	}
	sourceURL, ok, err := b.mirrorSource(ctx, repoName)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", fmt.Errorf("%w: %s has no registry source", xetmirror.ErrUpstreamNotFound, repoName)
	}
	origin, err := sourceOrigin(sourceURL)
	if err != nil {
		return nil, "", err
	}
	token, _ := sourceURL.User.Password()
	return origin, token, nil
}

func sourceOrigin(sourceURL *url.URL) (*url.URL, error) {
	if (sourceURL.Scheme != "http" && sourceURL.Scheme != "https") || sourceURL.Host == "" {
		return nil, fmt.Errorf("mirror source %s is not an http(s) URL", sourceURL.Redacted())
	}
	return &url.URL{Scheme: sourceURL.Scheme, Host: sourceURL.Host}, nil
}

func (b *Backend) mirrorSource(ctx context.Context, repoName string) (*url.URL, bool, error) {
	repoType, projectName, name, ok := utils.ParseFromRepoName(repoName)
	if !ok || repoType != "models" {
		return nil, false, nil
	}
	prj, err := b.projectRepo.GetProjectByName(ctx, projectName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if !prj.HasProxy() {
		return nil, false, nil
	}
	reg, err := b.registryRepo.GetRegistry(ctx, *prj.RegistryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	sourceURL, err := url.Parse(strings.TrimSuffix(reg.URL, "/") + "/" + prj.Organization + "/" + name)
	if err != nil {
		return nil, false, err
	}
	if bc := registry.AsBasic(reg.GetCredential()); bc != nil {
		sourceURL.User = url.UserPassword(bc.Username, bc.Password)
	}
	return sourceURL, true, nil
}
