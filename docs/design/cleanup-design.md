# MatrixHub Disk Cleanup Feature Design Document

## Context and Scope

MatrixHub stores Git repositories, large model and dataset files, database
metadata and job logs. Cleanup must reclaim orphaned data without damaging
content still reachable from any repository.

The original scope includes orphaned Git repositories and LFS objects, expired
sessions, historical sync task records and a storage-management UI. The Cleanup
API currently covers repository and LFS storage only. Session retention, task
retention, log/cache policies, S3 lifecycle management and the UI remain separate
concerns; they are not performed by the storage cleanup endpoints.

## Storage Layout

```text
<DataDir>/
  repositories/
    <project>/<model>.git/
    datasets/<project>/<dataset>.git/
  xet/
    storage/                 # CAS indexes, shards and xorbs
    chunks/                  # client cache
  token-signing-secret       # generated signing key when not configured
```

Git stores metadata and LFS pointers; xet stores deduplicated, compressed file
content. A pointer identifies the logical content by SHA-256 and its original
size. The old `DataDir/lfs` objects are neither migrated nor deleted by the xet
collector. Back up that data before upgrading and plan migration separately.

Metadata, sessions and task records live in the configured database. Job logs
and client caches have their own lifetimes; this API does not account for or
remove them. The current server wires a local xet store, not an S3 store.

## Orphaned Git Repositories

The domain service reads valid paths through `IModelRepo.ListAllPaths` and
`IDatasetRepo.ListAllPaths`. `IGitRepo.FindOrphanedRepos` compares those paths
with repositories on the repositories filesystem. A repository absent from
both sets is a candidate; its preview size is the total of stored files.

Deletion validates the relative path, rejecting parent traversal and the root,
then uses hfd's repository removal API to invalidate its cache as well as remove
files. Repository cleanup precedes LFS collection when both are requested, so
the subsequent scan no longer treats deleted repositories as live references.

## Orphaned LFS Objects

`IGitRepo.CollectLFS(ctx, dryRun)` delegates liveness and reclamation to hfd's
repository-aware `gc.Collector` and the xet store:

1. Read the stored object list, including original file sizes, before unlinking.
2. `Prune` scans repositories for reachable LFS pointers, including history,
   and identifies unreferenced SHA-256 index entries past the grace window.
3. A dry run returns candidates without unlinking or sweeping.
4. A real run unlinks those entries, then `SweepStep` reclaims eligible CAS data.

An OID referenced by another repository or a reachable historical commit must
remain live. Errors while determining liveness must not be treated as an empty
reference set. A failure after unlinking returns the partial result and error;
the service reports completed work and appends the failure to `errors`.

`apiServer.gcGrace` is a Go duration: unset or `0` uses hfd's one-hour default;
a negative value disables the window. The chart exposes `apiserver.gcGrace`.
The collector serializes its prune and sweep operations, but does not lock out
Git pushes or uploads. Run cleanup only while pushes, uploads and mirror syncs
are quiescent: shard mtime is not refreshed on a dedup hit, so the grace period
alone cannot protect content between upload and commit publication.

## API and Size Semantics

The API contract is defined in [cleanup.proto](../../api/proto/v1alpha1/cleanup.proto).

| Endpoint | Purpose | Permission |
| --- | --- | --- |
| `POST /api/v1alpha1/cleanup/preview` | List orphan candidates without deleting | `cleanup.get` |
| `POST /api/v1alpha1/cleanup/execute` | Execute cleanup or dry-run it | `cleanup.execute` |
| `GET /api/v1alpha1/cleanup/stats` | Report repository and LFS storage usage | `cleanup.get` |

- `orphaned_lfs_objects[].size_bytes` is the original logical file size.
- `total_reclaimable_bytes` and `orphaned_size_bytes` sum candidate repository
  sizes and logical LFS sizes. They are estimates, not exact physical savings:
  xet compression, shared chunks and shard layout affect what can be removed.
- `lfs_size_bytes` sums stored shards and xorbs; it excludes indexes, caches,
  legacy LFS data, database files and logs. `total_size_bytes` adds repository
  storage to that value.
- Real execution reports actual swept bytes in `space_reclaimed_bytes`, plus
  removed repository sizes. It may also reclaim data unlinked by an earlier run.
- Dry-run execution counts candidates, but LFS contributes zero to
  `space_reclaimed_bytes`; use preview for logical size estimates.

The xet listing also exposes per-object unique compressed bytes. Summing these
does not calculate exact batch savings: objects may share chunks with other
candidates, and physical reclamation happens at shard/xorb granularity.

## Ownership

- The [cleanup service](../../internal/domain/cleanup/cleanup_service.go) owns
  orchestration and aggregation through domain repository interfaces.
- The [Git repository adapter](../../internal/repo/git_cleanup_repo.go) owns
  filesystem traversal, object listing and the collector.
- The [hfd adapter](../../internal/apiserver/hfd/backend.go) constructs the
  storage and mirror shared by protocol handlers and the repository adapter.
- The [API handler](../../internal/apiserver/handler/cleanup_handler.go) maps
  requests and responses; authorization is enforced by middleware.

## Safety and Verification

Back up storage and metadata before executing cleanup. Use preview first and
stop concurrent writers before actual collection. Expired session/task cleanup,
S3 multipart/version cleanup and automatic scheduling require their own policies
and verification; they must not be inferred from a successful storage GC run.

Focused checks:

```bash
go test -count=1 ./internal/domain/cleanup ./internal/repo ./internal/apiserver/middleware
E2E_LABELS=cleanup make test.e2e
```

Use a disposable server for E2E. Coverage must verify path confinement, dry-run
immutability and sizes, real reclamation, preservation of live content, and a
second cleanup with no newly orphaned objects. The cleanup suite uploads two
distinct LFS files, deletes one model, collects garbage and verifies that the
other model still downloads intact.