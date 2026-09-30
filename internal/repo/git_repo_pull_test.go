package repo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	backendhttp "github.com/matrixhub-ai/hfd/pkg/backend/http"
	"github.com/matrixhub-ai/hfd/pkg/mirror"
	"github.com/matrixhub-ai/hfd/pkg/repository"
	xetclient "github.com/wzshiming/xet/client"
	xetlocal "github.com/wzshiming/xet/storage/local"

	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
)

func TestPullFromRemoteAuthenticatesThroughSharedMirror(t *testing.T) {
	ctx := context.Background()
	upstream := newRepoTestStorage(t, t.TempDir())
	src, err := repository.Init(ctx, upstream.RepositoriesFS(), repository.ResolvePath("org/repo"), "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.CreateCommit(ctx, "main", "init", "Test", "test@test.com",
		[]repository.CommitOperation{{Type: repository.CommitOperationAdd, Path: "README.md", Content: []byte("# src\n")}}, ""); err != nil {
		t.Fatal(err)
	}
	hub := backendhttp.NewHandler(backendhttp.WithStorage(upstream))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if username, password, ok := r.BasicAuth(); !ok || username != "user" || password != "hf_secret" {
			// Native git only retries with URL credentials after a 401 challenge.
			w.Header().Set("WWW-Authenticate", `Basic realm="hfd"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		hub.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	storage := newRepoTestStorage(t, dir)
	xs, err := xetlocal.NewStorage(xetlocal.WithBasePath(filepath.Join(dir, "xet", "storage")))
	if err != nil {
		t.Fatal(err)
	}
	client, err := xetclient.NewClient(xetclient.WithCacheDir(filepath.Join(dir, "xet", "chunks")))
	if err != nil {
		t.Fatal(err)
	}
	shared, err := mirror.NewMirror(mirror.WithXETStorage(xs), mirror.WithXETClient(client), mirror.WithDataDir(filepath.Join(dir, "xet")), mirror.WithRepositoriesFS(storage.RepositoriesFS()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(shared.Wait)
	repo := NewGitDB(storage, shared, nil, 0)
	gitPath := repository.ResolvePath("proj/name")
	if repository.IsRepository(storage.RepositoriesFS(), gitPath) {
		t.Fatal("destination must start absent")
	}

	if err := repo.PullFromRemote(ctx, &git.GitRepository{
		RemoteRegistryURL:  srv.URL + "/",
		RemoteProjectName:  "org",
		RemoteResourceName: "repo",
		ProjectName:        "proj",
		ResourceName:       "name",
		ResourceType:       "model",
		Credential:         &git.BasicCredential{Username: "user", Password: "hf_secret"},
		LogWriter:          io.Discard,
	}); err != nil {
		t.Fatalf("PullFromRemote() error = %v", err)
	}
	revs, err := repo.ListRevisions(ctx, "model", "proj", "name")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(revs.Branches, func(r *git.Revision) bool { return r.Name == "main" }) {
		t.Fatalf("branches after pull = %+v, want main", revs.Branches)
	}
	dest, err := repository.Open(storage.RepositoriesFS(), gitPath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.(*gitRepo).readBlobBytes(dest, "refs/heads/main", "README.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# src\n" {
		t.Fatalf("README.md after pull = %q, want upstream content", got)
	}
}
