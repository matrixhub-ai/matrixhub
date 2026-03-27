# Implementation Constraints

This document combines the current UI stack and the key rules around Mantine, TanStack Router, TanStack Form, Zod, table implementation, and i18n.

## Stack

- `pnpm`
- `Vite`
- `React 19`
- `TypeScript`
- `Mantine v8`
- `TanStack Router`
- `TanStack Form`
- `TanStack Query`
- `Zod`
- `react-i18next`
- `ESLint v9`

## Project Baselines

- `@tanstack/router-plugin` must be registered before `@vitejs/plugin-react`
- `babel-plugin-react-compiler` is enabled and should not be removed casually
- `VITE_UI_BASE_PATH` controls the deployed UI base path
- `src/routeTree.gen.ts` is generated and must only be produced by tooling
- If an agent creates, changes, or expands a shared wrapper, shared component convention, or other stable implementation pattern, the agent must update the relevant docs under `ui/agents/` in the same change
- In hand-written code, fix lint and type issues directly instead of suppressing them with broad directive comments. If an exception is truly unavoidable, scope it to the narrowest rule and line possible and explain why. Generated files are the exception.

## Mantine

- Prefer Mantine layout primitives such as `AppShell`, `Stack`, `Group`, `Flex`, and `Box`
- Prefer `Text` and `Title` for text rendering
- Prefer Mantine props and theme tokens for spacing
- When styling Mantine components that already express state or hierarchy, first use the component's own semantics and stateful API before adding explicit colors
- Prefer semantic Mantine color tokens or CSS variables such as `var(--mantine-primary-color-filled)`, `var(--mantine-primary-color-light)`, `var(--mantine-color-text)`, and `var(--mantine-color-dimmed)` over palette-index references like `var(--mantine-color-cyan-6)` or `gray.7`
- If the design system already defines an exact semantic token for the target color, use that exact project token first instead of approximating with a nearby Mantine semantic token
- Only fall back to palette-index color references when no semantic Mantine token matches the design intent, because indexed colors adapt poorly when dark theme support is added later
- When implementing from Figma, decide color in this order: component-native semantics, existing semantic tokens, then explicit Figma-driven color overrides only if the design intent is still not represented
- Do not copy Figma colors mechanically when the same intent is already covered by component defaults or existing semantic tokens
- Do not pile up raw `div`s or inline styles where Mantine already covers the use case
- Do not hardcode colors, font sizes, or spacing values as the default approach

## Figma Assets

- Committed SVG assets belong under `src/assets/svgs`
- Before committing a new icon SVG from Figma, first check whether `@tabler/icons-react` already provides a suitable icon and prefer the package when it does
- Only add a committed SVG icon when no suitable Tabler icon exists, or when the asset is brand-specific and must preserve its exact shape
- When a Figma icon has no suitable Tabler equivalent, confirm the download or save step before adding a new SVG asset
- Do not leave reusable SVG markup inline in route or component files unless the SVG must be generated dynamically
- Temporary exports and comparisons may live under `.planning/<task>/`

## Router

Baseline rules — see `tanstack-router.md` for full conventions.

- All route files belong in `src/routes`
- Every route file should explicitly export `Route`
- Route files own route definitions, layouts, redirects, and metadata
- Non-trivial pages should live in `src/features` and be mounted by the route
- Do not let complex business logic stay in route files long term

## Data Fetching

See `tanstack-query.md` for full conventions. Key rules:

- Use `queryOptions()` / `mutationOptions()` factories — never inline query config
- Use query key factory objects per feature
- Use `MutationCache` / `QueryCache` for global error/success notifications via Mantine — see `tanstack-integration.md`

## Forms

Baseline rules — see `tanstack-form.md` for full conventions and Mantine binding patterns.

- Use `@tanstack/react-form` for all forms in this project
- Use `Zod` as the default validation and schema definition approach for form data; let Zod provide the error messages — do not hand-write error strings that duplicate what the schema already expresses
- Do not use uncontrolled forms, native ad-hoc form state, Mantine `useForm`, or other form libraries for new form work
- Pass Zod schemas directly to field-level `validators` (e.g. `validators: { onSubmit: schema }`) — TanStack Form v1 supports Standard Schema natively, so Zod schemas work without an adapter; do not write manual `safeParse` calls
- Use `fieldError(field)` from `@/shared/utils/form` to extract the first error message for Mantine `error` props — Standard Schema errors are complex objects, not strings
- Use Mantine's own `label` and `withAsterisk` props on form field components for labelling — do not render separate `<Text>` elements as field labels
- When a field label needs inline help text, prefer `@/shared/components/FieldHintLabel` and let it inherit the host component's native label styling instead of hand-building page-local tooltip labels
- Keep Mantine as the field UI layer; bind TanStack Form state to Mantine component props such as `value`, `checked`, `onChange`, and `error`
- When reading TanStack Form state outside a `form.Field`, prefer `form.Subscribe` for render-only UI subscriptions; use `useStore(form.store, selector)` only when a hook or non-Field branch needs reactive access, and keep the selector as narrow as practical
- Connect form `onSubmit` to TanStack Query mutations — see `tanstack-integration.md` for the full pattern

## Error Notifications

See `tanstack-integration.md` for the complete error handling strategy.

- All API errors must surface through Mantine notifications (`@mantine/notifications`)
- Mutation errors: handled globally by `MutationCache.onError`
- Background query refetch errors: handled globally by `QueryCache.onError`
- Route load errors: handled by route `errorComponent`
- Form validation errors: shown inline via Mantine component `error` prop
- Do not add per-component `try-catch` or `notifications.show()` for API errors

## Tables

- Use `mantine-react-table` v2 beta as the current default library for new data-table work in this project
- Follow the official v2 docs at `https://v2.mantine-react-table.com` for API shape and setup
- Install the current table stack with `mantine-react-table@beta` and its direct peer dependencies used by this project: `@mantine/dates`, `@tabler/icons-react`, `clsx`, and `dayjs`
- Do not add a direct `@tanstack/react-table` dependency just to use `mantine-react-table`; the wrapped table package already brings the TanStack table dependency it uses
- Any place that needs a data table should use the project's wrapped table component or adapter instead of wiring `mantine-react-table` directly in the page
- If the required wrapper does not exist yet, create or extend the project table wrapper first, then use that wrapper in the page
- Put stable shared table wrappers under `src/shared/components/data-table/` unless the project later adopts a different shared location
- Centralize shared table styling and behavior in the wrapper, including pagination, loading states, empty states, row actions, selection behavior, and similar concerns
- For route-backed list pages that use the shared table wrapper, keep query/search params, pagination navigation, refresh triggers, and row-selection state in the page layer or a shared hook such as `src/shared/hooks/useRouteListState.ts`; the shared hook expects the search object to have `page` / `query` fields and derives selected row ids and current-page selected records from `records` + `getRecordId`; feature table components should stay focused on columns, cells, and feature-specific toolbar/actions
- Do not introduce a different table abstraction or second table library for the same class of UI without agreement

## Feature Page Splitting Guidelines

These rules apply to page implementations under `src/features/{feature}/pages`.

- Split page implementation by responsibility and complexity, not by file length
- When a single page file carries multiple distinct complex concerns, split it proactively. Typical cases include complex list rendering mixed with forms or dialogs, page composition mixed with heavy state or side effects, or data loading/submission mixed with large UI blocks
- Extract independently understandable complex sections into the current feature's `components/`
- Extract complex state and side effects into the current feature's `hooks/`
- Only move code into `shared` when it is clearly reused across features or has already become a stable common pattern
- Do not split for the sake of splitting. The goal is to reduce reading and modification cost

## i18n

- When adding new user-facing copy, put it in locale files first
- When adding a new page, update both `en` and `zh`
- The path under `src/locales/{lang}/**/*.json` becomes the translation key prefix
- Prefer `useTranslation()` inside components
- Do not keep spreading new hardcoded display copy

## Default Commands

```bash
pnpm dev
pnpm build
pnpm lint
pnpm typecheck
```
