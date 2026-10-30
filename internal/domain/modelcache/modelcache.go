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

// Package modelcache pulls proxy-project models from their upstream registry
// into MatrixHub storage on the server, without a client downloading them,
// and reports per-revision cache status.
package modelcache

import (
	"context"
	"errors"
	"time"
)

// Selection narrows the files of a revision to cache; the zero value selects
// every file.
type Selection struct {
	// AllowPatterns and IgnorePatterns use huggingface_hub fnmatch semantics.
	AllowPatterns  []string
	IgnorePatterns []string
}

// Request asks to cache a model revision.
type Request struct {
	Project string
	Name    string
	// Revision is a branch, tag or commit; empty means the default branch.
	Revision  string
	Selection Selection
	// DryRun only lists the upstream files and the local cache state.
	DryRun bool
}

// FileState is the cache state of one file.
type FileState int

const (
	FileMissing FileState = iota
	FileQueued
	FileDownloading
	FileCached
	FileFailed
)

// TaskState is the state of one run of a cache task.
type TaskState int

const (
	TaskRunning TaskState = iota + 1
	TaskSucceeded
	TaskFailed
)

// CacheState tells how much of a selection is in MatrixHub storage. Files
// stored in git are present once the commit is synced, so LFS objects decide
// between missing and partial.
type CacheState int

const (
	// CacheStateMissing means no LFS object of the selection is cached.
	CacheStateMissing CacheState = iota + 1
	CacheStatePartial
	// CacheStateCached means every selected file is cached.
	CacheStateCached
)

// FileStatus is the cache status of one selected file.
type FileStatus struct {
	Path            string
	Size            int64
	IsLFS           bool
	OID             string
	State           FileState
	DownloadedBytes int64
	Error           string
}

// Totals aggregates the file states of a selection.
type Totals struct {
	Files            int64
	Bytes            int64
	CachedFiles      int64
	CachedBytes      int64
	DownloadingFiles int64
	DownloadingBytes int64
	QueuedFiles      int64
	MissingFiles     int64
	FailedFiles      int64
}

// TaskStatus describes one run of a cache task.
type TaskStatus struct {
	// ID is opaque and unique per run: retrying a failed pull starts a new
	// run with a new ID.
	ID         string
	State      TaskState
	StartedAt  time.Time
	FinishedAt *time.Time
	Error      string
}

// CacheStatus is the cache status of a selection of a model revision.
type CacheStatus struct {
	Project           string
	Name              string
	RequestedRevision string
	// Revision is the branch, tag or commit that was listed:
	// RequestedRevision, or the name of the default branch when none was
	// requested. A dry run leaves it empty when the registry does not name
	// its default branch.
	Revision string
	Commit   string
	State    CacheState
	Totals   Totals
	Files    []FileStatus
	// Task is the latest run known for the selection; nil when there is
	// none.
	Task     *TaskStatus
	Warnings []string
}

var (
	ErrInvalidSelection     = errors.New("invalid file selection")
	ErrNotProxyProject      = errors.New("project is not a proxy project")
	ErrModelNotFound        = errors.New("model not found")
	ErrRevisionNotFound     = errors.New("model or revision not found")
	ErrUnsupportedRevision  = errors.New("only branches, tags and commits can be cached")
	ErrRevisionNotMirrored  = errors.New("revision is not on a branch or tag that MatrixHub mirrors")
	ErrTooManyFiles         = errors.New("revision has too many files to list")
	ErrUpstreamSync         = errors.New("upstream sync failed")
	ErrUpstreamAccessDenied = errors.New("upstream registry denied access")
	ErrUpstreamUnavailable  = errors.New("upstream registry unavailable")
	ErrTooManyTasks         = errors.New("too many running cache tasks")
	ErrClosed               = errors.New("model cache service is closed")
)

// IModelCacheService caches proxy-project models from upstream on the server.
//
//go:generate go tool mockgen -source=modelcache.go -destination=mocks/modelcache_mock.go -package=mocks
type IModelCacheService interface {
	// CacheModel starts, or attaches to, the task that pulls the selected
	// files of a revision into local storage. With DryRun it is read-only.
	CacheModel(ctx context.Context, req *Request) (*CacheStatus, error)
	// GetModelCache reports the local cache status of every file of a
	// revision and the latest task of that commit. It never contacts
	// upstream and never starts downloads.
	GetModelCache(ctx context.Context, project, name, revision string) (*CacheStatus, error)
	// Close stops running tasks and waits for them to exit.
	Close()
}

// Options tunes the in-memory task executor.
type Options struct {
	// PerTaskConcurrency bounds concurrent object downloads of one task.
	PerTaskConcurrency int
	// GlobalConcurrency bounds concurrent object downloads of all tasks. An
	// object holds its slot only while it downloads, not while it waits to be
	// stored, for a re-sync or for its next attempt.
	GlobalConcurrency int
	// MaxRunningTasks bounds the number of running tasks.
	MaxRunningTasks int
	// MaxAttempts bounds download attempts per object.
	MaxAttempts int
	// RetryBackoff is the delay before the second attempt; it doubles after.
	RetryBackoff time.Duration
	// PollInterval is how often a download's progress is checked.
	PollInterval time.Duration
	// StallTimeout fails an attempt whose download makes no progress for
	// this long. Zero means the default; a negative value disables it.
	StallTimeout time.Duration
	// RequeueMinInterval rate-limits the forced re-syncs of one model, which
	// re-queue its objects: a re-sync within the interval reports the last
	// one's result.
	RequeueMinInterval time.Duration
	// RefreshTimeout bounds how long a pull waits for one forced re-sync of
	// a model. The mirror cannot always stop git at once, so a re-sync that
	// outlives it finishes in the background.
	RefreshTimeout time.Duration
	// StoreTimeout fails an object whose download finished but is not
	// stored this long after, because a client still reads it.
	StoreTimeout time.Duration
	// FinishedTaskTTL is how long finished tasks are kept for status reports.
	FinishedTaskTTL time.Duration
	// Now returns the current time; tests may replace it.
	Now func() time.Time
}

func (o Options) withDefaults() Options {
	if o.PerTaskConcurrency <= 0 {
		o.PerTaskConcurrency = 2
	}
	if o.GlobalConcurrency <= 0 {
		o.GlobalConcurrency = 4
	}
	if o.MaxRunningTasks <= 0 {
		o.MaxRunningTasks = 8
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 3
	}
	if o.RetryBackoff <= 0 {
		o.RetryBackoff = 10 * time.Second
	}
	if o.PollInterval <= 0 {
		o.PollInterval = time.Second
	}
	if o.StallTimeout < 0 {
		o.StallTimeout = 0
	} else if o.StallTimeout == 0 {
		o.StallTimeout = 10 * time.Minute
	}
	if o.RequeueMinInterval <= 0 {
		o.RequeueMinInterval = 5 * time.Second
	}
	if o.RefreshTimeout <= 0 {
		o.RefreshTimeout = 10 * time.Minute
	}
	if o.StoreTimeout <= 0 {
		o.StoreTimeout = time.Hour
	}
	if o.FinishedTaskTTL <= 0 {
		o.FinishedTaskTTL = time.Hour
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}
