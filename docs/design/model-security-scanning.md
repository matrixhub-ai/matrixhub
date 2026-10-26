# Design: Model Artifact Malware Scanning & Security Admission

## Summary

Add a security scanning and admission-control subsystem to MatrixHub: every model
revision that enters the server (direct upload, git push, or upstream sync) gets a
scan task; file-level findings from a generic malware engine (ClamAV) plus a static
pickle analyzer are aggregated into a version-level verdict; download admission is
enforced at the resolve endpoint (and git fetch) according to a configurable policy;
results are exposed through the Hugging Face-compatible `model_info(securityStatus=)`
API, a REST report API, the web UI, and an audit trail.

Non-goals (per the competition brief and by design): no model loading, no
deserialization of untrusted objects, no execution of repository code, no detection
of model-output harmfulness / training-data poisoning / backdoors.

## Threat model & scanning boundary

A model repository contains weights (pickle/zip-based `.pt/.pth/.bin/.ckpt`,
safetensors), configs, tokenizer data, Python code, shared libraries and nested
archives. The risks in scope:

1. **Generic malware** inside any file (supply-chain droppers, infected binaries).
2. **Dangerous serialized content**: pickle opcodes that reference executable
   entrypoints (`os.system`, `subprocess.*`, `builtins.eval`, ...) — arbitrary code
   execution at `torch.load` time.
3. **Resource-exhaustion payloads**: archive/compression bombs aimed at the scanner
   or at downstream clients.

The scanning boundary is the **repository revision** (git commit SHA). Content
changes produce a new revision and therefore a new scan result; results are never
inherited across revisions.

## Architecture

Follows the project's DDD layering (`docs/code-architecture.md`):

```
upload / git push / mirror sync ──► postReceiveHook ──► ScanService.EnqueueRevision
                                                                    │
jobserver processor (claims pending tasks via DB CAS)               ▼
   ├─ enumerate files at revision (gitRepo.Tree)              scan_tasks
   ├─ per file: sha256 digest ──► file_result cache hit? ──► reuse
   ├─ scanners: clamav (INSTREAM) · pickle-static · heuristics
   ├─ aggregate file findings ──► version verdict (scan_reports)
   └─ audit events
                                                                    │
download admission: handleResolve / git-upload-pack ◄── policy ────┘
HF API: /api/models/{repo}?securityStatus=true&files_metadata=true
REST:   /api/scan/v1alpha1/... (reports, rescan, policy, audit)
UI:     model detail page badge + findings table + rescan button
```

### Domain (`internal/domain/scan`)

- Entities: `ScanTask`, `FileResult`, `VersionReport`, `ScanPolicy`, `AuditEvent`.
- Ports: `Store` (persistence), `TreeReader` (list files + open at revision —
  implemented on top of the existing `hfd` repository API), `Scanner`.
- `ScanService`: enqueue, claim/execute, aggregate, policy evaluation, rescan.

### Scanners

1. **clamav** — talks to `clamd` over a unix socket using the INSTREAM command
   (chunked, size-capped). No subprocess execution. Version = `SIGNATURES`
   reported by clamd (clamav version + DB version), reused as the rule version.
2. **pickle-static** — a pure-Go pickle **opcode walker**. It reads the pickle
   opcode stream (protocols 0–5) without building any objects: it tracks
   `GLOBAL` (protocol ≤3) and `STACK_GLOBAL` (protocol 4+) operands and matches
   the resolved `module.name` against:
   - a denylist (`os.system`, `os.popen*`, `posix.*`, `subprocess.*`,
     `builtins.eval/exec/execfile/compile/__import__`, `ctypes.*`,
     `pty.*`, `socket.*`, ...) → **critical**;
   - an allowlist of known model-framework globals (`torch.*`,
     `collections.*`, `numpy.*` multiarray, `torchvision.*`, ...) → clean;
   - anything else → **warning** "review required" (unknown import).
   Malformed/truncated pickles and opcode-budget exhaustion are reported as
   **failed** file results (never silently clean, never a crash). No imports of
   referenced modules ever happen.
3. **heuristics** — resource-safety checks: compression-bomb detection for
   zip/gzip/tar members (decompression capped; ratio + absolute caps), nesting
   depth limits, and magic-byte based file-type identification used by the
   report. Executable/archive file types are reported as informational.

### File-type routing

All files go through clamav + heuristics. The pickle analyzer runs on
`.pkl/.pickle/.pt/.pth/.bin/.ckpt/.safetensors?` — no: safetensors is JSON+raw
and structurally safe; it is only flagged as `safetensors` file type (clamav
still runs). `.pt/.pth/.bin/.ckpt` are zip containers (new zipfile format) or
legacy pickles: the scanner sniffs the magic (`PK` → walks zip members ending in
`data.pkl` and scans those members; raw pickle otherwise).

### State machine

```
pending ──► scanning ──► completed(verdict: pass|warning|blocked)
   │            │
   │            └──► failed        (scanner unavailable, internal error, timeout)
   └───────────────► cancelled     (manual cancel / shutdown)
```

Version-level security status (`scan_reports`, keyed by revision):

| situation                    | status     | HF `securityRepoStatus.status` |
|------------------------------|------------|-------------------------------|
| no task for revision         | `unscanned`| `null`-safe: `scanning` semantics chosen by policy (see below) |
| pending/scanning             | `scanning` | `scanning` |
| completed, no ≥block finding | `pass`/`warning` | `clean` (+details) |
| completed, ≥block finding    | `blocked`  | `infected` |
| failed/cancelled             | `failed`   | `error` |

Unscanned and scanning are **never** reported as clean; failed is reported as
`error`, never as safe.

### Triggers

- `postReceiveHookFunc` (covers HF NDJSON commit, git push over http/ssh, branch
  and tag updates, and mirror sync which reuses the same hook) → enqueue one
  scan task per updated ref tip.
- Lazy proxy sync (`CheckOrSyncFromRemote`) → enqueue after pull.
- Manual rescan → REST API, audit-logged, forces re-execution.

### Admission policy

`scan_policies`: platform default + optional per-project override.

- `mode`: `enforce` | `audit` | `off`.
- `block_severity`: lowest severity that blocks (`critical` | `high` | `medium`).
- `on_pending`, `on_failed`, `on_scanner_unavailable`: `block` | `allow`
  (defaults `block`; "受控放行" is the `allow` setting — the decision is always
  recorded in the audit log and shown in API/UI).

Enforcement points:

- **`/resolve` downloads** (HF-compatible path used by `huggingface_hub`):
  after revision resolution, look up the version status; deny with
  `403` + stable JSON error body
  `{"error":"RevisionBlocked","message":"...security policy: <reason> (report: <url>)"}`
  so HF-compatible clients surface a clear message.
- **git fetch (upload-pack, http+ssh)**: denied while any requested ref tip is
  blocked (or pending under a blocking policy) with a `GIT_E` style pre-advert
  error message.
- Upload is never blocked by scan results (scanning is post-receive);
  pre-receive upload-time checks are out of scope for v1.

### Result reuse

`scan_file_results` is keyed by `(content sha256, scanner_id, scanner_version)`:
identical content (e.g. the same weights file appearing in many revisions or
repos) reuses the physical finding without re-running scanners. The **context**
is protected by construction: file findings are pure content facts (no project,
user or policy data); version reports, policy evaluation and admission decisions
are always computed per (project, revision) from the current policy — never
copied. A scanner/rule version bump changes the cache key, forcing re-evaluation.
Manual rescan sets `force`, bypassing the cache.

### Execution safety & limits

- Scanners run inside the jobserver processor goroutine with a per-task context
  deadline (default 30m), max files per task (default 10k) and per-file size cap
  (default 2 GiB, streaming digest).
- pickle walker: bounded opcode count, bounded memo size, bounded stack depth.
- heuristics: capped decompression (default 4 GiB absolute / 1000:1 ratio),
  nesting depth ≤ 8.
- clamd INSTREAM: size-capped stream; clamd unavailability ⇒ task `failed`
  (visible, retryable), never "clean".
- No network access needed by any scanner (clamd via local unix/tcp socket);
  no environment/credential access; scanner code paths never evaluate repo data.
- Service restart: claimed tasks carry `claim_deadline`; on boot, stale claims
  are reset to `pending` and re-run.
- Production hardening (documented, optional deployment): run the apiserver
  scanner with systemd/cgroup CPU+memory limits; the Helm chart exposes the
  clamd socket / sidecar values.

### HF API compatibility

`/api/models/{ns}/{name}` (and `/revision/{rev}`):

- `securityStatus=true` → adds `securityRepoStatus`:
  `{"kind":"malware","status":"scanning|clean|infected|error", "details":{...}}`
  — `details` carries MatrixHub extras (verdict, per-severity counts, task id)
  which `huggingface_hub` ignores while parsing known fields.
- `files_metadata=true` (and `blobs=true`) → each sibling gains `size`,
  `lastModified`, and for LFS files an `lfs` object
  `{oid, size(, sha256 for compat)}`; findings summary is attached per file
  when a completed report exists.
- Results always correspond to the **requested revision** (resolved to a commit
  SHA); unscanned / scanning / failed / completed are distinct values.

### Audit & governance

`scan_audit_events` records: task creation (trigger + actor), policy decisions
(allow/deny + reason + policy snapshot), manual review actions (rescan,
override), admission outcomes. Queryable via REST (filter by project/model/
revision/actor/action) and rendered in the UI model security tab. Every event
resolves to operator + project + model + revision + (optional) file path.

## Data model (migration `2_scan`)

Tables: `scan_tasks`, `scan_file_results`, `scan_reports`, `scan_policies`,
`scan_audit_events` (see migration SQL for columns). SQLite + MySQL variants.

## REST API (gin, prefix `/api/scan/v1alpha1`)

The gRPC-gateway owns `/api/v1alpha1/*`, so the scan API is mounted on the
existing gin engine under a non-conflicting prefix:

- `GET  /reports/{repoType}/{project}/{name}/revision/{rev}` — version report
  (status, verdict, findings with digest/type/rule/severity/scanner+version/
  scanned-at/advice), paginated.
- `POST /reports/{repoType}/{project}/{name}/revision/{rev}/rescan` — manual
  rescan (audit).
- `GET/PUT /policies` (+ per-project) — policy management.
- `GET  /audit` — filtered audit query.
- `POST /tasks/{id}/cancel` — cancel a running scan.

## UI

Model detail page: security status badge (unscanned/scanning/pass/warning/
blocked/failed), findings table with severity chips + per-file detail drawer
(rule, scanner+rule version, digest, advice; **payloads are never shown**),
rescan button, policy banner, audit timeline. i18n en/zh.

## Failure semantics

| failure                | behavior                                                     |
|------------------------|--------------------------------------------------------------|
| clamd unreachable      | task `failed`; admission per `on_scanner_unavailable`        |
| file unreadable/corrupt| per-file `failed` result; version verdict `failed` if any required scanner failed for any file |
| timeout                | task `failed` (deadline), partial findings kept for triage   |
| malformed pickle       | per-file finding (medium) "malformed/truncated pickle"       |
| zip bomb               | per-file critical finding; decompression aborted at cap      |
| server restart         | stale claims re-queued; statuses stay truthful                |

## Known limitations

- No allow/deny decision on upload (post-receive scanning only).
- git fetch gating is per-ref-tip; historical non-tip revisions inside a fetch
  are not individually evaluated (resolve path is exact).
- Pickle allowlist covers common frameworks; unknown imports raise warnings
  (review) rather than false-clean.
- ClamAV coverage is signature-based; a pass is not proof of safety (surfaced in
  UI/report wording).
- Container-level (cgroup) isolation is deployment-dependent; in-process limits
  are enforced as listed above.

## Testing

Unit: pickle walker (safe/malicious/STACK_GLOBAL/truncated/budget), archive
bomb, aggregation matrix, policy matrix, digest reuse, HF status mapping +
handler query params, admission middleware.

E2E (sqlite, docker-compose optional clamd sidecar): safe upload → clean;
EICAR file → blocked + HF download denied with clear error; crafted pickle
(static-only sample) → blocked; content update → new revision re-scanned;
duplicate file → cache reuse; clamd stopped → failed status + policy-controlled
admission; server restart mid-scan → task re-run; unauthorized report query →
403.
