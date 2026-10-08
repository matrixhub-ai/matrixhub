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
	"net/url"
	"strings"

	"gorm.io/gorm"

	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// mirrorSource is the hfd mirror's SourceFunc: the repository's registry
// source URL with the registry's Basic credential as userinfo.
func (b *Backend) mirrorSource(ctx context.Context, repoName string) (string, bool, error) {
	repoType, projectName, name, ok := utils.ParseFromRepoName(repoName)
	if !ok || repoType != "models" {
		return "", false, nil
	}
	prj, err := b.projectRepo.GetProjectByName(ctx, projectName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if !prj.HasProxy() {
		return "", false, nil
	}
	reg, err := b.registryRepo.GetRegistry(ctx, *prj.RegistryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	sourceURL, err := url.Parse(strings.TrimSuffix(reg.URL, "/") + "/" + prj.Organization + "/" + name)
	if err != nil {
		return "", false, err
	}
	if bc := registry.AsBasic(reg.GetCredential()); bc != nil {
		sourceURL.User = url.UserPassword(bc.Username, bc.Password)
	}
	return sourceURL.String(), true, nil
}

// mirrorDestination is the hfd mirror's DestinationFunc. No repository is a
// standing push mirror (sync jobs pass DestinationURL per call), but hfd skips
// PushToRemote entirely, explicit URL included, when no callback is registered.
func (b *Backend) mirrorDestination(context.Context, string) (string, bool, error) {
	return "", false, nil
}
