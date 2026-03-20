# TanStack Query Rules

This document defines the conventions for using TanStack Query v5 in this project.

## Query Key Factory

Every feature must define a query key factory object. Keys are hierarchical arrays enabling granular cache invalidation.

```ts
// src/features/{feature}/{feature}.query.ts

export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params: ProjectsSearch) =>
    [...projectKeys.lists(), params] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (projectId: string) =>
    [...projectKeys.details(), projectId] as const,
}
```

Use `as const` on every key array for literal type narrowing.

## Query Options Factory

Always extract query configuration into `queryOptions()` functions. This is the single most important pattern for type safety and reuse across `useQuery`, `useSuspenseQuery`, `ensureQueryData`, and `setQueryData`.

```ts
import { queryOptions } from '@tanstack/react-query'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'

export function projectDetailQueryOptions(projectId: string) {
  return queryOptions({
    queryKey: projectKeys.detail(projectId),
    queryFn: () => Projects.GetProject({ name: projectId }),
  })
}

export function projectsQueryOptions(search: ProjectsSearch) {
  return queryOptions({
    queryKey: projectKeys.list(search),
    queryFn: () => Projects.ListProjects({ ... }),
  })
}
```

Do not pass explicit generics to `useQuery<Data, Error>()`. Let TypeScript infer types from `queryOptions()`.

## Custom Hooks

Wrap `useQuery` in a custom hook only when you need to compose additional options like `placeholderData` or conditional enabling.

```ts
export function useProjects(search: ProjectsSearch) {
  return useQuery({
    ...projectsQueryOptions(search),
    placeholderData: keepPreviousData,
  })
}
```

For route components where data is prefetched in the loader, prefer `useSuspenseQuery` directly in the component (no custom hook wrapper needed).

## Mutation Options Factory

Use `mutationOptions()` to extract mutation configuration. Attach `meta` for toast behavior and auto-invalidation (see `tanstack-integration.md`).

```ts
import { mutationOptions } from '@tanstack/react-query'

export function deleteProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (projectId: string) =>
      Projects.DeleteProject({ name: projectId }),
    meta: {
      successMessage: t('projects.deleteSuccess'),
      errorMessage: t('projects.deleteError'),
      invalidates: [projectKeys.lists()],
    },
  })
}
```

## Notification Meta Type

The `NotificationMeta` interface in `src/types/tanstack-query.ts` defines the shared meta shape for query and mutation notification behavior. Use `as NotificationMeta` when reading `mutation.meta` or `query.meta` in the global caches (see `src/queryClient.ts`).

When passing `meta` to `mutationOptions()` or `queryOptions()`, use the `NotificationMeta` type to ensure consistency:

```ts
import type { NotificationMeta } from '@/types/tanstack-query'

export function deleteProjectMutationOptions(): ... {
  return mutationOptions({
    mutationFn: ...,
    meta: {
      successMessage: '...',
      errorMessage: '...',
      invalidates: [projectKeys.lists()],
    } satisfies NotificationMeta,
  })
}
```

## Cache Invalidation

- `queryClient.invalidateQueries({ queryKey: projectKeys.all })` — invalidate everything for the feature
- `queryClient.invalidateQueries({ queryKey: projectKeys.lists() })` — invalidate all list queries
- `queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) })` — invalidate a single detail
- Prefer declarative invalidation through mutation `meta.invalidates` (handled globally by `MutationCache`)

## Optimistic Updates

For simple UI feedback, prefer `mutation.variables` during pending state:

```tsx
const mutation = useMutation(updateProjectMutationOptions())

// In JSX:
{mutation.isPending ? mutation.variables.name : project.name}
```

For complex cache manipulation, use the `onMutate` / `onError` / `onSettled` pattern:

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

## Do Not

- Duplicate query options inline across multiple components — extract to `queryOptions()`
- Manually refetch where cache invalidation would suffice
- Use per-hook `onSuccess` / `onError` callbacks on `useQuery` (removed in v5) — use the global `QueryCache` / `MutationCache` callbacks instead
- Add explicit generics when `queryOptions()` already provides type inference
- Place query/mutation logic inside route files — keep it in `src/features/{feature}/{feature}.query.ts`

## File Naming

```
src/features/{feature}/
  {feature}.query.ts     # query key factory + queryOptions + custom hooks
  {feature}.mutation.ts  # mutationOptions (when the feature has mutations)
```
