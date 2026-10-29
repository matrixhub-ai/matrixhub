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
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"github.com/stretchr/testify/require"
	cryptossh "golang.org/x/crypto/ssh"

	backend "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/ssh"
)

func generateHostKey(t *testing.T) cryptossh.Signer {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := cryptossh.NewSignerFromKey(privateKey)
	require.NoError(t, err)
	return signer
}

func runGit(t *testing.T, dir string, environment []string, arguments ...string) string {
	t.Helper()
	command := exec.CommandContext(t.Context(), "git", arguments...)
	command.Dir = dir
	command.Env = append(os.Environ(), environment...)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(arguments, " "), output)
	return string(output)
}

func startSSHServer(t *testing.T, server *backend.Server) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, listener.Close()) })
	go func() { _ = server.Serve(t.Context(), listener) }()
	return listener.Addr().String()
}

func TestSSHGitCloneAndPush(t *testing.T) {
	dataDir := t.TempDir()
	clientDir := t.TempDir()
	repositoryStorage := storage.NewStorage(storage.WithRootDir(dataDir))
	repositoryName := "org/model.git"
	repositoryPath := filepath.Join(repositoryStorage.RepositoriesDir(), repositoryName)
	require.NoError(t, os.MkdirAll(filepath.Dir(repositoryPath), 0o750))
	runGit(t, "", nil, "init", "--bare", "--initial-branch=main", repositoryPath)

	var (
		operations      []permission.Operation
		operationsMutex sync.Mutex
	)
	permissionHook := func(_ context.Context, operation permission.Operation, repo string, _ permission.Context) (bool, error) {
		if strings.TrimPrefix(repo, "/") != repositoryName {
			return false, fmt.Errorf("unexpected repository %q", repo)
		}
		operationsMutex.Lock()
		defer operationsMutex.Unlock()
		operations = append(operations, operation)
		return true, nil
	}
	preReceive := make(chan []receive.RefUpdate, 1)
	postReceive := make(chan []receive.RefUpdate, 1)
	server := backend.NewServer(
		backend.WithHostKey(generateHostKey(t)),
		backend.WithStorage(repositoryStorage),
		backend.WithPermissionHookFunc(permissionHook),
		backend.WithServices(nil),
		backend.WithMirror(nil),
		backend.WithPreReceiveHookFunc(func(_ context.Context, repo string, updates []receive.RefUpdate) (bool, error) {
			if strings.TrimPrefix(repo, "/") != repositoryName {
				return false, fmt.Errorf("unexpected repository %q", repo)
			}
			preReceive <- updates
			return true, nil
		}),
		backend.WithPostReceiveHookFunc(func(_ context.Context, repo string, updates []receive.RefUpdate) error {
			if strings.TrimPrefix(repo, "/") != repositoryName {
				return fmt.Errorf("unexpected repository %q", repo)
			}
			postReceive <- updates
			return nil
		}),
	)
	address := startSSHServer(t, server)
	port := address[strings.LastIndex(address, ":")+1:]
	environment := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p " + port,
	}
	remoteURL := "ssh://git@" + address + "/" + repositoryName
	cloneDir := filepath.Join(clientDir, "clone")
	runGit(t, "", environment, "clone", remoteURL, cloneDir)

	runGit(t, cloneDir, environment, "config", "user.email", "test@matrixhub.example")
	runGit(t, cloneDir, environment, "config", "user.name", "MatrixHub Test")
	require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("# MatrixHub\n"), 0o600))
	runGit(t, cloneDir, environment, "add", "README.md")
	runGit(t, cloneDir, environment, "commit", "-m", "initial commit")
	runGit(t, cloneDir, environment, "push", "origin", "main")
	select {
	case updates := <-preReceive:
		require.NotEmpty(t, updates)
	case <-time.After(time.Second):
		t.Fatal("pre-receive hook was not called")
	}
	select {
	case updates := <-postReceive:
		require.NotEmpty(t, updates)
	case <-time.After(time.Second):
		t.Fatal("post-receive hook was not called")
	}

	secondClone := filepath.Join(clientDir, "second-clone")
	runGit(t, "", environment, "clone", remoteURL, secondClone)
	content, err := os.ReadFile(filepath.Join(secondClone, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "# MatrixHub\n", string(content))
	operationsMutex.Lock()
	defer operationsMutex.Unlock()
	require.Contains(t, operations, permission.OperationReadRepo)
	require.Contains(t, operations, permission.OperationUpdateRepo)
}

func TestSSHAuthenticationAndLFSAuthenticate(t *testing.T) {
	dataDir := t.TempDir()
	repositoryStorage := storage.NewStorage(storage.WithRootDir(dataDir))
	repositoryName := "org/model.git"
	repositoryPath := filepath.Join(repositoryStorage.RepositoriesDir(), repositoryName)
	require.NoError(t, os.MkdirAll(filepath.Dir(repositoryPath), 0o750))
	runGit(t, "", nil, "init", "--bare", repositoryPath)

	validator := authenticate.NewSimpleBasicAuthValidator("alice", "secret")
	tokenSigner := authenticate.NewTokenSignValidator([]byte("lfs-signing-key"))
	server := backend.NewServer(
		backend.WithHostKey(generateHostKey(t)),
		backend.WithStorage(repositoryStorage),
		backend.WithBasicAuthValidator(validator),
		backend.WithTokenSignValidator(tokenSigner),
		backend.WithLFSURL("https://matrixhub.example/"),
	)
	address := startSSHServer(t, server)
	client, err := cryptossh.Dial("tcp", address, &cryptossh.ClientConfig{
		User:            "alice",
		Auth:            []cryptossh.AuthMethod{cryptossh.Password("secret")},
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(), // #nosec G106 -- local ephemeral test server
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	session, err := client.NewSession()
	require.NoError(t, err)
	output, err := session.Output("git-lfs-authenticate '/org/model.git' download")
	require.NoError(t, err)
	_ = session.Close()

	var response struct {
		Href      string            `json:"href"`
		Header    map[string]string `json:"header"`
		ExpiresIn int               `json:"expires_in"`
	}
	require.NoError(t, json.Unmarshal(output, &response))
	require.Equal(t, "https://matrixhub.example/org/model.git/info/lfs", response.Href)
	require.Equal(t, 3600, response.ExpiresIn)
	token := strings.TrimPrefix(response.Header["Authorization"], "Bearer ")
	require.NotEmpty(t, token)
	user, _, valid, err := tokenSigner.Validate(context.Background(), http.MethodPost, response.Href+"/objects/batch", token)
	require.NoError(t, err)
	require.True(t, valid)
	require.Equal(t, "alice", user)

	_, err = cryptossh.Dial("tcp", address, &cryptossh.ClientConfig{
		User:            "alice",
		Auth:            []cryptossh.AuthMethod{cryptossh.Password("wrong")},
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(), // #nosec G106 -- local ephemeral test server
	})
	require.Error(t, err)
}

func TestAuthorizedKeysCallback(t *testing.T) {
	authorized := generateHostKey(t).PublicKey()
	unauthorized := generateHostKey(t).PublicKey()
	callback := backend.AuthorizedKeysCallback([]cryptossh.PublicKey{authorized})

	permissions, err := callback(nil, authorized)
	require.NoError(t, err)
	require.NotNil(t, permissions)
	_, err = callback(nil, unauthorized)
	require.Error(t, err)
}

func TestSSHValidatorOptionsAcceptNilAndConfiguredValidators(t *testing.T) {
	basic := authenticate.BasicAuthValidatorFunc(func(context.Context, string, string) (string, bool, bool, error) {
		return "", false, false, fmt.Errorf("basic auth unavailable")
	})
	publicKey := authenticate.PublicKeyValidatorFunc(func(context.Context, string, string, []byte) (string, bool, bool, error) {
		return "", false, false, fmt.Errorf("public key auth unavailable")
	})
	server := backend.NewServer(
		backend.WithBasicAuthValidator(nil),
		backend.WithPublicKeyValidator(nil),
		backend.WithBasicAuthValidator(basic),
		backend.WithPublicKeyValidator(publicKey),
	)
	require.NotNil(t, server)
}
