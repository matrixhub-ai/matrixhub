# TanStack Form Rules

This document defines the conventions for using TanStack Form v1 in this project. Also see the form section in `implementation.md` for baseline rules.

## Schema-First Validation

Define Zod schemas first, then use them as validators. Do not duplicate validation logic by hand.

```ts
// src/features/{feature}/{feature}.schema.ts

import { z } from 'zod'

export const createProjectSchema = z.object({
  name: z.string().trim().min(1, 'Project name is required'),
  description: z.string().optional(),
})

export type CreateProjectInput = z.infer<typeof createProjectSchema>
```

## Basic Form Pattern

Bind TanStack Form state to Mantine components. Keep Mantine as the field UI layer.

```tsx
import { useForm } from '@tanstack/react-form'
import { TextInput, Textarea, Button, Stack } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { createProjectSchema } from '../project.schema'
import { createProjectMutationOptions } from '../project.mutation'

function CreateProjectForm({ onSuccess }: { onSuccess: () => void }) {
  const { t } = useTranslation()
  const mutation = useMutation(createProjectMutationOptions())

  const form = useForm({
    defaultValues: { name: '', description: '' },
    validators: {
      onChange: createProjectSchema,
    },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync(value)
      onSuccess()
    },
  })

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        form.handleSubmit()
      }}
    >
      <Stack>
        <form.Field name="name">
          {(field) => (
            <TextInput
              required
              label={t('projects.nameLabel')}
              value={field.state.value}
              onChange={(e) => field.handleChange(e.currentTarget.value)}
              onBlur={field.handleBlur}
              error={field.state.meta.errors?.[0]?.toString()}
            />
          )}
        </form.Field>

        <form.Field name="description">
          {(field) => (
            <Textarea
              label={t('projects.descriptionLabel')}
              value={field.state.value}
              onChange={(e) => field.handleChange(e.currentTarget.value)}
              onBlur={field.handleBlur}
              error={field.state.meta.errors?.[0]?.toString()}
            />
          )}
        </form.Field>

        <form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting, s.isPristine]}>
          {([canSubmit, isSubmitting, isPristine]) => (
            <Button
              type="submit"
              disabled={!canSubmit || isPristine}
              loading={isSubmitting}
            >
              {t('common.create')}
            </Button>
          )}
        </form.Subscribe>
      </Stack>
    </form>
  )
}
```

## Validation Timing

Choose the appropriate validation timing based on field type:

| Timing     | When to Use                                                                     |
| ---------- | ------------------------------------------------------------------------------- |
| `onChange` | Most text fields — immediate feedback                                           |
| `onBlur`   | Fields where real-time validation is distracting (e.g. email, complex patterns) |
| `onSubmit` | Expensive validators, cross-field rules, or server-side uniqueness checks       |

Field-level and form-level validators can be combined:

```tsx
<form.Field
  name="slug"
  validators={{
    onBlur: ({ value }) => {
      const result = slugSchema.safeParse(value)
      return result.success ? undefined : result.error.issues[0]?.message
    },
    onChangeAsyncDebounceMs: 500,
    onChangeAsync: async ({ value, signal }) => {
      const exists = await checkSlugExists(value, signal)
      return exists ? 'Slug already taken' : undefined
    },
  }}
>
  {(field) => (
    <TextInput
      label={t('projects.slugLabel')}
      value={field.state.value}
      onChange={(e) => field.handleChange(e.currentTarget.value)}
      onBlur={field.handleBlur}
      error={field.state.meta.errors?.[0]?.toString()}
    />
  )}
</form.Field>
```

## Form + Mutation Integration

Connect TanStack Form's `onSubmit` to TanStack Query mutations. The mutation handles API calls, error notifications, and cache invalidation. The form handles validation and UI state.

```tsx
const form = useForm({
  defaultValues: { name: '', description: '' },
  validators: { onChange: createProjectSchema },
  onSubmit: async ({ value }) => {
    // mutateAsync throws on error, which TanStack Form catches
    // and sets form.state.isSubmitting = false
    await mutation.mutateAsync(value)
    onSuccess()
  },
})
```

Do not duplicate error handling in both the form `onSubmit` and the mutation — let the global `MutationCache` handle error notifications.

## Edit Forms with Existing Data

For edit forms, pass the current entity data as `defaultValues`:

```tsx
const form = useForm({
  defaultValues: {
    name: project.name ?? '',
    description: project.description ?? '',
  },
  validators: { onChange: updateProjectSchema },
  onSubmit: async ({ value }) => {
    await updateMutation.mutateAsync({ projectId, ...value })
    onSuccess()
  },
})
```

## Mantine Component Binding Patterns

### TextInput / Textarea / NumberInput

```tsx
<TextInput
  value={field.state.value}
  onChange={(e) => field.handleChange(e.currentTarget.value)}
  onBlur={field.handleBlur}
  error={field.state.meta.errors?.[0]?.toString()}
/>
```

### Select

```tsx
<Select
  data={options}
  value={field.state.value}
  onChange={(val) => field.handleChange(val ?? '')}
  onBlur={field.handleBlur}
  error={field.state.meta.errors?.[0]?.toString()}
/>
```

### Switch / Checkbox

```tsx
<Switch
  checked={field.state.value}
  onChange={(e) => field.handleChange(e.currentTarget.checked)}
  error={field.state.meta.errors?.[0]?.toString()}
/>
```

## Do Not

- Use Mantine `useForm` or uncontrolled form patterns — use `@tanstack/react-form`
- Write inline validation logic when a Zod schema exists — reuse the schema
- Duplicate API error handling in form `onSubmit` catch blocks — rely on `MutationCache`
- Create form state outside TanStack Form (e.g. `useState` for field values)
- Skip `onBlur` binding — it is required for blur-time validators and touched state tracking

## File Naming

```
src/features/{feature}/
  {feature}.schema.ts    # Zod schemas and derived types
```

Form components live in the feature's `components/` or `pages/` as appropriate.
