# Security Scanning & Admission

Model repositories carry more than weights: configs, Python code, shared
libraries, archives and — critically — pickle-serialized model files that can
execute arbitrary code at `torch.load` time. MatrixHub's security subsystem
(design: [docs/design/model-security-scanning.md](design/model-security-scanning.md),
issue [#1066](https://github.com/matrixhub-ai/matrixhub/issues/1066)) scans
every repository revision and enforces a download-admission policy.

**Nothing in the scanning path deserializes, imports or executes repository
content.** The pickle analyzer walks raw opcodes; ClamAV receives a byte
stream over a local socket.

## What happens when you upload

1. A commit (HF NDJSON upload, git push, or upstream sync) fires the
   post-receive hook → one scan task is enqueued per updated ref **tip**
   (keyed by commit SHA — content changes always produce a new verdict).
2. The scan processor claims the task and analyzes every file with three
   static scanners:

| Scanner | Covers | Rule source |
|---|---|---|
| `clamav` | generic malware in any file (clamd INSTREAM, size-capped) | ClamAV daily signatures |
| `pickle-static` | dangerous serialization: pickle opcode walk resolving `GLOBAL`/`STACK_GLOBAL` targets; denylist (`os.*`, `subprocess.*`, `builtins.eval/exec/…`, `ctypes.*`, …) → **critical**; model-framework allowlist (`torch.*`, `collections.*`, …) → clean; unknown imports → **warning**; zip containers (`.pt/.pth/.bin/.ckpt`) walked to their `data.pkl` members | built-in opcode-walker/1 |
| `heuristics` | compression bombs (ratio + absolute caps), pathological nesting, file-type identification | built-in heur/1 |

3. File findings aggregate into a version verdict: any **critical → blocked**,
   else any **warning → warning**, else **pass**. Scanner failures fail the
   task — a scan that could not run is never recorded as safe.

## Statuses

Revision status (HF `securityRepoStatus.status` in parentheses):
`unscanned`/`scanning` (scanning) · `pass`/`warning` (clean) · `blocked`
(infected) · `failed` (error). Unscanned, scanning and failed are never
reported as clean.

## Admission policy

Platform default + optional per-project override
(`PUT /api/scan/v1alpha1/policies/{project}`):

```json
{
  "mode": "enforce",            // enforce | audit | off
  "blockSeverity": "critical",  // lowest severity that blocks
  "onPending": "block",         // block | allow while a scan is running
  "onFailed": "block"           // block | allow when scanning failed
}
```

Blocked downloads answer with a stable, HF-client-friendly error:

```
HTTP 403
{"error":"RevisionBlocked","message":"revision 816f22228b6c is blocked by
security policy: malicious or dangerous content detected (see scan report)"}
```

The same gate applies to git fetch (upload-pack) at the default branch tip.

## Hugging Face API compatibility

```python
from huggingface_hub import HfApi
info = HfApi(endpoint=MY_MATRIXHUB).model_info(
    "project/model", securityStatus=True, files_metadata=True)
info.security_repo_status   # {"kind": "malware", "status": "infected", "details": {...}}
info.siblings[0].size       # + blobId, lastModified, lfs{oid, sha256, size}
```

## REST API

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/scan/v1alpha1/reports/{type}/{project}/{name}/revision/{rev}` | explainable report (per-file severity, rules, digests, scanner+rule versions, counts; **payloads are never echoed**) |
| POST | `.../revision/{rev}/rescan` | manual rescan (bypasses the content-digest cache, audit-logged) |
| GET | `.../status` | quick status + HF mapping |
| GET/PUT | `/api/scan/v1alpha1/policies[/{project}]` | policy management |
| GET | `/api/scan/v1alpha1/audit` | audit trail (filters: project/name/revision/actor/action) |

## Result reuse

File-level results are cached by `(sha256 content digest, scanner, rule
version)`: identical weights files across revisions or repositories are not
re-scanned. Version verdicts and policy decisions are always recomputed per
(project, revision) — context, rules and tenants are never mixed. A ClamAV
signature update changes the rule version and invalidates the cache
automatically.

## Enabling

```yaml
jobServer:
  scan:
    pollInterval: 5s
    maxConcurrent: 2
    taskMaxDuration: 30m
    maxFiles: 10000
    maxFileSizeBytes: 4294967296
    clamAVSocket: /var/run/clamav/clamd.ctl   # omit → scan tasks fail per policy
```

Docker Compose with a clamd sidecar:
`deploy/docker-compose.security.yml` + `deploy/config-security-sqlite.yaml`.

## Execution safety

Task deadline, file-count and per-file size caps; bounded pickle walking
(opcode/memo/stack budgets); bounded decompression (absolute + ratio);
scanners never touch the network (clamd via local socket) and never see
credentials. Worker crashes self-heal: claimed tasks carry a lease and are
re-queued by the periodic stale-claim sweep.

## Known limitations

- ClamAV is signature-based; "pass" is not proof of safety (UI and report
  wording reflect this).
- The pickle allowlist covers common frameworks; unknown imports raise
  warnings for manual review rather than false-clean.
- git fetch gating evaluates the default branch tip; per-path `/resolve`
  downloads are gated at the exact revision.
- Container-level (cgroup) hardening is deployment-dependent; in-process
  limits are enforced as listed above.

## Trying it

Upload a model containing the harmless
[EICAR](https://en.wikipedia.org/wiki/EICAR_test_file) test string or a
static pickle sample referencing `os.system`, then watch the model's
**Security** tab (or the report API) flip the revision to `blocked` and
downloads start answering `RevisionBlocked`. The e2e suite
(`test/e2e_apiserver/security_scan/`) exercises the full flow including
content updates, manual rescan, audit and anonymous-access rejection.
