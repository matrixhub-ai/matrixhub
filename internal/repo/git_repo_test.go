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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	hfdlfs "github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	hfdstorage "github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

const (
	repoTestProject = "test-project"
	repoTestName    = "test-model"
	// repoTestMissingOID references an LFS object that no test stores locally.
	repoTestMissingOID = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

func TestIsCommitSHA(t *testing.T) {
	for _, test := range []struct {
		name string
		rev  string
		want bool
	}{
		{name: "lowercase full SHA", rev: "0123456789abcdef0123456789abcdef01234567", want: true},
		{name: "mixed case full SHA", rev: "0123456789ABCDEF0123456789abcdefABCDEF01", want: true},
		{name: "empty", rev: ""},
		{name: "short SHA", rev: "0123456"},
		{name: "41 hex characters", rev: "0123456789abcdef0123456789abcdef012345678"},
		{name: "40 characters with non-hex letter", rev: "0123456789abcdef0123456789abcdef0123456g"},
		{name: "40 character branch name", rev: "release/2026-10-30/candidate-build-00001"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, isCommitSHA(test.rev))
		})
	}
}

func TestGitRepoBuildURL(t *testing.T) {
	for _, test := range []struct {
		name     string
		repoType string
		revision string
		path     string
		want     string
	}{
		{
			name:     "empty revision defaults to main",
			repoType: "model",
			path:     "README.md",
			want:     "/test-project/test-model/resolve/main/README.md",
		},
		{
			name:     "short branch name",
			repoType: "models",
			revision: "dev",
			path:     "weights/model.safetensors",
			want:     "/test-project/test-model/resolve/dev/weights/model.safetensors",
		},
		{
			name:     "qualified branch ref",
			repoType: "model",
			revision: "refs/heads/dev",
			path:     "README.md",
			want:     "/test-project/test-model/resolve/dev/README.md",
		},
		{
			name:     "qualified tag ref",
			repoType: "model",
			revision: "refs/tags/v1.0",
			path:     "README.md",
			want:     "/test-project/test-model/resolve/v1.0/README.md",
		},
		{
			name:     "commit SHA",
			repoType: "model",
			revision: "0123456789abcdef0123456789abcdef01234567",
			path:     "README.md",
			want:     "/test-project/test-model/resolve/0123456789abcdef0123456789abcdef01234567/README.md",
		},
		{
			name:     "dataset prefix",
			repoType: "dataset",
			revision: "main",
			path:     "data/train.csv",
			want:     "/datasets/test-project/test-model/resolve/main/data/train.csv",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := (&gitRepo{}).buildURL(test.repoType, repoTestProject, repoTestName, test.revision, test.path)
			require.Equal(t, test.want, got)
		})
	}
}

// refTestRepo is a repository where the short name "same" is both a branch
// pointing at the first commit and a tag pointing at the second one.
type refTestRepo struct {
	git     *gitRepo
	repo    *repository.Repository
	initial string
	first   string
	second  string
}

func newRefTestRepo(t *testing.T) refTestRepo {
	t.Helper()
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	g := NewGitDB(store, nil).(*gitRepo)
	require.NoError(t, g.CreateRepository(ctx, "model", repoTestProject, repoTestName))

	commit := func(message string, files map[string]string) string {
		ops := make([]git.CommitOperation, 0, len(files))
		for path, content := range files {
			ops = append(ops, git.CommitOperation{Type: git.CommitOperationAdd, Path: path, Content: []byte(content)})
		}
		hash, err := g.CreateCommit(ctx, "model", repoTestProject, repoTestName, "main", &git.Commit{
			Message:     message,
			AuthorName:  "Test",
			AuthorEmail: "test@example.com",
		}, ops)
		require.NoError(t, err)
		return hash
	}
	first := commit("first", map[string]string{"README.md": "v1", "weights/model.bin": "weights"})
	second := commit("second", map[string]string{"README.md": "second"})

	repo, err := repository.Open(store.ResolvePath(repoTestProject + "/" + repoTestName))
	require.NoError(t, err)
	evictRepoTestRepository(t, repo)
	initial, err := repo.ResolveRevision(first + "~1")
	require.NoError(t, err)
	require.NoError(t, repo.CreateBranch("same", first))
	require.NoError(t, repo.CreateTag("same", second))
	require.NoError(t, repo.CreateTag("v1.0", first))

	return refTestRepo{git: g, repo: repo, initial: initial, first: first, second: second}
}

func TestResolveRef(t *testing.T) {
	fixture := newRefTestRepo(t)
	// Like git, a full SHA names the commit even when a branch has that name.
	require.NoError(t, fixture.repo.CreateBranch(fixture.second, fixture.first))

	for _, test := range []struct {
		name string
		rev  string
		want string
	}{
		{name: "empty revision", rev: "", want: ""},
		{name: "qualified branch ref", rev: "refs/heads/main", want: "refs/heads/main"},
		{name: "qualified ref is not looked up", rev: "refs/tags/missing", want: "refs/tags/missing"},
		{name: "full commit SHA", rev: fixture.first, want: fixture.first},
		{name: "full commit SHA wins over a branch with that name", rev: fixture.second, want: fixture.second},
		{name: "branch", rev: "main", want: "refs/heads/main"},
		{name: "tag", rev: "v1.0", want: "refs/tags/v1.0"},
		{name: "branch wins over tag with the same name", rev: "same", want: "refs/heads/same"},
		{name: "short commit SHA", rev: fixture.first[:7], want: fixture.first[:7]},
		{name: "unknown revision", rev: "missing", want: "missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, resolveRef(fixture.repo, test.rev))
		})
	}
}

func TestGitRepoListCommitsResolvesRevisions(t *testing.T) {
	ctx := context.Background()
	fixture := newRefTestRepo(t)

	for _, test := range []struct {
		name      string
		revision  string
		page      int
		pageSize  int
		wantIDs   []string
		wantTotal int64
		wantErr   string
	}{
		{
			name:      "branch wins over tag with the same name",
			revision:  "same",
			page:      1,
			pageSize:  10,
			wantIDs:   []string{fixture.first, fixture.initial},
			wantTotal: 2,
		},
		{
			name:      "qualified tag ref",
			revision:  "refs/tags/same",
			page:      1,
			pageSize:  10,
			wantIDs:   []string{fixture.second, fixture.first, fixture.initial},
			wantTotal: 3,
		},
		{
			name:      "tag",
			revision:  "v1.0",
			page:      1,
			pageSize:  10,
			wantIDs:   []string{fixture.first, fixture.initial},
			wantTotal: 2,
		},
		{
			name:      "short commit SHA",
			revision:  fixture.first[:7],
			page:      1,
			pageSize:  10,
			wantIDs:   []string{fixture.first, fixture.initial},
			wantTotal: 2,
		},
		{
			name:     "unknown revision",
			revision: "missing",
			page:     1,
			pageSize: 10,
			wantErr:  "failed to resolve revision",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			commits, total, err := fixture.git.ListCommits(ctx, "model", repoTestProject, repoTestName, test.revision, test.page, test.pageSize)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Nil(t, commits)
				require.Zero(t, total)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.wantTotal, total)
			ids := make([]string, 0, len(commits))
			for _, commit := range commits {
				ids = append(ids, commit.ID)
			}
			require.Equal(t, test.wantIDs, ids)
		})
	}
}

func TestGitRepoGetBlobResolvesRevisions(t *testing.T) {
	ctx := context.Background()
	fixture := newRefTestRepo(t)

	for _, test := range []struct {
		name       string
		revision   string
		path       string
		want       git.TreeEntry
		wantCommit string
		wantErr    string
	}{
		{
			name:     "branch wins over tag with the same name",
			revision: "same",
			path:     "README.md",
			want: git.TreeEntry{
				Name: "README.md", Path: "README.md", Type: git.FileTypeFile, Size: 2,
				URL: "/test-project/test-model/resolve/same/README.md",
			},
			wantCommit: fixture.first,
		},
		{
			name:     "tag",
			revision: "v1.0",
			path:     "README.md",
			want: git.TreeEntry{
				Name: "README.md", Path: "README.md", Type: git.FileTypeFile, Size: 2,
				URL: "/test-project/test-model/resolve/v1.0/README.md",
			},
			wantCommit: fixture.first,
		},
		{
			name:     "default branch",
			revision: "",
			path:     "README.md",
			want: git.TreeEntry{
				Name: "README.md", Path: "README.md", Type: git.FileTypeFile, Size: 6,
				URL: "/test-project/test-model/resolve/main/README.md",
			},
			wantCommit: fixture.second,
		},
		{
			name:     "full commit SHA",
			revision: fixture.first,
			path:     "README.md",
			want: git.TreeEntry{
				Name: "README.md", Path: "README.md", Type: git.FileTypeFile, Size: 2,
				URL: "/test-project/test-model/resolve/" + fixture.first + "/README.md",
			},
			wantCommit: fixture.first,
		},
		{
			name:     "unknown revision",
			revision: "missing",
			path:     "README.md",
			wantErr:  "failed to resolve revision",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			entry, err := fixture.git.GetBlob(ctx, "model", repoTestProject, repoTestName, test.revision, test.path)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Nil(t, entry)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, entry.Commit)
			require.Equal(t, test.wantCommit, entry.Commit.ID)
			require.False(t, entry.Commit.CommitterDate.IsZero())
			require.Equal(t, entry.Commit.CommitterDate, entry.Commit.CreatedAt)
			entry.Commit = nil
			require.NotEmpty(t, entry.Hash)
			entry.Hash = ""
			require.Equal(t, test.want, *entry)
		})
	}
}

func TestExtractMetadataRejectsMissingRepository(t *testing.T) {
	g := NewGitDB(hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir())), nil)

	_, err := g.ExtractMetadata(context.Background(), "model", repoTestProject, repoTestName)

	require.ErrorContains(t, err, "failed to open git repo")
}

func TestReadSafetensorsHeaderFrom(t *testing.T) {
	header := []byte(`{"__metadata__":{"format":"pt"}}`)
	prefix := func(length uint64) []byte {
		return binary.LittleEndian.AppendUint64(nil, length)
	}
	valid := append(prefix(uint64(len(header))), header...)

	for _, test := range []struct {
		name          string
		input         []byte
		want          []byte
		wantRemaining []byte
		wantErr       error
		wantErrText   string
	}{
		{
			name:          "stops after the header",
			input:         append(append([]byte{}, valid...), "tensor-bytes"...),
			want:          valid,
			wantRemaining: []byte("tensor-bytes"),
		},
		{
			name:  "empty header",
			input: prefix(0),
			want:  prefix(0),
		},
		{
			name:    "missing length prefix",
			input:   nil,
			wantErr: io.EOF,
		},
		{
			name:    "truncated length prefix",
			input:   []byte{1, 2, 3},
			wantErr: io.ErrUnexpectedEOF,
		},
		{
			name:    "truncated header",
			input:   append(prefix(uint64(len(header))), header[:4]...),
			wantErr: io.ErrUnexpectedEOF,
		},
		{
			name:        "header over the size limit",
			input:       prefix(maxSafetensorsHeaderBytes + 1),
			wantErrText: "safetensors header too large: 67108865 bytes",
		},
		{
			name:        "length prefix overflowing int",
			input:       prefix(math.MaxUint64),
			wantErrText: "safetensors header too large: 18446744073709551615 bytes",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := bytes.NewReader(test.input)

			got, err := readSafetensorsHeaderFrom(reader)

			switch {
			case test.wantErr != nil:
				require.ErrorIs(t, err, test.wantErr)
				require.Nil(t, got)
			case test.wantErrText != "":
				require.EqualError(t, err, test.wantErrText)
				require.Nil(t, got)
			default:
				require.NoError(t, err)
				require.Equal(t, test.want, got)
				remaining, err := io.ReadAll(reader)
				require.NoError(t, err)
				require.Equal(t, string(test.wantRemaining), string(remaining))
			}
		})
	}
}

func TestParseSafetensorsIndex(t *testing.T) {
	for _, test := range []struct {
		name          string
		indexJSON     string
		wantTotalSize int64
		wantPaths     []string
	}{
		{
			name: "sharded index lists each shard once",
			indexJSON: `{"metadata":{"total_size":4096},"weight_map":{
				"model.embed_tokens.weight":"model-00001-of-00002.safetensors",
				"model.norm.weight":"./model-00001-of-00002.safetensors",
				"lm_head.weight":"model-00002-of-00002.safetensors",
				"model.layers.0.weight":"nested/../model-00002-of-00002.safetensors"}}`,
			wantTotalSize: 4096,
			wantPaths:     []string{"model-00001-of-00002.safetensors", "model-00002-of-00002.safetensors"},
		},
		{
			name: "skips paths outside the repository and non-safetensors files",
			indexJSON: `{"weight_map":{
				"a":"shards/model.safetensors",
				"b":"../escape.safetensors",
				"c":"shards/../../escape.safetensors",
				"d":"/absolute.safetensors",
				"e":"pytorch_model.bin",
				"f":"",
				"g":"."}}`,
			wantPaths: []string{"shards/model.safetensors"},
		},
		{
			name:          "index without weight map",
			indexJSON:     `{"metadata":{"total_size":1}}`,
			wantTotalSize: 1,
		},
		{
			name:      "malformed JSON",
			indexJSON: `{"weight_map":`,
		},
		{
			name:      "unexpected field type",
			indexJSON: `{"metadata":{"total_size":"4096"},"weight_map":{"a":"model.safetensors"}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			index := parseSafetensorsIndex([]byte(test.indexJSON))

			require.Equal(t, test.wantTotalSize, index.Metadata.TotalSize)
			require.ElementsMatch(t, test.wantPaths, index.safetensorsPaths())
		})
	}
}

func TestLFSPointerSize(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)
	files := map[string][]byte{
		"pointer.safetensors":       repoTestLFSPointer(repoTestMissingOID, "1024"),
		"regular.safetensors":       []byte("not an lfs pointer"),
		"empty-object.safetensors":  repoTestLFSPointer(repoTestMissingOID, "0"),
		"size-overflow.safetensors": repoTestLFSPointer(repoTestMissingOID, "9223372036854775808"),
		"oversized.safetensors":     bytes.Repeat([]byte("a"), 2048),
	}
	commitRepoTestFiles(t, ctx, repo, files)

	for _, test := range []struct {
		path     string
		wantSize int64
		wantOK   bool
	}{
		{path: "pointer.safetensors", wantSize: 1024, wantOK: true},
		{path: "regular.safetensors"},
		{path: "empty-object.safetensors"},
		{path: "size-overflow.safetensors"},
		{path: "oversized.safetensors"},
		{path: "missing.safetensors"},
	} {
		t.Run(test.path, func(t *testing.T) {
			size, ok := lfsPointerSize(repo, "main", test.path)

			require.Equal(t, test.wantOK, ok)
			require.Equal(t, test.wantSize, size)
		})
	}
}

func TestGitRepoOpenBlobContent(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)
	storedObject := []byte("content of an LFS object fetched earlier")
	commitRepoTestFiles(t, ctx, repo, map[string][]byte{
		"config.json":         []byte(`{"torch_dtype":"bfloat16"}`),
		"stored.safetensors":  storeRepoTestLFSObject(t, store, storedObject),
		"pending.safetensors": repoTestLFSPointer(repoTestMissingOID, "1024"),
	})
	// A mirror without in-flight downloads stands in for a proxy whose object
	// was neither queued nor fetched.
	idleMirror := mirror.NewMirror(mirror.WithCacheDir(t.TempDir()))

	for _, test := range []struct {
		name    string
		mirror  *mirror.Mirror
		path    string
		want    []byte
		wantErr string
	}{
		{
			name: "regular file is read from git",
			path: "config.json",
			want: []byte(`{"torch_dtype":"bfloat16"}`),
		},
		{
			name: "LFS pointer is resolved from the local object store",
			path: "stored.safetensors",
			want: storedObject,
		},
		{
			name:    "missing LFS object without mirror",
			path:    "pending.safetensors",
			wantErr: "lfs object " + repoTestMissingOID + " is not available locally",
		},
		{
			name:    "missing LFS object not downloading in mirror",
			mirror:  idleMirror,
			path:    "pending.safetensors",
			wantErr: "lfs object " + repoTestMissingOID + " is not available locally",
		},
		{
			name:    "missing file",
			path:    "missing.safetensors",
			wantErr: "file not found",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := NewGitDB(store, test.mirror).(*gitRepo)

			rc, err := g.openBlobContent(repo, "main", test.path)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				require.Nil(t, rc)
				return
			}

			require.NoError(t, err)
			content, err := io.ReadAll(rc)
			require.NoError(t, rc.Close())
			require.NoError(t, err)
			require.Equal(t, test.want, content)
		})
	}
}

func TestExtractMetadataReadsShardHeadersWithoutIndexTotalSize(t *testing.T) {
	ctx := context.Background()
	store := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	repo := initRepoTestRepository(t, ctx, store)

	gitShard := buildRepoTestSafetensorsFile(t, map[string][]int64{"model.embed_tokens.weight": {2, 3}}, 12)
	lfsShard := buildRepoTestSafetensorsFile(t, map[string][]int64{"lm_head.weight": {4, 5}}, 40)
	index, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"format": "pt"},
		"weight_map": map[string]string{
			"model.embed_tokens.weight": "model-00001-of-00003.safetensors",
			"model.norm.weight":         "./model-00001-of-00003.safetensors",
			"lm_head.weight":            "model-00002-of-00003.safetensors",
			"model.layers.0.weight":     "model-00003-of-00003.safetensors",
		},
	})
	require.NoError(t, err)
	commitRepoTestFiles(t, ctx, repo, map[string][]byte{
		"model.safetensors.index.json":     index,
		"model-00001-of-00003.safetensors": gitShard,
		"model-00002-of-00003.safetensors": storeRepoTestLFSObject(t, store, lfsShard),
		"model-00003-of-00003.safetensors": repoTestLFSPointer(repoTestMissingOID, "4096"),
	})

	files, err := NewGitDB(store, nil).ExtractMetadata(ctx, "models", repoTestProject, repoTestName)

	require.NoError(t, err)
	require.Equal(t, index, files.SafetensorsIndexJSON)
	// Readable shards contribute their header, whether the content lives in git
	// or in the LFS store; shards whose object is missing fall back to the size
	// recorded in their pointer.
	require.Equal(t, map[string][]byte{
		"model-00001-of-00003.safetensors": repoTestSafetensorsHeader(t, gitShard),
		"model-00002-of-00003.safetensors": repoTestSafetensorsHeader(t, lfsShard),
	}, files.SafetensorsFiles)
	require.Equal(t, map[string]int64{"model-00003-of-00003.safetensors": 4096}, files.SafetensorsSizes)
}

func TestReadSafetensorsHeaderGivesUpOnStalledMirrorDownload(t *testing.T) {
	ctx := context.Background()
	// git talks to the loopback test server directly, even when a proxy is
	// configured for the developer's machine.
	t.Setenv("NO_PROXY", "127.0.0.1")
	t.Setenv("no_proxy", "127.0.0.1")

	upstream := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	upstreamRepo := initRepoTestRepositoryAt(t, ctx, upstream, "upstream/model")
	commitRepoTestFiles(t, ctx, upstreamRepo, map[string][]byte{
		"model.safetensors": repoTestLFSPointer(repoTestMissingOID, "1024"),
	})

	// The upstream serves git but holds every LFS batch request until the test
	// ends, so the object stays in flight in the mirror tee cache.
	release := make(chan struct{})
	lfsBatch := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/info/lfs/objects/batch") {
			http.NotFound(w, r)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.Header().Set("Content-Type", "application/vnd.git-lfs+json")
		_, _ = fmt.Fprintf(w, `{"objects":[{"oid":%q,"size":1024,"error":{"code":404,"message":"object not found"}}]}`, repoTestMissingOID)
	})
	server := httptest.NewServer(backendhttp.NewHandler(
		backendhttp.WithStorage(upstream),
		backendhttp.WithNext(lfsBatch),
	))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })

	local := hfdstorage.NewStorage(hfdstorage.WithRootDir(t.TempDir()))
	gitMirror := mirror.NewMirror(
		mirror.WithLFSStorage(hfdlfs.NewLocal(local.LFSDir())),
		mirror.WithCacheDir(t.TempDir()),
	)
	g := NewGitDB(local, gitMirror).(*gitRepo)
	pull := &git.GitRepository{
		RemoteRegistryURL:  server.URL,
		RemoteProjectName:  "upstream",
		RemoteResourceName: "model",
		ProjectName:        repoTestProject,
		ResourceName:       repoTestName,
		ResourceType:       "model",
		LogWriter:          io.Discard,
	}
	require.NoError(t, g.PullFromRemote(ctx, pull))
	repo, err := repository.Open(local.ResolvePath(repoTestProject + "/" + repoTestName))
	require.NoError(t, err)
	evictRepoTestRepository(t, repo)

	// PullFromRemote only queues the object: nothing is on disk, so content is
	// served by the in-flight tee cache download.
	require.False(t, hfdlfs.NewLocal(local.LFSDir()).Exists(repoTestMissingOID))
	// Under load the mirror occasionally skips queueing the pointer of a freshly
	// mirrored repository until the next sync, and the tee cache briefly hides
	// an object while its background worker promotes it. Sync again until the
	// object is in flight; it then stays cached because the upstream holds the
	// batch request until the test ends.
	require.Eventually(t, func() bool {
		if gitMirror.Get(repoTestMissingOID) != nil {
			return true
		}
		return g.PullFromRemote(ctx, pull) == nil && gitMirror.Get(repoTestMissingOID) != nil
	}, 30*time.Second, 50*time.Millisecond)
	rc, err := g.openBlobContent(repo, "main", "model.safetensors")
	require.NoError(t, err)
	require.NoError(t, rc.Close())

	// The header bytes never arrive, so the read must give up when its budget
	// expires during the stalled download instead of blocking metadata
	// extraction. The wait is bounded so a regression fails instead of hanging.
	readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	type headerResult struct {
		header []byte
		err    error
	}
	done := make(chan headerResult, 1)
	go func() {
		header, err := g.readSafetensorsHeader(readCtx, repo, "main", "model.safetensors")
		done <- headerResult{header: header, err: err}
	}()
	var result headerResult
	select {
	case result = <-done:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "readSafetensorsHeader did not return after its context expired")
	}

	require.ErrorIs(t, result.err, context.DeadlineExceeded)
	require.ErrorContains(t, result.err, "read safetensors header model.safetensors")
	require.Nil(t, result.header)
}

func initRepoTestRepositoryAt(t *testing.T, ctx context.Context, store *hfdstorage.Storage, name string) *repository.Repository {
	t.Helper()

	repoPath := store.ResolvePath(name)
	require.NoError(t, os.MkdirAll(filepath.Dir(repoPath), 0750))
	repo, err := repository.Init(ctx, repoPath, "main")
	require.NoError(t, err)
	evictRepoTestRepository(t, repo)
	return repo
}

// evictRepoTestRepository removes repo when the test ends. hfd caches every
// opened repository process-wide and deadlocks once more than 128 are cached,
// so evicting test repositories keeps repeated runs (-count=N) from hanging.
func evictRepoTestRepository(t *testing.T, repo *repository.Repository) {
	t.Helper()
	t.Cleanup(func() { _ = repo.Remove() })
}

func commitRepoTestFiles(t *testing.T, ctx context.Context, repo *repository.Repository, files map[string][]byte) {
	t.Helper()

	ops := make([]repository.CommitOperation, 0, len(files))
	for path, content := range files {
		ops = append(ops, repository.CommitOperation{Type: repository.CommitOperationAdd, Path: path, Content: content})
	}
	_, err := repo.CreateCommit(ctx, "main", "add files", "Test", "test@example.com", ops, "")
	require.NoError(t, err)
}

func repoTestLFSPointer(oid, size string) []byte {
	return []byte("version https://git-lfs.github.com/spec/v1\n" +
		"oid sha256:" + oid + "\n" +
		"size " + size + "\n")
}

// storeRepoTestLFSObject puts content into the local LFS store and returns the
// pointer that references it.
func storeRepoTestLFSObject(t *testing.T, store *hfdstorage.Storage, content []byte) []byte {
	t.Helper()

	sum := sha256.Sum256(content)
	oid := hex.EncodeToString(sum[:])
	require.NoError(t, hfdlfs.NewLocal(store.LFSDir()).Put(oid, bytes.NewReader(content), int64(len(content))))
	return repoTestLFSPointer(oid, strconv.Itoa(len(content)))
}

func repoTestSafetensorsHeader(t *testing.T, content []byte) []byte {
	t.Helper()

	require.GreaterOrEqual(t, len(content), 8)
	headerLength := binary.LittleEndian.Uint64(content[:8])
	require.LessOrEqual(t, 8+headerLength, uint64(len(content)))
	return content[:8+headerLength]
}
