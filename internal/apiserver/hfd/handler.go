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
	"fmt"
	"net/http"
	"os"

	backendhf "github.com/matrixhub-ai/hfd/pkg/backend/hf"
	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	backendlfs "github.com/matrixhub-ai/hfd/pkg/backend/lfs"
	backendssh "github.com/matrixhub-ai/hfd/pkg/backend/ssh"
	hfdssh "github.com/matrixhub-ai/hfd/pkg/ssh"
	xetauth "github.com/wzshiming/xet/auth"
	xetserver "github.com/wzshiming/xet/server"
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
		// LFS batches and locks only authorize: they neither create nor sync a model.
		backendlfs.WithPreOpenHookFunc(b.authorize),
		backendlfs.WithTokenSignValidator(b.auth.tokenSignValidator),
	)
	handler = backendhttp.NewHandler(
		backendhttp.WithStorage(b.storage.storage),
		backendhttp.WithNext(handler),
		backendhttp.WithPreOpenHookFunc(b.preOpenHook),
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
		// xetd's defaults: hf_xet keeps about one connection per reconstruction term.
		xetserver.WithTermSplit(16, 1<<20),
	)
}

// SSHServer builds the SSH protocol server, or nil when SSH is disabled.
// Call after Bind.
func (b *Backend) SSHServer() (*backendssh.Server, error) {
	if b.config.APIServer.SSHPort == 0 {
		return nil, nil
	}

	hostKeyPath := b.config.APIServer.SSHHostKeyPath

	data, err := os.ReadFile(hostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read SSH host key: %w", err)
	}
	hostKey, err := hfdssh.ParseHostKeyFile(data)
	if err != nil {
		return nil, fmt.Errorf("parse SSH host key %s: %w", hostKeyPath, err)
	}

	return backendssh.NewServer(
		backendssh.WithStorage(b.storage.storage),
		backendssh.WithHostKey(hostKey),
		backendssh.WithPreOpenHookFunc(b.preOpenHook),
		backendssh.WithPostReceiveHookFunc(b.postReceiveHook),
		backendssh.WithLFSURL(b.config.APIServer.HostURL),
		backendssh.WithBasicAuthValidator(b.auth.basicAuthValidator),
		backendssh.WithPublicKeyValidator(b.auth.publicKeyValidator),
		// Clients without an accepted key or password are anonymous, as over HTTP; preOpenHook admits them to public reads only.
		backendssh.WithAnonymousAccess(),
		backendssh.WithTokenSignValidator(b.auth.tokenSignValidator),
	), nil
}
