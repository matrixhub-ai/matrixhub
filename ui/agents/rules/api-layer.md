# API Layer

This doc records only the stable conventions for using the generated API SDK in `ui/`.

Do not document page-specific API decisions, temporary endpoint quirks, or speculative global patterns here. Add them only after they become shared project conventions.

## Stable Facts

- The backend exposes a gRPC API through an HTTP/JSON gateway.
- The TypeScript SDK is generated under `api/ts/` at the repo root.
- In `ui/`, import SDK modules through the `@matrixhub/api-ts/*` alias.
- Generated `.pb.ts` files are read-only. Do not edit them manually; shared generated-runtime customizations must live in `scripts/patch_ts_sdk.sh`.
- `api/ts/fetch.pb.ts` exposes `setFetchFn`/`resetFetchFn` so the generated SDK can use an app-provided fetch implementation. This repo-maintained patch is applied after TypeScript SDK generation by `scripts/patch_ts_sdk.sh`.
- `src/shared/utils/apiRequestHeaders.ts` installs app-level API request header providers by configuring the generated SDK fetch source at startup. Providers apply to `/api` and `/apis` requests made through the generated SDK.
- The default provider adds the current i18next language as `Accept-Language`.

Example:

```ts
import type { ListUsersRequest, User } from '@matrixhub/api-ts/v1alpha1/user.pb'
import { Users } from '@matrixhub/api-ts/v1alpha1/user.pb'
```

## Default Usage

- Prefer calling generated SDK methods directly when the SDK already covers the endpoint.
- Add wrappers or abstractions only after a real shared need appears.
- Add global API headers through `src/shared/utils/apiRequestHeaders.ts` instead of per-call SDK wrappers. Explicit headers passed by a specific call take precedence over provider defaults.

## Type Rules

- Prefer SDK request, response, and entity types over hand-written duplicates.
- Use `import type` for type-only imports.
- If the UI needs a different shape, derive a thin UI-specific type with `Pick`, `Omit`, or explicit mapping.
- Generated proto3 fields are often optional; handle `undefined` explicitly in UI code.
- Import shared types from their canonical SDK module instead of redefining them locally.

## Do Not

- Manually edit generated `.pb.ts` files.
- Write raw REST calls for endpoints that already exist in the SDK.
- Add per-request global headers around generated SDK calls.
- Import from `../api/ts/...` directly. Use `@matrixhub/api-ts/...`.
- Freeze the current SDK file tree as a long-term rule.
- Document speculative global API patterns such as shared `pathPrefix`, global error mapping, auth refresh, or query library integration before the project actually adopts them.
