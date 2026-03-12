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

package repo

import (
	"context"
	"os"
	"testing"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/pkg/repository"
)

var (
	fakeProject = "public"
	fakeName    = "test-repo" // 解压后的目录名
)

func TestNewGitRepo(t *testing.T) {
	g := NewGitDB()
	require.NotNil(t, g)
}

func TestCreateDeleteGitRepo(t *testing.T) {
	g := NewGitDB()
	err := g.CreateRepository(context.Background(), "project", "name")
	require.NoError(t, err)

	err = g.DeleteRepository(context.Background(), "project", "name")
	require.NoError(t, err)
}

func TestRepoOperations(t *testing.T) {
	fakeRepo()
	defer cleanup()

	g := NewGitDB()

	t.Run("ListRevisions", func(t *testing.T) {
		rev, err := g.ListRevisions(context.Background(), fakeProject, fakeName)
		require.NoError(t, err)
		require.NotEmpty(t, rev.Branches)
	})

	t.Run("ListCommits", func(t *testing.T) {
		commits, _, err := g.ListCommits(context.Background(), fakeProject, fakeName, "", 1, 10)
		require.NoError(t, err)
		require.NotEmpty(t, commits)
	})

	t.Run("GetCommit", func(t *testing.T) {
		commits, err := g.GetCommit(context.Background(), fakeProject, fakeName, "84efb79a883218797b952cc96ab965a6558e7598")
		require.NoError(t, err)
		require.NotEmpty(t, commits)
		require.NotEmpty(t, commits.Diffs[0].Diff)
	})

	t.Run("GetBlob", func(t *testing.T) {
		tn, err := g.GetBlob(context.Background(), fakeProject, fakeName, "", "docs/mcp-integration.md")
		require.NoError(t, err)
		require.NotEmpty(t, tn)
	})

	t.Run("GetTree", func(t *testing.T) {
		tree, err := g.GetTree(context.Background(), fakeProject, fakeName, "", "docs/image")
		require.NoError(t, err)
		require.NotEmpty(t, tree)
		require.Equal(t, 7, len(tree))
	})
}

func fakeRepo() {
	path := repoPath(rootDir, fakeProject, fakeName)
	if repository.IsRepository(path) {
		return
	}

	fs := osfs.New(path)
	sto := filesystem.NewStorageWithOptions(
		fs,
		cache.NewObjectLRUDefault(),
		filesystem.Options{ExclusiveAccess: true})

	_, err := git.Clone(sto, nil, &git.CloneOptions{
		URL: "https://gitee.com/Barryda/QuickDesk.git",
	})
	if err != nil {
		panic(err)
	}

}

func cleanup() {
	_ = os.RemoveAll(rootDir)
}
