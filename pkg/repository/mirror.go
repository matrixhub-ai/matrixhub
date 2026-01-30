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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	formatcfg "github.com/go-git/go-git/v5/plumbing/format/config"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"

	"github.com/matrixhub-ai/matrixhub/internal/utils"
)

// InitMrror initializes a new bare git repository as a mirror of the remote repository at sourceURL.
func InitMrror(ctx context.Context, repoPath string, sourceURL string) (*Repository, error) {
	sourceURL = strings.TrimSuffix(sourceURL, "/")
	sourceURL = strings.TrimSuffix(sourceURL, ".git") + ".git"
	infoRefsURL := sourceURL + "/info/refs?service=git-upload-pack"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoRefsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/x-git-upload-pack-advertisement")
	req.Header.Set("User-Agent", "go-git/5.x")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch info/refs: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	if resp.Header.Get("Content-Type") != "application/x-git-upload-pack-advertisement" {
		return nil, fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}

	advRefs, err := parseAdvRefs(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse advertisement references from pkt-line: %w", err)
	}

	refs := advRefs.Capabilities.Get(capability.SymRef)

	if len(refs) == 0 {
		return nil, errors.New("no default branch found in remote references")
	}

	rawSymRef := refs[0]
	const headPrefix = "HEAD:refs/heads/"
	if !strings.HasPrefix(rawSymRef, headPrefix) {
		return nil, fmt.Errorf("unexpected SymRef format: %q", rawSymRef)
	}

	branch := strings.TrimPrefix(rawSymRef, headPrefix)
	if strings.TrimSpace(branch) == "" {
		return nil, errors.New("empty default branch in SymRef capability")
	}
	opt := &git.PlainInitOptions{
		Bare:         true,
		ObjectFormat: formatcfg.SHA1,
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName(branch),
		},
	}

	formats := advRefs.Capabilities.Get(capability.ObjectFormat)
	if len(formats) != 0 && formats[0] == "sha256" {
		opt.ObjectFormat = formatcfg.SHA256
	}

	repo, err := git.PlainInitWithOptions(repoPath, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		return nil, err
	}
	cfg.Init.DefaultBranch = branch
	cfg.Remotes = map[string]*config.RemoteConfig{
		"origin": {
			Name:   "origin",
			URLs:   []string{sourceURL},
			Mirror: true,
			Fetch: []config.RefSpec{
				"+refs/heads/*:refs/heads/*",
			},
		},
	}

	err = repo.Storer.SetConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Repository{
		repo:     repo,
		repoPath: repoPath,
	}, nil
}

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
		"--prune",
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

func parseAdvRefs(r io.Reader) (*packp.AdvRefs, error) {
	advRefs := packp.NewAdvRefs()
	err := advRefs.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode advRefs: %w", err)
	}

	if len(advRefs.References) == 0 {
		return nil, errors.New("no references found in advRefs")
	}
	return advRefs, nil
}
