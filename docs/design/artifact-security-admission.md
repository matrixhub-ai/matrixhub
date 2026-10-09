# Model artifact scanning and revision-bound admission

## Purpose and scope

This opt-in contribution implements the DaoCloud model-artifact scanning and admission challenge for the 2026 Shanghai Open Source Competition. It starts from MatrixHub commit `6890726abe272767d26ce1a6b0580b0718f232c0` and uses the existing storage, authorization and JobServer infrastructure.

The verified workflow is direct upload, asynchronous static scanning, revision-specific HF metadata, and policy-controlled downloads. Model output moderation, training-data poisoning and model backdoors are outside the scanning scope. Proxy/sync ingestion is a later integration step.

## Components

- `internal/domain/artifactscan`: scan entities, use cases, aggregation, project policies and owned scanner/storage interfaces.
- `internal/repo`: persisted tasks, reports, policies, audit events and scanner/Git/LFS adapters.
- `internal/jobserver/artifact_scan.go`: task claiming, execution and abandoned-attempt recovery.
- API adapters: report and management endpoints, authentication/authorization, HF metadata and download decisions.
- Git HTTP/SSH and LFS adapters: authorize the repository and evaluate the content served.
- `deploy/security-prototype`: external ClamAV/Fickling worker and a local Compose deployment.
- Model Security UI: revision status, file findings, project policy, rescan/cancel and audit events.

The opt-in REST contract is recorded in [artifact-security.json](../../api/openapi/artifact-security.json). SQL migration 2 is provided for SQLite and MySQL; SQLite is the tested database. API and migration integration remain review topics.

## Revision identity and Git transfer

A report belongs to an immutable repository revision. Content changes enqueue a new scan; historical results remain associated with their original revision. Verified file digests, scanner/rule versions and the project context scope reusable file results. Revision aggregation and policy decisions are evaluated for their current context.

Git fetch can serve history beyond a default branch tip. Each Git read captures refs and reachable history in a private snapshot. Both admission and upload-pack operate on that same snapshot, so a concurrent push cannot change the approved transfer. The snapshot is bounded to 256 refs, 256 reachable commits, a 128 MiB pack and a 30-second creation budget. Over-budget snapshots are denied.

HF resolve and repository-scoped LFS access apply the same policy service after authorization. LFS bytes are checked against expected digest/size. HF model information supports revision-specific security status and file metadata; pending and failed checks remain visible to clients.

## States, policy and task lifecycle

Unscanned, pending, scanning, passed, warning, blocked, failed and cancelled outcomes remain distinguishable. Strict policy admits completed passing revisions. Configurable project policy controls warning thresholds and treatment of pending/failure states.

Completed findings survive an incomplete scan. A known high-risk finding blocks download even if another check fails and the project otherwise allows failures. Reports expose both the risk finding and the incomplete check. Unsupported input and analyzer budget failures do not become a successful result.

Manual rescan creates a new attempt and bypasses reuse. Rule/configuration changes require reassessment. Attempt ownership fences late results after cancellation or replacement; recovery handles abandoned claims. Audit records preserve task actions and actual admission decisions.

## Static analysis and isolation

ClamAV scans generic malware signatures. Fickling 0.1.12 performs bounded static Pickle analysis in a limited child process. No scan loads a model, deserializes an object, imports repository modules or runs repository scripts.

ZIP processing validates member count, expanded size, compression ratio and duplicate names. PyTorch tensor recognition requires a restricted metadata profile and retains a medium-risk manual-review warning. Numeric members still undergo supported Pickle-call detection; filenames and member order alone cannot grant an exemption. Known dangerous calls remain blocked.

Nested/encrypted archives, unsupported serialization and exceeded analysis budgets produce an incomplete result. Safetensors currently receives generic ClamAV checks and filename classification, without a dedicated structure validator.

The worker runs without root/capabilities on an internal network, with read-only root, bounded temporary storage, CPU/memory/process limits and deadlines. Business credentials are not mounted. ClamAV has a separate signature-update network. See [deployment and reproduction](../../deploy/security-prototype/README-stage2.md) and [third-party notices](../../deploy/security-prototype/THIRD_PARTY.md).

## Current capacity

| Budget | Value |
| --- | --- |
| File transfer/scan | 1 GiB per file |
| Pickle metadata | 8 MiB per segment, 32 MiB total; 10-second static child budget |
| Files per revision | 256 |
| ZIP | 100 entries; 1280 MiB expanded; ratio 200 |
| Revision execution | 900 seconds in Compose |
| Scanner HTTP / worker / ClamAV | 600 / 540 / 480 seconds |
| Scheduling | Single execution loop; tasks queue |

256 MiB and 1 GiB controlled containers were uploaded, scanned and downloaded with matching bytes and SHA-256. They were rescanned under the final rule fingerprint. These measurements cover individual artifacts; high-concurrency capacity has not been measured.

## Recorded verification

The author-run environment uses SQLite, Docker Compose, Go 1.26.9, ClamAV 1.5.4 and Fickling 0.1.12, with `huggingface_hub 1.29.0` as the official client.

| Scope | Result |
| --- | --- |
| Seven affected Go packages, all top-level tests with race checking | 120 passed; HF package 64 passed |
| Authorization failure subcases | 12, denied/error across create/delete for model/dataset/space |
| Controlled scanner cases / additional container-risk regressions | 30 / 8 passed |
| Official HF client / admission / web session probes | 7 / 15 / 5 passed |
| Git SSH / lifecycle recovery probes | 2 / 3 passed |
| Fixed public PyTorch checkpoint and policy linkage | warning; 4 policy checks passed |
| Controlled large artifacts | 256 MiB and 1 GiB, matching download digests |
| Go static checks / UI lint, typecheck and build | passed |

The HF test fixture now supplies authorization and commit-service doubles while retaining actual Git writes and reads; production permission checks were not changed. Original upload/read/branch/tag/LFS assertions remain in place.

Reproduce the seven-package tests:

```bash
go test -race -json \
  ./internal/domain/artifactscan ./internal/repo ./internal/apiserver \
  ./internal/apiserver/handler/hf ./internal/apiserver/handler/http \
  ./internal/apiserver/handler/ssh ./internal/apiserver/middleware \
  -count=1
```

Sanitized evidence is under `security-admission-evidence/20261010`, `20261010-guards` and `20261010-hf-fixture`. The initial capacity run and final-rule rescan are separate measurements. No live malware is used; EICAR and inert serialization samples are examined statically. The results are engineering regression evidence; an independent malicious-model accuracy benchmark has not been performed.

## Review topics

- External scanner deployment and LGPL/GPL dependency boundary.
- API authorization, versioned contract and migration placement.
- Snapshot/history admission scope and budgets.
- Backend/UI split and documentation-website integration.

MySQL, high concurrency, proxy/sync ingestion and Xet uploads are not validated. The upstream E2E suite has not been run against this contribution; recorded verification covers the packages and live probes above.
