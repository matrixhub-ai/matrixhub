# Trivy Go Alerts Remediation Design

## Goal

Resolve MatrixHub code-scanning alerts 13, 14, and 16 without removing PostgreSQL support or hiding reachable high-severity vulnerabilities.

## Context

The alerts are produced by the image-scanning job in `.github/workflows/call-release-image.yaml`. Trivy scans the compiled `/usr/local/bin/matrixhub` binary, writes SARIF, and uploads it to GitHub code scanning under the `trivy-image-scan` category.

The repository defaults to MySQL in local, Docker Compose, and Helm configurations, but the database layer intentionally supports both MySQL and PostgreSQL. PostgreSQL support must remain.

## Design

### Alert 13: CVE-2026-26958

The latest upstream `main` already resolves `filippo.io/edwards25519` to `v1.2.0`, which is newer than the first fixed release, `v1.1.1`. No additional dependency change is required for this alert. Verification must confirm that the selected version remains at least `v1.1.1` after the other module graph changes.

### Alert 14: CVE-2026-32286

Replace the legacy `github.com/jackc/pgconn` import in `internal/infra/db/errors.go` with `github.com/jackc/pgx/v5/pgconn`.

The active GORM PostgreSQL driver already uses pgx/v5. Matching its `PgError` type preserves PostgreSQL unique-constraint detection and allows the legacy `github.com/jackc/pgconn` module and its vulnerable `github.com/jackc/pgproto3/v2` dependency to leave the module graph.

Add a regression test that passes a pgx/v5 `PgError` with `pgerrcode.UniqueViolation` to `IsUniqueViolationError`. The test must fail against the legacy import before the implementation changes.

### Alert 16: GO-2026-5932

Keep `golang.org/x/crypto` because MatrixHub uses unaffected `bcrypt` and `ssh` packages. MatrixHub does not import the affected `openpgp` packages.

Set `limit-severities-for-sarif: true` on the Trivy action. The workflow already declares `severity: HIGH,CRITICAL`, but SARIF includes every severity unless this input is enabled. This prevents the unreachable, unknown-severity OpenPGP module alert from being uploaded while retaining high- and critical-severity image findings.

Do not add `.trivyignore` entries and do not dismiss high-severity findings globally.

## Files

- `internal/infra/db/errors.go`: migrate PostgreSQL error matching to pgx/v5.
- `internal/infra/db/errors_test.go`: add MySQL/PostgreSQL unique-violation regression coverage.
- `go.mod`, `go.sum`: remove obsolete pgconn/pgproto3 dependencies while retaining the already-fixed edwards25519 version.
- `.github/workflows/call-release-image.yaml`: enforce severity filtering for SARIF output.

## Verification

1. Observe the new pgx/v5 PostgreSQL error test fail before changing production code.
2. Run `go test ./internal/infra/db -count=1` after the migration.
3. Run `go mod tidy` and `go mod verify`.
4. Confirm `filippo.io/edwards25519` still resolves to `v1.1.1` or later; upstream currently selects `v1.2.0`.
5. Confirm legacy `github.com/jackc/pgconn` and `github.com/jackc/pgproto3/v2` are absent from the selected module graph.
6. Run `govulncheck ./...`.
7. Run the relevant Go package tests and record unrelated baseline failures separately.
8. Validate the workflow YAML and confirm `limit-severities-for-sarif` is enabled on the SARIF-producing Trivy step.

## Known Baseline Failures

The correctly scoped baseline command `go test ./cmd/... ./internal/... -short -count=1` currently fails before this change in:

- `internal/apiserver/handler/hf`, where the existing test fixture reaches a nil dependency in `handleCreateRepo`.
- `internal/apiserver/handler/http`, where tests assume the Git default branch is `master`.

These failures are outside this security-remediation scope. The existing CI command omits the leading `./` in `go test cmd/... internal/...`, so it exercises Go toolchain packages rather than MatrixHub packages; correcting that CI issue is also outside this change.

## Non-goals

- Removing PostgreSQL support.
- Refactoring the database abstraction.
- Fixing unrelated HF or HTTP test failures.
- Changing the repository-wide unit-test command.
- Adding broad vulnerability ignores.
