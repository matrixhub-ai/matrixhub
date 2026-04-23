# MatrixHub Disk Cleanup Feature Design Document

## Context

MatrixHub is an AI model registry that stores a large number of model files and datasets. As usage grows over time, disk space continues to increase. A disk cleanup feature is needed to:

1. Clean up orphaned data (database records deleted but disk files remain)
2. Clean up expired data (e.g., old task records, expired sessions)
3. Provide disk space management capabilities

## Confirmed Requirements Scope

Based on user confirmation, this implementation includes the following features:

| Cleanup Type | Priority | Trigger Method |
|--------------|----------|----------------|
| Orphaned Git Repositories | High | Manual API trigger |
| Orphaned LFS Objects | High | Manual API trigger |
| Expired Session Data | Medium | Manual API trigger |
| Historical Sync Task Records | Low | Manual API trigger |

**UI Requirements**: A simple UI interface is needed to provide disk usage statistics and cleanup buttons.

## Storage Architecture Detailed Analysis

### 1. Local Filesystem Storage (`DataDir`)

**Configuration Entry**: `internal/infra/config/config.go:35` - `DataDir string`

```
<DataDir>/
├── repositories/                    # Git repository storage (primary storage)
│   ├── {project}/                   # Project directory
│   │   ├── {model_name}/            # Model repository
│   │   │   ├── .git/                # Git metadata
│   │   │   │   ├── objects/         # Git object storage (commit, tree, blob)
│   │   │   │   ├── refs/            # Branch and tag references
│   │   │   │   ├── HEAD             # Current branch pointer
│   │   │   │   └── config           # Repository configuration
│   │   │   ├── README.md            # Model documentation
│   │   │   ├── config.json          # Model configuration
│   │   │   └── *.safetensors        # Model weight files (LFS pointer)
│   │   └── ...
│   └── datasets/                    # Dataset directory
│       └── {project}/
│           └── {dataset_name}/      # Dataset repository
│               └── ...
└── lfs/                             # LFS large file storage
    └── objects/                     # LFS object files
        ├── {oid[:2]}/               # First two characters subdirectory
        │   └── {oid[2:4]}/         # Middle two characters subdirectory
        │       └── {oid}            # Actual file (named by SHA256 hash)
        └── ...
```

**Key Code**:

| Location | Function | Description |
|----------|----------|-------------|
| `internal/apiserver/apiserver.go:191-193` | Git repository initialization | `gitstorage.NewStorage(gitstorage.WithRootDir(server.config.DataDir))` |
| `internal/apiserver/apiserver.go:195` | LFS storage initialization | `lfs.NewLocal(storage.LFSDir())` |
| `internal/repo/git_repo.go:57-61` | Repository path resolution | `g.storage.ResolvePath(repoName)` |

**Disk Usage Estimation**:

| Content | Storage Format | Usage Characteristics |
|---------|----------------|----------------------|
| Git Objects | Compressed binary | Many small files, incremental storage |
| Git History | Delta compression | Grows with number of commits |
| LFS Objects | Original files | **Primary usage**, model weight files typically GB~TB scale |

### 2. Database Storage

**Supported Databases**: MySQL / PostgreSQL (via GORM)

**Table Structure and Disk Usage**:

| Table Name | Record Type | Growth Rate | Disk Usage |
|------------|-------------|-------------|------------|
| `models` | Model metadata | One row per model | Low |
| `datasets` | Dataset metadata | One row per dataset | Low |
| `projects` | Project information | One row per project | Low |
| `users` | User information | One row per user | Low |
| `sessions` | Session data | Frequent read/write | **Medium** (needs periodic cleanup) |
| `access_tokens` | Access tokens | One row per token | Low |
| `sync_tasks` | Sync task records | One row per sync | **Medium** (continuous accumulation) |
| `sync_jobs` | Sync job records | One row per resource | **Medium** (continuous accumulation) |
| `sync_policies` | Sync policies | One row per policy | Low |
| `registries` | Registry configurations | One row per registry | Low |
| `members_roles_projects` | Project member relations | One row per member | Low |
| `roles` | Role definitions | Fixed data | Low |
| `labels` | Labels | Grows with usage | Low |
| `models_labels` | Model label associations | One row per label | Low |
| `datasets_labels` | Dataset label associations | One row per label | Low |

**Database Connection Configuration**: `internal/infra/db/config.go`

```go
type Config struct {
    DSN      string // Connection string
    Migrate  bool   // Auto migrate
    Debug    bool   // Debug mode
    SQLPath  string // Migration SQL path
}
```

### 3. Log File Storage

**Log Configuration**: `internal/infra/log/config.go`

```go
type Config struct {
    Level      string // Log level
    OutputPath string // Output path (stdout or file)
}
```

**Log Rotation**: Uses `gopkg.in/natefinch/lumberjack.v2` for automatic rotation

- Log files are typically stored in system default location or configured `OutputPath`
- Automatic rotation, generally no manual cleanup needed

### 4. Mirror Cache Storage

**Cache Mechanism**: `internal/apiserver/apiserver.go:206-213`

```go
sharedMirror := mirror.NewMirror(
    mirror.WithLFSCache(lfsTeeCache),
    mirror.WithTTL(time.Minute),  // 1 minute TTL
)
```

**Cache Location**: Memory + temporary files (managed by hfd library)

**Characteristics**:
- TTL auto-expiration
- Used for model caching in proxy mode
- Usually no manual cleanup needed

### 5. Potential S3/MinIO Storage

**Current Status**: Currently uses local filesystem for LFS object storage by default

**Extension Point**: `internal/apiserver/apiserver.go:195`

```go
// Current implementation
lfsStorage := lfs.NewLocal(storage.LFSDir())

// Can be extended to S3 backend (requires configuration)
// lfsStorage := lfs.NewS3(s3Client, bucket)
```

**If S3 storage is enabled, cleanup needed**:
- Orphaned LFS objects in Bucket
- Incomplete multipart uploads
- Expired object versions (if versioning is enabled)

### Storage Usage Summary

| Storage Type | Location | Primary Usage | Cleanup Priority |
|--------------|----------|---------------|------------------|
| Git Repositories | `DataDir/repositories/` | Model code, configuration | High (orphaned repos) |
| LFS Objects | `DataDir/lfs/objects/` | **Model weight files** | High (orphaned objects) |
| Database | MySQL/PostgreSQL | Metadata, sessions, task records | Medium (expired data) |
| Log Files | System path | Runtime logs | Low (auto rotation) |
| Mirror Cache | Memory/temp files | Proxy cache | Low (auto expiration) |
| S3 Objects | S3/MinIO | LFS objects (if enabled) | Low (not enabled by default) |

## Content Requiring Cleanup (Complete Analysis)

### 1. Orphaned Git Repositories (High Priority)

**Scenario**: Model/dataset records deleted from database but corresponding Git repositories and LFS files not properly deleted.

**Causes**:
- Deletion operation failed midway (DB delete succeeded but Git delete failed)
- Direct database record deletion
- System anomaly causing data inconsistency

**Cleanup Strategy**:
1. Scan `<DataDir>/repositories/` directory
2. Get all model/dataset disk paths
3. Compare with database records
4. Delete repositories without corresponding records

**Code Location**: `internal/domain/model/model_service.go:139-154` - DeleteModel deletes Git repository before deleting database record

### 2. Orphaned LFS Objects (High Priority)

**Scenario**: LFS object files exist but are not referenced by any Git repository.

**Cleanup Strategy**:
1. Scan LFS object directory
2. Check if each object is referenced by `.gitattributes` or LFS pointer files in Git repositories
3. Delete unreferenced objects

**LFS Storage Location**:
- Local filesystem: `<DataDir>/lfs/objects/`
- S3/MinIO: Objects in bucket (if S3 backend is configured)

**Code Location**:
- `internal/apiserver/apiserver.go:194` - LFS storage initialization
- `github.com/matrixhub-ai/hfd/pkg/lfs` - LFS storage abstraction

#### 2.1 LFS Pointer File Format

LFS pointer file format stored in Git repository (`internal/apiserver/handler/hf/handler_hf_upload.go:335`):

```
version https://git-lfs.github.com/spec/v1
oid sha256:<64-character SHA256 hash>
size <file size>
```

#### 2.2 LFS Object Storage Path

Actual large file storage location:

```
<DataDir>/lfs/objects/<oid[:2]>/<oid[2:4]>/<full oid>
```

For example, if OID is `abc123...`, file path is:
```
<DataDir>/lfs/objects/ab/c1/abc123...
```

#### 2.3 Detection Method

**Core Steps**:

1. **Scan LFS Directory**: Traverse `<DataDir>/lfs/objects/` to get all OIDs
2. **Scan Git Repositories**: For all repositories, traverse every branch/tag, check all commits for LFS pointers
3. **Calculate Difference**: Orphaned OID = OIDs in LFS directory - OIDs referenced in Git repositories

**Methods to Detect LFS Pointers**:

```go
// Method 1: Use blob.LFSPointer()
blob, err := entry.Blob()
if err != nil {
    return err
}
ptr, _ := blob.LFSPointer()
if ptr != nil {
    oid := ptr.OID()  // SHA256 hash
    size := ptr.Size()
}

// Method 2: Use lfs.DecodePointer() to parse small files
if blob.Size() <= lfs.MaxLFSPointerSize {
    reader, err := blob.NewReader()
    if err == nil {
        ptr, err := lfs.DecodePointer(reader)
        if err == nil && ptr != nil {
            oid := ptr.OID()
        }
    }
}
```

#### 2.4 Key Issue: References in Historical Commits

**Problem Description**: A file deleted in current HEAD may still exist in historical commits, and the corresponding LFS object should not be cleaned.

**Solution**: Must scan all reachable commits, not just each branch's HEAD.

```go
// Correct approach: Traverse all reachable commits
func collectAllReachableLFSOIDs(repo *repository.Repository) (map[string]bool, error) {
    referencedOIDs := make(map[string]bool)

    // Get all branches
    branches, _ := repo.Branches()

    for _, branch := range branches {
        // Get all commits for this branch
        commits, err := repo.Commits("refs/heads/"+branch, nil)
        if err != nil {
            continue
        }

        // For each commit, get its tree and check LFS pointers
        for _, commit := range commits {
            entries, err := repo.Tree(commit.Hash().String(), "", &repository.TreeOptions{Recursive: true})
            if err != nil {
                continue
            }

            for _, entry := range entries {
                blob, _ := entry.Blob()
                ptr, _ := blob.LFSPointer()
                if ptr != nil {
                    referencedOIDs[ptr.OID()] = true
                }
            }
        }
    }

    // Same for all tags
    tags, _ := repo.Tags()
    for _, tag := range tags {
        commits, _ := repo.Commits("refs/tags/"+tag, nil)
        // ... same processing as above
    }

    return referencedOIDs, nil
}
```

**Performance Impact**: Scanning all commits significantly increases time, but this is a necessary cost for data safety.

#### 2.5 Handling Unreachable Objects

**Scenario**: After `git gc` or `git prune`, some commits become unreachable (dangling commits).

**Judgment Method**:
- **Conservative Strategy**: Only clean LFS objects that have been cleaned by gc (i.e., no references found in Git repository)
- **Aggressive Strategy**: Execute `git gc` first to clean unreachable objects, then detect LFS

**Recommendation**: Use conservative strategy to avoid accidentally deleting data that users might recover.

#### 2.6 Multiple Repositories Referencing Same OID

**Scenario**: Different repositories may have uploaded the same file (e.g., same model weights), with identical OIDs.

**Solution**:
- Scan **all** repositories during detection
- Only mark as orphaned when **no** repository references that OID

#### 2.7 Core Function Design

```go
// internal/domain/cleanup/lfs_detector.go

package cleanup

import (
    "context"
    "os"
    "path/filepath"

    "github.com/matrixhub-ai/hfd/pkg/repository"
)

// LFSOrphanDetector detects orphaned LFS objects
type LFSOrphanDetector struct {
    dataDir     string  // Configured DataDir
    lfsDir      string  // LFS storage directory path
    storage     *storage.Storage
}

// OrphanedLFSObject represents an orphaned LFS object
type OrphanedLFSObject struct {
    OID      string
    Size     int64
    Path     string  // Filesystem path
}

// DetectOrphanedLFS performs orphaned LFS detection
func (d *LFSOrphanDetector) DetectOrphanedLFS(ctx context.Context) ([]*OrphanedLFSObject, error) {
    // Step 1: Scan LFS directory, get all OIDs
    allOIDs, err := d.scanLFSObjects()
    if err != nil {
        return nil, err
    }

    // Step 2: Collect all referenced OIDs
    referencedOIDs, err := d.collectReferencedOIDs(ctx)
    if err != nil {
        return nil, err
    }

    // Step 3: Calculate difference
    orphaned := make([]*OrphanedLFSObject, 0)
    for oid, info := range allOIDs {
        if !referencedOIDs[oid] {
            orphaned = append(orphaned, &OrphanedLFSObject{
                OID:  oid,
                Size: info.size,
                Path: info.path,
            })
        }
    }

    return orphaned, nil
}

// collectRepoLFSOIDs collects all LFS OIDs from a single repository (including historical commits)
func (d *LFSOrphanDetector) collectRepoLFSOIDs(repo *repository.Repository) (map[string]bool, error) {
    oids := make(map[string]bool)

    // Process all branches
    branches, err := repo.Branches()
    if err != nil {
        return oids, nil
    }

    for _, branch := range branches {
        d.collectOIDsFromRevision(repo, "refs/heads/"+branch, oids)
    }

    // Process all tags
    tags, err := repo.Tags()
    if err != nil {
        return oids, nil
    }

    for _, tag := range tags {
        d.collectOIDsFromRevision(repo, "refs/tags/"+tag, oids)
    }

    return oids, nil
}

// collectOIDsFromRevision collects LFS OIDs from specified revision (traversing all commits)
func (d *LFSOrphanDetector) collectOIDsFromRevision(repo *repository.Repository, rev string, oids map[string]bool) {
    commits, err := repo.Commits(rev, nil)
    if err != nil {
        return
    }

    for _, commit := range commits {
        entries, err := repo.Tree(commit.Hash().String(), "", &repository.TreeOptions{Recursive: true})
        if err != nil {
            continue
        }

        for _, entry := range entries {
            blob, err := entry.Blob()
            if err != nil {
                continue
            }

            ptr, _ := blob.LFSPointer()
            if ptr != nil {
                oids[ptr.OID()] = true
            }
        }
    }
}
```

#### 2.8 Performance Optimization Strategies

Since traversing all commits of all repositories is required, performance is a key issue:

**1. Parallel Repository Processing**
```go
func (d *LFSOrphanDetector) collectReferencedOIDsParallel(ctx context.Context) (map[string]bool, error) {
    // First collect all repository paths
    repoPaths := d.listAllRepoPaths()

    // Use worker pool for parallel processing
    var mu sync.Mutex
    referencedOIDs := make(map[string]bool)

    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(10) // Limit concurrency

    for _, path := range repoPaths {
        g.Go(func() error {
            oids, err := d.collectRepoLFSOIDsFromPath(path)
            if err != nil {
                return nil // Skip errors
            }
            mu.Lock()
            for oid := range oids {
                referencedOIDs[oid] = true
            }
            mu.Unlock()
            return nil
        })
    }

    return referencedOIDs, g.Wait()
}
```

**2. OID Caching (Avoid Repeated Parsing)**
- For the same commit hash, its tree content doesn't change, can cache parsed LFS OIDs
- Use `map[commitHash]map[oid]bool` for caching

**3. Optional Fast Mode**
- Only scan each branch's HEAD (excluding history)
- Suitable for scenarios where users explicitly know no historical files need to be retained

#### 2.9 Key Dependencies

| Package | Purpose | Key Methods |
|---------|---------|-------------|
| `hfd/pkg/repository` | Git repository operations | `Open()`, `Commits()`, `Tree()`, `Branches()`, `Tags()`, `IsRepository()` |
| `hfd/pkg/lfs` | LFS pointer parsing | `DecodePointer()`, `MaxLFSPointerSize` |
| `hfd/pkg/storage` | Storage path management | `ResolvePath()`, `LFSDir()` |

#### 2.10 Files to Implement

| File | Function |
|------|----------|
| `internal/domain/cleanup/lfs_detector.go` | LFS orphan detection core logic |
| `internal/domain/cleanup/cleanup_service.go` | Cleanup service (integrating LFS + repository cleanup) |
| `internal/repo/cleanup_repo.go` | Database queries (get valid repository list) |
| `internal/apiserver/handler/cleanup_handler.go` | API Handler |

### 3. S3/MinIO Object Storage Cleanup (Requires Evaluation)

**Current Status Analysis**:

MatrixHub supports S3-compatible backend storage (MinIO, AWS S3, etc.). Current code uses `lfs.NewLocal()` to initialize local LFS storage, but the architecture supports switching to S3 backend.

**Potential Cleanup Requirements**:

| Scenario | Description | Cleanup Method |
|----------|-------------|----------------|
| Orphaned LFS Objects | Objects in S3 Bucket that are unreferenced | Need to call S3 API to list and delete |
| Incomplete Multipart Uploads | Large file multipart upload interrupted residuals | Clean up multipart uploads |
| Expired Object Versions | If versioning is enabled | Clean up old version objects |

**Implementation Suggestion**:

```go
// Extend LFS storage interface to support cleanup
type LFSCleanup interface {
    // ListObjects lists all LFS objects
    ListObjects(ctx context.Context) ([]LFSObject, error)
    // DeleteObject deletes specified object
    DeleteObject(ctx context.Context, oid string) error
}

// S3 backend implementation
type S3LFSCleanup struct {
    client *s3.Client
    bucket string
}

func (s *S3LFSCleanup) ListObjects(ctx context.Context) ([]LFSObject, error) {
    // Call S3 ListObjectsV2
}

func (s *S3LFSCleanup) DeleteObject(ctx context.Context, oid string) error {
    // Call S3 DeleteObject
}
```

**Priority Assessment**: Low (currently uses local storage by default, S3 support requires configuration to enable)

### 4. Expired Session Data (Medium Priority)

**Current Status**: `sessions` table already has `expiry` field, but needs periodic cleanup.

**Cleanup Strategy**:
- Delete session records where `expiry < NOW()`

**Code Location**: `db/migrations/sql/mysql/0_init.up.sql:221-226` - sessions table definition

### 5. Historical Sync Task/Job Records (Low Priority)

**Scenario**: Sync task records continue to accumulate.

**Cleanup Strategy**:
- Keep last N days/N records
- Delete old completed task records (status = succeeded/failed/stopped)

**Code Location**:
- `internal/domain/syncpolicy/sync_task.go` - SyncTask model
- `internal/domain/syncjob/sync_job.go` - SyncJob model

### 6. Temporary Files (Low Priority)

**Scenario**: Temporary files left over from upload process.

**Potential Locations**:
- Git temporary files: temporary objects in `.git/objects/`
- LFS temporary files: temporary storage during upload
- HTTP request body temporary files

**Further Investigation Needed**: Check if `github.com/matrixhub-ai/hfd` library has temporary file handling mechanisms.

### 7. Other Potential Cleanup Items

| Type | Scenario | Priority | Description |
|------|----------|----------|-------------|
| Access Token | Expired access tokens | Low | `access_tokens.expire_at` field |
| Log Files | System log rotation | Low | Handled by log library (lumberjack) |
| Cache Files | Mirror cache | Medium | `mirror.WithTTL(time.Minute)` already has TTL |

### Cleanup Content Summary Table

| Cleanup Type | Storage Location | Current Implementation Status | Priority | Complexity |
|--------------|------------------|------------------------------|----------|------------|
| Orphaned Git Repositories | Local disk | ✅ Implemented | High | Medium |
| Orphaned LFS Objects (Local) | Local disk | ✅ Implemented | High | High |
| Orphaned LFS Objects (S3) | S3/MinIO | ❌ Not implemented | Low | High |
| Expired Sessions | MySQL/PostgreSQL | ✅ Implemented | Medium | Low |
| Historical Sync Tasks | MySQL/PostgreSQL | ✅ Implemented | Low | Low |
| Expired Access Tokens | MySQL/PostgreSQL | ❌ Not implemented | Low | Low |
| Temporary Files | Local disk | ❌ Not implemented | Low | Medium |
| Mirror Cache | Local disk | ⚠️ TTL configured | Medium | Low |

## Design Proposal

### API Design

```protobuf
// api/proto/v1alpha1/cleanup.proto

service CleanupService {
  // Get disk usage statistics
  rpc GetStorageStats(GetStorageStatsRequest) returns (StorageStats);

  // Preview cleanup content (dry-run)
  rpc PreviewCleanup(PreviewCleanupRequest) returns (CleanupPreview);

  // Execute cleanup
  rpc ExecuteCleanup(ExecuteCleanupRequest) returns (CleanupResult);
}

message StorageStats {
  int64 total_size_bytes = 1;       // Total used space (bytes)
  int64 models_size_bytes = 2;      // Models space usage
  int64 datasets_size_bytes = 3;    // Datasets space usage
  int64 lfs_size_bytes = 4;         // LFS objects space usage
  int64 orphaned_size_bytes = 5;    // Orphaned data size
  int32 session_count = 6;          // Session record count
  int32 expired_session_count = 7;  // Expired session count
  int32 sync_task_count = 8;        // Sync task record count
}

message PreviewCleanupRequest {
  bool include_orphaned_repos = 1;    // Include orphaned repositories
  bool include_orphaned_lfs = 2;      // Include orphaned LFS
  bool include_expired_sessions = 3;  // Include expired sessions
  bool include_old_sync_tasks = 4;    // Include historical sync tasks
  int32 sync_task_retention_days = 5; // Sync task retention days (default 30)
}

message CleanupPreview {
  repeated OrphanedRepo orphaned_repos = 1;
  repeated OrphanedLFS orphaned_lfs_objects = 2;
  int32 expired_sessions_count = 3;
  int32 old_sync_tasks_count = 4;
  int64 total_reclaimable_bytes = 5;
}

message OrphanedRepo {
  string path = 1;           // Repository path
  string type = 2;          // model or dataset
  string project_name = 3;  // Project name (parsed from path)
  string resource_name = 4; // Resource name (parsed from path)
  int64 size_bytes = 5;      // Repository size
}

message OrphanedLFS {
  string oid = 1;       // LFS object ID
  int64 size_bytes = 2; // Object size
}

message ExecuteCleanupRequest {
  bool clean_orphaned_repos = 1;      // Clean orphaned repositories
  bool clean_orphaned_lfs = 2;        // Clean orphaned LFS
  bool clean_expired_sessions = 3;    // Clean expired sessions
  bool clean_old_sync_tasks = 4;      // Clean historical sync tasks
  int32 sync_task_retention_days = 5; // Sync task retention days
  bool dry_run = 6;                   // Preview only, don't execute
}

message CleanupResult {
  int32 repos_deleted = 1;           // Number of repositories deleted
  int32 lfs_objects_deleted = 2;     // Number of LFS objects deleted
  int32 sessions_deleted = 3;        // Number of sessions deleted
  int32 sync_tasks_deleted = 4;      // Number of sync tasks deleted
  int64 space_reclaimed_bytes = 5;   // Space reclaimed
  repeated string errors = 6;        // Error messages
}
```

### Implementation Architecture

```
internal/
├── domain/
│   └── cleanup/
│       ├── cleanup.go           # Domain model
│       └── cleanup_service.go   # Cleanup service
├── repo/
│   └── cleanup_repo.go          # Database queries
└── apiserver/
    └── handler/
        └── cleanup_handler.go   # API Handler
```

### Core Cleanup Logic

```go
// internal/domain/cleanup/cleanup_service.go

type CleanupService struct {
    cleanupRepo ICleanupRepo
    gitStorage  *gitstorage.Storage
    dataDir     string
}

// FindOrphanedRepos finds orphaned Git repositories
func (s *CleanupService) FindOrphanedRepos(ctx context.Context) ([]*OrphanedRepo, error) {
    // 1. Get all models and datasets from database
    // 2. Build database path set
    // 3. Traverse disk directory, compare to find orphaned repositories
}

// FindOrphanedLFS finds orphaned LFS objects
func (s *CleanupService) FindOrphanedLFS(ctx context.Context) ([]*OrphanedLFS, error) {
    // 1. Collect all referenced LFS OIDs
    // 2. Scan LFS directory, find unreferenced objects
}

// Cleanup executes cleanup
func (s *CleanupService) Cleanup(ctx context.Context, opts *CleanupOptions) (*CleanupResult, error) {
    // Execute various cleanups based on options
}
```

### Security Considerations

1. **Dry-run Mode**: Preview by default, execute only after confirmation
2. **Permission Control**: Only platform_admin can execute cleanup
3. **Backup Recommendation**: Prompt user to backup before execution
4. **Error Handling**: Single cleanup failure doesn't affect other items
5. **Audit Logging**: Record cleanup operation audit logs

## Implementation Steps

### Phase 1: Basic Cleanup Functionality (MVP)

1. Implement `StorageStats` - Disk usage statistics
2. Implement `PreviewCleanup` - Preview orphaned repositories and LFS objects
3. Implement `ExecuteCleanup` - Clean orphaned data
4. Add API Handler and Proto definitions

**Key Files**:
- `api/proto/v1alpha1/cleanup.proto` - New
- `internal/domain/cleanup/cleanup.go` - New
- `internal/domain/cleanup/cleanup_service.go` - New
- `internal/repo/cleanup_repo.go` - New
- `internal/apiserver/handler/cleanup_handler.go` - New
- `internal/repo/repos.go` - Modify (add cleanup repo)
- `internal/apiserver/apiserver.go` - Modify (register cleanup handler)

### Phase 2: Session and Task Cleanup

1. Implement expired session cleanup
2. Implement historical sync task/job cleanup
3. Add cleanup configuration management

### Phase 3: S3 Storage Cleanup (Optional)

1. Extend LFS storage interface
2. Implement S3 backend cleanup
3. Support multipart upload cleanup

### Phase 4: Automatic Cleanup (Optional)

1. Add scheduled task framework
2. Implement automatic cleanup scheduling
3. Add cleanup policy configuration UI

## Verification Plan

### Unit Tests

```bash
go test ./internal/domain/cleanup/...
go test ./internal/repo/cleanup_repo_test.go
```

### Integration Tests

1. Create test models and datasets
2. Manually delete database records (simulate orphaned state)
3. Call PreviewCleanup to verify detection is correct
4. Call ExecuteCleanup to verify cleanup is correct
5. Verify normal data is not accidentally deleted

### Manual Verification

```bash
# 1. Start MatrixHub
make local-run-api

# 2. Create test data
# 3. Call cleanup preview via API
curl http://localhost:3001/apis/v1alpha1/cleanup/stats
# Preview orphaned data
curl -X POST http://localhost:3001/api/v1alpha1/cleanup/preview \
-H "Content-Type: application/json" \
-d '{"include_orphaned_repos": true, "include_orphaned_lfs": true}'

# Dry-run cleanup (don't actually delete)
curl -X POST http://localhost:3001/api/v1alpha1/cleanup/execute \
-H "Content-Type: application/json" \
-d '{"clean_orphaned_repos": true, "clean_orphaned_lfs": true, "dry_run": true}'
```