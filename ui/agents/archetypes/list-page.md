# Archetype: List Page

A route-backed paginated list with search params, a table, and (usually) create / delete actions.

---

## Reference implementation

Mirror the shape of the projects list. Read these files before building a new list page.

| File | Role |
|---|---|
| `src/routes/(auth)/(app)/projects/index.tsx` | Route adapter — `validateSearch`, `loaderDeps`, `loader`, mount page |
| `src/features/projects/projects.schema.ts` | Zod schemas: `projectsSearchSchema`, `ProjectsSearch`, input types for mutations |
| `src/features/projects/projects.query.ts` | `projectKeys` factory, `projectsQueryOptions`, `projectDetailQueryOptions` |
| `src/features/projects/projects.mutation.ts` | `createProjectMutationOptions`, `deleteProjectMutationOptions` with `NotificationMeta` |
| `src/features/projects/pages/ProjectsPage.tsx` | Page: `Route.useSearch()`, `useQuery`, render table + toolbar |
| `src/features/projects/components/ProjectsTable.tsx` | Table — columns/cells only |

---

## Required patterns

From `patterns.md`:

- §1 Architecture at a glance — definition/runtime flow
- §2–§3 Query keys + `queryOptions` / `mutationOptions` factories
- §4 `NotificationMeta` behavior (set `errorMessage`; `successMessage` only when needed)
- §11 Route: search params with Zod + `fallback`
- §12 Route: `prefetchQuery` loader (non-blocking) + `loaderDeps`
- §13 Route: `useQuery` after prefetch
- §17 Tables — go through the project wrapper (`src/shared/components/DataTable.tsx`)
- (Optional) `useRouteListState` from `src/shared/hooks/useRouteListState.ts` — use when the table needs row selection + batch operations; skip for read-only tables

---

## Archetype-specific gotchas

- **`loaderDeps` is mandatory when the loader reads search.** Without it, the loader sees stale params on back-navigation.
- **Do not wrap `useQuery(projectsQueryOptions(search))` in a custom hook** just for aesthetics (`patterns.md §3`).
- **Invalidation points.** Every create/delete mutation must include `invalidates: [projectKeys.lists()]` in `meta`. If the list silently fails to refresh after a mutation, this is the first thing to check.
- **Success toast policy.** List pages usually refetch on mutation success, which already communicates success to the user — omit `meta.successMessage` unless the UX explicitly needs a toast.
- **Search params live in the URL, not local state.** Bind toolbar inputs through `Route.useNavigate({ search })`, not `useState`.

---

## Must answer before coding

1. Which route and which feature does the page belong to? Is the existing route adapter sufficient, or does a new one need to be added?
2. Which search params (pagination, sort, query string) live in the URL?
3. Which create/update/delete actions must be available, and which ones trigger list invalidation?
4. Which locale keys are new? Confirm en + zh updates.
5. Does this page need a table wrapper extension, or is the existing `DataTable` sufficient?

---

## Self-check before handoff

Subset of `workflows/review-checklist.md`:

- `validateSearch` + Zod schema present with `fallback` defaults.
- `loaderDeps` declared when the loader reads search.
- Every mutation has `meta.errorMessage` and `meta.invalidates`.
- Table usage goes through the project wrapper, not raw `mantine-react-table`.
- Locale keys present in both `en` and `zh`.

---

## Complete planning sample

For a worked end-to-end planning example, see `ui/agents/examples/page-plan-example.md` (models-within-a-project list).
