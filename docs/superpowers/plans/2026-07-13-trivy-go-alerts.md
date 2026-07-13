# Trivy Go Alerts Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the legacy PostgreSQL protocol dependency that triggers CVE-2026-32286, retain the already-fixed edwards25519 version, and make the existing HIGH/CRITICAL Trivy SARIF policy effective.

**Architecture:** Keep MatrixHub's MySQL and PostgreSQL behavior unchanged. Align PostgreSQL error inspection with the pgx/v5 driver already used by GORM, then let `go mod tidy` remove the obsolete pgconn/pgproto3 packages from the MatrixHub runtime. The modules can remain in the broader module graph because golang-migrate declares them for an unused driver path; the release binary must not contain them. Treat the OpenPGP finding as an unreachable module-level result and filter SARIF by the workflow's declared severities rather than adding a vulnerability ignore.

**Tech Stack:** Go 1.25.12, GORM PostgreSQL/pgx v5, Go modules, Trivy GitHub Action, GitHub SARIF upload.

---

### Task 1: Cover and migrate PostgreSQL unique-violation detection

**Files:**
- Create: `internal/infra/db/errors_test.go`
- Modify: `internal/infra/db/errors.go:17-33`

- [ ] **Step 1: Write the failing regression test**

Create `internal/infra/db/errors_test.go` with the following content:

```go
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

package db

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "mysql unique violation", err: &mysql.MySQLError{Number: 1062}, want: true},
		{name: "mysql other error", err: &mysql.MySQLError{Number: 1064}, want: false},
		{name: "postgres unique violation", err: &pgconn.PgError{Code: pgerrcode.UniqueViolation}, want: true},
		{
			name: "wrapped postgres unique violation",
			err:  fmt.Errorf("insert project: %w", &pgconn.PgError{Code: pgerrcode.UniqueViolation}),
			want: true,
		},
		{name: "postgres other error", err: &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}, want: false},
		{name: "unrelated error", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolationError(tt.err); got != tt.want {
				t.Fatalf("IsUniqueViolationError() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" test ./internal/infra/db -run TestIsUniqueViolationError -count=1 -v
```

Expected: FAIL for both PostgreSQL unique-violation cases because production still tests against the legacy `github.com/jackc/pgconn.PgError` type.

- [ ] **Step 3: Implement the minimal pgx/v5 migration**

In `internal/infra/db/errors.go`, replace:

```go
"github.com/jackc/pgconn"
```

with:

```go
"github.com/jackc/pgx/v5/pgconn"
```

Do not change `IsUniqueViolationError` logic beyond the import migration.

- [ ] **Step 4: Run the test and verify GREEN**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" test ./internal/infra/db -run TestIsUniqueViolationError -count=1 -v
```

Expected: all six subtests PASS.

- [ ] **Step 5: Run package-level regression tests**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" test ./internal/infra/db -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the behavior change**

```bash
git add internal/infra/db/errors.go internal/infra/db/errors_test.go
git commit -m "fix(db): use pgx v5 postgres errors"
```

### Task 2: Remove the obsolete PostgreSQL modules

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Reconcile the module graph**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" mod tidy
```

Expected: `github.com/jackc/pgx/v5` becomes direct; legacy `github.com/jackc/pgconn` and `github.com/jackc/pgproto3/v2` disappear from `go.mod`/`go.sum`; edwards25519 remains newer than `v1.1.1`.

- [ ] **Step 2: Verify dependency integrity**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" mod verify
"$GO" list -m -f '{{.Version}}' filippo.io/edwards25519
if "$GO" list -deps ./cmd/matrixhub | rg '^github.com/jackc/(pgconn|pgproto3/v2)($|/)'; then
  echo 'legacy PostgreSQL package still linked into MatrixHub'
  exit 1
fi
```

Expected: all modules verify, edwards25519 reports `v1.2.0` or later, and the release command has no dependency on either legacy package tree.

- [ ] **Step 3: Re-run the database tests**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" test ./internal/infra/db -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit module cleanup**

```bash
git add go.mod go.sum
git commit -m "fix(deps): remove legacy pgproto3"
```

### Task 3: Enforce the declared Trivy SARIF severity policy

**Files:**
- Modify: `.github/workflows/call-release-image.yaml:131-138`

- [ ] **Step 1: Verify the configuration assertion is RED**

```bash
rg -q '^          limit-severities-for-sarif: true$' .github/workflows/call-release-image.yaml
```

Expected: exit status 1 because the limiter is absent.

- [ ] **Step 2: Add the SARIF severity limiter**

Change the Trivy action inputs to:

```yaml
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'HIGH,CRITICAL'
          limit-severities-for-sarif: true
```

- [ ] **Step 3: Verify the configuration assertion is GREEN**

```bash
rg -q '^          limit-severities-for-sarif: true$' .github/workflows/call-release-image.yaml
ruby -e 'require "yaml"; YAML.load_file(".github/workflows/call-release-image.yaml"); puts "workflow yaml valid"'
```

Expected: both commands exit 0 and print `workflow yaml valid`.

- [ ] **Step 4: Commit the workflow change**

```bash
git add .github/workflows/call-release-image.yaml
git commit -m "fix(ci): limit Trivy SARIF severities"
```

### Task 4: Perform final security and regression verification

**Files:**
- Verify only; no new files.

- [ ] **Step 1: Run focused Go tests**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
"$GO" test ./internal/infra/... -short -count=1
```

Expected: PASS.

- [ ] **Step 2: Run govulncheck**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
tools_dir=$(mktemp -d)
GOBIN="$tools_dir" "$GO" install golang.org/x/vuln/cmd/govulncheck@latest
"$tools_dir/govulncheck" ./...
rm -rf "$tools_dir"
```

Expected: no reachable vulnerabilities.

- [ ] **Step 3: Inspect compiled binary module metadata**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
build_dir=$(mktemp -d)
"$GO" build -o "$build_dir/matrixhub" ./cmd/matrixhub
"$GO" version -m "$build_dir/matrixhub" > "$build_dir/modules.txt"
rg 'filippo.io/edwards25519\s+v1.2.0' "$build_dir/modules.txt"
if rg 'github.com/jackc/(pgconn|pgproto3/v2)' "$build_dir/modules.txt"; then
  echo 'legacy PostgreSQL module present in binary'
  exit 1
fi
rm -rf "$build_dir"
```

Expected: fixed edwards25519 is present and neither legacy PostgreSQL module appears.

- [ ] **Step 4: Compare full tests with the recorded baseline**

```bash
GO="$HOME/.cache/codex/toolchains/go1.25.12/bin/go"
results=$(mktemp)
set +e
"$GO" test -json ./cmd/... ./internal/... -short -count=1 > "$results"
test_rc=$?
set -e
jq -r 'select(.Action == "fail" and .Package != null) | .Package' "$results" | sort -u
rm -f "$results"
printf 'full test exit: %d\n' "$test_rc"
```

Expected: non-zero only for the recorded baseline packages `internal/apiserver/handler/hf` and `internal/apiserver/handler/http`; no touched package fails.

- [ ] **Step 5: Verify final diff and branch state**

```bash
git diff upstream/main...HEAD --check
git diff upstream/main...HEAD --stat
git status --short --branch
```

Expected: no whitespace errors, only the design/plan, database error handling/tests, module files, and Trivy workflow changed, and the worktree is clean.
