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
