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

package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/permission"
	"github.com/matrixhub-ai/hfd/pkg/receive"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	"github.com/matrixhub-ai/hfd/pkg/storage"
	"golang.org/x/crypto/ssh"

	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
	"github.com/matrixhub-ai/matrixhub/internal/infra/utils"
)

// Signer is an alias for ssh.Signer to avoid requiring callers to import golang.org/x/crypto/ssh.
type Signer = ssh.Signer

// PublicKey is an alias for ssh.PublicKey to avoid requiring callers to import golang.org/x/crypto/ssh.
type PublicKey = ssh.PublicKey

// Server implements the SSH protocol (ssh://) server for git operations.
type Server struct {
	storage             *storage.Storage
	config              *ssh.ServerConfig
	permissionHookFunc  permission.PermissionHookFunc
	preReceiveHookFunc  receive.PreReceiveHookFunc
	postReceiveHookFunc receive.PostReceiveHookFunc
	tokenSignValidator  authenticate.TokenSignValidator
	lfsURL              string
	mirror              *mirror.Mirror
	modelService        model.IModelService
}

// Option configures the SSH server.
type Option func(*Server)

// WithPublicKeyCallback sets the public key authentication callback for the SSH server.
func WithPublicKeyCallback(fn func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)) Option {
	return func(s *Server) {
		s.config.NoClientAuth = false
		s.config.PublicKeyCallback = fn
	}
}

// WithPermissionHookFunc sets the permission hook for verifying operations.
func WithPermissionHookFunc(fn permission.PermissionHookFunc) Option {
	return func(s *Server) {
		s.permissionHookFunc = fn
	}
}

// WithServices sets the services for the SSH server.
func WithServices(modelService model.IModelService) Option {
	return func(s *Server) {
		s.modelService = modelService
	}
}

// WithPreReceiveHookFunc sets the pre-receive hook called before a git push is processed.
// If the hook returns an error, the push is rejected.
func WithPreReceiveHookFunc(fn receive.PreReceiveHookFunc) Option {
	return func(s *Server) {
		s.preReceiveHookFunc = fn
	}
}

// WithPostReceiveHookFunc sets the post-receive hook called after a git push is processed.
// Errors from this hook are logged but do not affect the push result.
func WithPostReceiveHookFunc(fn receive.PostReceiveHookFunc) Option {
	return func(s *Server) {
		s.postReceiveHookFunc = fn
	}
}

// WithLFSURL sets the base HTTP URL for the server, used by git-lfs-authenticate
// to tell LFS clients the LFS API endpoint. For example: "http://localhost:8080".
func WithLFSURL(lfsURL string) Option {
	return func(s *Server) {
		s.lfsURL = lfsURL
	}
}

// WithMirror sets the mirror to use for repository synchronization. If not provided,
// a mirror will be created when mirrorSourceFunc is set.
func WithMirror(m *mirror.Mirror) Option {
	return func(s *Server) {
		s.mirror = m
	}
}

func permissionsExtensions(user string) *ssh.Permissions {
	return &ssh.Permissions{
		Extensions: map[string]string{
			"x-user": user,
		},
	}
}

func getUserFromPermissions(perms *ssh.Permissions) string {
	if perms == nil {
		return authenticate.Anonymous
	}
	if user, ok := perms.Extensions["x-user"]; ok {
		return user
	}
	return authenticate.Anonymous
}

// WithBasicAuthValidator configures the SSH server to use the given validator
// for SSH password authentication.
func WithBasicAuthValidator(auth authenticate.BasicAuthValidator) Option {
	if auth == nil {
		return func(s *Server) {}
	}
	return func(s *Server) {
		s.config.NoClientAuth = false
		s.config.PasswordCallback = func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if user, _, ok, err := auth.Validate(context.Background(), conn.User(), string(password)); err != nil {
				slog.WarnContext(context.Background(), "password validation error", "error", err)
				return nil, fmt.Errorf("password validation error")
			} else if ok {
				return permissionsExtensions(user), nil
			}
			return nil, fmt.Errorf("invalid username or password")
		}
	}
}

// WithPublicKeyValidator configures the SSH server to use the given validator
// for SSH public key authentication.
func WithPublicKeyValidator(auth authenticate.PublicKeyValidator) Option {
	if auth == nil {
		return func(s *Server) {}
	}
	return func(s *Server) {
		s.config.NoClientAuth = false
		s.config.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if user, _, ok, err := auth.Validate(context.Background(), conn.User(), key.Type(), key.Marshal()); err != nil {
				slog.WarnContext(context.Background(), "public key validation error", "error", err)
				return nil, fmt.Errorf("public key validation error")
			} else if ok {
				return permissionsExtensions(user), nil
			}
			return nil, fmt.Errorf("invalid public key")
		}
	}
}

// WithTokenSignValidator configures the SSH server to include authentication
// headers in git-lfs-authenticate responses so that LFS clients can authenticate
// with the HTTP server.
func WithTokenSignValidator(auth authenticate.TokenSignValidator) Option {
	return func(s *Server) {
		s.tokenSignValidator = auth
	}
}

// WithStorage sets the storage backend for the server, which is used to resolve
func WithStorage(storage *storage.Storage) Option {
	return func(s *Server) {
		s.storage = storage
	}
}

// WithHostKey adds the given SSH host key to the server configuration.
func WithHostKey(hostKey ssh.Signer) Option {
	return func(s *Server) {
		s.config.AddHostKey(hostKey)
	}
}

// NewServer creates a new SSH protocol server.
func NewServer(opts ...Option) *Server {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	s := &Server{
		config: config,
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// AuthorizedKeysCallback returns a PublicKeyCallback that checks incoming keys
// against the provided list of authorized public keys.
func AuthorizedKeysCallback(authorizedKeys []ssh.PublicKey) func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	keyMap := make(map[string]bool, len(authorizedKeys))
	for _, k := range authorizedKeys {
		keyMap[string(k.Marshal())] = true
	}
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if keyMap[string(key.Marshal())] {
			return &ssh.Permissions{}, nil
		}
		return nil, fmt.Errorf("public key not found in authorized keys")
	}
}

// Serve accepts connections on the listener and handles them.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(ctx, conn)
	}
}

// ListenAndServe listens on the given address and serves SSH protocol requests.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()
	return s.Serve(ctx, listener)
}

// handleConnection handles a single SSH connection.
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	serverConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		slog.WarnContext(ctx, "ssh protocol: handshake failed", "error", err)
		return
	}
	defer func() { _ = serverConn.Close() }()

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	user := getUserFromPermissions(serverConn.Permissions)
	ctx = authenticate.WithContext(ctx, authenticate.UserInfo{User: user})

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			slog.ErrorContext(ctx, "ssh protocol: could not accept channel", "error", err)
			return
		}

		go s.handleSession(ctx, channel, requests)
	}
}

// handleSession handles an SSH session channel.
func (s *Server) handleSession(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request) {
	defer func() { _ = channel.Close() }()

	var envs []string

	for req := range requests {
		switch req.Type {
		case "env":
			var env setenvRequest
			err := ssh.Unmarshal(req.Payload, &env)
			if err != nil {
				_ = req.Reply(false, nil)
				slog.WarnContext(ctx, "ssh protocol: failed to parse env request", "error", err)
				sendExitStatus(channel, 1, "")
				return
			}
			switch env.Name {
			case "GIT_PROTOCOL":
				if repository.IsValidGitProtocol(env.Value) {
					envs = append(envs, "GIT_PROTOCOL="+env.Value)
				}
				_ = req.Reply(true, nil)
			default:
				_ = req.Reply(false, nil)
			}
		case "exec":
			var exec execMsg
			err := ssh.Unmarshal(req.Payload, &exec)
			if err != nil {
				_ = req.Reply(false, nil)
				slog.WarnContext(ctx, "ssh protocol: failed to parse env request", "error", err)
				sendExitStatus(channel, 1, "")
				return
			}

			cmd, err := parseCommand(exec.Command)
			if err != nil {
				_ = req.Reply(false, nil)
				slog.WarnContext(ctx, "ssh protocol: failed to parse exec command", "error", err)
				sendExitStatus(channel, 1, "")
				return
			}

			_ = req.Reply(true, nil)

			switch cmd.service {
			case repository.GitLFSAuthenticate:
				s.executeLFSAuthenticate(ctx, channel, cmd.repoName, cmd.operation)
			case repository.GitLFSTransfer:
				sendExitStatus(channel, 1, "git-lfs-transfer is not supported\n")
			default:
				s.executeCommand(ctx, channel, cmd.service, cmd.repoName, envs...)
			}
			return
		case "auth-agent-req@openssh.com":
			_ = req.Reply(false, nil)
		default:
			_ = req.Reply(false, nil)
			sendExitStatus(channel, 1, "unsupported request type\n")
			return
		}
	}
}

// executeCommand runs a git service command and pipes I/O through the SSH channel.
func (s *Server) executeCommand(ctx context.Context, channel ssh.Channel, service string, repoName string, env ...string) {
	repoPath := s.storage.ResolvePath(repoName)
	if repoPath == "" {
		sendExitStatus(channel, 1, "repository not found\n")
		return
	}

	isModelRepo := false
	var modelProject, modelName string
	if service == repository.GitReceivePack && s.modelService != nil {
		repoType, project, name, ok := utils.ParseFromRepoName(repoName)
		if ok && repoType == "models" {
			isModelRepo = true
			modelProject = project
			modelName = name
		}
	}

	if s.mirror != nil {
		switch service {
		case repository.GitUploadPack:
			isMirrorSrc, err := s.mirror.IsMirrorSource(ctx, repoName)
			if err != nil {
				slog.ErrorContext(ctx, "ssh protocol: failed to check mirror status", "repo", repoName, "error", err)
				sendExitStatus(channel, 1, "")
				return
			}
			if !isMirrorSrc {
				slog.WarnContext(ctx, "ssh protocol: pull from mirror repository denied", "repo", repoName)
				sendExitStatus(channel, 1, "pull from mirror repository denied")
				return
			}
		case repository.GitReceivePack:
			// Local model repositories are allowed to be created and pushed directly.
			// Mirror destination checks only apply to repositories managed as mirrors.
			if isModelRepo {
				break
			}
			isMirrorDest, err := s.mirror.IsMirrorDestination(ctx, repoName)
			if err != nil {
				slog.ErrorContext(ctx, "ssh protocol: failed to check mirror destination status", "repo", repoName, "error", err)
				sendExitStatus(channel, 1, "")
				return
			}
			if !isMirrorDest {
				slog.WarnContext(ctx, "ssh protocol: push to mirror destination repository denied", "repo", repoName)
				sendExitStatus(channel, 1, "push to mirror destination repository denied")
				return
			}
		}
	}

	if s.permissionHookFunc != nil {
		op := permission.OperationReadRepo
		if service == repository.GitReceivePack {
			op = permission.OperationUpdateRepo
		}
		if ok, err := s.permissionHookFunc(ctx, op, repoName, permission.Context{}); err != nil {
			slog.WarnContext(ctx, "ssh protocol: permission hook error", "service", service, "repo", repoName, "error", err)
			sendExitStatus(channel, 1, "")
			return
		} else if !ok {
			sendExitStatus(channel, 1, "permission denied")
			return
		}
	}

	if isModelRepo {
		if _, err := s.modelService.EnsureModel(ctx, modelProject, modelName); err != nil {
			slog.WarnContext(ctx, "ssh protocol: failed to ensure model", "repo", repoName, "error", err)
			sendExitStatus(channel, 1, "")
			return
		}
	}

	_, err := s.openRepo(ctx, repoPath, repoName, service)
	if err != nil {
		if err == repository.ErrRepositoryNotExists {
			sendExitStatus(channel, 1, "repository not found\n")
			return
		}
		slog.WarnContext(ctx, "ssh protocol: failed to open repository", "repo", repoName, "error", err)
		sendExitStatus(channel, 1, "")
		return
	}

	// For receive-pack with permission/receive hooks: use pipe-based approach
	// to intercept pkt-line commands for permission checking before the push completes.
	if service == repository.GitReceivePack && (s.preReceiveHookFunc != nil || s.postReceiveHookFunc != nil || s.mirror != nil) {
		s.executeReceivePackWithHooks(ctx, channel, service, repoName, repoPath, env...)
		return
	}

	cmd := exec.CommandContext(ctx, service, ".")
	cmd.Dir = repoPath
	cmd.Stdin = channel
	cmd.Stdout = channel
	cmd.Stderr = channel.Stderr()
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	if err := cmd.Run(); err != nil {
		slog.ErrorContext(ctx, "ssh protocol: command failed", "service", service, "error", err)
		sendExitStatus(channel, 1, "")
		return
	}

	exitCode := cmd.ProcessState.ExitCode()
	sendExitStatus(channel, uint32(exitCode), "")
}

// executeReceivePackWithHooks handles git-receive-pack using a pipe to intercept
// pkt-line ref update commands. This allows the permission hook to inspect and
// reject pushes before git-receive-pack processes the pack data.
func (s *Server) executeReceivePackWithHooks(ctx context.Context, channel ssh.Channel, service string, repoName, repoPath string, env ...string) {
	pr, pw := io.Pipe()
	defer func() { _ = pr.Close() }()

	cmd := exec.CommandContext(ctx, service, ".")
	cmd.Dir = repoPath
	cmd.Stdin = pr
	cmd.Stdout = channel
	cmd.Stderr = channel.Stderr()
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		slog.ErrorContext(ctx, "ssh protocol: command start failed", "service", service, "error", err)
		sendExitStatus(channel, 1, "")
		return
	}

	// After git-receive-pack sends the ref advertisement through stdout→channel,
	// the client sends pkt-line commands back through the channel. ParseRefUpdates
	// reads these commands and returns a replay reader for forwarding.
	updates, replay := receive.ParseRefUpdates(channel, repoPath)

	// Pre-receive hook — can reject the push before pack data is processed.
	if s.preReceiveHookFunc != nil && len(updates) > 0 {
		if ok, err := s.preReceiveHookFunc(ctx, repoName, updates); err != nil {
			slog.WarnContext(ctx, "ssh protocol: pre-receive hook denied push", "repo", repoName, "error", err)
			_ = cmd.Process.Kill()
			_ = pw.Close()
			_ = cmd.Wait()
			sendExitStatus(channel, 1, "")
			return
		} else if !ok {
			slog.WarnContext(ctx, "ssh protocol: pre-receive hook denied push", "repo", repoName)
			_ = cmd.Process.Kill()
			_ = pw.Close()
			_ = cmd.Wait()
			sendExitStatus(channel, 1, "pre-receive denied")
			return
		}
	}

	// Permission granted — forward the buffered pkt-line data and remaining
	// channel input to git-receive-pack through the pipe.
	go func() {
		defer func() { _ = pw.Close() }()
		if _, err := io.Copy(pw, replay); err != nil {
			slog.WarnContext(ctx, "ssh protocol: error forwarding data to receive-pack", "repo", repoName, "error", err)
		}
	}()

	if err := cmd.Wait(); err != nil {
		slog.ErrorContext(ctx, "ssh protocol: command failed", "service", service, "error", err)
		sendExitStatus(channel, 1, "")
		return
	}

	// Fire post-receive hook with the ref updates.
	s.afterReceivePack(ctx, repoName, updates)

	exitCode := cmd.ProcessState.ExitCode()
	sendExitStatus(channel, uint32(exitCode), "")
}

func (s *Server) afterReceivePack(ctx context.Context, repoName string, updates []receive.RefUpdate) {
	if len(updates) == 0 {
		return
	}

	if s.postReceiveHookFunc != nil {
		if hookErr := s.postReceiveHookFunc(ctx, repoName, updates); hookErr != nil {
			slog.WarnContext(ctx, "ssh protocol: post-receive hook error", "repo", repoName, "error", hookErr)
		}
	}
}

func (s *Server) openRepo(ctx context.Context, repoPath, repoName, service string) (*repository.Repository, error) {
	if s.mirror != nil && service == repository.GitUploadPack && s.modelService != nil {
		repoType, project, name, ok := utils.ParseFromRepoName(repoName)
		if ok && repoType == "models" {
			if err := s.modelService.CheckOrSyncFromRemote(ctx, project, name); err != nil {
				log.Errorf("failed to sync from remote for %s/%s: %v", project, name, err)
				return nil, err
			}
		}
	}
	return repository.Open(repoPath)
}

// lfsAuthResponse is the JSON response returned by git-lfs-authenticate.
type lfsAuthResponse struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in,omitempty"`
}

func lfsHref(httpURL, repoName string) string {
	href := strings.TrimRight(httpURL, "/") + "/" + strings.TrimPrefix(repoName, "/")
	if !strings.HasSuffix(href, ".git") {
		href += ".git"
	}
	href += "/info/lfs"
	return href
}

// executeLFSAuthenticate handles the git-lfs-authenticate command by returning
// a JSON response with the LFS API endpoint URL.
func (s *Server) executeLFSAuthenticate(ctx context.Context, channel ssh.Channel, repoName string, operation string) {
	if s.lfsURL == "" {
		sendExitStatus(channel, 1, "server not configured for host url")
		return
	}

	if operation != "download" && operation != "upload" {
		slog.ErrorContext(ctx, "ssh protocol: git-lfs-authenticate: invalid operation", "operation", operation)
		sendExitStatus(channel, 1, "invalid LFS operation")
		return
	}

	repoPath := s.storage.ResolvePath(repoName)
	if repoPath == "" {
		sendExitStatus(channel, 1, "repository not found")
		return
	}

	if s.permissionHookFunc != nil {
		op := permission.OperationReadRepo
		if operation == "upload" {
			op = permission.OperationUpdateRepo
		}
		if ok, err := s.permissionHookFunc(ctx, op, repoName, permission.Context{}); err != nil {
			slog.WarnContext(ctx, "ssh protocol: permission hook error", "operation", operation, "repo", repoName, "error", err)
			sendExitStatus(channel, 1, "")
			return
		} else if !ok {
			sendExitStatus(channel, 1, "permission denied")
			return
		}
	}

	// Build the LFS API href
	href := lfsHref(s.lfsURL, repoName)

	resp := lfsAuthResponse{
		Href:      href,
		Header:    make(map[string]string),
		ExpiresIn: 3600,
	}

	// Include authentication headers when a token signer is configured,
	// so LFS clients can authenticate with the HTTP server.
	if s.tokenSignValidator != nil {
		userInfo, _ := authenticate.GetUserInfo(ctx)
		batchURL := href + "/objects/batch"
		if token, err := s.tokenSignValidator.Sign(ctx, http.MethodPost, batchURL, userInfo.User, time.Duration(resp.ExpiresIn)*time.Second); err != nil {
			slog.WarnContext(ctx, "ssh protocol: failed to sign LFS auth token", "error", err)
		} else if token != "" {
			resp.Header["Authorization"] = "Bearer " + token
		}
	}

	data, err := json.Marshal(resp)
	if err != nil {
		slog.ErrorContext(ctx, "ssh protocol: failed to marshal LFS auth response", "error", err)
		sendExitStatus(channel, 1, "")
		return
	}

	if _, err := channel.Write(data); err != nil {
		slog.ErrorContext(ctx, "ssh protocol: failed to write LFS auth response", "error", err)
		sendExitStatus(channel, 1, "")
		return
	}

	sendExitStatus(channel, 0, "")
}

// parsedCommand holds the result of parsing an SSH exec command.
type parsedCommand struct {
	service   string
	repoName  string
	operation string // only for git-lfs-authenticate and git-lfs-transfer
}

// parseCommand parses an SSH exec command like "git-upload-pack '/repo.git'",
// "git-upload-pack /repo.git", or "git-lfs-authenticate '/repo.git' download".
func parseCommand(cmdLine string) (*parsedCommand, error) {
	parts := strings.SplitN(strings.TrimSpace(cmdLine), " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid command format: %q", cmdLine)
	}

	service := parts[0]
	rest := parts[1]

	switch service {
	case repository.GitUploadPack, repository.GitReceivePack:
		repoName := strings.Trim(rest, "'")
		return &parsedCommand{service: service, repoName: repoName}, nil

	case repository.GitLFSAuthenticate, repository.GitLFSTransfer:
		// Format: git-lfs-authenticate <path> <operation>
		// or: git-lfs-transfer <path> <operation>
		subParts := strings.SplitN(rest, " ", 2)
		if len(subParts) != 2 {
			return nil, fmt.Errorf("invalid %s format: %q", service, cmdLine)
		}
		repoName := strings.Trim(subParts[0], "'")
		operation := strings.TrimSpace(subParts[1])
		return &parsedCommand{service: service, repoName: repoName, operation: operation}, nil

	default:
		return nil, fmt.Errorf("unsupported service: %s", service)
	}
}

// sendExitStatus sends the exit status to the SSH client.
func sendExitStatus(channel ssh.Channel, status uint32, msg string) {
	if msg != "" {
		_, _ = fmt.Fprint(channel.Stderr(), msg)
	}
	_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(&exitStatusMsg{Status: status}))
}

// exitStatusMsg copy from golang.org/x/crypto/ssh.exitStatusMsg
type exitStatusMsg struct {
	Status uint32
}

// setenvRequest copy from golang.org/x/crypto/ssh.setenvRequest
type setenvRequest struct {
	Name  string
	Value string
}

// execMsg copy from golang.org/x/crypto/ssh.execMsg
type execMsg struct {
	Command string
}
