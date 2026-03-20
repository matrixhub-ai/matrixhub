# TanStack Router Rules

This document defines the conventions for using TanStack Router v1 in this project. Also see `implementation.md` and `structure.md` for baseline routing rules.

## Route File Responsibilities

Route files in `src/routes/` are thin adapters. They own:

- `createFileRoute()` definition
- Search param validation via `validateSearch`
- Data prefetching via `loader`
- Redirects and `beforeLoad` guards
- Mounting the feature page component

Route files do **not** own complex UI, business logic, or large state management.

## Search Params with Zod

Always validate search params with Zod schemas. Use `fallback` for search params with defaults.

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

In the route file:

```tsx
export const Route = createFileRoute('/(auth)/(app)/projects/')({
  validateSearch: projectsSearchSchema,
  // ...
})
```

## Route Loaders and Data Prefetching

Use `ensureQueryData` in loaders to prefetch data. This fetches only if the cache is empty or stale, avoiding redundant requests on back-navigation.

```tsx
export const Route = createFileRoute('/(auth)/(app)/projects/$projectId')({
  loader: async ({ context: { queryClient }, params: { projectId } }) => {
    await queryClient.ensureQueryData(projectDetailQueryOptions(projectId))
  },
  component: ProjectDetailPage,
})
```

### Parallel Prefetching

When a route needs multiple data sources, fire them in parallel:

```tsx
loader: async ({ context: { queryClient }, params: { projectId } }) => {
  await Promise.allSettled([
    queryClient.ensureQueryData(projectDetailQueryOptions(projectId)),
    queryClient.ensureQueryData(projectModelsQueryOptions(projectId)),
  ])
}
```

Use `Promise.allSettled` (not `Promise.all`) so one failure does not block the others.

### Search-Dependent Loaders

When the loader depends on search params, use `loaderDeps` to tell the router which params to watch:

```tsx
export const Route = createFileRoute('/(auth)/(app)/projects/')({
  validateSearch: projectsSearchSchema,
  loaderDeps: ({ search }) => ({ search }),
  loader: async ({ context: { queryClient }, deps: { search } }) => {
    await queryClient.ensureQueryData(projectsQueryOptions(search))
  },
  component: ProjectsPage,
})
```

## Component Data Access

### With Prefetched Data (Suspense)

When the loader already prefetched the data, use `useSuspenseQuery` in the component. The data type will never be `undefined`.

```tsx
function ProjectDetailPage() {
  const { projectId } = Route.useParams()
  const { data } = useSuspenseQuery(projectDetailQueryOptions(projectId))
  // data is guaranteed non-undefined
}
```

### Without Prefetching

When the component fetches on its own (e.g. conditional queries), use `useQuery` and handle the loading/error states:

```tsx
function ProjectDetailPage() {
  const { projectId } = Route.useParams()
  const { data, isPending, error } = useQuery(projectDetailQueryOptions(projectId))

  if (isPending) return <LoadingOverlay visible />
  if (error) return <Alert color="red">{error.message}</Alert>
  // ...
}
```

## Route-Level Hooks

### useSearch / useParams / useNavigate

Use the type-safe route-scoped hooks:

```tsx
const search = Route.useSearch()
const params = Route.useParams()
const navigate = Route.useNavigate()

// Navigate with search param updates
navigate({
  search: (prev) => ({ ...prev, page: 2 }),
})
```

### getRouteApi

When you need route hooks outside the route component file, use `getRouteApi`:

```tsx
import { getRouteApi } from '@tanstack/react-router'

const routeApi = getRouteApi('/(auth)/(app)/projects/')

function ProjectsToolbar() {
  const search = routeApi.useSearch()
  const navigate = routeApi.useNavigate()
  // ...
}
```

## Router Context

The router is created with `queryClient` in context (see `src/router.tsx`). This makes `queryClient` available in all route loaders via `context.queryClient`.

Do not create a second `QueryClient`. Always use the one from router context.

## Route-Level Error Boundaries

Use `errorComponent` for route-specific error handling:

```tsx
export const Route = createFileRoute('/(auth)/(app)/projects/$projectId')({
  loader: async ({ context: { queryClient }, params: { projectId } }) => {
    await queryClient.ensureQueryData(projectDetailQueryOptions(projectId))
  },
  component: ProjectDetailPage,
  errorComponent: ({ error }) => (
    <Alert color="red" title="Failed to load project">
      {error.message}
    </Alert>
  ),
})
```

## Do Not

- Place complex page implementations in route files — mount feature pages from `src/features`
- Manually edit `src/routeTree.gen.ts` — it is auto-generated
- Use raw `useLocation` or `window.location` for navigation — use route-scoped `useNavigate`
- Skip `validateSearch` for routes with search params — always validate with Zod
- Use `prefetchQuery` in loaders when `ensureQueryData` suffices (avoid unnecessary refetches)
- Create a second `QueryClient` — use the one injected via router context
