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
	"net/http"
	"net/http/httptest"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"

	domaingit "github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func modelScopeRevisionFixture(t *testing.T) (*gitRepo, *domaingit.GitRepository, *repository.Repository, string) {
	t.Helper()
	source := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	upstream, err := repository.Init(t.Context(), source.ResolvePath("team/demo"), "master")
	require.NoError(t, err)
	_, err = upstream.CreateCommit(t.Context(), "master", "upstream model", "upstream", "upstream@example.com", []repository.CommitOperation{
		{Type: repository.CommitOperationAdd, Path: "config.json", Content: []byte(`{"revision":"master"}`)},
		{Type: repository.CommitOperationAdd, Path: "nested/tokenizer.json", Content: []byte(`{}`)},
	}, "")
	require.NoError(t, err)
	server := httptest.NewServer(backendhttp.NewHandler(backendhttp.WithStorage(source), backendhttp.WithNext(http.NotFoundHandler())))
	t.Cleanup(server.Close)
	local := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	g := NewGitDB(local, nil).(*gitRepo)
	return g, modelScopeRemote(server.URL), upstream, local.ResolvePath("local/demo")
}

func TestModelScopeHFDefaultAliasesMasterAndTracksUpdates(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		t.Run(map[bool]string{false: "cold repository", true: "existing initializer"}[initialized], func(t *testing.T) {
			g, remote, upstream, path := modelScopeRevisionFixture(t)
			var initial string
			if initialized {
				require.NoError(t, g.CreateRepository(t.Context(), "models", "local", "demo"))
				local, err := repository.Open(path)
				require.NoError(t, err)
				initial, err = local.ResolveRevision("main")
				require.NoError(t, err)
			}
			_, err := fetchModelScopeRefs(t.Context(), remote, path)
			require.NoError(t, err)
			local, err := repository.Open(path)
			require.NoError(t, err)
			main, err := local.ResolveRevision("main")
			require.NoError(t, err)
			master, err := local.ResolveRevision("master")
			require.NoError(t, err)
			require.Equal(t, master, main)
			data, err := g.readBlobBytes(local, "main", "config.json")
			require.NoError(t, err)
			require.JSONEq(t, `{"revision":"master"}`, string(data))
			rev, file, err := local.SplitRevisionAndPath("main/nested/tokenizer.json")
			require.NoError(t, err)
			_, err = local.Blob(rev, file)
			require.NoError(t, err)
			raw, err := gogit.PlainOpen(path)
			require.NoError(t, err)
			if initialized {
				backup, err := raw.Storer.Reference(plumbing.ReferenceName("refs/matrixhub/modelscope-initial-main/" + initial))
				require.NoError(t, err)
				require.Equal(t, initial, backup.Hash().String())
				_, err = local.Commits(initial, &repository.CommitsOptions{Limit: 1})
				require.NoError(t, err)
			}
			_, err = upstream.CreateCommit(t.Context(), "master", "update", "upstream", "upstream@example.com", []repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "new.json", Content: []byte(`{}`)}}, "")
			require.NoError(t, err)
			_, err = fetchModelScopeRefs(t.Context(), remote, path)
			require.NoError(t, err)
			local, err = repository.Open(path)
			require.NoError(t, err)
			updated, err := local.ResolveRevision("main")
			require.NoError(t, err)
			require.NotEqual(t, main, updated)
			_, err = local.Blob("main", "new.json")
			require.NoError(t, err)
			// A real main introduced upstream must replace the alias, without
			// rewriting the independent master branch through that symbolic ref.
			require.NoError(t, upstream.CreateBranch("main", "master"))
			realMain, err := upstream.CreateCommit(t.Context(), "main", "real main", "upstream", "upstream@example.com", []repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "real-main.json", Content: []byte(`{}`)}}, "")
			require.NoError(t, err)
			_, err = fetchModelScopeRefs(t.Context(), remote, path)
			require.NoError(t, err)
			local, err = repository.Open(path)
			require.NoError(t, err)
			got, err := local.ResolveRevision("main")
			require.NoError(t, err)
			require.Equal(t, realMain, got)
			got, err = local.ResolveRevision("master")
			require.NoError(t, err)
			require.Equal(t, updated, got)
		})
	}
}

func TestModelScopeHFDefaultPreservesLocalHistoryAndRealMainTag(t *testing.T) {
	for _, mode := range []string{"local model history", "upstream main tag"} {
		t.Run(mode, func(t *testing.T) {
			g, remote, upstream, path := modelScopeRevisionFixture(t)
			require.NoError(t, g.CreateRepository(t.Context(), "models", "local", "demo"))
			local, err := repository.Open(path)
			require.NoError(t, err)
			if mode == "local model history" {
				own, err := local.CreateCommit(t.Context(), "main", "local data", "user", "user@example.com", []repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "user.json", Content: []byte(`{}`)}}, "")
				require.NoError(t, err)
				_, err = fetchModelScopeRefs(t.Context(), remote, path)
				require.NoError(t, err)
				local, err = repository.Open(path)
				require.NoError(t, err)
				got, err := local.ResolveRevision("main")
				require.NoError(t, err)
				require.Equal(t, own, got)
			} else {
				sha, err := upstream.ResolveRevision("master")
				require.NoError(t, err)
				require.NoError(t, upstream.CreateTag("main", sha))
				_, err = fetchModelScopeRefs(t.Context(), remote, path)
				require.NoError(t, err)
				local, err = repository.Open(path)
				require.NoError(t, err)
				got, err := local.ResolveRevision("main")
				require.NoError(t, err)
				require.Equal(t, sha, got)
			}
		})
	}
}

func TestModelScopeHFDefaultUsesActualHEADAndPreservesEditedAlias(t *testing.T) {
	g, remote, upstream, path := modelScopeRevisionFixture(t)
	require.NoError(t, upstream.CreateBranch("release/v1", "master"))
	head, err := upstream.CreateCommit(t.Context(), "release/v1", "release default", "upstream", "upstream@example.com", []repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "release.json", Content: []byte(`{}`)}}, "")
	require.NoError(t, err)
	source, err := gogit.PlainOpen(upstream.RepoPath())
	require.NoError(t, err)
	require.NoError(t, source.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("release/v1"))))
	_, err = fetchModelScopeRefs(t.Context(), remote, path)
	require.NoError(t, err)
	local, err := repository.Open(path)
	require.NoError(t, err)
	got, err := local.ResolveRevision("main")
	require.NoError(t, err)
	require.Equal(t, head, got)
	own, err := local.CreateCommit(t.Context(), "main", "local edit", "user", "user@example.com", []repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "user.json", Content: []byte(`{}`)}}, "")
	require.NoError(t, err)
	_, err = fetchModelScopeRefs(t.Context(), remote, path)
	require.NoError(t, err)
	local, err = repository.Open(path)
	require.NoError(t, err)
	got, err = local.ResolveRevision("main")
	require.NoError(t, err)
	require.Equal(t, own, got)
	metadata, err := g.ExtractMetadata(t.Context(), "models", "local", "demo")
	require.NoError(t, err)
	require.Equal(t, "release/v1", metadata.DefaultBranch)
}
