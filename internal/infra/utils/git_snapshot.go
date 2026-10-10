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

package utils

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitReadSnapshot materializes exactly the captured refs and their reachable
// objects, without alternates, hard links, source hooks or later ref updates.
// The private directory lives until the upload-pack subprocess exits.
func GitReadSnapshot(ctx context.Context, source, temporaryRoot string) (string, func(), error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	original, err := gogit.PlainOpen(source)
	if err != nil {
		return "", nil, err
	}
	iterator, err := original.References()
	if err != nil {
		return "", nil, err
	}
	var refs []*plumbing.Reference
	var roots []string
	err = iterator.ForEach(func(ref *plumbing.Reference) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(refs) >= 256 {
			return errors.New("snapshot ref budget exceeded")
		}
		if strings.HasPrefix(ref.Name().String(), "refs/replace/") {
			return errors.New("replacement refs are unsupported by snapshot admission")
		}
		refs = append(refs, ref)
		if ref.Type() == plumbing.HashReference {
			// A tag pointing at a raw blob/tree has no commit-level scan report.
			hash := ref.Hash()
			for depth := 0; ; depth++ {
				if _, err := original.CommitObject(hash); err == nil {
					break
				}
				if depth >= 16 {
					return errors.New("snapshot ref does not resolve to a commit")
				}
				tag, err := original.TagObject(hash)
				if err != nil {
					return errors.New("snapshot ref does not resolve to a commit")
				}
				hash = tag.Target
			}
			roots = append(roots, ref.Hash().String())
		}
		return nil
	})
	iterator.Close()
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(temporaryRoot, 0700); err != nil {
		return "", nil, err
	}
	path, err := os.MkdirTemp(temporaryRoot, "git-read-snapshot-")
	if err != nil {
		return "", nil, err
	}
	release := func() { _ = os.RemoveAll(path) }
	success := false
	defer func() {
		if !success {
			release()
		}
	}()
	snapshot, err := gogit.PlainInit(path, true)
	if err != nil {
		return "", nil, err
	}
	if len(roots) > 0 {
		packPath := filepath.Join(path, "incoming.pack")
		if err := os.MkdirAll(filepath.Dir(packPath), 0700); err != nil {
			return "", nil, err
		}
		output, err := os.Create(packPath)
		if err != nil {
			return "", nil, err
		}
		cmd := exec.CommandContext(ctx, "git", "-c", "pack.threads=1", "-C", source, "pack-objects", "--stdout", "--revs")
		cmd.Env = append(os.Environ(), "GIT_NO_REPLACE_OBJECTS=1")
		cmd.Stdin = strings.NewReader(strings.Join(roots, "\n") + "\n")
		cmd.Stdout = &snapshotBudgetWriter{writer: output, remaining: 128 << 20}
		runErr := cmd.Run()
		closeErr := output.Close()
		if runErr != nil {
			return "", nil, fmt.Errorf("snapshot pack creation failed: %w", runErr)
		}
		if closeErr != nil {
			return "", nil, closeErr
		}
		index := exec.CommandContext(ctx, "git", "-C", path, "index-pack", "--strict", packPath)
		digest, err := index.Output()
		if err != nil {
			return "", nil, fmt.Errorf("snapshot pack validation failed: %w", err)
		}
		checksum := strings.TrimSpace(string(digest))
		decoded, err := hex.DecodeString(checksum)
		if err != nil || len(decoded) != 20 {
			return "", nil, errors.New("invalid snapshot pack checksum")
		}
		destination := filepath.Join(path, "objects", "pack", "pack-"+checksum)
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			return "", nil, err
		}
		if err := os.Rename(packPath, destination+".pack"); err != nil {
			return "", nil, err
		}
		if err := os.Rename(filepath.Join(path, "incoming.idx"), destination+".idx"); err != nil {
			return "", nil, err
		}
	}
	for _, ref := range refs {
		if err := snapshot.Storer.SetReference(ref); err != nil {
			return "", nil, err
		}
	}
	cfg, err := snapshot.Config()
	if err != nil {
		return "", nil, err
	}
	for _, option := range []string{"allowAnySHA1InWant", "allowReachableSHA1InWant", "allowTipSHA1InWant"} {
		cfg.Raw.SetOption("uploadpack", "", option, "false")
	}
	if err := snapshot.SetConfig(cfg); err != nil {
		return "", nil, err
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	success = true
	return path, release, nil
}

type snapshotBudgetWriter struct {
	writer    io.Writer
	remaining int64
}

func (w *snapshotBudgetWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errors.New("snapshot pack exceeds 128 MiB budget")
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}
