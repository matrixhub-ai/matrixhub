package repository

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	ErrRepositoryNotExists = git.ErrRepositoryNotExists
)

type Repository struct {
	repo     *git.Repository
	repoPath string
}

func IsRepository(repoPath string) bool {
	stat, err := os.Stat(filepath.Join(repoPath, "HEAD"))
	if err == nil && stat.Size() != 0 {
		return true
	}
	return false
}

func Init(repoPath string, defaultBranch string) (*Repository, error) {
	repo, err := git.PlainInitWithOptions(repoPath, &git.PlainInitOptions{
		Bare: true,
		InitOptions: git.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName(defaultBranch),
		},
	})
	if err != nil {
		return nil, err
	}
	return &Repository{
		repo:     repo,
		repoPath: repoPath,
	}, nil
}

func Open(repoPath string) (*Repository, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{})
	if err != nil {
		return nil, err
	}
	return &Repository{
		repo:     repo,
		repoPath: repoPath,
	}, nil
}

func (r *Repository) DefaultBranch() string {
	head := filepath.Join(r.repoPath, "HEAD")
	data, err := os.ReadFile(head)
	if err != nil {
		return "main"
	}

	prefix := "ref: refs/heads/"
	if len(data) > len(prefix) && string(data[:len(prefix)]) == prefix {
		return string(bytes.TrimSpace(data[len(prefix):]))
	}
	return "main"
}

func (r *Repository) Branches() ([]string, error) {
	branchesIter, err := r.repo.Branches()
	if err != nil {
		return nil, err
	}

	var branches []string
	err = branchesIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		branches = append(branches, name)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(branches) == 0 {
		return []string{r.DefaultBranch()}, nil
	}
	return branches, nil
}

func (r *Repository) Remove() error {
	return os.RemoveAll(r.repoPath)
}

func (r *Repository) IsMirror() (bool, string, error) {
	config, err := r.repo.Config()
	if err != nil {
		return false, "", err
	}
	isMirror := false
	sourceURL := ""
	if config != nil {
		if remote, ok := config.Remotes["origin"]; ok {
			if remote.Mirror {
				isMirror = true
			}
			if len(remote.URLs) > 0 {
				sourceURL = remote.URLs[0]
			}
		}
	}
	return isMirror, sourceURL, nil
}

func (r *Repository) SetMirrorRemote(sourceURL string) error {
	cfg, err := r.repo.Config()
	if err != nil {
		return err
	}
	cfg.Init.DefaultBranch = r.DefaultBranch()
	cfg.Remotes = map[string]*config.RemoteConfig{
		"origin": {
			Name:   "origin",
			URLs:   []string{sourceURL},
			Mirror: true,
			Fetch: []config.RefSpec{
				"+refs/heads/*:refs/heads/*",
				"+refs/tags/*:refs/tags/*",
			},
		},
	}

	err = r.repo.Storer.SetConfig(cfg)
	if err != nil {
		return err
	}
	return nil
}
