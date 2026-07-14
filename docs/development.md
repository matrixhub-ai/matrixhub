# Local Development Guide

This document describes how to run MatrixHub frontend and backend services locally.
For backend package boundaries and dependency rules, see
[Architecture Guide](architecture.md).

## Prerequisites

- Go 1.23+
- Node.js 18+ for the app UI; Node.js 20+ for the documentation website
- pnpm 8+
- Docker

## Local Development

Start with MySQL, then choose the frontend mode that matches your work.

### 1. Start MySQL

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

### 2. Configure the Backend Database DSN

```bash
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true"
```

### Option A: Backend-focused Development With Built UI

Use this when changing backend code and you only need a working UI. The UI is
built once and served by the Go API server from the same origin.

Make sure the selected config points `ui.staticDir` to the built frontend:

```yaml
ui:
  staticDir: ./ui/dist
```

Build the frontend:

```bash
cd ui
pnpm install
pnpm build
cd ..
```

Start the backend from the repository root:

```bash
go run ./cmd/matrixhub apiserver -c config/config.yaml
```

If you keep environment-specific values in a local config file, pass that config
file explicitly:

```bash
go run ./cmd/matrixhub apiserver -c config/config-local.yaml
```

Make shortcuts:

```bash
make local-run-built-ui
LOCAL_CONFIG=config/config-local.yaml make local-run-built-ui
```

Open http://localhost:3001.

### Option B: Full-stack Development

Use this when changing frontend code or when you need Vite hot reload.

Start the backend from the repository root:

```bash
# Use default config file
go run ./cmd/matrixhub apiserver
# Or specify config file
go run ./cmd/matrixhub apiserver -c config/config.yaml
```

Start the frontend dev server in another terminal:

```bash
cd ui
pnpm install # Install dependencies on first run
VITE_APP_API_URL=http://127.0.0.1:3001 pnpm dev
```

If your backend uses a different address or port, point `VITE_APP_API_URL` to
that address instead.

Open http://localhost:5173.

Make shortcuts:

```bash
make local-run       # Start backend and frontend dev server together
make local-run-api   # Start backend only
make local-run-ui    # Start frontend dev server only
```

## Documentation Website Local Development

The documentation website lives in `website/` and uses Docusaurus.

For active documentation development with hot reload:

```bash
cd website
npm install
npm run start
```

Open http://localhost:3000.

To preview the production build locally:

```bash
cd website
npm install
npm run build
npm run serve
```

Make shortcuts:

```bash
make -C website build
make serve-website
```

## Static Checks

Run the locally runnable CI static checks from the repository root:

```bash
make verify
```

For focused runs while iterating:

```bash
make verify.go
make verify.ui
make verify.workflow
make verify.govulncheck
```

`make verify.go` runs Go lint plus mock generation consistency checks.
`make verify.ui` runs ESLint, TypeScript type checking, and the UI build.
`make verify.workflow` runs GitHub Actions workflow linting and action reference
validation. `make verify.govulncheck` checks reachable Go vulnerabilities.
GitHub CodeQL analysis still runs in CI and is not included in `make verify`.

To auto-fix supported Go lint/import formatting issues:

```bash
make lint-fix
```

## Unit Tests

Run all unit tests from the repository root:

```bash
make test.unit
```

`make test.unit` temporarily excludes `./internal/apiserver/handler/hf`; 
add that package back after the HF handler tests are fixed.

Run unit tests with coverage:

```bash
make test.unit.coverage
```

For focused package runs while iterating:

```bash
go test -count=1 ./internal/domain/syncpolicy/...
```

The Makefile delegates package selection to `scripts/unit-test.sh`, where the
default package list is defined. Override `UNIT_TEST_PKGS` to include new
unit-test package roots:

```bash
UNIT_TEST_PKGS="./cmd/... ./internal/... ./pkg/..." make test.unit
```

Override `UNIT_TEST_EXCLUDE_PKGS` to change the temporary exclusions:

```bash
UNIT_TEST_EXCLUDE_PKGS= make test.unit
```

When interfaces change, regenerate mocks before running or committing tests:

```bash
make generate-mocks
```

## End-to-End Tests

E2E tests live under `test/e2e_apiserver` and run against a live MatrixHub API
server. Start MySQL and the backend API server first; the frontend UI does not
need to be running.

Known issue: E2E tests modify the target MatrixHub database and `dataDir`, and
the current cleanup does not guarantee that database rows or filesystem data are
restored to their original state. Use a disposable local database and data
directory, not a shared or important environment.

```bash
export MATRIXHUB_DATABASE_DSN="matrixhub:changeme@tcp(127.0.0.1:3306)/matrixhub?charset=utf8mb4&multiStatements=true&parseTime=true"
go run ./cmd/matrixhub apiserver -c config/config.yaml
```

Then run E2E tests from another terminal:

```bash
make test.e2e
```

`make test.e2e` runs E2E tests against
`MATRIXHUB_BASE_URL=http://localhost:3001` by default. The default level is
`smoke`; currently `smoke` and `all` run the same E2E package set with different
timeouts. Override the level or base URL when needed:

```bash
level=all make test.e2e
MATRIXHUB_BASE_URL=http://localhost:3002 make test.e2e
```

The E2E runner invokes the `ginkgo` CLI. If it is not installed:

```bash
go install github.com/onsi/ginkgo/v2/ginkgo@v2.22.1
```

For a KIND-based E2E environment:

```bash
make test.e2e.kind
```

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

4. **Run the relevant unit tests.** See [Unit Tests](#unit-tests) for local
commands.

#### gomock tips

- **Argument matchers**: `gomock.Any()`, `gomock.Eq(v)`, or a literal value. Use literals/`Eq` to assert the exact arguments a method was called with.
- **Return / capture**: `.Return(...)` for static results; `.DoAndReturn(func(...) {...})` to capture arguments or compute the result dynamically.
- **Call counts**: expectations default to exactly once. Use `.Times(n)`, `.MinTimes(n)`, `.MaxTimes(n)`, or `.AnyTimes()` — handy for background loops that poll an unpredictable number of times.
- **Negative assertions**: simply set **no** `EXPECT()` for a method; gomock fails the test if it is called.
- **Regenerate after changing an interface**: rerun `make generate-mocks` and commit the updated files under `mocks/` (they are generated — never hand-edit).
- **Dependabot bumps `go.uber.org/mock`**: the `tool` directive keeps `mockgen` in sync automatically. CI re-runs `go generate` and fails if `mocks/` drifts. If that happens, check out the Dependabot branch, run `make generate-mocks`, commit any `mocks/` diff, and push.

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
