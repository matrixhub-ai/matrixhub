# Archetype: Modal Form

A button opens a Mantine `Modal` containing a TanStack Form bound to a mutation. Used for create / edit flows.

---

## Reference implementation

| File | Role |
|---|---|
| `src/features/projects/components/CreateProjectModal.tsx` | Create form, Zod validation, calls create mutation |
| `src/features/projects/members/components/AddMemberModal.tsx` | Create form with select + async validation |
| `src/features/admin/registries/components/RegistryFormModal.tsx` | Create / edit reuse via `defaultValues` |
| `src/shared/components/ModalWrapper.tsx` | Shared modal chrome |
| `src/shared/utils/form.ts` → `fieldError` | First-error extractor for Mantine `error` prop |
| `src/shared/components/FieldHintLabel.tsx` | Label with inline help text |

---

## Required patterns

From `patterns.md`:

- §7 Form: schema-first + Standard Schema — `{feature}.schema.ts`, `fieldError`, Mantine `label` / `withAsterisk`
- §8 Form: Mantine binding — one `TextInput` example covers the template for Select / Textarea / NumberInput / Switch / Checkbox
- §9 Form + mutation integration — `mutateAsync` in `onSubmit`, `form.Subscribe` for submit-button state
- §10 Validation timing — `onChange` for most text fields, `onBlur` for formats, `onSubmit`/`onChangeAsync` for server checks

---

## Archetype-specific gotchas

- **Reset on open/close.** When the modal reopens, reset the form: call `form.reset()` from a `useEffect` keyed on the `opened` prop. Forgetting this leaks stale values between sessions.
- **Do not duplicate error handling.** `MutationCache.onError` already toasts the error — do not add a `try/catch` in the form `onSubmit`.
- **Success behavior.** On success, close the modal in the `onSubmit` callback after `await mutation.mutateAsync(...)`. Omit `meta.successMessage` unless the UX needs a toast on top of the closing modal.
- **Disabling the submit button.** Use `<form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting, s.isPristine]}>`. Do not read `form.state` imperatively for rendering.
- **`onBlur={field.handleBlur}` is required.** Without it, blur-time validators never fire and touched-state tracking breaks.
- **Edit forms** pass current entity data as `defaultValues`, using `?? ''` for nullable strings.

---

## Must answer before coding

1. Is this create-only, edit-only, or a shared create/edit shape? If shared, which fields differ?
2. Which fields need server-side validation (uniqueness, permission)?
3. Which mutation keys must invalidate on success?
4. What is the modal layout — single-column `Stack`, two-column via `SimpleGrid`, or tabbed?

---

## Self-check before handoff

Subset of `workflows/review-checklist.md`:

- Form uses `@tanstack/react-form` + Zod schema — no Mantine `useForm`, no `useState` for field values.
- `fieldError(field)` feeds Mantine `error` props.
- `onBlur={field.handleBlur}` is present on every bound field.
- `form.reset()` runs when the modal opens.
- No `try/catch` around `mutateAsync` just to toast errors.

---

## Known duplication — future abstraction

The lifecycle pattern — `form.reset()` + `useEffect` on `opened` + `useStore` selectors for submit button state — repeats across create/edit modals in this repo. When a shared `useModalForm` or `<ModalForm>` lands, update this archetype file **in the same PR** (per the shared-changes rule in `AGENTS.md`).
