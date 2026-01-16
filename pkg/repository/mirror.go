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

package repository

import (
	"context"
	"fmt"

	"github.com/matrixhub-ai/matrixhub/internal/utils"
)

func (r *Repository) SyncMirror(ctx context.Context) error {
	branch := r.DefaultBranch()
	err := r.fetchShallow(ctx, branch)
	if err != nil {
		return err
	}

	err = r.fetchShallow(ctx, "*")
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) fetchShallow(ctx context.Context, branch string) error {
	args := []string{
		"fetch",
		"--depth=1",
		"origin",
		fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branch, branch),
		"--progress",
	}
	cmd := utils.Command(ctx, "git", args...)
	cmd.Dir = r.repoPath
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
