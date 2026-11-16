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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitHTTP "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitstorage "github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
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

// Pass credentials directly to the HTTP transport, without invoking a process
// or writing tokens to Git configuration.
func modelScopeGitAuth(gr *domaingit.GitRepository) transport.AuthMethod {
	if cred := gr.Credential; cred != nil && cred.Password != "" {
		username := cred.Username
		if username == "" {
			username = "oauth2"
		}
		return &gitHTTP.BasicAuth{Username: username, Password: cred.Password}
	}
	return nil
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

// The shared repository reader caches pack indexes. Decode ModelScope metadata
// packs into loose objects so cached readers can immediately see later fetches.
// Embedding only Storer deliberately hides the optional PackfileWriter fast path.
// Actual weight payloads remain in the separate LFS content store.
type modelScopeObjectStorer struct{ gitstorage.Storer }

func fetchModelScopeRefs(ctx context.Context, gr *domaingit.GitRepository, repoPath string) ([]string, error) {
	gitConfig, err := modelScopeGitConfig(ctx, gr)
	if err != nil {
		return nil, err
	}
	remoteName := gr.RemoteProjectName + "/" + gr.RemoteResourceName
	sourceURL := remoteRepositoryURL(gr, remoteName)
	remoteConfig := &config.RemoteConfig{Name: "modelscope", URLs: []string{sourceURL}}
	auth := modelScopeGitAuth(gitConfig)
	remoteRefs, err := gogit.NewRemote(memory.NewStorage(), remoteConfig).ListContext(ctx, &gogit.ListOptions{
		Auth: auth, InsecureSkipTLS: gr.Insecure,
	})
	if err != nil {
		return nil, fmt.Errorf("list modelscope revisions: %w", err)
	}
	var defaultBranch plumbing.ReferenceName
	refs := make([]string, 0)
	refSpecs := make([]config.RefSpec, 0)
	for _, ref := range remoteRefs {
		if ref.Name() == plumbing.HEAD && ref.Type() == plumbing.SymbolicReference && ref.Target().IsBranch() {
			defaultBranch = ref.Target()
		}
		if ref.Name().IsBranch() || ref.Name().IsTag() {
			if err := ref.Name().Validate(); err != nil {
				return nil, fmt.Errorf("invalid modelscope revision: %w", err)
			}
			name := ref.Name().String()
			refs = append(refs, name)
			refSpecs = append(refSpecs, config.RefSpec("+"+name+":"+name))
		}
	}
	if defaultBranch == "" || len(refs) == 0 {
		return nil, fmt.Errorf("modelscope repository has no default branch or revisions")
	}
	if err := defaultBranch.Validate(); err != nil {
		return nil, fmt.Errorf("invalid modelscope default branch: %w", err)
	}
	var repo *gogit.Repository
	if repository.IsRepository(repoPath) {
		repo, err = gogit.PlainOpen(repoPath)
	} else {
		repo, err = gogit.PlainInit(repoPath, true)
	}
	if err != nil {
		return nil, err
	}
	// Use an in-memory remote configuration so neither endpoint credentials nor
	// user-supplied options reach a process or the local repository config.
	remote := gogit.NewRemote(modelScopeObjectStorer{repo.Storer}, remoteConfig)
	if err := remote.FetchContext(ctx, &gogit.FetchOptions{
		Auth: auth, InsecureSkipTLS: gr.Insecure, RefSpecs: refSpecs, Tags: gogit.NoTags,
	}); err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return nil, fmt.Errorf("fetch modelscope revisions: %w", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, defaultBranch)); err != nil {
		return nil, err
	}
	if err := updateModelScopeHFDefault(repo, defaultBranch, refs); err != nil {
		return nil, fmt.Errorf("update ModelScope default revision compatibility: %w", err)
	}
	return refs, nil
}

// Keep HF's default revision usable for ModelScope repositories without changing
// HF's shared handlers or the actual upstream default branch. Real upstream
// branches/tags win. Only our untouched initializer or a managed alias may move.
func updateModelScopeHFDefault(repo *gogit.Repository, defaultBranch plumbing.ReferenceName, refs []string) error {
	main := plumbing.NewBranchReferenceName("main")
	marker := plumbing.ReferenceName("refs/matrixhub/modelscope-hf-main")
	tagMain := false
	for _, name := range refs {
		if name == main.String() {
			return nil
		}
		if name == plumbing.NewTagReferenceName("main").String() {
			tagMain = true
		}
	}
	current, err := repo.Storer.Reference(main)
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return err
	}
	if err == nil {
		owned, markerErr := repo.Storer.Reference(marker)
		managed := markerErr == nil && current.Type() == plumbing.HashReference && current.Hash() == owned.Hash()
		if !managed {
			commit, commitErr := repo.CommitObject(current.Hash())
			if commitErr != nil {
				return commitErr
			}
			if commit.NumParents() != 0 || strings.TrimSpace(commit.Message) != "Initial commit" || commit.Author.Name != "HuggingFace" || commit.Author.Email != "hf@users.noreply.huggingface.co" {
				return nil
			}
			tree, treeErr := commit.Tree()
			if treeErr != nil {
				return treeErr
			}
			if len(tree.Entries) != 1 || tree.Entries[0].Name != repository.GitattributesFileName {
				return nil
			}
			file, fileErr := tree.File(repository.GitattributesFileName)
			if fileErr != nil {
				return fileErr
			}
			content, readErr := file.Contents()
			if readErr != nil {
				return readErr
			}
			if !bytes.Equal([]byte(content), repository.GitattributesText) {
				return nil
			}
			backup := plumbing.ReferenceName("refs/matrixhub/modelscope-initial-main/" + current.Hash().String())
			if err := repo.Storer.SetReference(plumbing.NewHashReference(backup, current.Hash())); err != nil {
				return err
			}
		}
		if tagMain {
			return repo.Storer.RemoveReference(main)
		}
	} else if tagMain {
		return nil
	}
	target, err := repo.Storer.Reference(defaultBranch)
	if err != nil {
		return err
	}
	// Record the last managed target so local edits are never mistaken for an
	// alias on a later refresh. Upstream updates advance both refs together.
	if err := repo.Storer.SetReference(plumbing.NewHashReference(main, target.Hash())); err != nil {
		return err
	}
	return repo.Storer.SetReference(plumbing.NewHashReference(marker, target.Hash()))
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

	// Reopen after fetching so object indexes reflect new packs.
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
