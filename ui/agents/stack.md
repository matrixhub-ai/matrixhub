# Stack

Technology choices, global wirings, and day-to-day commands. Everything here is stable across tasks.

---

## Stack

- `pnpm` — package manager
- `Vite` — dev server / build
- `React 19`
- `TypeScript`
- `Mantine v8`
- `TanStack Router`
- `TanStack Form`
- `TanStack Query`
- `Zod`
- `react-i18next`
- `ESLint v9`

For how these libraries combine in code, see `patterns.md`. For visual/Mantine conventions, see `visual.md`.

---

## Project baselines

- `@tanstack/router-plugin` must be registered **before** `@vitejs/plugin-react` in `vite.config.ts`.
- `babel-plugin-react-compiler` is enabled and should not be removed casually.
- `VITE_UI_BASE_PATH` controls the deployed UI base path.
- `src/routeTree.gen.ts` is generated. **Never hand-edit it.** The TanStack Router plugin produces it.
- In hand-written code, fix lint and type issues directly. Do not suppress with broad directives (file-level `eslint-disable`, `eslint-disable-next-line`, `@ts-ignore`, `@ts-expect-error`) when a real fix is practical. If an exception is unavoidable, scope it to the narrowest rule and line possible and explain why. Generated files are the exception.
- If a change creates or modifies a shared wrapper, shared component convention, or other stable project pattern, update the relevant `ui/agents/` docs in the same PR.

---

## Global wirings (reference files)

- `src/main.tsx` — app entry, `@mantine/notifications/styles.css` import, `Notifications` mounted inside `MantineProvider`.
- `src/router.tsx` — the single `Router` instance with `queryClient` in context.
- `src/queryClient.ts` — `QueryClient` with `QueryCache` + `MutationCache` that drive global notifications via `NotificationMeta`.
- `src/mantineTheme.ts` — Mantine theme.
- `src/i18n/` — i18next bootstrap, language detection, day.js locale.

Do not create a second `QueryClient`. Loaders read it from `context.queryClient`.

---

## i18n

- When adding new user-facing copy, put it in locale files first — never a new hardcoded string in a component.
- When adding a new page, update **both `en` and `zh`** locale files.
- The path under `src/locales/{lang}/**/*.json` becomes the translation key prefix.
- Prefer `useTranslation()` inside components; the runtime lives in `src/i18n/`, not in business code.

---

## API SDK (summary)

The generated TypeScript SDK is imported via the `@matrixhub/api-ts/*` alias. Generated `.pb.ts` files are read-only. Full usage rules: see `patterns.md §18`.

---

## Default commands

```bash
pnpm dev        # dev server
pnpm build      # production build
pnpm lint       # ESLint
pnpm typecheck  # TypeScript
```

Before submitting, at minimum run `pnpm lint` and `pnpm typecheck` on the relevant parts of your change.
