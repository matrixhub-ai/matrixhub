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
	backendssh "github.com/matrixhub-ai/hfd/pkg/backend/ssh"
	hfdserver "github.com/matrixhub-ai/hfd/pkg/server"
	hfdssh "github.com/matrixhub-ai/hfd/pkg/ssh"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

// Handler wraps next with hfd's HTTP chain and matrixhub authentication; call after Bind.
func (b *Backend) Handler(next http.Handler) http.Handler {
	options := b.serverOptions()
	options.Next = next
	return hfdserver.NewHTTPHandler(options)
}

func (b *Backend) serverOptions() hfdserver.Options {
	return hfdserver.Options{
		Storage:    b.storage.storage,
		XETStorage: b.storage.xetStorage,
		Mirror:     b.storage.sharedMirror,
		Authenticators: &authenticate.Authenticators{
			BasicAuth: b.auth.basicAuthValidator,
			Token:     b.auth.tokenValidator,
			TokenSign: b.auth.tokenSignValidator,
			PublicKey: b.auth.publicKeyValidator,
		},
		Authenticate:  b.authenticateHTTP,
		CASAuthorizer: b.storage.casIssuer,
		Permission:    b.permissionHookFunc,
		PreOpen:       b.preOpenHook,
		PreReceive:    b.preReceiveHook,
		PostReceive:   b.postReceiveHook,
		HostURL:       b.config.APIServer.HostURL,
	}
}

// authenticateHTTP tries sessions/HF tokens before git and signed LFS validators.
func (b *Backend) authenticateHTTP(next http.Handler) http.Handler {
	handler := authenticate.NewHandler(
		authenticate.WithNext(gitAuthChallenge(next)),
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

	return hfdserver.NewSSHServer(b.serverOptions(), hostKey)
}
