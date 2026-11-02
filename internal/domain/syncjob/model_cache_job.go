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

package syncjob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	// ErrNotProxyProject reports a model cache job for a project without an
	// upstream registry.
	ErrNotProxyProject = errors.New("only models of proxy projects can be cached")
	// ErrInvalidRevision reports a revision that a model cache job cannot
	// pull.
	ErrInvalidRevision = errors.New("only branches, tags and commits can be cached")
)

func (sjs *SyncJobService) CreateModelCacheJob(ctx context.Context, projectName, name, revision string) (*SyncJob, error) {
	if err := validateRevision(revision); err != nil {
		return nil, err
	}
	prj, err := sjs.projectRepo.GetProjectByName(ctx, projectName)
	if err != nil {
		return nil, err
	}
	if !prj.HasProxy() {
		return nil, ErrNotProxyProject
	}
	params, err := json.Marshal(ModelCacheParams{Revision: revision})
	if err != nil {
		return nil, err
	}
	job := &SyncJob{
		JobType:            JobTypeModelCache,
		RemoteRegistryID:   *prj.RegistryID,
		RemoteProjectName:  prj.Organization,
		RemoteResourceName: name,
		ProjectName:        prj.Name,
		ResourceName:       name,
		ResourceType:       "model",
		SyncType:           "pull",
		Status:             SyncJobStatusPending,
		Params:             params,
	}
	if err := sjs.CreateSyncJob(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// validateRevision accepts the default branch (empty), branch and tag names,
// and commit ids. MatrixHub mirrors only branches and tags, so a ref such as
// refs/pr/1 is never pulled; and a revision expression such as main~1 would
// resolve locally but not upstream.
func validateRevision(revision string) error {
	if revision == "" {
		return nil
	}
	if strings.HasPrefix(revision, "refs/") &&
		!strings.HasPrefix(revision, "refs/heads/") &&
		!strings.HasPrefix(revision, "refs/tags/") {
		return fmt.Errorf("%w: %s", ErrInvalidRevision, revision)
	}
	if !isRefName(revision) {
		return fmt.Errorf("%w: %q is not a branch, tag or commit name", ErrInvalidRevision, revision)
	}
	// git allows '@' in ref names, but the revision syntax of go-git, which
	// MatrixHub resolves revisions with, rejects it.
	if strings.Contains(revision, "@") {
		return fmt.Errorf("%w: MatrixHub cannot resolve %q, which contains '@'", ErrInvalidRevision, revision)
	}
	return nil
}

// isRefName applies the rules of git check-ref-format to a branch or tag
// name; commit ids follow them too.
func isRefName(name string) bool {
	if name == "@" || strings.HasPrefix(name, "-") || strings.HasPrefix(name, "/") ||
		strings.HasSuffix(name, "/") || strings.HasSuffix(name, ".") ||
		strings.Contains(name, "..") || strings.Contains(name, "//") || strings.Contains(name, "@{") {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}

// executeModelCacheJob pulls the model as a pull job does, and checks that
// upstream has the revision. Unlike a pull job, it never creates the project,
// and it creates the model record only after the pull, so that a model that
// upstream does not have leaves no record behind.
//
// A pull only queues the LFS files of the model for download, so the job can
// end before they are stored.
func (sjs *SyncJobService) executeModelCacheJob(ctx context.Context, syncJob *SyncJob, logWriter io.Writer) error {
	params, err := syncJob.ModelCacheParams()
	if err != nil {
		return err
	}
	prj, err := sjs.projectRepo.GetProjectByName(ctx, syncJob.ProjectName)
	if err != nil {
		return fmt.Errorf("get project(%s): %w", syncJob.ProjectName, err)
	}
	if !prj.HasProxy() {
		return fmt.Errorf("project %s: %w", syncJob.ProjectName, ErrNotProxyProject)
	}
	reg, err := sjs.registryRepo.GetRegistry(ctx, syncJob.RemoteRegistryID)
	if err != nil {
		return fmt.Errorf("get registry(id=%d): %w", syncJob.RemoteRegistryID, err)
	}

	revision := params.Revision
	if revision == "" {
		revision = "the default branch"
	}
	writeLog(logWriter, "[INFO] model cache job start, id=%d, registry=%s (id=%d), source=%s/%s -> target=%s/%s, revision=%s\n",
		syncJob.ID, reg.URL, syncJob.RemoteRegistryID, syncJob.RemoteProjectName, syncJob.RemoteResourceName,
		syncJob.ProjectName, syncJob.ResourceName, revision)

	if err = sjs.gitRepo.PullFromRemote(ctx, pullRepository(syncJob, reg, logWriter)); err != nil {
		return err
	}
	writeLog(logWriter, "[INFO] pull completed\n")
	if err = sjs.ensureLocalResource(ctx, syncJob, prj, nil); err != nil {
		// Another job, or a download through the proxy, may have created
		// the model meanwhile.
		if _, getErr := sjs.modelRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName); getErr != nil {
			return err
		}
	}
	if err = sjs.syncResourceMetadata(ctx, syncJob); err != nil {
		return err
	}

	// The pull fetched every branch and tag of upstream, with their commits,
	// so a revision that does not resolve now cannot be cached.
	if params.Revision != "" {
		if _, _, err = sjs.gitRepo.ListCommits(ctx, syncJob.ResourceType, syncJob.ProjectName, syncJob.ResourceName,
			params.Revision, 1, 1); err != nil {
			return fmt.Errorf("revision %s: %w", params.Revision, err)
		}
	}
	writeLog(logWriter, "[INFO] job completed successfully; the LFS files of the model are queued for download\n")
	return nil
}

func writeLog(w io.Writer, format string, args ...any) {
	if w != nil {
		_, _ = fmt.Fprintf(w, format, args...)
	}
}
