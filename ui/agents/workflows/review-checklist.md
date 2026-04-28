# Review Checklist

Used in two situations:

1. **Author self-check** before opening a PR.
2. **Reviewer standard** during review — human or AI (e.g. GitHub PR review bots).

Every item below is phrased as a verifiable predicate. Tick or fail each one against the actual diff.

---

## Scope & Placement

- [ ] The change scope is small enough to describe in one sentence.
- [ ] If a new page was added, page planning was done first (see the relevant archetype under `ui/agents/archetypes/`).
- [ ] Route files contain only route concerns (`createFileRoute`, `validateSearch`, `loader`, mount) — no complex UI or business logic.
- [ ] Non-trivial page UI lives in `src/features/{feature}/`, not in `src/routes/`.
- [ ] `src/routeTree.gen.ts` has no hand-written diffs.

## Data Flow

- [ ] New queries/mutations live in `src/features/{feature}/{feature}.{query,mutation}.ts` — not inline in route or page code.
- [ ] Query keys come from a hierarchical factory (`features/{feature}/{feature}.query.ts`).
- [ ] Every mutation that changes list data sets `meta.invalidates`.
- [ ] Every user-triggered mutation sets `meta.errorMessage`. `meta.successMessage` is present only when the UI wouldn't otherwise communicate success.
- [ ] No per-component `try/catch` / `notifications.show()` for mutation errors — `MutationCache.onError` handles them.
- [ ] Initial-load failures surface via route `errorComponent`, not via a notification.

## Forms

- [ ] Forms use `@tanstack/react-form` + Zod schema — no Mantine `useForm`, no `useState` for field values, no other form library.
- [ ] Zod schemas are passed directly to field `validators`; no manual `safeParse` in validators.
- [ ] `fieldError(field)` from `@/shared/utils/form` feeds every Mantine `error` prop.
- [ ] Every bound field has `onBlur={field.handleBlur}`.
- [ ] No separate `<Text>` used as a field label — Mantine `label` / `withAsterisk` or `FieldHintLabel` is used instead.
- [ ] No hand-written error strings that duplicate Zod messages.

## Tables

- [ ] Data tables go through the project wrapper (`src/shared/components/DataTable.tsx` or the current shared location). `mantine-react-table` is not wired directly in a feature page.
- [ ] No direct `@tanstack/react-table` dependency was added.
- [ ] When the table needs row selection + batch operations, pagination / search / selection state uses `useRouteListState`. Read-only or single-action tables may manage state directly in the page.

## Visual

- [ ] Colors follow the decision order: component-native semantics → project semantic tokens → Mantine semantic tokens → palette-index (last resort). No mechanical copy of Figma hex colors when a semantic token applies.
- [ ] Spacing/font-size/color are not hardcoded when a Mantine prop or theme token covers the intent.
- [ ] New icon SVGs were only added after confirming no `@tabler/icons-react` match exists. Committed SVG assets are under `src/assets/svgs`.

## i18n

- [ ] Every new user-facing string is in `src/locales/en/**/*.json` **and** `src/locales/zh/**/*.json`.
- [ ] No new hardcoded display copy was added to components.
- [ ] Translation keys match the folder path under `src/locales/{lang}/`.

## Shared Primitives

- [ ] If a shared wrapper, shared component convention, or other stable project pattern was created or changed, the corresponding `ui/agents/` doc was updated in **this** PR.

## Basic Validation

- [ ] `pnpm lint` passes on the relevant parts of the change.
- [ ] `pnpm typecheck` passes on the relevant parts of the change.
- [ ] No broad suppressions (`eslint-disable`, `@ts-ignore`, `@ts-expect-error`) were added. If an exception was unavoidable, it is scoped to the narrowest rule + line and explained inline.
- [ ] Any work that could not be validated locally is called out explicitly in the PR description.

---

## When this checklist itself should be updated

- The default stack changes.
- Directory boundaries in `architecture.md` change.
- A new stable convention is adopted project-wide.
- A new archetype is added under `ui/agents/archetypes/`.
