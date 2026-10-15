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
package utils

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func snapshotGit(t *testing.T, path string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", path}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	result, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", result)
	return strings.TrimSpace(string(result))
}

func snapshotSource(t *testing.T) (string, string) {
	t.Helper()
	source := t.TempDir()
	snapshotGit(t, source, "init", "-b", "main")
	snapshotGit(t, source, "config", "user.name", "Snapshot test")
	snapshotGit(t, source, "config", "user.email", "snapshot@example.invalid")
	require.NoError(t, os.WriteFile(filepath.Join(source, "config.json"), []byte("{\"safe\":true}\n"), 0600))
	snapshotGit(t, source, "add", ".")
	snapshotGit(t, source, "commit", "-m", "approved content")
	return source, snapshotGit(t, source, "rev-parse", "HEAD")
}

func TestGitReadSnapshotSurvivesSourceUpdateAndRemoval(t *testing.T) {
	source, approved := snapshotSource(t)
	snapshotGit(t, source, "tag", "-a", "v1", "-m", "approved tag")
	path, release, err := GitReadSnapshot(t.Context(), source, t.TempDir())
	require.NoError(t, err)
	defer release()
	require.NoError(t, os.WriteFile(filepath.Join(source, "config.json"), []byte("unscanned update\n"), 0600))
	snapshotGit(t, source, "add", ".")
	snapshotGit(t, source, "commit", "-m", "unscanned content")
	unscanned := snapshotGit(t, source, "rev-parse", "HEAD")
	require.NotEqual(t, approved, unscanned)
	require.Equal(t, approved, snapshotGit(t, path, "rev-parse", "HEAD"))
	cmd := exec.CommandContext(t.Context(), "git", "-C", path, "cat-file", "-e", unscanned)
	require.Error(t, cmd.Run(), "unscanned object must not be present in the snapshot")
	require.NoFileExists(t, filepath.Join(path, "objects", "info", "alternates"))
	require.NoError(t, os.RemoveAll(source))
	clone := filepath.Join(t.TempDir(), "client")
	snapshotGit(t, path, "clone", "--no-local", path, clone)
	require.Equal(t, approved, snapshotGit(t, clone, "rev-parse", "HEAD"))
	data, err := os.ReadFile(filepath.Join(clone, "config.json"))
	require.NoError(t, err)
	require.Equal(t, "{\"safe\":true}\n", string(data))
	require.Equal(t, approved, snapshotGit(t, clone, "rev-parse", "v1^{}"))
	release()
	require.NoDirExists(t, path)
}

func TestGitReadSnapshotRejectsUnscannableRefs(t *testing.T) {
	for _, kind := range []string{"blob-tag", "replacement-ref", "ref-budget"} {
		t.Run(kind, func(t *testing.T) {
			source, approved := snapshotSource(t)
			switch kind {
			case "blob-tag":
				blob := snapshotGit(t, source, "rev-parse", "HEAD:config.json")
				snapshotGit(t, source, "update-ref", "refs/tags/raw-blob", blob)
			case "replacement-ref":
				snapshotGit(t, source, "update-ref", "refs/replace/"+approved, approved)
			case "ref-budget":
				for i := 0; i < 256; i++ {
					snapshotGit(t, source, "update-ref", "refs/tags/tag-"+strconv.Itoa(i), approved)
				}
			}
			temporary := t.TempDir()
			path, release, err := GitReadSnapshot(t.Context(), source, temporary)
			require.Error(t, err)
			require.Empty(t, path)
			require.Nil(t, release)
			entries, err := os.ReadDir(temporary)
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
}

func TestGitReadSnapshotCancelledAndEmpty(t *testing.T) {
	source, _ := snapshotSource(t)
	temporary := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	path, release, err := GitReadSnapshot(ctx, source, temporary)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, path)
	require.Nil(t, release)
	empty := t.TempDir()
	snapshotGit(t, empty, "init", "--bare")
	path, release, err = GitReadSnapshot(t.Context(), empty, temporary)
	require.NoError(t, err)
	defer release()
	require.Empty(t, snapshotGit(t, path, "for-each-ref"))
}

func TestGitReadSnapshotPackBudget(t *testing.T) {
	var target bytes.Buffer
	writer := &snapshotBudgetWriter{writer: &target, remaining: 3}
	n, err := writer.Write([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, 3, n)
	_, err = writer.Write([]byte("x"))
	require.ErrorContains(t, err, "budget")
	require.Equal(t, "abc", target.String())
}
