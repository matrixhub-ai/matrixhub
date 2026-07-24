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
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	pkgssh "github.com/matrixhub-ai/hfd/pkg/ssh"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	backendssh "github.com/matrixhub-ai/matrixhub/internal/apiserver/handler/ssh"
	"golang.org/x/crypto/ssh"
)

// runGitCmd runs a git command in the specified directory.
func runGitCmd(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Git command failed: git %s\nError: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func TestSSHProtocolServer(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshprotocol-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Create a temporary directory for client operations
	clientDir, err := os.MkdirTemp("", "sshprotocol-test-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	// Create a bare repository
	repoName := "test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	// Generate a host key for the SSH server
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	// Start SSH server on a random port
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().(*net.TCPAddr)
	sshURL := "ssh://git@" + addr.String() + "/" + repoName

	// Configure SSH to skip host key verification
	sshCmd := "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p " + strings.Split(addr.String(), ":")[1]
	env := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=" + sshCmd,
	}

	t.Run("CloneEmptyRepository", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "clone-empty")

		runGitCmd(t, "", env, "clone", sshURL, cloneDir)

		// Verify .git directory exists
		hfdir := filepath.Join(cloneDir, ".git")
		if _, err := os.Stat(hfdir); os.IsNotExist(err) {
			t.Errorf(".git directory not found in cloned repository")
		}
	})

	t.Run("PushToRepository", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-empty")

		runGitCmd(t, workDir, env, "config", "user.email", "test@test.com")
		runGitCmd(t, workDir, env, "config", "user.name", "Test User")

		// Create a test file
		testFile := filepath.Join(workDir, "README.md")
		if err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		runGitCmd(t, workDir, env, "add", "README.md")
		runGitCmd(t, workDir, env, "commit", "-m", "Initial commit")
		runGitCmd(t, workDir, env, "push", "-u", "origin", "master")
	})

	t.Run("CloneWithContent", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "clone-with-content")

		runGitCmd(t, "", env, "clone", sshURL, cloneDir)

		// Verify README.md exists
		readmePath := filepath.Join(cloneDir, "README.md")
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("Failed to read README.md: %v", err)
		}
		if string(content) != "# Test Repository\n" {
			t.Errorf("Unexpected content: %s", content)
		}
	})

	t.Run("FetchFromRepository", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-with-content")
		runGitCmd(t, workDir, env, "fetch", "origin")
	})

	t.Run("PushMoreCommits", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-with-content")

		runGitCmd(t, workDir, env, "config", "user.email", "test@test.com")
		runGitCmd(t, workDir, env, "config", "user.name", "Test User")

		testFile := filepath.Join(workDir, "file2.txt")
		if err := os.WriteFile(testFile, []byte("Second file\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		runGitCmd(t, workDir, env, "add", "file2.txt")
		runGitCmd(t, workDir, env, "commit", "-m", "Add second file")
		runGitCmd(t, workDir, env, "push")
	})

	t.Run("PullChanges", func(t *testing.T) {
		workDir := filepath.Join(clientDir, "clone-empty")

		runGitCmd(t, workDir, env, "config", "user.email", "test@test.com")
		runGitCmd(t, workDir, env, "config", "user.name", "Test User")

		runGitCmd(t, workDir, env, "pull")

		// Verify file2.txt exists
		file2Path := filepath.Join(workDir, "file2.txt")
		if _, err := os.Stat(file2Path); os.IsNotExist(err) {
			t.Errorf("file2.txt not found after pull")
		}
	})
}

func generateHostKey() (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating host key: %w", err)
	}
	return ssh.NewSignerFromKey(priv)
}

func generateClientKeyFile(path string) (ssh.PublicKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating client key: %w", err)
	}
	privKeyPEM, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, fmt.Errorf("marshaling private key: %w", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(privKeyPEM), 0600); err != nil {
		return nil, fmt.Errorf("writing private key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, err
	}
	return signer.PublicKey(), nil
}

func TestSSHPublicKeyAuth(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshauth-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	clientDir, err := os.MkdirTemp("", "sshauth-test-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	// Create a bare repository
	repoName := "auth-test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	// Generate host key and authorized client key
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	goodKeyFile := filepath.Join(clientDir, "id_good")
	goodPubKey, err := generateClientKeyFile(goodKeyFile)
	if err != nil {
		t.Fatalf("Failed to generate good client key: %v", err)
	}

	// Start SSH server with public key auth
	callback := backendssh.AuthorizedKeysCallback([]ssh.PublicKey{goodPubKey})
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage), backendssh.WithPublicKeyCallback(callback))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().(*net.TCPAddr)
	sshURL := "ssh://git@" + addr.String() + "/" + repoName
	port := strings.Split(addr.String(), ":")[1]

	goodSSHCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s -p %s", goodKeyFile, port)
	goodEnv := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=" + goodSSHCmd,
	}

	t.Run("CloneWithAuthorizedKey", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "clone-auth")
		runGitCmd(t, "", goodEnv, "clone", sshURL, cloneDir)

		hfdir := filepath.Join(cloneDir, ".git")
		if _, err := os.Stat(hfdir); os.IsNotExist(err) {
			t.Errorf(".git directory not found in cloned repository")
		}
	})

	t.Run("CloneWithUnauthorizedKeyFails", func(t *testing.T) {
		badKeyFile := filepath.Join(clientDir, "id_bad")
		_, err := generateClientKeyFile(badKeyFile)
		if err != nil {
			t.Fatalf("Failed to generate bad client key: %v", err)
		}

		badSSHCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s -p %s", badKeyFile, port)
		badEnv := []string{
			"GIT_TERMINAL_PROMPT=0",
			"GIT_SSH_COMMAND=" + badSSHCmd,
		}

		cloneDir := filepath.Join(clientDir, "clone-bad-auth")
		cmd := exec.CommandContext(t.Context(), "git", "clone", sshURL, cloneDir)
		cmd.Env = append(os.Environ(), badEnv...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("Expected clone to fail with unauthorized key, but it succeeded: %s", output)
		}
	})
}

func TestParseAuthorizedKeys(t *testing.T) {
	// Generate a test key
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Create an authorized_keys entry
	pubKey := signer.PublicKey()
	authorizedKey := ssh.MarshalAuthorizedKey(pubKey)

	t.Run("SingleKey", func(t *testing.T) {
		keys, err := pkgssh.ParseAuthorizedKeys(authorizedKey)
		if err != nil {
			t.Fatalf("Failed to parse authorized keys: %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("Expected 1 key, got %d", len(keys))
		}
		if string(keys[0].Marshal()) != string(pubKey.Marshal()) {
			t.Error("Parsed key does not match original")
		}
	})

	t.Run("MultipleKeys", func(t *testing.T) {
		_, priv2, _ := ed25519.GenerateKey(rand.Reader)
		signer2, _ := ssh.NewSignerFromKey(priv2)
		authorizedKey2 := ssh.MarshalAuthorizedKey(signer2.PublicKey())

		combined := append(authorizedKey, authorizedKey2...)
		keys, err := pkgssh.ParseAuthorizedKeys(combined)
		if err != nil {
			t.Fatalf("Failed to parse authorized keys: %v", err)
		}
		if len(keys) != 2 {
			t.Fatalf("Expected 2 keys, got %d", len(keys))
		}
	})

	t.Run("InvalidData", func(t *testing.T) {
		_, err := pkgssh.ParseAuthorizedKeys([]byte("invalid-key-data"))
		if err == nil {
			t.Error("Expected error for invalid key data")
		}
	})
}

func TestAuthorizedKeysCallback(t *testing.T) {
	// Generate two key pairs
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	signer1, _ := ssh.NewSignerFromKey(priv1)
	pub1 := signer1.PublicKey()

	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	signer2, _ := ssh.NewSignerFromKey(priv2)
	pub2 := signer2.PublicKey()

	// Only authorize key1
	callback := backendssh.AuthorizedKeysCallback([]ssh.PublicKey{pub1})

	t.Run("AuthorizedKeyAccepted", func(t *testing.T) {
		_, err := callback(nil, pub1)
		if err != nil {
			t.Errorf("Expected authorized key to be accepted, got: %v", err)
		}
	})

	t.Run("UnauthorizedKeyRejected", func(t *testing.T) {
		_, err := callback(nil, pub2)
		if err == nil {
			t.Error("Expected unauthorized key to be rejected")
		}
	})
}

func TestSSHLFSAuthenticate(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshlfs-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	// Create a bare repository
	repoName := "lfs-test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	// Generate host key
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	httpURL := "http://localhost:8080"

	// Start SSH server with HTTP URL configured
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage), backendssh.WithLFSURL(httpURL))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().String()

	t.Run("LFSAuthenticateDownload", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User:            "git",
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			t.Fatalf("Failed to dial SSH: %v", err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer session.Close()

		output, err := session.Output("git-lfs-authenticate '/lfs-test-repo' download")
		if err != nil {
			t.Fatalf("git-lfs-authenticate failed: %v", err)
		}

		// Parse the JSON response
		var resp struct {
			Href      string            `json:"href"`
			Header    map[string]string `json:"header"`
			ExpiresIn int               `json:"expires_in"`
		}
		if err := json.Unmarshal(output, &resp); err != nil {
			t.Fatalf("Failed to parse LFS auth response: %v\nOutput: %s", err, output)
		}

		expectedHref := "http://localhost:8080/lfs-test-repo.git/info/lfs"
		if resp.Href != expectedHref {
			t.Errorf("href = %q, want %q", resp.Href, expectedHref)
		}
		if resp.ExpiresIn != 3600 {
			t.Errorf("expires_in = %d, want 3600", resp.ExpiresIn)
		}
	})

	t.Run("LFSAuthenticateUpload", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User:            "git",
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			t.Fatalf("Failed to dial SSH: %v", err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer session.Close()

		output, err := session.Output("git-lfs-authenticate '/lfs-test-repo' upload")
		if err != nil {
			t.Fatalf("git-lfs-authenticate failed: %v", err)
		}

		var resp struct {
			Href string `json:"href"`
		}
		if err := json.Unmarshal(output, &resp); err != nil {
			t.Fatalf("Failed to parse LFS auth response: %v\nOutput: %s", err, output)
		}

		expectedHref := "http://localhost:8080/lfs-test-repo.git/info/lfs"
		if resp.Href != expectedHref {
			t.Errorf("href = %q, want %q", resp.Href, expectedHref)
		}
	})

	t.Run("LFSTransferReturnsError", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User:            "git",
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			t.Fatalf("Failed to dial SSH: %v", err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer session.Close()

		err = session.Run("git-lfs-transfer '/lfs-test-repo' download")
		if err == nil {
			t.Fatal("Expected git-lfs-transfer to fail, but it succeeded")
		}
	})
}

func TestSSHLFSAuthenticateNoHTTPURL(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshlfs-nourl-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	repoName := "lfs-test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	// Start SSH server WITHOUT HTTP URL configured
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().String()

	config := &ssh.ClientConfig{
		User:            "git",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		t.Fatalf("Failed to dial SSH: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	err = session.Run("git-lfs-authenticate '/lfs-test-repo' download")
	if err == nil {
		t.Fatal("Expected git-lfs-authenticate to fail when HTTP URL is not configured")
	}
}

func TestSSHPasswordAuth(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshpwdauth-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	clientDir, err := os.MkdirTemp("", "sshpwdauth-test-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	// Create a bare repository
	repoName := "pwd-auth-test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	// Generate host key
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	// Start SSH server with password auth
	auth := authenticate.NewSimpleBasicAuthValidator("testuser", "testpass")
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage), backendssh.WithBasicAuthValidator(auth))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().(*net.TCPAddr)

	t.Run("PasswordAuthViaDial", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User: "testuser",
			Auth: []ssh.AuthMethod{
				ssh.Password("testpass"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		client, err := ssh.Dial("tcp", addr.String(), config)
		if err != nil {
			t.Fatalf("Failed to dial SSH with password: %v", err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer session.Close()
	})

	t.Run("WrongPasswordViaDial", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User: "testuser",
			Auth: []ssh.AuthMethod{
				ssh.Password("wrongpass"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		_, err := ssh.Dial("tcp", addr.String(), config)
		if err == nil {
			t.Fatal("Expected SSH dial to fail with wrong password")
		}
	})
}

func TestSSHPublicKeyAuthViaAuthenticator(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshpkauth-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	clientDir, err := os.MkdirTemp("", "sshpkauth-test-client")
	if err != nil {
		t.Fatalf("Failed to create temp client dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(clientDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	// Create a bare repository
	repoName := "pk-auth-test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	// Generate host key and client key
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	goodKeyFile := filepath.Join(clientDir, "id_good")
	goodPubKey, err := generateClientKeyFile(goodKeyFile)
	if err != nil {
		t.Fatalf("Failed to generate good client key: %v", err)
	}

	// Start SSH server with public key auth
	auth := authenticate.NewSimplePublicKeyValidator([][]byte{goodPubKey.Marshal()})
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage), backendssh.WithPublicKeyValidator(auth))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().(*net.TCPAddr)
	sshURL := "ssh://git@" + addr.String() + "/" + repoName
	port := strings.Split(addr.String(), ":")[1]

	goodSSHCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s -p %s", goodKeyFile, port)
	goodEnv := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=" + goodSSHCmd,
	}

	t.Run("CloneWithAuthorizedKeyViaAuthenticator", func(t *testing.T) {
		cloneDir := filepath.Join(clientDir, "clone-pk-auth")
		runGitCmd(t, "", goodEnv, "clone", sshURL, cloneDir)

		hfdir := filepath.Join(cloneDir, ".git")
		if _, err := os.Stat(hfdir); os.IsNotExist(err) {
			t.Errorf(".git directory not found in cloned repository")
		}
	})

	t.Run("CloneWithUnauthorizedKeyViaAuthenticatorFails", func(t *testing.T) {
		badKeyFile := filepath.Join(clientDir, "id_bad")
		_, err := generateClientKeyFile(badKeyFile)
		if err != nil {
			t.Fatalf("Failed to generate bad client key: %v", err)
		}

		badSSHCmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s -p %s", badKeyFile, port)
		badEnv := []string{
			"GIT_TERMINAL_PROMPT=0",
			"GIT_SSH_COMMAND=" + badSSHCmd,
		}

		cloneDir := filepath.Join(clientDir, "clone-bad-pk-auth")
		cmd := exec.CommandContext(t.Context(), "git", "clone", sshURL, cloneDir)
		cmd.Env = append(os.Environ(), badEnv...)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("Expected clone to fail with unauthorized key, but it succeeded: %s", output)
		}
	})
}

func TestSSHLFSAuthenticateWithAuthenticator(t *testing.T) {
	// Create a temporary directory for repositories
	repoDir, err := os.MkdirTemp("", "sshlfs-auth-test-repos")
	if err != nil {
		t.Fatalf("Failed to create temp repo dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	storage := storage.NewStorage(storage.WithRootDir(repoDir))

	// Create a bare repository
	repoName := "lfs-auth-test-repo.git"
	repoPath := filepath.Join(storage.RepositoriesDir(), repoName)
	runGitCmd(t, "", nil, "init", "--bare", repoPath)

	// Generate host key
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	httpURL := "http://localhost:8080"
	basicAuth := authenticate.NewSimpleBasicAuthValidator("admin", "secret")
	tokenSignValidator := authenticate.NewTokenSignValidator([]byte("secret"))

	// Start SSH server with authenticator and LFS URL
	server := backendssh.NewServer(backendssh.WithHostKey(hostKey), backendssh.WithStorage(storage),
		backendssh.WithLFSURL(httpURL),
		backendssh.WithBasicAuthValidator(basicAuth),
		backendssh.WithTokenSignValidator(tokenSignValidator),
	)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	go func() {
		_ = server.Serve(t.Context(), listener)
	}()

	addr := listener.Addr().String()

	t.Run("LFSAuthenticateIncludesAuthHeaders", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User: "admin",
			Auth: []ssh.AuthMethod{
				ssh.Password("secret"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			t.Fatalf("Failed to dial SSH: %v", err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer session.Close()

		output, err := session.Output("git-lfs-authenticate '/lfs-auth-test-repo' download")
		if err != nil {
			t.Fatalf("git-lfs-authenticate failed: %v", err)
		}

		// Parse the JSON response
		var resp struct {
			Href      string            `json:"href"`
			Header    map[string]string `json:"header"`
			ExpiresIn int               `json:"expires_in"`
		}
		if err := json.Unmarshal(output, &resp); err != nil {
			t.Fatalf("Failed to parse LFS auth response: %v\nOutput: %s", err, output)
		}

		expectedHref := "http://localhost:8080/lfs-auth-test-repo.git/info/lfs"
		if resp.Href != expectedHref {
			t.Errorf("href = %q, want %q", resp.Href, expectedHref)
		}

		// Verify auth headers are included
		authHeader, ok := resp.Header["Authorization"]
		if !ok {
			t.Fatal("Expected Authorization header in LFS auth response")
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Fatalf("Expected Bearer token, got %q", authHeader)
		}
		// Validate the signed token can be verified and contains the right subject
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		batchURL := expectedHref + "/objects/batch"
		user, _, valid, _ := tokenSignValidator.Validate(context.Background(), http.MethodPost, batchURL, tokenStr)
		if !valid {
			t.Fatal("Expected signed token to be valid")
		}
		if user != "admin" {
			t.Errorf("Expected user 'admin', got %q", user)
		}
	})
}
