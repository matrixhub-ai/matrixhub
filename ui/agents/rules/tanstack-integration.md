# TanStack Integration & Error Notification Rules

This document defines how TanStack Query, TanStack Form, and TanStack Router work together, and how errors are surfaced to users via Mantine notifications.

## Architecture Overview

```
Router (navigation + prefetch)
  └─ loader: queryClient.ensureQueryData(queryOptions)
       └─ Component (UI)
            ├─ useSuspenseQuery(queryOptions)    → read data
            ├─ useMutation(mutationOptions)      → write data
            └─ useForm({ onSubmit: mutation })   → form → mutation
```

- **Router** prefetches data in loaders using `queryOptions`
- **Query** manages server state (reads via `useQuery`/`useSuspenseQuery`, writes via `useMutation`)
- **Form** manages local form state and validation, delegates submission to mutations
- **Notifications** are triggered globally by `QueryCache` and `MutationCache` — no per-component error handling needed

## Global Error Notification Setup

### Dependencies

Ensure `@mantine/notifications` is installed and its styles are imported:

```tsx
// main.tsx
import { Notifications } from '@mantine/notifications'
import '@mantine/notifications/styles.css'

// Inside MantineProvider:
<MantineProvider theme={mantineTheme} cssVariablesResolver={cssVariablesResolver}>
  <Notifications position="top-right" />
  <RouterProvider router={router} />
</MantineProvider>
```

### QueryClient with Global Caches

Configure `QueryCache` and `MutationCache` in `src/queryClient.ts` for centralized notification handling:

```ts
import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'
import { notifications } from '@mantine/notifications'
import { i18n } from './i18n'

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error)
}

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // Only notify for background refetch failures
      // (queries that already had data in cache)
      if (query.state.data !== undefined && !query.meta?.skipNotification) {
        notifications.show({
          title: i18n.t('common.backgroundRefreshFailed'),
          message: getErrorMessage(error),
          color: 'red',
        })
      }
    },
  }),
  mutationCache: new MutationCache({
    onSuccess: (_data, _variables, _context, mutation) => {
      const meta = mutation.meta
      if (meta?.skipNotification) return

      // Show success notification
      if (meta?.successMessage) {
        notifications.show({
          message: meta.successMessage,
          color: 'green',
        })
      }

      // Auto-invalidate related queries
      if (meta?.invalidates) {
        for (const key of meta.invalidates) {
          queryClient.invalidateQueries({ queryKey: key })
        }
      }
    },
    onError: (error, _variables, _context, mutation) => {
      const meta = mutation.meta
      if (meta?.skipNotification) return

      notifications.show({
        title: meta?.errorMessage ?? i18n.t('common.operationFailed'),
        message: getErrorMessage(error),
        color: 'red',
      })
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: false,
      retry: 0,
    },
  },
})
```

### Type Registration

Register the meta shape in `src/types/tanstack-query.d.ts`:

```ts
import '@tanstack/react-query'

declare module '@tanstack/react-query' {
  interface Register {
    defaultError: Error
    defaultMeta: {
      /** Notification message shown on success */
      successMessage?: string
      /** Notification title shown on error */
      errorMessage?: string
      /** Skip all notifications for this query/mutation */
      skipNotification?: boolean
      /** Query keys to invalidate on mutation success */
      invalidates?: readonly unknown[][]
    }
  }
}
```

## Full-Stack Feature Pattern

A complete feature follows this data flow. Use the `projects` feature as a reference.

### 1. Schema (`{feature}.schema.ts`)

```ts
import { z } from 'zod'

export const createProjectSchema = z.object({
  name: z.string().trim().min(1, 'Required'),
  description: z.string().optional(),
})

export type CreateProjectInput = z.infer<typeof createProjectSchema>
```

### 2. Query (`{feature}.query.ts`)

```ts
import { queryOptions } from '@tanstack/react-query'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'

export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (params: ProjectsSearch) => [...projectKeys.lists(), params] as const,
  details: () => [...projectKeys.all, 'detail'] as const,
  detail: (id: string) => [...projectKeys.details(), id] as const,
}

export function projectsQueryOptions(search: ProjectsSearch) {
  return queryOptions({
    queryKey: projectKeys.list(search),
    queryFn: () => Projects.ListProjects({ ... }),
  })
}

export function projectDetailQueryOptions(id: string) {
  return queryOptions({
    queryKey: projectKeys.detail(id),
    queryFn: () => Projects.GetProject({ name: id }),
  })
}
```

### 3. Mutation (`{feature}.mutation.ts`)

```ts
import { mutationOptions } from '@tanstack/react-query'
import { Projects } from '@matrixhub/api-ts/v1alpha1/project.pb'
import { projectKeys } from './project.query'

export function createProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (input: CreateProjectInput) =>
      Projects.CreateProject(input),
    meta: {
      successMessage: i18n.t('projects.createSuccess'),
      errorMessage: i18n.t('projects.createError'),
      invalidates: [projectKeys.lists()],
    },
  })
}

export function deleteProjectMutationOptions() {
  return mutationOptions({
    mutationFn: (id: string) =>
      Projects.DeleteProject({ name: id }),
    meta: {
      successMessage: i18n.t('projects.deleteSuccess'),
      errorMessage: i18n.t('projects.deleteError'),
      invalidates: [projectKeys.lists()],
    },
  })
}
```

### 4. Route (`src/routes/.../projects/index.tsx`)

```tsx
import { createFileRoute } from '@tanstack/react-router'
import { projectsSearchSchema } from '@/features/projects/project.schema'
import { projectsQueryOptions } from '@/features/projects/project.query'
import { ProjectsPage } from '@/features/projects/pages/ProjectsPage'

export const Route = createFileRoute('/(auth)/(app)/projects/')({
  validateSearch: projectsSearchSchema,
  loaderDeps: ({ search }) => ({ search }),
  loader: async ({ context: { queryClient }, deps: { search } }) => {
    await queryClient.ensureQueryData(projectsQueryOptions(search))
  },
  component: ProjectsPage,
})
```

### 5. Page Component (`src/features/.../pages/ProjectsPage.tsx`)

```tsx
import { useSuspenseQuery, useMutation } from '@tanstack/react-query'
import { projectsQueryOptions } from '../project.query'
import { deleteProjectMutationOptions } from '../project.mutation'

export function ProjectsPage() {
  const search = Route.useSearch()
  const { data } = useSuspenseQuery(projectsQueryOptions(search))
  const deleteMutation = useMutation(deleteProjectMutationOptions())

  // No try-catch needed — MutationCache handles error notifications
  const handleDelete = (id: string) => deleteMutation.mutate(id)

  return (/* ... */)
}
```

### 6. Form Component (`src/features/.../components/CreateProjectForm.tsx`)

```tsx
import { useForm } from '@tanstack/react-form'
import { useMutation } from '@tanstack/react-query'
import { createProjectSchema } from '../project.schema'
import { createProjectMutationOptions } from '../project.mutation'

function CreateProjectForm({ onSuccess }: { onSuccess: () => void }) {
  const mutation = useMutation(createProjectMutationOptions())

  const form = useForm({
    defaultValues: { name: '', description: '' },
    validators: { onChange: createProjectSchema },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync(value)
      onSuccess()
    },
  })

  return (
    <form onSubmit={(e) => { e.preventDefault(); form.handleSubmit() }}>
      {/* Mantine fields bound to TanStack Form — see tanstack-form.md */}
    </form>
  )
}
```

## Notification Behavior Summary

| Scenario                       | Notification                                             | Mechanism                 |
| ------------------------------ | -------------------------------------------------------- | ------------------------- |
| Mutation succeeds              | Green notification with `meta.successMessage`            | `MutationCache.onSuccess` |
| Mutation fails                 | Red notification with `meta.errorMessage` + error detail | `MutationCache.onError`   |
| Background query refetch fails | Red notification (data was previously cached)            | `QueryCache.onError`      |
| Initial query load fails       | No notification — route `errorComponent` handles it      | Route error boundary      |
| Form validation error          | Inline field error via Mantine `error` prop              | TanStack Form validators  |

### Skipping Notifications

For silent operations (background sync, polling):

```ts
useMutation({
  mutationFn: someBackgroundSync,
  meta: { skipNotification: true },
})
```

For queries that should not toast on refetch failure:

```ts
queryOptions({
  queryKey: ['...'],
  queryFn: () => ...,
  meta: { skipNotification: true },
})
```

## Error Handling Layers

```
Layer 1: Form validation     → inline field errors (Mantine error prop)
Layer 2: Mutation errors      → global notification (MutationCache)
Layer 3: Query refetch errors → global notification (QueryCache, only when cached data exists)
Layer 4: Route load errors    → route errorComponent (full-page error state)
```

Do not add `try-catch` blocks in components for API calls. Let the global caches handle notifications, and let route error boundaries handle page-level errors.

## Do Not

- Add per-component `notifications.show()` calls for mutation errors — use `MutationCache`
- Wrap mutation calls in `try-catch` just to show a toast — `MutationCache.onError` does this
- Show notifications for initial query failures — use route `errorComponent` instead
- Forget to add `meta.invalidates` on mutations that change list data
- Use `window.alert` or `console.error` as error feedback — always use Mantine notifications
- Forget `@mantine/notifications/styles.css` import — notifications will be unstyled without it
