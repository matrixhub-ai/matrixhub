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
	"time"

	"github.com/matrixhub-ai/matrixhub/internal/domain/dataset"
	"github.com/matrixhub-ai/matrixhub/internal/domain/git"
	"github.com/matrixhub-ai/matrixhub/internal/domain/job"
	"github.com/matrixhub-ai/matrixhub/internal/domain/model"
	"github.com/matrixhub-ai/matrixhub/internal/domain/project"
	"github.com/matrixhub-ai/matrixhub/internal/domain/registry"
	"github.com/matrixhub-ai/matrixhub/internal/infra/log"
)

//go:generate go tool mockgen -source=sync_job_service.go -destination=mocks/sync_job_service_mock.go -package=mocks
type ISyncJobService interface {
	GetSyncJob(ctx context.Context, id int) (*SyncJob, error)
	CreateSyncJob(ctx context.Context, param *SyncJob) error
	CreateSyncJobs(ctx context.Context, params []*SyncJob) error
	UpdateSyncJob(ctx context.Context, param *SyncJob) error
	ExecuteSyncJob(ctx context.Context, param *SyncJob) error
	ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status SyncJobStatus, resourceType string) ([]*SyncJob, int64, error)

	ClaimPendingSyncJobs(ctx context.Context, nowMs int64) ([]job.DueJob, error)
	ExecuteSyncJobWithLog(ctx context.Context, jobID int) error

	SetOnJobDone(fn func(ctx context.Context, taskID int) error)

	// CreateModelCacheJob creates a pending job that caches a revision of a
	// model of a proxy project; see JobTypeModelCache. The model may not
	// exist locally yet. A job server runs the job.
	CreateModelCacheJob(ctx context.Context, projectName, name, revision string) (*SyncJob, error)
}

// resourceTypeDataset is the sync job resource type for datasets; every other
// value is handled as a model.
const resourceTypeDataset = "dataset"

var (
	// ErrNotProxyProject reports a model cache job for a project without an
	// upstream registry.
	ErrNotProxyProject = errors.New("only models of proxy projects can be cached")
	// ErrInvalidRevision reports a revision that a model cache job cannot
	// pull.
	ErrInvalidRevision = errors.New("only branches, tags and commits can be cached")
)

// MetadataSyncer refreshes the database metadata (README, size, labels) of a
// resource from its on-disk Git repository.
type MetadataSyncer interface {
	SyncMetadata(ctx context.Context, project, name string) error
}

type SyncJobService struct {
	syncJobRepo   ISyncJobRepo
	registryRepo  registry.IRegistryRepo
	projectRepo   project.IProjectRepo
	modelRepo     model.IModelRepo
	datasetRepo   dataset.IDatasetRepo
	gitRepo       git.IGitRepo
	modelMetadata MetadataSyncer
	datasetMeta   MetadataSyncer
	logStore      LogStore
	onJobDone     func(ctx context.Context, taskID int) error
	onJobCreated  func(jobID int) bool
}

type LogStore interface {
	Writer(jobID int) (io.WriteCloser, error)
	Reader(jobID int) (io.ReadCloser, error)
}

func NewSyncJobService(srepo ISyncJobRepo, rrepo registry.IRegistryRepo, prepo project.IProjectRepo, mrepo model.IModelRepo, dsrepo dataset.IDatasetRepo, grepo git.IGitRepo, modelMeta, datasetMeta MetadataSyncer, logStore LogStore) ISyncJobService {
	return &SyncJobService{
		syncJobRepo:   srepo,
		registryRepo:  rrepo,
		projectRepo:   prepo,
		modelRepo:     mrepo,
		datasetRepo:   dsrepo,
		gitRepo:       grepo,
		modelMetadata: modelMeta,
		datasetMeta:   datasetMeta,
		logStore:      logStore,
	}
}

// SetOnJobDone sets the callback invoked after a job finishes (used for task status aggregation).
func (sjs *SyncJobService) SetOnJobDone(fn func(ctx context.Context, taskID int) error) {
	sjs.onJobDone = fn
}

// SetOnJobCreated registers a best-effort wakeup called after a job is persisted.
func (sjs *SyncJobService) SetOnJobCreated(fn func(jobID int) bool) {
	sjs.onJobCreated = fn
}

func (sjs *SyncJobService) GetSyncJob(ctx context.Context, id int) (*SyncJob, error) {
	return sjs.syncJobRepo.GetSyncJob(ctx, id)
}

func (sjs *SyncJobService) CreateSyncJob(ctx context.Context, syncJob *SyncJob) error {
	return sjs.CreateSyncJobs(ctx, []*SyncJob{syncJob})
}

// CreateSyncJobs persists the full batch before triggering any job, so task
// status aggregation cannot observe a partially-created set of jobs.
func (sjs *SyncJobService) CreateSyncJobs(ctx context.Context, syncJobs []*SyncJob) error {
	if err := sjs.syncJobRepo.CreateSyncJobs(ctx, syncJobs); err != nil {
		return err
	}
	for _, syncJob := range syncJobs {
		if sjs.onJobCreated != nil && !sjs.onJobCreated(syncJob.ID) {
			log.Warnw("sync job trigger was not queued; periodic polling will recover it", "jobID", syncJob.ID)
		}
	}
	return nil
}

func (sjs *SyncJobService) UpdateSyncJob(ctx context.Context, syncJob *SyncJob) error {
	return sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob)
}

func (sjs *SyncJobService) ListSyncJobsByTaskID(ctx context.Context, taskID int, page, pageSize int, status SyncJobStatus, resourceType string) ([]*SyncJob, int64, error) {
	return sjs.syncJobRepo.ListSyncJobsByTaskID(ctx, taskID, page, pageSize, status, resourceType)
}

func (sjs *SyncJobService) ExecuteSyncJob(ctx context.Context, syncJob *SyncJob) error {
	defer func() {
		syncJob.CompletedTimestamp = time.Now().Unix()
		if err := sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob); err != nil {
			fmt.Printf("update sync job failed: %v\n", err)
		}
	}()

	return sjs.executeJob(ctx, syncJob, nil)
}

const claimJobBatchLimit = 32

// ClaimPendingSyncJobs selects pending jobs and CAS-claims them for execution.
func (sjs *SyncJobService) ClaimPendingSyncJobs(ctx context.Context, _ int64) ([]job.DueJob, error) {
	candidates, err := sjs.syncJobRepo.SelectPendingJobs(ctx, claimJobBatchLimit)
	if err != nil {
		return nil, err
	}
	var out []job.DueJob
	for _, j := range candidates {
		claimed, err := sjs.syncJobRepo.UpdateJobStatusCAS(ctx, j.ID, SyncJobStatusPending, SyncJobStatusRunning)
		if err != nil {
			return nil, err
		}
		if !claimed {
			continue
		}
		out = append(out, job.DueJob{
			ID:          j.ID,
			PolicyID:    0,
			TriggerType: 0,
			FireAtMs:    time.Now().UnixMilli(),
		})
	}
	return out, nil
}

// ClaimPendingSyncJob atomically claims a specific job requested by a creation trigger.
func (sjs *SyncJobService) ClaimPendingSyncJob(ctx context.Context, jobID int) (job.DueJob, bool, error) {
	syncJob, err := sjs.syncJobRepo.GetSyncJob(ctx, jobID)
	if err != nil {
		return job.DueJob{}, false, err
	}
	if syncJob.Status != SyncJobStatusPending {
		return job.DueJob{}, false, nil
	}
	claimed, err := sjs.syncJobRepo.UpdateJobStatusCAS(ctx, syncJob.ID, SyncJobStatusPending, SyncJobStatusRunning)
	if err != nil || !claimed {
		return job.DueJob{}, claimed, err
	}
	return job.DueJob{ID: syncJob.ID, FireAtMs: time.Now().UnixMilli()}, true, nil
}

// ReleaseSyncJobClaim returns a job that did not start executing to pending.
func (sjs *SyncJobService) ReleaseSyncJobClaim(ctx context.Context, jobID int) error {
	_, err := sjs.syncJobRepo.UpdateJobStatusCAS(ctx, jobID, SyncJobStatusRunning, SyncJobStatusPending)
	return err
}

// ExecuteSyncJobWithLog executes a sync job with logging and status reporting.
func (sjs *SyncJobService) ExecuteSyncJobWithLog(ctx context.Context, jobID int) error {
	syncJob, err := sjs.syncJobRepo.GetSyncJob(ctx, jobID)
	if err != nil {
		return err
	}

	var logWriter io.WriteCloser
	if sjs.logStore != nil {
		logWriter, err = sjs.logStore.Writer(jobID)
		if err != nil {
			log.Errorw("open job log failed", "jobID", jobID, "error", err)
		}
	}
	if logWriter != nil {
		defer func() { _ = logWriter.Close() }()
	}

	// 如果 job 已被外部停止（例如用户手动停止 task），跳过执行
	if syncJob.Status == SyncJobStatusStopped {
		if logWriter != nil {
			_, _ = fmt.Fprintln(logWriter, "[INFO] Job was stopped before execution started, skipping")
		}
		syncJob.CompletedTimestamp = time.Now().Unix()
		if updateErr := sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob); updateErr != nil {
			log.Errorw("update sync job status failed", "jobID", jobID, "error", updateErr)
		}
		sjs.reportJobDone(ctx, syncJob)
		return nil
	}

	err = sjs.executeJob(ctx, syncJob, logWriter)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			syncJob.Status = SyncJobStatusStopped
		} else {
			syncJob.Status = SyncJobStatusFailed
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "\n[ERROR] %v\n", err)
		}
	} else {
		syncJob.Status = SyncJobStatusSucceeded
	}
	syncJob.CompletedTimestamp = time.Now().Unix()

	if updateErr := sjs.syncJobRepo.UpdateSyncJob(ctx, syncJob); updateErr != nil {
		log.Errorw("update sync job status failed", "jobID", jobID, "error", updateErr)
	}

	sjs.reportJobDone(ctx, syncJob)

	return err
}

// reportJobDone reports the end of a job to its sync task, if it has one.
func (sjs *SyncJobService) reportJobDone(ctx context.Context, syncJob *SyncJob) {
	if sjs.onJobDone == nil || !syncJob.HasSyncTask() {
		return
	}
	if repErr := sjs.onJobDone(ctx, *syncJob.SyncTaskID); repErr != nil {
		log.Errorw("report task status failed", "taskID", *syncJob.SyncTaskID, "error", repErr)
	}
}

// executeJob runs a job as its type says.
func (sjs *SyncJobService) executeJob(ctx context.Context, syncJob *SyncJob, logWriter io.Writer) error {
	switch syncJob.JobType {
	case JobTypeSync, "":
		return sjs.executeGitJob(ctx, syncJob, logWriter)
	case JobTypeModelCache:
		return sjs.executeModelCacheJob(ctx, syncJob, logWriter)
	default:
		return fmt.Errorf("unknown job type: %s", syncJob.JobType)
	}
}

func (sjs *SyncJobService) executeGitJob(ctx context.Context, syncJob *SyncJob, logWriter io.Writer) error {
	reg, err := sjs.registryRepo.GetRegistry(ctx, syncJob.RemoteRegistryID)
	if err != nil {
		return fmt.Errorf("get registry(id=%d): %w", syncJob.RemoteRegistryID, err)
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] job start, id=%d, registry=%s (id=%d), resource_type=%s\n",
			syncJob.ID, reg.URL, syncJob.RemoteRegistryID, syncJob.ResourceType)
		var sourceProject, sourceResource, targetProject, targetResource string
		if syncJob.SyncType == "push" {
			sourceProject, sourceResource = syncJob.ProjectName, syncJob.ResourceName
			targetProject, targetResource = syncJob.RemoteProjectName, syncJob.RemoteResourceName
		} else {
			sourceProject, sourceResource = syncJob.RemoteProjectName, syncJob.RemoteResourceName
			targetProject, targetResource = syncJob.ProjectName, syncJob.ResourceName
		}
		_, _ = fmt.Fprintf(logWriter, "[INFO] source=%s/%s -> target=%s/%s, sync_type=%s\n",
			sourceProject, sourceResource, targetProject, targetResource, syncJob.SyncType)
	}

	switch syncJob.SyncType {
	case "pull":
		return sjs.executePullJob(ctx, syncJob, reg, logWriter)
	case "push":
		return sjs.executePushJob(ctx, syncJob, reg, logWriter)
	default:
		return fmt.Errorf("unknown sync type: %s", syncJob.SyncType)
	}
}

func (sjs *SyncJobService) executePullJob(ctx context.Context, syncJob *SyncJob, reg *registry.Registry, logWriter io.Writer) error {
	prj, err := sjs.projectRepo.GetProjectByName(ctx, syncJob.ProjectName)
	if err != nil {
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] project %s not found, creating new project\n", syncJob.ProjectName)
		}
		prj = &project.Project{
			Name: syncJob.ProjectName,
			Type: project.ProjectTypePublic,
		}
		prj, err = sjs.projectRepo.CreateProject(ctx, prj)
		if err != nil {
			return fmt.Errorf("create project(%s): %w", syncJob.ProjectName, err)
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] created project %s (id=%d)\n", prj.Name, prj.ID)
		}
	}
	if err = sjs.ensureLocalResource(ctx, syncJob, prj, logWriter); err != nil {
		return err
	}

	if err = sjs.gitRepo.PullFromRemote(ctx, pullRepository(syncJob, reg, logWriter)); err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] pull completed\n")
	}

	// Git content alone is not enough: README, size and labels live in the
	// database and are only refreshed by an explicit metadata sync.
	if err = sjs.syncResourceMetadata(ctx, syncJob); err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] metadata refreshed\n")
	}

	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] job completed successfully\n")
	}
	return nil
}

// pullRepository describes the pull of a job's resource from its registry.
func pullRepository(syncJob *SyncJob, reg *registry.Registry, logWriter io.Writer) *git.GitRepository {
	gr := &git.GitRepository{
		RemoteRegistryURL:  reg.URL,
		RemoteProjectName:  syncJob.RemoteProjectName,
		RemoteResourceName: syncJob.RemoteResourceName,
		ProjectName:        syncJob.ProjectName,
		ResourceName:       syncJob.ResourceName,
		ResourceType:       syncJob.ResourceType,
		LogWriter:          logWriter,
	}
	if reg.GetCredential() != nil {
		if bc := registry.AsBasic(reg.GetCredential()); bc != nil {
			gr.Credential = &git.BasicCredential{
				Username: bc.Username,
				Password: bc.Password,
			}
		}
	}
	return gr
}

// ensureLocalResource creates the local database record for the synced
// resource when it does not exist yet.
func (sjs *SyncJobService) ensureLocalResource(ctx context.Context, syncJob *SyncJob, prj *project.Project, logWriter io.Writer) error {
	if syncJob.ResourceType == resourceTypeDataset {
		ds, _ := sjs.datasetRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
		if ds != nil {
			if logWriter != nil {
				_, _ = fmt.Fprintf(logWriter, "[INFO] dataset exists, pulling from remote\n")
			}
			return nil
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] dataset not found, creating dataset record and cloning from remote\n")
		}
		created, err := sjs.datasetRepo.Create(ctx, &dataset.Dataset{
			Name:        syncJob.ResourceName,
			ProjectID:   prj.ID,
			ProjectName: syncJob.ProjectName,
		})
		if err != nil {
			return err
		}
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] created dataset record (id=%d)\n", created.ID)
		}
		return nil
	}

	mod, _ := sjs.modelRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
	if mod != nil {
		if logWriter != nil {
			_, _ = fmt.Fprintf(logWriter, "[INFO] model exists, pulling from remote\n")
		}
		return nil
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] model not found, creating model record and cloning from remote\n")
	}
	created, err := sjs.modelRepo.Create(ctx, &model.Model{
		Name:        syncJob.ResourceName,
		ProjectID:   prj.ID,
		ProjectName: syncJob.ProjectName,
	})
	if err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] created model record (id=%d)\n", created.ID)
	}
	return nil
}

// syncResourceMetadata refreshes the database metadata of the synced resource
// from the freshly pulled Git repository.
func (sjs *SyncJobService) syncResourceMetadata(ctx context.Context, syncJob *SyncJob) error {
	syncer := sjs.modelMetadata
	if syncJob.ResourceType == resourceTypeDataset {
		syncer = sjs.datasetMeta
	}
	if syncer == nil {
		return nil
	}
	if err := syncer.SyncMetadata(ctx, syncJob.ProjectName, syncJob.ResourceName); err != nil {
		return fmt.Errorf("sync metadata for %s %s/%s: %w",
			syncJob.ResourceType, syncJob.ProjectName, syncJob.ResourceName, err)
	}
	return nil
}

func (sjs *SyncJobService) executePushJob(ctx context.Context, syncJob *SyncJob, reg *registry.Registry, logWriter io.Writer) error {
	if syncJob.ResourceType == resourceTypeDataset {
		ds, err := sjs.datasetRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
		if err != nil || ds == nil {
			return fmt.Errorf("local dataset %s/%s not found: %w", syncJob.ProjectName, syncJob.ResourceName, err)
		}
	} else {
		mod, err := sjs.modelRepo.GetByProjectAndName(ctx, syncJob.ProjectName, syncJob.ResourceName)
		if err != nil || mod == nil {
			return fmt.Errorf("local model %s/%s not found: %w", syncJob.ProjectName, syncJob.ResourceName, err)
		}
	}

	gr := &git.GitRepository{
		RemoteRegistryURL:  reg.URL,
		RemoteProjectName:  syncJob.RemoteProjectName,
		RemoteResourceName: syncJob.RemoteResourceName,
		ProjectName:        syncJob.ProjectName,
		ResourceName:       syncJob.ResourceName,
		ResourceType:       syncJob.ResourceType,
		LogWriter:          logWriter,
	}
	if reg.GetCredential() != nil {
		if bc := registry.AsBasic(reg.GetCredential()); bc != nil {
			gr.Credential = &git.BasicCredential{
				Username: bc.Username,
				Password: bc.Password,
			}
		}
	}

	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] pushing to remote registry\n")
	}
	if err := sjs.gitRepo.PushToRemote(ctx, gr); err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] push completed\n")
	}
	return nil
}

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

	if logWriter != nil {
		revision := params.Revision
		if revision == "" {
			revision = "the default branch"
		}
		_, _ = fmt.Fprintf(logWriter, "[INFO] model cache job start, id=%d, registry=%s (id=%d), source=%s/%s -> target=%s/%s, revision=%s\n",
			syncJob.ID, reg.URL, syncJob.RemoteRegistryID, syncJob.RemoteProjectName, syncJob.RemoteResourceName,
			syncJob.ProjectName, syncJob.ResourceName, revision)
	}

	if err = sjs.gitRepo.PullFromRemote(ctx, pullRepository(syncJob, reg, logWriter)); err != nil {
		return err
	}
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] pull completed\n")
	}
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
	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] metadata refreshed\n")
	}

	// The pull fetched every branch and tag of upstream, with their commits,
	// so a revision that does not resolve now cannot be cached.
	if params.Revision != "" {
		if _, _, err = sjs.gitRepo.ListCommits(ctx, syncJob.ResourceType, syncJob.ProjectName, syncJob.ResourceName,
			params.Revision, 1, 1); err != nil {
			return fmt.Errorf("revision %s: %w", params.Revision, err)
		}
	}

	if logWriter != nil {
		_, _ = fmt.Fprintf(logWriter, "[INFO] job completed successfully; the LFS files of the model are queued for download\n")
	}
	return nil
}
