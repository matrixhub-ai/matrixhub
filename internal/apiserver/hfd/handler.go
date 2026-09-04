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

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
	backendcas "github.com/matrixhub-ai/hfd/pkg/backend/cas"
	backendhf "github.com/matrixhub-ai/hfd/pkg/backend/hf"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	backendlfs "github.com/matrixhub-ai/hfd/pkg/backend/lfs"
	backendssh "github.com/matrixhub-ai/hfd/pkg/backend/ssh"
	hfdssh "github.com/matrixhub-ai/hfd/pkg/ssh"
	xetserver "github.com/wzshiming/xet/server"

	"github.com/matrixhub-ai/matrixhub/internal/apiserver/middleware"
)

// Handler wraps next with the hfd protocol chain (xet CAS, HF API, LFS, git
// HTTP) and the auth layers. Call after Bind.
func (b *Backend) Handler(next http.Handler) http.Handler {
	handler := next
	storage := b.storage.storage
	xetStorage := b.storage.xetStorage
	xetAuthFn := b.storage.xetAuthFn
	sharedMirror := b.storage.sharedMirror
	permissionHookFunc := b.permissionHookFunc
	preOpenHookFunc := b.preOpenHook
	preReceiveHookFunc := b.preReceiveHook
	postReceiveHookFunc := b.postReceiveHook
	basicAuthValidator := b.auth.basicAuthValidator
	tokenValidator := b.auth.tokenValidator
	tokenSignValidator := b.auth.tokenSignValidator

	// xet CAS data plane: serves chunk uploads/downloads authenticated by
	// minted CAS tokens, falling through to the gin engine.
	handler = xetserver.NewHandler(
		xetserver.WithStorage(xetStorage),
		xetserver.WithAuthFunc(xetAuthFn),
		xetserver.WithNext(handler),
	)

	handler = backendcas.NewHandler(
		backendcas.WithMirror(sharedMirror),
		backendcas.WithPermissionHookFunc(permissionHookFunc),
		backendcas.WithNext(handler),
	)

	handler = backendhf.NewHandler(
		backendhf.WithStorage(storage),
		backendhf.WithNext(handler),
		backendhf.WithMirror(sharedMirror),
		backendhf.WithPreOpenHookFunc(preOpenHookFunc),
		backendhf.WithPermissionHookFunc(permissionHookFunc),
		backendhf.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendhf.WithPostReceiveHookFunc(postReceiveHookFunc),
	)

	handler = backendlfs.NewHandler(
		backendlfs.WithStorage(storage),
		backendlfs.WithNext(handler),
		backendlfs.WithMirror(sharedMirror),
		backendlfs.WithPermissionHookFunc(permissionHookFunc),
		backendlfs.WithTokenSignValidator(tokenSignValidator),
	)

	// No WithMirror here: the http/ssh git backends use the mirror only for
	// the mirror-only access gate (checkMirrorAccess), which would 403 plain
	// clone/push since matrixhub's source/destination funcs return false.
	// LFS/xet serving happens in the hf/lfs/cas backends, which keep the mirror.
	handler = backendhttp.NewHandler(
		backendhttp.WithStorage(storage),
		backendhttp.WithNext(handler),
		backendhttp.WithPreOpenHookFunc(preOpenHookFunc),
		backendhttp.WithPermissionHookFunc(permissionHookFunc),
		backendhttp.WithPreReceiveHookFunc(preReceiveHookFunc),
		backendhttp.WithPostReceiveHookFunc(postReceiveHookFunc),
	)

	// Authentication layers, inner→outer. Order rationale: HFAuthn (outermost)
	// authenticates sessions/HF tokens and never rejects, so hfd validators
	// skip already-authenticated requests; the CAS token recognizer accepts
	// minted xet tokens before the per-URL validators would 401 them; the hfd
	// handler validates git basic/token and SSH-signed LFS tokens (as authcodec
	// blobs); identityNormalizer (innermost) then decodes blob UserInfo into
	// the matrixhub identity ctx so no path clobbers another's result.
	handler = identityNormalizer(handler)

	handler = authenticate.NewHandler(
		authenticate.WithNext(handler),
		authenticate.WithBasicAuthValidator(basicAuthValidator),
		authenticate.WithTokenValidator(tokenValidator),
		authenticate.WithTokenSignValidator(tokenSignValidator),
	)

	handler = authenticate.TokenValidatorHandler(authenticate.NewTokenRecognizer("xet-cas", xetAuthFn), handler)

	handler = middleware.HFAuthnMiddleware(b.akRepo, b.sessionRepo, b.userRepo, b.robotRepo)(handler)

	return handler
}

// SSHServer builds the SSH protocol server, or nil when SSH is disabled.
// Call after Bind.
func (b *Backend) SSHServer() *backendssh.Server {
	if b.config.APIServer.SSHPort == 0 {
		return nil
	}

	hostKeyPath := b.config.APIServer.SSHHostKeyPath

	data, _ := os.ReadFile(hostKeyPath)
	hostKeySigner, _ := hfdssh.ParseHostKeyFile(data)
	// TODO: handle error and edge cases for host key file (e.g. file not exist, invalid format, etc.)

	sshOpts := []backendssh.Option{
		backendssh.WithStorage(b.storage.storage),
		backendssh.WithHostKey(hostKeySigner),
		// No WithMirror: it only feeds the mirror-only access gate, which would
		// deny plain clone/push (see Handler).
		backendssh.WithPreOpenHookFunc(b.preOpenHook),
		backendssh.WithPermissionHookFunc(b.permissionHookFunc),
		backendssh.WithPreReceiveHookFunc(b.preReceiveHook),
		backendssh.WithPostReceiveHookFunc(b.postReceiveHook),
		backendssh.WithLFSURL(b.config.APIServer.HostURL),
		backendssh.WithBasicAuthValidator(b.auth.basicAuthValidator),
		backendssh.WithPublicKeyValidator(b.auth.publicKeyValidator),
		backendssh.WithTokenSignValidator(b.auth.tokenSignValidator),
	}

	return backendssh.NewServer(sshOpts...)
}
