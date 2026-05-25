# MatrixHub Architecture Rules (DDD)

This file is the shared entry point for backend/Go architecture constraints.

For UI work, follow `ui/AGENTS.md`.

## Dependency direction (MUST)

Arrows mean **"A depends on / imports B"**.

```
handler (middleware / transport) ───────────────→ domain
job processor (async/cron/worker entrypoints) ─→ domain
repo (implementation) ─────────────────────────→ domain
domain ──(public/common only)──────────────────→ infra
repo ─────────────────────────────────────────→ infra
```

### Allowed

- **handler/job processor → domain(service interface)**.
- **handler/job processor → domain(repo interface)** when no domain service is needed.
- **domain(service impl) → domain(repo interface)**.
- **repo → domain + infra** (repo is the implementation layer).
- **domain → infra (public/common only)**.

### Forbidden (hard rules)

- **domain MUST NOT depend on anything except `infra` public/common packages**.
- **domain MUST NOT depend on handler/job processor/transport**.
- **handler/job processor MUST NOT import `internal/repo` implementations** (only domain interfaces).
- **`internal/infra` (shared/pkg) MUST NOT contain business rules**.

## Ports & adapters

- **Ports (interfaces)** live in `internal/domain/...`.
- **Adapters (implementations)** live in `internal/repo/...` and use `internal/infra/...`.
- **External interactions** (DB/gorm/HTTP APIs/SDK/MQ/object storage) should be implemented in `internal/repo/...` whenever possible.

## Folders (backend)

- **`internal/apiserver/handler`**: HTTP/GRPC handlers, middleware, request/response mapping, auth extraction.
- **`internal/jobserver/...`**: background job entrypoints (pollers/processors/workers). Same dependency rules as handlers.
- **`internal/domain/...`**: domain entities/value objects, domain services/use-cases, domain interfaces (ports).
- **`internal/repo/...`**: implementations of domain interfaces (adapters). Depends on `internal/domain`, uses `internal/infra`.
- **`internal/infra/...`**: public/common packages (e.g. log, i18n, errors), database wiring, gorm models, third-party SDK clients, utilities.
