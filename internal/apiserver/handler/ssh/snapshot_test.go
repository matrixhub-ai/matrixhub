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
package ssh_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/storage"
	backend "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/ssh"
	domainGit "github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
	"github.com/matrixhub-ai/matrixhub/internal/repo"
	"github.com/stretchr/testify/require"
	ssh "golang.org/x/crypto/ssh"
)

func sshSnapshotGit(t *testing.T, path string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", path}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	data, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", data)
	return strings.TrimSpace(string(data))
}
func TestGitSnapshotSSHUsesCapturedRefsAndRejectsNewHistory(t *testing.T) {
	store := storage.NewStorage(storage.WithRootDir(t.TempDir()))
	original := store.ResolvePath("tester/model")
	require.NoError(t, os.MkdirAll(original, 0700))
	sshSnapshotGit(t, original, "init", "-b", "main")
	sshSnapshotGit(t, original, "config", "user.name", "Snapshot test")
	sshSnapshotGit(t, original, "config", "user.email", "snapshot@example.invalid")
	require.NoError(t, os.WriteFile(filepath.Join(original, "config.json"), []byte("approved\n"), 0600))
	sshSnapshotGit(t, original, "add", ".")
	sshSnapshotGit(t, original, "commit", "-m", "approved")
	approved := sshSnapshotGit(t, original, "rev-parse", "HEAD")
	temporary := t.TempDir()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(private)
	require.NoError(t, err)
	var captures, releases atomic.Int32
	server := backend.NewServer(backend.WithStorage(store), backend.WithHostKey(signer), backend.WithReadSnapshot(
		func(ctx context.Context, name, path string) (string, func(), error) {
			if name != "tester/model" {
				return "", nil, domainGit.ErrReadNotAdmitted
			}
			snapshot, release, err := utils.GitReadSnapshot(ctx, path, temporary)
			if err != nil {
				return "", nil, err
			}
			revisions, err := repo.ArtifactRevisionsAtPath(ctx, snapshot, 256)
			if err != nil {
				release()
				return "", nil, err
			}
			if len(revisions) != 1 || revisions[0] != approved {
				release()
				return "", nil, domainGit.ErrReadNotAdmitted
			}
			if captures.Add(1) == 1 {
				require.NoError(t, os.WriteFile(filepath.Join(original, "config.json"), []byte("unscanned\n"), 0600))
				sshSnapshotGit(t, original, "add", ".")
				sshSnapshotGit(t, original, "commit", "-m", "unscanned")
			}
			return snapshot, func() { release(); releases.Add(1) }, nil
		}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, listener) }()
	defer func() { cancel(); listener.Close(); <-done }()
	client, err := ssh.Dial("tcp", listener.Addr().String(), &ssh.ClientConfig{User: "git", HostKeyCallback: ssh.FixedHostKey(signer.PublicKey()), Timeout: 5 * time.Second})
	require.NoError(t, err)
	defer client.Close()
	session, err := client.NewSession()
	require.NoError(t, err)
	session.Stdin = strings.NewReader("0000")
	output, err := session.CombinedOutput("git-upload-pack '/tester/model.git'")
	require.NoError(t, err, "%s", output)
	require.Contains(t, string(output), approved)
	updated := sshSnapshotGit(t, original, "rev-parse", "HEAD")
	require.NotEqual(t, approved, updated)
	require.NotContains(t, string(output), updated)
	session.Close()
	session, err = client.NewSession()
	require.NoError(t, err)
	session.Stdin = strings.NewReader("0000")
	output, err = session.CombinedOutput("git-upload-pack '/tester/model.git'")
	require.Error(t, err)
	require.Contains(t, string(output), "repository snapshot is not admitted")
	require.NotContains(t, string(output), updated)
	session.Close()
	require.Equal(t, captures.Load(), releases.Load())
	entries, err := os.ReadDir(temporary)
	require.NoError(t, err)
	require.Empty(t, entries)
}
