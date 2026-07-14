# Jobserver Design

## Context

MatrixHub uses the jobserver to run background work in-process with the API
server. The current sync policy scheduler follows a delayed-job style design
for a single API server replica.

This document captures the current runtime behavior. It can be expanded as the
jobserver design evolves.

## Sync Policy Scheduling

Sync scheduling uses `sync_policies.next_run_at`, stored in milliseconds. The
jobserver runs in-process with the API server under `internal/jobserver`; there
is no separate cron daemon process.

The sync policy flow is:

1. The `sync_policy` processor polls due policies.
2. It CAS-advances `next_run_at`.
3. It calls `CreatePendingSyncTask` to insert a `sync_tasks` row with pending
   status.
4. The `sync_task` processor claims pending tasks.
5. It calls `ExecuteSyncTask` to generate `sync_jobs` rows.
6. The `sync_job` processor runs the git work.

## Configuration

The relevant config block is `jobServer` in `config.yaml`.

- Set `jobServer.enabled: false` to disable inserting pending `sync_tasks` from
  the poller. Manual API `CreateSyncTask` will only bump `next_run_at`.
- Per-sync-policy tuning lives under `jobServer.syncPolicy`, including
  `pollInterval`, `maxConcurrent`, and `taskMaxDuration`.

## Database Schema

Scheduling columns are in `0_init.up.sql`:

- `cron`
- `last_run_at`
- `next_run_at`
- `idx_due`

Apply migrations with `database.migrate: true` or run the SQL under
`db/migrations/sql/mysql/`.

## Cron Syntax

Cron expressions are validated with `github.com/robfig/cron/v3`.

The scheduler supports five-field cron expressions and descriptors such as
`@daily`.

Example:

```text
0 * * * *
```

This runs hourly.

## Current Limitations

The current design is single-replica oriented. These capabilities are not part
of the current release:

- Stop task
- Heartbeat
- Reaper
- Multi-replica fencing with `running_by` / `running_until`

## Verification

Verify jobserver logic without a database:

```bash
go test -v -count=1 ./internal/jobserver/...
```

Verify the full scheduler path end to end with MySQL and a migrated schema:

1. Start MatrixHub with the jobserver enabled.
2. Create or trigger a sync policy, either by scheduled cron or manual
   `CreateSyncTask`.
3. Confirm that pending `sync_tasks` rows are created.
4. Watch logs for `jobserver: processor loop start`.
