# Code Patterns

Cross-library protocols used everywhere in `src/features/` and `src/routes/`. Read this alongside an archetype file from `archetypes/` when building a new page.

Organised by pattern (not by library). When a section says "see §N", follow that anchor inside this file.

---

## §1. Architecture at a glance

Two loader strategies exist depending on page type:

```
Detail page (single resource, must exist before render):
  loader: await queryClient.ensureQueryData(queryOptions)   ← blocks navigation
    └─ Component
         └─ useSuspenseQuery(queryOptions)   → data is never undefined

List page (collection, tolerates loading state):
  loader: void queryClient.prefetchQuery(queryOptions)      ← non-blocking
    └─ Component
         └─ useQuery(queryOptions)           → handles isPending / keepPreviousData
```

**Definition flow** — how files are wired:
`schema → query → mutation → route → page/form`

**Runtime flow** — what happens during navigation:
`loader → page → form submit → cache invalidation → notification`

Roles:

- **Router** prefetches data in loaders using `queryOptions`.
- **Query** manages server state (reads + writes).
- **Form** manages local field state and validation; delegates submission to a mutation.
- **Notifications** fire globally from `QueryCache` / `MutationCache` — no per-component error handling.

Reference: `src/queryClient.ts`, `src/router.tsx`, `src/types/tanstack-query.ts`.

---

## §2. Query keys

Every feature defines a hierarchical key factory. Always use `as const` for literal narrowing.

```ts
// src/features/{feature}/{feature}.query.ts
export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params: ProjectsSearch) => [...projectKeys.lists(), params] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (projectId: string) => [...projectKeys.details(), projectId] as const,
}
```

---

## §3. `queryOptions` / `mutationOptions` factories

Always extract into factory functions. This is the single most important pattern for reuse across `useQuery`, `useSuspenseQuery`, `ensureQueryData`, `setQueryData`.

```ts
import { queryOptions, mutationOptions } from '@tanstack/react-query'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'

export function projectDetailQueryOptions(projectId: string) {
  return queryOptions({
    queryKey: projectKeys.detail(projectId),
    queryFn: () => Projects.GetProject({ name: projectId }),
  })
}

export function deleteProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (projectId: string) => Projects.DeleteProject({ name: projectId }),
    meta: {
      errorMessage: i18n.t('projects.deleteError'),
      invalidates: [projectKeys.lists()],
    } satisfies NotificationMeta,
  })
}
```

Rules:

- Never pass explicit generics to `useQuery<Data, Error>()`. Let the factory infer.
- Introduce a custom hook (e.g. `useProjects`) only when it adds real behavior (`placeholderData`, conditional `enabled`, composed queries). A thin wrapper over `useSuspenseQuery(options)` is dead weight.
- Keep query/mutation code in `src/features/{feature}/{feature}.{query,mutation}.ts` — never inline in route files.

---

## §4. `NotificationMeta` + global caches

All user-facing API feedback flows through `MutationCache` / `QueryCache` in `src/queryClient.ts`, using the `NotificationMeta` interface in `src/types/tanstack-query.ts`.

Behavior:

- `MutationCache.onSuccess` — shows a green notification **only if `meta.successMessage` is provided**, then auto-invalidates `meta.invalidates` query keys.
- `MutationCache.onError` — shows a red notification with `meta.errorMessage` as the title + the error detail.
- `QueryCache.onError` — shows a red notification **only** for background refetch failures (query that previously had data). Initial load failures surface through route `errorComponent`.
- Both caches respect `meta.skipNotification` for background/silent operations.

When to set `successMessage`:

- **Set it** when success is not otherwise visible to the user (e.g. background action, same-page state mutation without list refresh).
- **Omit it** when the UI already reflects success: a refetched list, a closed modal, a navigation away. Silence is the default.

Read `mutation.meta` / `query.meta` as `NotificationMeta | undefined` — pnpm strict hoisting prevents module augmentation on `@tanstack/query-core`, so casts live at the read site.

---

## §5. Cache invalidation

Granularities, from widest to narrowest:

```ts
queryClient.invalidateQueries({ queryKey: projectKeys.all })         // all feature queries
queryClient.invalidateQueries({ queryKey: projectKeys.lists() })     // all list queries
queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) })  // one detail
```

Prefer the declarative form — set `meta.invalidates` on `mutationOptions`, let `MutationCache.onSuccess` invoke `invalidateQueries` for you. Avoid imperative `queryClient.invalidateQueries` in component code unless the scenario is genuinely outside the mutation lifecycle.

---

## §6. Optimistic updates

For simple UI feedback, prefer `mutation.variables` during pending state:

```tsx
const mutation = useMutation(updateProjectMutationOptions())
return <>{mutation.isPending ? mutation.variables.name : project.name}</>
```

For cache-level optimism, use the full pattern:

```ts
useMutation({
  mutationFn: updateProject,
  onMutate: async (newData) => {
    await queryClient.cancelQueries({ queryKey: projectKeys.detail(id) })
    const previous = queryClient.getQueryData(projectKeys.detail(id))
    queryClient.setQueryData(projectKeys.detail(id), (old) => ({ ...old, ...newData }))
    return { previous }
  },
  onError: (_err, _vars, context) => {
    queryClient.setQueryData(projectKeys.detail(id), context?.previous)
  },
  onSettled: () => {
    queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) })
  },
})
```

---

## §7. Form: schema-first + Standard Schema

Define the Zod schema first in `{feature}.schema.ts`. TanStack Form v1 accepts Standard Schema natively — pass Zod 4 schemas directly to field validators, no adapter, no manual `safeParse`.

```ts
// src/features/{feature}/{feature}.schema.ts
export const createProjectSchema = z.object({
  name: z.string().trim().min(1, 'Project name is required'),
  description: z.string().optional(),
})
export type CreateProjectInput = z.infer<typeof createProjectSchema>
```

Let Zod own error messages — do not hand-write error strings that duplicate what the schema expresses.

Use `fieldError(field)` from `@/shared/utils/form` to extract the first error for Mantine's `error` prop. Standard Schema errors are objects, not strings.

Use Mantine's own `label` and `withAsterisk` props for labelling. Never render a separate `<Text>` as a field label. For labels with inline help text, use `@/shared/components/FieldHintLabel`.

---

## §8. Form: Mantine binding

Keep Mantine as the field UI layer. Bind TanStack Form state onto Mantine component props.

```tsx
<form.Field name="name">
  {(field) => (
    <TextInput
      required
      label={t('projects.nameLabel')}
      value={field.state.value}
      onChange={(e) => field.handleChange(e.currentTarget.value)}
      onBlur={field.handleBlur}
      error={fieldError(field)}
    />
  )}
</form.Field>
```

`Select`, `Textarea`, `NumberInput`, `Switch`, `Checkbox` follow the same template — read `value` / `checked` from `field.state`, write via `field.handleChange`, wire `onBlur={field.handleBlur}` and `error={fieldError(field)}`. Switch/Checkbox use `checked` + `e.currentTarget.checked`. Select uses `(val) => field.handleChange(val ?? '')`.

---

## §9. Form + mutation integration

Let the form own validation and UI state; let the mutation own API call + error notification + cache invalidation.

```tsx
const mutation = useMutation(createProjectMutationOptions())

const form = useForm({
  defaultValues: { name: '', description: '' },
  validators: { onChange: createProjectSchema },
  onSubmit: async ({ value }) => {
    await mutation.mutateAsync(value)  // throws on error; form catches + resets isSubmitting
    onSuccess()
  },
})
```

- Do not add a `try/catch` around `mutateAsync` just to toast the error — `MutationCache.onError` already does it.
- Edit forms: pass existing entity fields as `defaultValues` (e.g. `name: project.name ?? ''`).
- Reading form state outside `<form.Field>`: prefer `<form.Subscribe selector={...}>` for render subscriptions; use `useStore(form.store, selector)` only when a non-Field hook/branch needs reactive access, and keep the selector narrow.
- `onBlur={field.handleBlur}` is required — without it, `onBlur` validators don't fire and touched-state tracking breaks.

---

## §10. Validation timing

| Timing     | When to use                                                             |
| ---------- | ----------------------------------------------------------------------- |
| `onChange` | Most text fields — immediate feedback                                   |
| `onBlur`   | Fields where real-time validation is distracting (email, complex regex) |
| `onSubmit` | Expensive validators, cross-field rules, server-side uniqueness         |

Async server checks compose naturally on the same field:

```tsx
<form.Field
  name="slug"
  validators={{
    onBlur: slugSchema,
    onChangeAsyncDebounceMs: 500,
    onChangeAsync: async ({ value, signal }) => {
      const exists = await checkSlugExists(value, signal)
      return exists ? 'Slug already taken' : undefined
    },
  }}
/>
```

---

## §11. Route: search params with Zod

Always validate search params with a Zod schema. Use `fallback` (from `@tanstack/router-zod-adapter`) for defaults so an invalid or missing value is coerced rather than throwing.

```ts
import { z } from 'zod'
import { fallback } from '@tanstack/router-zod-adapter'

export const projectsSearchSchema = z.object({
  q: fallback(z.string(), ''),
  sort: fallback(z.enum(['updatedAt', 'name']), 'updatedAt'),
  order: fallback(z.enum(['asc', 'desc']), 'desc'),
  page: fallback(z.number().int().min(1), 1),
})
export type ProjectsSearch = z.infer<typeof projectsSearchSchema>
```

```tsx
export const Route = createFileRoute('/(auth)/(app)/projects/')({
  validateSearch: projectsSearchSchema,
  // ...
})
```

---

## §12. Route: loaders and prefetching

| Page type | Typical loader          | Blocks navigation?                   | Typical component hook |
| --------- | ----------------------- | ------------------------------------ | ---------------------- |
| Detail    | `await ensureQueryData` | Yes — data ready before render       | `useSuspenseQuery`     |
| List      | `void prefetchQuery`    | No — component handles loading state | `useQuery`             |

Parallel detail fetches use `Promise.allSettled` so one failure does not block the others.

`loaderDeps` is mandatory when the loader reads search params — without it, the loader sees stale params on back-navigation.

---

## §13. Route: component data access

Match the hook to the loader strategy (§12):

- **`useSuspenseQuery`** — after `await ensureQueryData`. `data` is never `undefined`.
- **`useQuery`** — after `void prefetchQuery`, or without any prefetch. Handle `isPending` / `error` explicitly.

---

## §14. Route: hooks & context

Use the type-safe route-scoped hooks:

```tsx
const search = Route.useSearch()
const params = Route.useParams()
const navigate = Route.useNavigate()

navigate({ search: (prev) => ({ ...prev, page: 2 }) })
```

Outside the route component file, use `getRouteApi`:

```tsx
import { getRouteApi } from '@tanstack/react-router'
const routeApi = getRouteApi('/(auth)/(app)/projects/')
```

The router is created with `queryClient` in context (see `src/router.tsx`). Loaders receive it via `context.queryClient`. Do not create a second `QueryClient` anywhere.

Do not use raw `useLocation` or `window.location` for navigation — use `Route.useNavigate()` / `getRouteApi(...).useNavigate()`.

---

## §15. Route: error boundaries

Use `errorComponent` for route-specific error UI:

```tsx
errorComponent: ({ error }) => (
  <Alert color="red" title="Failed to load project">{error.message}</Alert>
),
```

Initial route load failures belong here, not in a notification.

---

## §16. Error layer map

```
Layer 1: Form validation     → inline field errors (Mantine `error` prop)
Layer 2: Mutation errors     → global notification (MutationCache.onError)
Layer 3: Query refetch errors → global notification (QueryCache.onError, only when cached data exists)
Layer 4: Route load errors   → route errorComponent (full-page error state)
```

Notification behavior matrix:

| Scenario                       | Notification                                        | Mechanism                 |
| ------------------------------ | --------------------------------------------------- | ------------------------- |
| Mutation succeeds              | Green with `meta.successMessage` (if set)           | `MutationCache.onSuccess` |
| Mutation fails                 | Red with `meta.errorMessage` + error detail         | `MutationCache.onError`   |
| Background query refetch fails | Red (data was previously cached)                    | `QueryCache.onError`      |
| Initial query load fails       | No notification — route `errorComponent` handles it | Route error boundary      |
| Form validation error          | Inline field error via Mantine `error` prop         | TanStack Form validators  |

Skip notifications for silent ops:

```ts
useMutation({ mutationFn: someBackgroundSync, meta: { skipNotification: true } })
queryOptions({ queryKey, queryFn, meta: { skipNotification: true } })
```

Ensure `@mantine/notifications/styles.css` is imported in `main.tsx` — without it, notifications render unstyled.

---

## §17. Tables

`mantine-react-table` v2 beta is the default table library. Follow the v2 docs at `https://v2.mantine-react-table.com`.

Peer deps kept in the project: `@mantine/dates`, `@tabler/icons-react`, `clsx`, `dayjs`. Do **not** add a direct `@tanstack/react-table` dependency — `mantine-react-table` brings the `@tanstack/table-core` it uses.

All data tables go through the project wrapper. If the wrapper does not yet cover your case, extend the wrapper first, then use it in the page. The wrapper owns pagination, loading, empty state, row actions, selection behavior, and shared styling.

Route-backed list pages hold search params, pagination navigation, refresh triggers, and row-selection state in the page layer or a shared hook — `src/shared/hooks/useRouteListState.ts`. The hook expects the search object to have `page` and `query` fields, and derives selected row ids + current-page selected records from `records` + `getRecordId`. Feature table components stay focused on columns, cells, and feature-specific toolbar actions.

Do not introduce a second table abstraction or a second table library for the same class of UI.

---

## §18. API SDK

The backend exposes gRPC via an HTTP/JSON gateway. The TypeScript SDK is generated under `api/ts/` at the repo root and imported in `ui/` via the `@matrixhub/api-ts/*` alias.

```ts
import type { ListUsersRequest, User } from '@matrixhub/api-ts/v1alpha1/user.pb'
import { Users } from '@matrixhub/api-ts/v1alpha1/user.pb'
```

- Generated `.pb.ts` files are read-only.
- Prefer SDK types over hand-written duplicates. Derive UI-specific shapes with `Pick` / `Omit` / explicit mapping — do not redefine.
- Use `import type` for type-only imports.
- proto3 fields are often optional — handle `undefined` in UI code.
- Call SDK methods directly when the SDK covers the endpoint. Add a wrapper only after a real shared need appears.

---

## §19. Consolidated Do Not

**Query / Mutation**

- Do not duplicate query options inline across components — extract to `queryOptions()`.
- Do not manually refetch where cache invalidation would suffice.
- Do not use per-hook `onSuccess` / `onError` callbacks on `useQuery` (removed in v5) — use the global `QueryCache` / `MutationCache` callbacks.
- Do not add explicit generics when `queryOptions()` already infers types.
- Do not place query/mutation logic inside route files — keep it in `src/features/{feature}/{feature}.{query,mutation}.ts`.
- Do not add per-component `notifications.show()` / `try-catch` for mutation errors — `MutationCache.onError` covers it.
- Do not show notifications for initial query failures — use route `errorComponent`.
- Do not forget `meta.invalidates` on mutations that change list data.
- Do not use `window.alert` or `console.error` as error feedback.
- Do not forget `@mantine/notifications/styles.css` in `main.tsx`.

**Form**

- Do not use Mantine `useForm`, uncontrolled patterns, or any other form library for new forms.
- Do not write inline validation when a Zod schema exists.
- Do not write manual `safeParse` calls in validators.
- Do not hand-write error strings that duplicate Zod messages.
- Do not render a separate `<Text>` as a field label.
- Do not store field values in `useState`; use TanStack Form.
- Do not skip `onBlur={field.handleBlur}`.
- Do not duplicate API error handling inside the form `onSubmit` catch.

**Router**

- Do not place complex page UI in route files — mount from `src/features`.
- Do not manually edit `src/routeTree.gen.ts`.
- Do not use raw `useLocation` / `window.location` for navigation.
- Do not skip `validateSearch` for routes with search params.
- Do not mix up loader strategies — detail pages use `await ensureQueryData`, list pages use `void prefetchQuery` (§12).
- Do not create a second `QueryClient`.

**API / SDK**

- Do not edit generated `.pb.ts` files.
- Do not write raw REST calls for endpoints the SDK covers.
- Do not import from `../api/ts/...` — use the `@matrixhub/api-ts/*` alias.
- Do not freeze the current SDK file tree as long-term documentation.
- Do not document speculative global patterns (shared `pathPrefix`, global error mapping, auth refresh) before the project adopts them.

**Tables**

- Do not wire `mantine-react-table` directly in a feature page — use the project wrapper.
- Do not add a direct `@tanstack/react-table` dependency.
- Do not introduce a second table library for the same class of UI without agreement.
