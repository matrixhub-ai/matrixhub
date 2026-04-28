# Archetype: Detail Page

A route-backed detail view keyed by a URL parameter, prefetched by the loader, read via Suspense in the component.

---

## Reference implementation

| File | Role |
|---|---|
| `src/routes/(auth)/(app)/projects/$projectId/route.tsx` | Parent route — `await ensureQueryData`, params, layout + tabs |
| `src/features/projects/projects.query.ts` | `projectDetailQueryOptions` |
| `src/features/projects/pages/ProjectSettingsPage.tsx` | Sub-page using `routeApi.useParams()` + `useSuspenseQuery` |

---

## Required patterns

From `patterns.md`:

- §1 Architecture at a glance
- §12 Loaders — single or parallel prefetch (`Promise.allSettled` for multi-source)
- §13 Component data access — `useSuspenseQuery` (data is never `undefined`)
- §14 Route hooks — `Route.useParams`, `Route.useNavigate`, `getRouteApi` when reading outside the route component
- §15 Route `errorComponent` — the right home for initial-load failures

---

## Archetype-specific gotchas

- **Use `useSuspenseQuery`, not `useQuery`, when the loader prefetched.** Suspense + typed non-undefined data is the entire point.
- **Parallel prefetches use `Promise.allSettled`, not `Promise.all`.** One failing call must not block the others.
- **Initial load failures belong in `errorComponent`.** Do not convert them into a notification toast.
- **Route context carries the single `QueryClient`.** Do not instantiate a second one inside the feature.

---

## Must answer before coding

1. Which params drive the query? Are they validated by the route?
2. Is this page composed of 1 data source or multiple? If multiple, are they all strictly required, or do some tolerate partial failure?
3. Which mutations can be triggered from this page, and which query keys must they invalidate?
4. What is the error UX — full-page error boundary, inline per-section, or both?

---

## Self-check before handoff

Subset of `workflows/review-checklist.md`:

- Loader uses `ensureQueryData`; multi-source uses `Promise.allSettled`.
- Component reads data via `useSuspenseQuery`.
- `errorComponent` is defined when the loader can fail in a user-meaningful way.
- No ad-hoc `try/catch` around mutations — `MutationCache` handles error notifications.
