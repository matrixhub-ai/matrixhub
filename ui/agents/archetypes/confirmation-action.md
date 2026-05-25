# Archetype: Confirmation Action

A destructive action (delete, disable, reset) triggered from a row or toolbar button, guarded by a Mantine confirmation modal.

---

## Reference implementation

| File | Role |
|---|---|
| `src/features/admin/users/components/DeleteUserAction.tsx` | Row-level delete with confirmation |
| `src/features/admin/registries/components/DeleteRegistryAction.tsx` | Row-level delete with confirmation |
| `src/features/admin/replications/components/DeleteReplicationAction.tsx` | Row-level delete with confirmation |
| `src/features/projects/components/DeleteProjectModal.tsx` | Project-level delete |
| `src/features/projects/members/components/RemoveMemberModal.tsx` | Remove member with role context |

---

## Required patterns

From `patterns.md`:

- §3 `mutationOptions` factory — keep the delete mutation in `{feature}.mutation.ts`
- §4 `NotificationMeta` — set `errorMessage`; set `successMessage` only when the list will not visibly refresh
- §5 Cache invalidation — declarative `meta.invalidates`, widest correct scope

---

## Archetype-specific gotchas

- **Show the target identity in the modal body.** Name / id / role — whatever lets a reviewer confirm they are deleting the right thing.
- **Disable the confirm button while `mutation.isPending`.** Do not leave the user clicking repeatedly during the request.
- **Close the modal after `await mutation.mutateAsync(...)` resolves.** Errors bubble up and `MutationCache.onError` toasts; success naturally closes.
- **Invalidate the narrowest correct key.** Deleting a single row invalidates the list (`keys.lists()`), not everything (`keys.all`). Use `keys.all` only when multiple dependent lists exist.
- **Success message is usually unnecessary.** A disappearing row is the UX confirmation. Only set `meta.successMessage` when the deletion is invisible (e.g. background cleanup with no list refetch).

---

## Must answer before coding

1. Is the action reversible? If not, the modal body must make the consequence plain.
2. What is the scope of invalidation — one list, all lists for the feature, or cross-feature?
3. Does the action affect the currently-viewed detail page (then navigate away on success)?
4. Is there a permission constraint — should the action button be hidden rather than disabled?

---

## Self-check before handoff

Subset of `workflows/review-checklist.md`:

- Delete mutation lives in `{feature}.mutation.ts`, with `errorMessage` + `invalidates` in `meta`.
- Confirm button is disabled while `isPending`.
- The modal body shows the target's identity.
- No `try/catch` wrapping `mutateAsync` just to toast the error.

---

## Known duplication — future abstraction

The confirm-button + internal modal + mutation wiring repeats across 9+ action files. When a shared `useActionModal` / `<ConfirmationActionButton>` lands, update this archetype file **in the same PR** (per the shared-changes rule in `AGENTS.md`).
