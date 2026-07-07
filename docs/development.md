# Local Development Guide

This document describes how to run MatrixHub frontend and backend services locally.

## Prerequisites

- Go 1.23+
- Node.js 18+
- pnpm 8+
- Docker

## Quick Start

### Local Development

If you need to modify code and debug locally, you can start services manually.

#### 1. Start MySQL Database

```bash
docker run -d \
  --name matrixhub-mysql \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=matrixhub \
  -e MYSQL_USER=matrixhub \
  -e MYSQL_PASSWORD=changeme \
  -p 3306:3306 \
  mysql:8.4
```

#### 2. Configure Environment Variables

```bash
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true"
```

#### 3. Start Backend Service

```bash
# Use default config file
go run ./cmd/matrixhub apiserver

# Or specify config file
go run ./cmd/matrixhub apiserver -c config/dev-config.yaml
```

#### 4. Start Frontend Service

```bash
cd ui
pnpm install  # Install dependencies on first run
pnpm dev
```

The frontend dev server will start at http://localhost:5173 and automatically proxy API requests to the backend.

## Using Makefile

The project provides convenient Makefile commands:

**⚠️ Important**: Before using `local-run` and `local-run-api` commands, **you must start MySQL database first**.

```bash
# 1. Start MySQL database (if not already running)
docker run -d \
  --name matrixhub-mysql \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=matrixhub \
  -e MYSQL_USER=matrixhub \
  -e MYSQL_PASSWORD=changeme \
  -p 3306:3306 \
  mysql:8.4

# 2. Run locally (frontend + backend)
make local-run

# Or run separately:
make local-run-api   # Run backend API server only
make local-run-web   # Run frontend only
```

**Tips**:
- `local-run-web` does not depend on MySQL and can run independently
- `local-run-api` and `local-run` require MySQL to be running
- If environment variables are not configured, ensure the database DSN in `config/config.yaml` is correct

## Configuration

### Environment Variables

`config.yaml` supports configuring database connection via environment variables:

```bash
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true"
```

### Frontend Proxy Configuration

The frontend is configured with Vite proxy, which automatically forwards `/api/*` and `/apis/*` requests to the backend server (http://127.0.0.1:3001) in development mode, no additional configuration needed.

Configuration file: `ui/vite.config.ts`

## Accessing the Application

- **Frontend UI**: http://localhost:5173
- **Backend API**: http://localhost:3001
- **API Health Check**: http://localhost:3001/health

## Development Tips

### Database

1. **First Run**: Setting `database.migrate: true` will automatically create database tables
2. **Debug Mode**: Setting `debug: true` shows detailed SQL logs
3. **Data Persistence**: Docker Compose uses named volumes, data persists after container restart

### Frontend

1. **Hot Reload**: Code changes automatically refresh the browser
2. **API Proxy**: No need to handle CORS in development mode
3. **TypeScript**: Use `pnpm typecheck` for type checking

### Backend

1. **Code Changes**: Requires manual service restart
2. **Dependency Management**: Use `go mod tidy` to organize dependencies
3. **Configuration Validation**: Config file is automatically validated on startup

### Writing Unit Tests

New backend unit tests, and existing unit tests when they need mocks, should follow a single mock/assertion stack:

- **Mocks**: [`go.uber.org/mock`](https://github.com/uber-go/mock) (`gomock`), generated from domain interfaces with `mockgen` (pinned in `go.mod` via the [`tool` directive](https://go.dev/doc/modules/tool)).
- **Assertions**: [`stretchr/testify/require`](https://github.com/stretchr/testify) (`require` stops the test on the first failed assertion; prefer it over `assert`).

Thanks to the DDD layering, domain services depend only on **interfaces** (ports). To unit-test a service you mock its dependencies and inject them, with **no database or network** involved. Mirror the layout: put external behavior in `internal/repo` adapters so the domain stays mockable.

**Reference example**: [`internal/domain/syncpolicy/sync_policy_service_test.go`](../internal/domain/syncpolicy/sync_policy_service_test.go) (domain service UT with gomock). For background job scheduling, rely on [`internal/jobserver/processor/processor_test.go`](../internal/jobserver/processor/processor_test.go) (poll-loop mechanics) and the e2e [`sync_policy_processor_test.go`](../test/e2e_apiserver/sync_policy/sync_policy_processor_test.go) (full scheduler path with real DB).

#### Basic steps

1. **Mark the interface for generation.** Add a `//go:generate` directive next to the interface (ports live in `internal/domain/...`). Mocks are emitted into a `mocks/` subpackage. Use `go tool mockgen` so the generator version always matches `go.uber.org/mock` in `go.mod`:

```go
//go:generate go tool mockgen -source=sync_policy.go -destination=mocks/sync_policy_repo_mock.go -package=mocks
type ISyncPolicyRepo interface {
    // ...
}
```

2. **Generate the mocks** (no separate `mockgen` install needed — Go resolves it from `go.mod`):

```bash
make generate-mocks   # runs `go generate ./...`
```

3. **Write the test** in an external `_test` package (e.g. `package syncpolicy_test`) to avoid import cycles with the generated `mocks` package. Construct a controller, inject mocks, set expectations, assert with `require`:

```go
func TestSyncPolicyService_CreateSyncPolicy(t *testing.T) {
    ctrl := gomock.NewController(t) // auto-verifies expectations via t.Cleanup
    repo := mocks.NewMockISyncPolicyRepo(ctrl)

    // Expect exactly one call; gomock fails the test on unexpected/missing calls.
    repo.EXPECT().CreateSyncPolicy(gomock.Any(), gomock.Any()).Return(nil)

    // Leave unused dependencies nil — only wire what the tested method touches.
    svc := syncpolicy.NewSyncPolicyService(repo, nil, nil, nil, nil)

    err := svc.CreateSyncPolicy(context.Background(), &syncpolicy.SyncPolicy{
        TriggerType: syncpolicy.TriggerTypeManual,
    })
    require.NoError(t, err)
}
```

4. **Run the tests**:

```bash
# Single package
go test -count=1 ./internal/domain/syncpolicy/...
go test -count=1 ./internal/jobserver/...

# All backend unit tests (matches CI; excludes e2e under test/)
go test -short -count=1 ./cmd/... ./internal/...

# go test ./... also runs test/e2e_apiserver/... and requires a running
# apiserver + MySQL. For e2e, use: make test.e2e
```

#### gomock tips

- **Argument matchers**: `gomock.Any()`, `gomock.Eq(v)`, or a literal value. Use literals/`Eq` to assert the exact arguments a method was called with.
- **Return / capture**: `.Return(...)` for static results; `.DoAndReturn(func(...) {...})` to capture arguments or compute the result dynamically.
- **Call counts**: expectations default to exactly once. Use `.Times(n)`, `.MinTimes(n)`, `.MaxTimes(n)`, or `.AnyTimes()` — handy for background loops that poll an unpredictable number of times.
- **Negative assertions**: simply set **no** `EXPECT()` for a method; gomock fails the test if it is called.
- **Regenerate after changing an interface**: rerun `make generate-mocks` and commit the updated files under `mocks/` (they are generated — never hand-edit).
- **Dependabot bumps `go.uber.org/mock`**: the `tool` directive keeps `mockgen` in sync automatically. CI re-runs `go generate` and fails if `mocks/` drifts. If that happens, check out the Dependabot branch, run `make generate-mocks`, commit any `mocks/` diff, and push.

### Sync policies and jobserver (delayed-job, single replica)

Sync scheduling uses `sync_policies.next_run_at` (milliseconds). The **jobserver** runs in-process with the API server (`internal/jobserver`): the **sync_policy** processor polls due policies, CAS-advances `next_run_at`, then calls `CreatePendingSyncTask` to insert a `sync_tasks` row (status pending). A separate **sync_task** processor claims pending tasks and calls `ExecuteSyncTask` to generate `sync_jobs` rows; the **sync_job** processor then runs the git work. There is no separate cron daemon process.

- **Config** (`config.yaml`): `jobServer` — set `enabled: false` to disable inserting pending `sync_tasks` from the poller (manual API `CreateSyncTask` will only bump `next_run_at`). Per-sync-policy tuning (`pollInterval`, `maxConcurrent`, `taskMaxDuration`) lives under `jobServer.syncPolicy`.
- **Migrations**: scheduling columns (`cron`, `last_run_at`, `next_run_at`, `idx_due`) are in `0_init.up.sql`; apply with `database.migrate: true` or run SQL under `db/migrations/sql/mysql/`.
- **Cron syntax**: validated with `github.com/robfig/cron/v3` (five fields plus descriptors such as `@daily`). Example: `0 * * * *` (hourly).
- **Not in this release**: Stop task, heartbeat, reaper, multi-replica fencing (`running_by` / `running_until`).

**Verify jobserver logic (no database required)**:

```bash
go test -v -count=1 ./internal/jobserver/...
```

**End-to-end** (requires MySQL + migrated schema + a sync policy with `next_run_at` due): start `go run ./cmd/matrixhub apiserver`, trigger a policy (scheduled cron or `CreateSyncTask` for manual bump), then list `sync_tasks` (pending rows) / watch logs for `jobserver: processor loop start`.

## Troubleshooting

### MySQL Connection Failed

```bash
# Check MySQL container status
docker ps | grep matrixhub-mysql

# View MySQL logs
docker logs matrixhub-mysql

# Restart MySQL
docker restart matrixhub-mysql
```

### Port Conflicts

If ports are occupied, you can modify:

**Backend Port**:

Modify `apiServer.port` in `config.yaml`

**Frontend Port**:
```bash
cd ui
pnpm dev --port 3000
```

### Dependency Issues

**Go Dependencies**:
```bash
go mod tidy
go mod download
```

**Frontend Dependencies**:
```bash
cd ui
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

### Generate HTTP Client SDK

When swagger API definitions change, regenerate the HTTP client SDK for testing:

```bash
make gen_openapi_sdk
```

Output: `test/client` directory.
