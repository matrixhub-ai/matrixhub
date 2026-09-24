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

package hfd

import (
	"net/http"
	"os"
	"strings"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	backendhf "github.com/matrixhub-ai/hfd/pkg/backend/hf"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	backendlfs "github.com/matrixhub-ai/hfd/pkg/backend/lfs"
	backendssh "github.com/matrixhub-ai/hfd/pkg/backend/ssh"
	hfdssh "github.com/matrixhub-ai/hfd/pkg/ssh"
	xetauth "github.com/wzshiming/xet/auth"
	xetserver "github.com/wzshiming/xet/server"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// Handler builds the HTTP protocol chain; call after Bind.
func (b *Backend) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	var handler http.Handler = backendhf.NewHandler(
		backendhf.WithStorage(b.storage.storage),
		backendhf.WithNext(next),
		backendhf.WithMirror(b.storage.sharedMirror),
		backendhf.WithPreOpenHookFunc(b.preOpenHook),
		backendhf.WithPermissionHookFunc(b.permissionHookFunc),
		backendhf.WithPreReceiveHookFunc(b.preReceiveHook),
		backendhf.WithPostReceiveHookFunc(b.postReceiveHook),
		backendhf.WithCreateRepoFunc(b.createRepo),
		backendhf.WithDeleteRepoFunc(b.deleteRepo),
		backendhf.WithListReposFunc(b.listRepos),
		backendhf.WithWhoamiFunc(b.whoami),
	)
	handler = backendlfs.NewHandler(
		backendlfs.WithStorage(b.storage.storage),
		backendlfs.WithNext(handler),
		backendlfs.WithMirror(b.storage.sharedMirror),
		backendlfs.WithPermissionHookFunc(b.permissionHookFunc),
		backendlfs.WithTokenSignValidator(b.auth.tokenSignValidator),
	)
	handler = backendhttp.NewHandler(
		backendhttp.WithStorage(b.storage.storage),
		backendhttp.WithNext(handler),
		backendhttp.WithPreOpenHookFunc(b.preOpenHook),
		backendhttp.WithPermissionHookFunc(b.permissionHookFunc),
		backendhttp.WithPreReceiveHookFunc(b.preReceiveHook),
		backendhttp.WithPostReceiveHookFunc(b.postReceiveHook),
	)
	handler = b.authenticateHTTP(handler)
	if b.storage.xetStorage == nil {
		return handler
	}
	// A nil *Issuer must fail closed rather than become a non-nil Authorizer.
	var authorizer xetauth.Authorizer = xetauth.AuthorizerFunc(func(*http.Request, xetauth.Grant) error {
		return xetauth.ErrUnauthenticated
	})
	if b.storage.casIssuer != nil {
		authorizer = b.storage.casIssuer
	}
	return xetserver.NewHandler(
		xetserver.WithStorage(b.storage.xetStorage),
		xetserver.WithAuthorizer(authorizer),
		xetserver.WithNext(handler),
	)
}

// authenticateHTTP tries sessions/HF tokens before git and signed LFS validators.
func (b *Backend) authenticateHTTP(next http.Handler) http.Handler {
	handler := authenticate.NewHandler(
		authenticate.WithNext(normalizeHTTPIdentity(gitAuthChallenge(next))),
		authenticate.WithBasicAuthValidator(b.auth.basicAuthValidator),
		authenticate.WithTokenValidator(b.auth.tokenValidator),
		authenticate.WithTokenSignValidator(b.auth.tokenSignValidator),
	)
	return middleware.HFAuthnMiddleware(b.akRepo, b.sessionRepo, b.userRepo, b.robotRepo)(handler)
}

func gitAuthChallenge(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if authenticate.IsAnonymous(authenticate.IdentityFrom(request.Context())) {
			service := request.URL.Query().Get("service")
			discovery := request.Method == http.MethodGet && strings.HasSuffix(request.URL.Path, "/info/refs") &&
				(service == "git-upload-pack" || service == "git-receive-pack")
			pack := request.Method == http.MethodPost && (strings.HasSuffix(request.URL.Path, "/git-upload-pack") ||
				strings.HasSuffix(request.URL.Path, "/git-receive-pack"))
			if discovery || pack {
				writer = &gitChallengeWriter{ResponseWriter: writer}
			}
		}
		next.ServeHTTP(writer, request)
	})
}

type gitChallengeWriter struct {
	http.ResponseWriter
}

func (writer *gitChallengeWriter) WriteHeader(statusCode int) {
	if statusCode == http.StatusForbidden {
		writer.Header().Set("WWW-Authenticate", `Basic realm="hfd"`)
		statusCode = http.StatusUnauthorized
	}
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *gitChallengeWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

// SSHServer builds the SSH protocol server, or nil when SSH is disabled.
// Call after Bind.
func (b *Backend) SSHServer() *backendssh.Server {
	if b.config.APIServer.SSHPort == 0 {
		return nil
	}

	hostKeyPath := b.config.APIServer.SSHHostKeyPath

	data, err := os.ReadFile(hostKeyPath)
	if err != nil {
		log.Fatalw("read SSH host key failed", "error", err)
	}
	hostKey, err := hfdssh.ParseHostKeyFile(data)
	if err != nil {
		log.Fatalw("parse SSH host key failed", "error", err)
	}

	return backendssh.NewServer(
		backendssh.WithStorage(b.storage.storage),
		backendssh.WithHostKey(hostKey),
		backendssh.WithPermissionHookFunc(b.permissionHookFunc),
		backendssh.WithPreOpenHookFunc(b.preOpenHook),
		backendssh.WithPreReceiveHookFunc(b.preReceiveHook),
		backendssh.WithPostReceiveHookFunc(b.postReceiveHook),
		backendssh.WithLFSURL(b.config.APIServer.HostURL),
		backendssh.WithBasicAuthValidator(b.auth.basicAuthValidator),
		backendssh.WithPublicKeyValidator(b.auth.publicKeyValidator),
		backendssh.WithTokenSignValidator(b.auth.tokenSignValidator),
	)
}
