# Archetype: Route-Only Change

A change that touches only route plumbing — search params, loaders, redirects — without modifying feature UI. Typical reasons: adding a new URL param, reshaping an existing param, adding a guard, wiring a new prefetch.

---

## Reference implementation

| File | Role |
|---|---|
| `src/routes/(auth)/(app)/projects/index.tsx` | Search params + loader + mount |
| `src/routes/(auth)/(app)/projects/$projectId/route.tsx` | Parent route with `beforeLoad` |
| `src/routes/(auth)/route.tsx` | Auth guard |
| `src/router.tsx` | Router instance with `queryClient` context |

---

## Required patterns

From `patterns.md`:

- §11 Search params with Zod + `fallback`
- §12 Loaders — detail pages `await` to block navigation, list pages fire-and-forget (`void`); `loaderDeps` when loader reads search
- §14 Route hooks & context — single `QueryClient`, `getRouteApi` outside the route file

---

## Archetype-specific gotchas

- **Updating `validateSearch` shape is a user-facing contract change.** Existing bookmarks / inbound links must degrade gracefully — use `fallback` for every new param so missing/invalid values don't redirect to error.
- **Add `loaderDeps` when the loader newly depends on search.** Adding a search param without declaring it in `loaderDeps` means the loader sees stale data after back-navigation.
- **Do not hand-edit `src/routeTree.gen.ts`.** After changing route files, the generator runs on `pnpm dev` / `pnpm build` and refreshes the tree.
- **Guards belong in `beforeLoad` / a parent route**, not in the component. Returning `redirect(...)` from `beforeLoad` is the canonical pattern.
- **Do not import a second `QueryClient`.** Loaders receive the one in router context.

---

## Must answer before coding

1. What is the existing search schema — are new params additive or do they reshape existing ones?
2. Does the loader need to fire new queries, or just more of the same data with different params?
3. Is the change backwards-compatible for existing URLs (old params still work)?
4. Does this change affect any existing feature page that reads `Route.useSearch()`?

---

## Self-check before handoff

Subset of `workflows/review-checklist.md`:

- New/changed search params use `fallback` for defaults.
- `loaderDeps` reflects the params the loader reads.
- No manual edit of `src/routeTree.gen.ts`.
- No second `QueryClient` introduced.
- If the route contract changed, downstream feature pages still read search correctly.
