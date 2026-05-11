# Archetype: List Page

A route-backed paginated list with search params, a table, and (usually) create / delete actions.

---

## Reference implementation

Read `src/features/admin/users/` + `src/routes/(auth)/admin/users.tsx` before building a new list page. For a simpler variant without row selection, see `src/features/projects/`.

**Flexibility:** search schema can be inlined in the route file when trivial (1–2 params); `useRouteListState` can be skipped when no row selection / batch operations are needed; a custom hook wrapping `useQuery` is only justified when it adds behavior like `placeholderData`.

---

## Required patterns

From `patterns.md`:

- §1 Architecture at a glance — definition/runtime flow
- §2–§3 Query keys + `queryOptions` / `mutationOptions` factories
- §4 `NotificationMeta` behavior (set `errorMessage`; `successMessage` only when needed)
- §11 Route: search params with Zod + `.default().catch()`
- §12 Route: `prefetchQuery` loader (non-blocking) + `loaderDeps`
- §13 Route: `useQuery` after prefetch
- §17 Tables — go through the shared wrapper (`src/shared/components/DataTable.tsx`)
- (Optional) `useRouteListState` from `src/shared/hooks/useRouteListState.ts` — use when the table needs row selection + batch operations; skip for read-only or single-action tables

---

## Archetype-specific gotchas

- **`loaderDeps` is mandatory when the loader reads search.** Without it, the loader sees stale params on back-navigation.
- **Invalidation points.** Every create/delete mutation must include `invalidates: [keys.lists()]` in `meta`. If the list silently fails to refresh after a mutation, this is the first thing to check.
- **Success toast policy.** List pages usually refetch on mutation success, which already communicates success to the user — omit `meta.successMessage` unless the UX explicitly needs a toast.
- **Search params live in the URL, not local state.** Bind toolbar inputs through `getRouteApi().useNavigate()`, not `useState`.

---

## Must answer before coding

1. Which route and which feature does the page belong to? Is the existing route adapter sufficient, or does a new one need to be added?
2. Which search params (pagination, sort, query string) live in the URL?
3. Which create/update/delete actions must be available, and which ones trigger list invalidation?
4. Does the table need row selection + batch operations (`useRouteListState`) or is it single-action only?
5. Which locale keys are new? Ensure all languages under `src/locales/` are updated.

---

## Self-check before handoff

Subset of `workflows/review-checklist.md`:

- `validateSearch` + Zod schema present with `.default().catch()` defaults.
- `loaderDeps` declared when the loader reads search.
- Loader uses `prefetchQuery` (non-blocking), not `ensureQueryData`.
- Every mutation has `meta.errorMessage` and `meta.invalidates`.
- Table usage goes through the shared wrapper, not raw `mantine-react-table`.
- Locale keys present in all languages under `src/locales/`.

---

## Complete planning sample

For a worked end-to-end planning example, see `ui/agents/examples/page-plan-example.md` (models-within-a-project list).
