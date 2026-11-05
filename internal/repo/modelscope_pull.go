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

package repo

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/matrixhub-ai/hfd/pkg/lfs"
	"github.com/matrixhub-ai/hfd/pkg/repository"

	domaingit "github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func validateModelScopeRepository(gr *domaingit.GitRepository) error {
	endpoint, err := url.Parse(gr.RemoteRegistryURL)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("modelscope requires an HTTP(S) registry base URL")
	}
	for _, name := range []string{gr.ProjectName, gr.ResourceName, gr.RemoteProjectName, gr.RemoteResourceName} {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
			return fmt.Errorf("invalid modelscope repository path")
		}
	}
	if gr.ResourceType != "model" && gr.ResourceType != "models" && gr.ResourceType != "" {
		return fmt.Errorf("modelscope pull currently supports models only")
	}
	return nil
}

// pullModelScope fetches Git metadata and downloads LFS content using the
// ModelScope file API. ModelScope's signed LFS URLs do not reliably support the
// HEAD probes used by the generic tee cache. A job must not finish until its
// files pass the local LFS store's size and SHA256 validation.
func (g *gitRepo) pullModelScope(ctx context.Context, gr *domaingit.GitRepository) error {
	if err := validateModelScopeRepository(gr); err != nil {
		return err
	}
	repoPath := g.gitPath(gr.ResourceType, gr.ProjectName, gr.ResourceName)
	// Serialize writes to a local repository, but do not deduplicate different
	// upstreams or credentials into a misleading shared success result.
	value, _ := g.modelScopeLocks.LoadOrStore(repoPath, make(chan struct{}, 1))
	lock := value.(chan struct{})
	select {
	case lock <- struct{}{}:
		defer func() { <-lock }()
	case <-ctx.Done():
		return ctx.Err()
	}
	return g.pullModelScopeFiles(ctx, gr, repoPath)
}

func modelScopeGit(ctx context.Context, gr *domaingit.GitRepository, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	// Scope credentials to this endpoint, keep tokens out of command arguments
	// and on-disk Git config, and never change global Git/TLS configuration.
	config := []string{"credential.helper="}
	if cred := gr.Credential; cred != nil && cred.Password != "" {
		username := cred.Username
		if username == "" {
			username = "oauth2"
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + cred.Password))
		config = append(config, "http."+strings.TrimRight(gr.RemoteRegistryURL, "/")+"/.extraHeader=Authorization: Basic "+encoded)
	}
	if gr.Insecure {
		config = append(config, "http.sslVerify=false")
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_CONFIG_COUNT=%d", len(config)))
	for i, entry := range config {
		key, value, _ := strings.Cut(entry, "=")
		cmd.Env = append(cmd.Env, fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", i, key), fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", i, value))
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("modelscope git operation failed: %w", err)
	}
	return output, nil
}

type modelScopeRefState struct {
	gate      chan struct{}
	fetchedAt time.Time
	source    string
}

func (g *gitRepo) lockModelScopeRefs(ctx context.Context, repoPath string) (*modelScopeRefState, error) {
	value, _ := g.modelScopeRefStates.LoadOrStore(repoPath, &modelScopeRefState{gate: make(chan struct{}, 1)})
	state := value.(*modelScopeRefState)
	select {
	case state.gate <- struct{}{}:
		return state, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (g *gitRepo) RefreshModelScopeRefs(ctx context.Context, gr *domaingit.GitRepository) (bool, error) {
	if err := validateModelScopeRepository(gr); err != nil {
		return false, err
	}
	repoPath := g.gitPath(gr.ResourceType, gr.ProjectName, gr.ResourceName)
	state, err := g.lockModelScopeRefs(ctx, repoPath)
	if err != nil {
		return false, err
	}
	defer func() { <-state.gate }()
	source := remoteRepositoryURL(gr, gr.RemoteProjectName+"/"+gr.RemoteResourceName)
	if source == state.source && time.Since(state.fetchedAt) < time.Minute && repository.IsRepository(repoPath) {
		return false, nil
	}
	if _, err = fetchModelScopeRefs(ctx, gr, repoPath); err != nil {
		return false, err
	}
	state.fetchedAt = time.Now()
	state.source = source
	return true, nil
}

func fetchModelScopeRefs(ctx context.Context, gr *domaingit.GitRepository, repoPath string) ([]string, error) {
	gitConfig, err := modelScopeGitConfig(ctx, gr)
	if err != nil {
		return nil, err
	}
	remoteName := gr.RemoteProjectName + "/" + gr.RemoteResourceName
	sourceURL := remoteRepositoryURL(gr, remoteName)
	refsOutput, err := modelScopeGit(ctx, gitConfig, "ls-remote", "--symref", sourceURL)
	if err != nil {
		return nil, err
	}
	defaultBranch := ""
	refs := make([]string, 0)
	for _, line := range strings.Split(string(refsOutput), "\n") {
		value, ref, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		if ref == "HEAD" && strings.HasPrefix(value, "ref: refs/heads/") {
			defaultBranch = strings.TrimPrefix(value, "ref: refs/heads/")
		}
		if (strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/")) && !strings.HasSuffix(ref, "^{}") {
			refs = append(refs, ref)
		}
	}
	if defaultBranch == "" || len(refs) == 0 {
		return nil, fmt.Errorf("modelscope repository has no default branch or revisions")
	}
	if !repository.IsRepository(repoPath) {
		if _, err := modelScopeGit(ctx, gitConfig, "init", "--bare", repoPath, "--initial-branch", defaultBranch); err != nil {
			return nil, err
		}
	}
	args := []string{"-C", repoPath, "fetch", "--no-tags", sourceURL}
	for _, ref := range refs {
		args = append(args, "+"+ref+":"+ref)
	}
	if _, err := modelScopeGit(ctx, gitConfig, args...); err != nil {
		return nil, err
	}
	if _, err := modelScopeGit(ctx, gitConfig, "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/"+defaultBranch); err != nil {
		return nil, err
	}
	return refs, nil
}

func newModelScopeClient(gr *domaingit.GitRepository) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: gr.Insecure} // #nosec G402 -- explicit registry setting
	transport.ResponseHeaderTimeout = 30 * time.Second
	client := &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many modelscope download redirects")
		}
		if req.URL.Host != via[0].URL.Host || req.URL.Scheme != via[0].URL.Scheme {
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
		}
		return nil
	}}
	return client
}

func openModelScopeResponse(ctx context.Context, client *http.Client, gr *domaingit.GitRepository, revision, file, byteRange string) (*http.Response, error) {
	query := url.Values{"Revision": {revision}, "FilePath": {file}}
	endpoint := strings.TrimRight(gr.RemoteRegistryURL, "/") + "/api/v1/models/" + url.PathEscape(gr.RemoteProjectName) + "/" + url.PathEscape(gr.RemoteResourceName) + "/repo?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Encoding", "identity")
	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}
	if cred := gr.Credential; cred != nil && cred.Password != "" {
		req.Header.Set("Authorization", "Bearer "+cred.Password)
		req.AddCookie(&http.Cookie{Name: "m_session_id", Value: cred.Password})
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download modelscope file %s: %w", file, err)
	}
	if resp.StatusCode != http.StatusOK && (resp.StatusCode != http.StatusPartialContent || byteRange == "") {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("modelscope file %s returned HTTP %d", file, resp.StatusCode)
	}
	return resp, nil
}

func (g *gitRepo) OpenModelScopeFile(ctx context.Context, gr *domaingit.GitRepository, revision, file, byteRange string) (*domaingit.RemoteFile, error) {
	if err := validateModelScopeRepository(gr); err != nil {
		return nil, err
	}
	client := newModelScopeClient(gr)
	defer client.CloseIdleConnections()
	resp, err := openModelScopeResponse(ctx, client, gr, revision, file, byteRange)
	if err != nil {
		return nil, err
	}
	return &domaingit.RemoteFile{
		Body: resp.Body, ContentLength: resp.ContentLength, ContentRange: resp.Header.Get("Content-Range"),
		Partial: resp.StatusCode == http.StatusPartialContent,
	}, nil
}

func (g *gitRepo) pullModelScopeFiles(ctx context.Context, gr *domaingit.GitRepository, repoPath string) error {
	state, err := g.lockModelScopeRefs(ctx, repoPath)
	if err != nil {
		return err
	}
	refs, err := fetchModelScopeRefs(ctx, gr, repoPath)
	if err == nil {
		state.fetchedAt = time.Now()
		state.source = remoteRepositoryURL(gr, gr.RemoteProjectName+"/"+gr.RemoteResourceName)
	}
	<-state.gate
	if err != nil {
		return err
	}

	// Reopen after the external Git fetch so object indexes reflect new packs.
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return err
	}
	client := newModelScopeClient(gr)
	defer client.CloseIdleConnections()

	store := lfs.NewLocal(g.storage.LFSDir())
	for _, ref := range refs {
		hash, err := repo.ResolveRevision(plumbing.Revision(ref))
		if err != nil {
			return err
		}
		commit, err := repo.CommitObject(*hash)
		if err != nil {
			tag, tagErr := repo.TagObject(*hash)
			if tagErr != nil {
				return err
			}
			commit, err = tag.Commit()
			if err != nil {
				return err
			}
		}
		files, err := commit.Files()
		if err != nil {
			return err
		}
		err = files.ForEach(func(file *object.File) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if file.Size > lfs.MaxLFSPointerSize {
				return nil
			}
			reader, err := file.Reader()
			if err != nil {
				return err
			}
			pointer, pointerErr := lfs.DecodePointer(reader)
			_ = reader.Close()
			if pointerErr != nil || pointer == nil {
				return nil
			}
			if info, err := store.Info(pointer.OID()); err == nil && info.Size() == pointer.Size() {
				return nil
			}
			resp, err := openModelScopeResponse(ctx, client, gr, commit.Hash.String(), file.Name, "")
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()

			if err := store.Put(pointer.OID(), io.LimitReader(resp.Body, pointer.Size()+1), pointer.Size()); err != nil {
				return fmt.Errorf("validate modelscope file %s: %w", file.Name, err)
			}
			if gr.LogWriter != nil {
				_, _ = fmt.Fprintf(gr.LogWriter, "[INFO] verified %s (%d bytes)\n", file.Name, pointer.Size())
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ModelScope SDK tokens are exchanged for distinct Git credentials by login.
// Keep the original SDK token for HTTP downloads and the Git token in memory.
func modelScopeGitConfig(ctx context.Context, gr *domaingit.GitRepository) (*domaingit.GitRepository, error) {
	if gr.Credential == nil || gr.Credential.Password == "" {
		return gr, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	body, err := json.Marshal(map[string]string{"AccessToken": gr.Credential.Password})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(gr.RemoteRegistryURL, "/")+"/api/v1/login", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: gr.Insecure} // #nosec G402 -- explicit registry setting
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("modelscope authentication failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modelscope authentication returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code int
		Data struct{ GitAccessToken, Username string }
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid modelscope authentication response")
	}
	if result.Code != http.StatusOK || result.Data.GitAccessToken == "" {
		return nil, fmt.Errorf("modelscope authentication did not return a Git token")
	}
	username := result.Data.Username
	if username == "" {
		username = gr.Credential.Username
	}
	copy := *gr
	copy.Credential = &domaingit.BasicCredential{Username: username, Password: result.Data.GitAccessToken}
	return &copy, nil
}
