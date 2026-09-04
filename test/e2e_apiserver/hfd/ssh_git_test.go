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

package hfd_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	v1alpha1current_user "github.com/matrixhub-ai/matrixhub/test/client/v1alpha1/current_user"
	"github.com/matrixhub-ai/matrixhub/test/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// sshHostPort returns the SSH endpoint from MATRIXHUB_SSH_HOST/MATRIXHUB_SSH_PORT
// (defaults 127.0.0.1:2222), shared with the git_ssh suite.
func sshHostPort() (host, port string) {
	GinkgoHelper()
	p, err := tools.GetSSHPort()
	Expect(err).NotTo(HaveOccurred())
	return tools.GetSSHHost(), strconv.Itoa(p)
}

// generateSSHKeyPair writes a fresh ed25519 private key (OpenSSH PEM, 0600)
// into dir and returns its path plus the public key in authorized_keys format.
func generateSSHKeyPair(dir string) (privateKeyPath, publicKey string) {
	GinkgoHelper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	Expect(err).NotTo(HaveOccurred())

	pemBlock, err := ssh.MarshalPrivateKey(priv, "gitproto-e2e")
	Expect(err).NotTo(HaveOccurred())
	privateKeyPath = filepath.Join(dir, "id_ed25519")
	Expect(os.WriteFile(privateKeyPath, pem.EncodeToMemory(pemBlock), 0o600)).To(Succeed())

	sshPub, err := ssh.NewPublicKey(pub)
	Expect(err).NotTo(HaveOccurred())
	return privateKeyPath, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

// sshKeyDir returns a DeferCleanup-managed temp dir for key material.
// os.MkdirTemp is used instead of GinkgoT().TempDir() because the path is
// interpolated into GIT_SSH_COMMAND, which the shell splits on spaces, and
// testing tempdirs embed the spec name.
func sshKeyDir() string {
	GinkgoHelper()
	dir, err := os.MkdirTemp("", "gitproto-ssh-")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

// sshEnv builds the GIT_SSH_COMMAND environment for the given key and port:
// batch mode (never prompt), host key checking off, and only the given
// identity offered. The key path is POSIX shell-quoted because
// GIT_SSH_COMMAND is parsed by the shell and $TMPDIR may contain spaces or
// quotes.
func sshEnv(privateKeyPath, port string) []string {
	quotedKeyPath := "'" + strings.ReplaceAll(privateKeyPath, "'", `'\''`) + "'"
	return []string{fmt.Sprintf(
		"GIT_SSH_COMMAND=ssh -i %s -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes -o BatchMode=yes",
		quotedKeyPath, port,
	)}
}

// sshRepoRemote is the scp-style remote for a model repository. The ssh://URL
// form is deliberately not used: it yields an absolute "/project/name.git"
// repository path, which the server-side hooks fail to map to a project. The
// SSH username is ignored by the server — identity comes from the key
// fingerprint — "git" is used by convention.
func (f *protoFixture) sshRepoRemote(host string) string {
	return fmt.Sprintf("git@%s:%s/%s.git", host, f.project, modelName)
}

var _ = Describe("GitProto over SSH", Label("gitproto"), func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("should push and clone a model repository over SSH with a registered key", Label("GP00101", "smoke", "git", "ssh"), func() {
		host, port := sshHostPort()
		f := setupFixture(ctx)

		keyPath, publicKey := generateSSHKeyPair(sshKeyDir())
		_, _, err := f.currentUserApi.CurrentUserCreateSSHKey(ctx, v1alpha1current_user.V1alpha1CreateSshKeyRequest{
			Name:      "gitproto-e2e",
			PublicKey: publicKey,
		})
		Expect(err).NotTo(HaveOccurred(), "register SSH public key")
		DeferCleanup(func() {
			// ssh_keys rows are not cascaded by user deletion; remove them via
			// the API (best-effort) while the session is still valid.
			cleanupCtx := context.Background()
			if list, _, err := f.currentUserApi.CurrentUserListSSHKeys(cleanupCtx); err == nil {
				for _, item := range list.Items {
					_, _, _ = f.currentUserApi.CurrentUserDeleteSSHKey(cleanupCtx, item.Id)
				}
			}
		})

		work := GinkgoT().TempDir()
		env := sshEnv(keyPath, port)
		remote := f.sshRepoRemote(host)

		// Clone first: the server provisions the repository with an initial
		// commit on first authenticated contact (see GP00001).
		_, err = runGit(work, env, "clone", remote, "repo")
		Expect(err).NotTo(HaveOccurred(), "clone over SSH")
		repoDir := filepath.Join(work, "repo")

		content := fmt.Sprintf("gitproto ssh e2e %s seed=%d\n", f.project, GinkgoRandomSeed())
		gitCommitFile(repoDir, "README.md", content, "add README via SSH")

		_, err = runGit(repoDir, env, "push", "origin", "HEAD")
		Expect(err).NotTo(HaveOccurred(), "push over SSH")

		_, err = runGit(work, env, "clone", remote, "verify")
		Expect(err).NotTo(HaveOccurred(), "clone for verification")

		got, err := os.ReadFile(filepath.Join(work, "verify", "README.md"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(got)).To(Equal(content), "cloned content must match pushed content")
	})

	It("should reject SSH access with an unregistered key", Label("GP00102", "git", "ssh"), func() {
		host, port := sshHostPort()
		f := setupFixture(ctx)

		// Generate a key pair but never register it: the SSH handshake must
		// fail, so git exits non-zero before any repository access.
		keyPath, _ := generateSSHKeyPair(sshKeyDir())

		work := GinkgoT().TempDir()
		out, err := runGit(work, sshEnv(keyPath, port), "clone", f.sshRepoRemote(host), "repo")
		Expect(err).To(HaveOccurred(), "clone with unregistered key must fail")
		Expect(out).To(ContainSubstring("fatal:"), "git must report a fatal error")
		Expect(filepath.Join(work, "repo")).NotTo(BeADirectory(), "no repository may be created")
	})
})
