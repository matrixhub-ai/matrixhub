# MatrixHub Architecture Guide

This document defines the backend architecture rules for MatrixHub. It is the
human-readable source of truth for Go package boundaries, dependency direction,
and where business logic should live.

For UI work, follow [`ui/AGENTS.md`](../ui/AGENTS.md).

## Architecture Model

MatrixHub follows a lightweight DDD and ports-and-adapters structure:

- Domain packages define entities, value objects, use cases, and interfaces.
- Handlers and job processors call domain services or domain interfaces.
- Repository packages implement domain interfaces and talk to infrastructure.
- Infrastructure packages provide common utilities, database wiring, models,
  clients, and other shared technical building blocks.

## Dependency Direction

An arrow means "A imports or depends on B".

```text
handler (middleware / transport)        -> domain
job processor (async / cron / worker)   -> domain
repo (adapter implementation)           -> domain
repo (adapter implementation)           -> infra
domain                                  -> infra public/common only
```

## Allowed Dependencies

- Handler and job processor packages may depend on domain service interfaces.
- Handler and job processor packages may depend on domain repository interfaces
  when no domain service is needed.
- Domain service implementations may depend on domain repository interfaces.
- Repository implementations may depend on domain and infra packages.
- Domain packages may depend on infra public/common packages only.

## Forbidden Dependencies

These are hard rules:

- Domain packages must not depend on anything except infra public/common
  packages.
- Domain packages must not depend on handlers, job processors, transport
  packages, or repository implementations.
- Handler and job processor packages must not import `internal/repo`
  implementations. They should depend on domain interfaces instead.
- `internal/infra` must not contain business rules.

## Ports And Adapters

- Ports, meaning interfaces, live in `internal/domain/...`.
- Adapters, meaning implementations, live in `internal/repo/...`.
- External interactions such as DB/GORM, HTTP APIs, SDKs, message queues, and
  object storage should be implemented in `internal/repo/...` whenever possible.

Domain services should express application behavior in terms of domain
interfaces. Repository adapters should translate between domain concepts and
external systems.

## Backend Folders

- `internal/apiserver/handler`: HTTP/GRPC handlers, middleware,
  request/response mapping, and auth extraction.
- `internal/jobserver/...`: background job entrypoints, pollers, processors,
  and workers. These follow the same dependency rules as handlers.
- `internal/domain/...`: domain entities, value objects, services/use cases, and
  interfaces.
- `internal/repo/...`: implementations of domain interfaces. These packages
  depend on `internal/domain` and use `internal/infra`.
- `internal/infra/...`: public/common packages, logging, i18n, errors, database
  wiring, GORM models, third-party SDK clients, and utilities.

## Placement Checklist

When adding or changing code:

- Put business decisions in `internal/domain/...`.
- Put interface definitions in the domain package that owns the use case.
- Put database, SDK, object storage, or other external integration code in
  `internal/repo/...`.
- Keep handlers focused on protocol concerns such as auth extraction, request
  parsing, validation shape, response mapping, and error translation.
- Keep job processors focused on scheduling, polling, claiming work, and calling
  domain services.
- Keep `internal/infra/...` generic and reusable; do not encode MatrixHub
  business policy there.
