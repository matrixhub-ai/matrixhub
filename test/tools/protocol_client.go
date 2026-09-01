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

package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EnvMatrixHubSSHHost = "MATRIXHUB_SSH_HOST"
	EnvMatrixHubSSHPort = "MATRIXHUB_SSH_PORT"
	DefaultSSHHost      = "127.0.0.1"
	DefaultSSHPort      = 2222
)

// SSHClientFixture holds an isolated key, known_hosts file, and SSH config for
// Git subprocesses.
type SSHClientFixture struct {
	Host           string
	Port           int
	PrivateKeyPath string
	PublicKey      string
	ConfigPath     string
}

func GetSSHHost() string {
	if host := os.Getenv(EnvMatrixHubSSHHost); host != "" {
		return host
	}
	return DefaultSSHHost
}

func GetSSHPort() (int, error) {
	value := os.Getenv(EnvMatrixHubSSHPort)
	if value == "" {
		return DefaultSSHPort, nil
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s %q", EnvMatrixHubSSHPort, value)
	}
	return port, nil
}

// NewSSHClientFixture generates a client key and pins the ephemeral server's
// host key in a temporary known_hosts file.
func NewSSHClientFixture(ctx context.Context, root string) (*SSHClientFixture, error) {
	port, err := GetSSHPort()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}

	privateKeyPath := filepath.Join(root, "id_ed25519")
	keygen := RunCommand(ctx, root, nil, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", privateKeyPath)
	if keygen.Err != nil {
		return nil, errorsFromCommand(keygen)
	}
	publicKeyBytes, err := os.ReadFile(privateKeyPath + ".pub")
	if err != nil {
		return nil, err
	}

	host := GetSSHHost()
	knownHostsPath := filepath.Join(root, "known_hosts")
	var scan CommandResult
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		scan = RunCommand(attemptCtx, root, nil, "ssh-keyscan", "-T", "5", "-p", strconv.Itoa(port), "-t", "rsa", host)
		cancel()
		if scan.Err == nil && strings.TrimSpace(scan.Stdout) != "" {
			break
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for SSH server: %w; last result: %s", ctx.Err(), scan.FailureMessage())
		case <-time.After(500 * time.Millisecond):
		}
	}
	if err := os.WriteFile(knownHostsPath, []byte(scan.Stdout), 0600); err != nil {
		return nil, err
	}

	configPath := filepath.Join(root, "ssh_config")
	config := fmt.Sprintf("Host *\n  Port %d\n  BatchMode yes\n  IdentitiesOnly yes\n  IdentityFile %s\n  StrictHostKeyChecking yes\n  UserKnownHostsFile %s\n  LogLevel ERROR\n",
		port, quoteSSHConfigPath(privateKeyPath), quoteSSHConfigPath(knownHostsPath))
	if err := os.WriteFile(configPath, []byte(config), 0600); err != nil {
		return nil, err
	}

	return &SSHClientFixture{
		Host:           host,
		Port:           port,
		PrivateKeyPath: privateKeyPath,
		PublicKey:      strings.TrimSpace(string(publicKeyBytes)),
		ConfigPath:     configPath,
	}, nil
}

func (f *SSHClientFixture) GitEnvironment() map[string]string {
	return map[string]string{
		"GIT_CONFIG_GLOBAL":   os.DevNull,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TERMINAL_PROMPT": "0",
		"GCM_INTERACTIVE":     "never",
		"GIT_SSH_COMMAND":     "ssh -F " + shellQuote(f.ConfigPath),
		"LC_ALL":              "C",
		"SSH_AUTH_SOCK":       "",
	}
}

func (f *SSHClientFixture) RepositoryURL(project, model string) string {
	return fmt.Sprintf("git@%s:%s/%s.git", f.Host, project, model)
}

// HFCLIEnvironment isolates token and cache state from the developer machine.
func HFCLIEnvironment(root, token string) map[string]string {
	return map[string]string{
		"HF_ENDPOINT":                  GetBaseURL(),
		"HF_TOKEN":                     token,
		"HF_HOME":                      filepath.Join(root, "hf"),
		"HF_HUB_CACHE":                 filepath.Join(root, "hf", "hub"),
		"HF_HUB_DISABLE_PROGRESS_BARS": "1",
		"HF_HUB_DISABLE_TELEMETRY":     "1",
		"HF_HUB_DISABLE_UPDATE_CHECK":  "1",
		"HF_HUB_DISABLE_XET":           "1",
		"NO_COLOR":                     "1",
		"LC_ALL":                       "C",
	}
}

func errorsFromCommand(result CommandResult) error {
	return fmt.Errorf("%s", result.FailureMessage())
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func quoteSSHConfigPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}
