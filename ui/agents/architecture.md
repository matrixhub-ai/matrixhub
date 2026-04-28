# Architecture

Where code lives in `ui/`. Read before writing any new file.

---

## Top-level layout

```
ui/src/
├── routes/          # Route adapters — file-based routing owned by TanStack Router
├── features/        # Feature implementations (pages, components, hooks, query/mutation/schema)
├── shared/          # Stable primitives reused across features
├── context/         # React contexts (current user, project role)
├── i18n/            # Runtime wiring: language detection, resource loading, dayjs locale
├── locales/         # Translation resources (en/, zh/), keys mirror their folder path
├── types/           # Cross-feature types (e.g. NotificationMeta)
├── utils/           # Generic utilities not tied to a feature
├── assets/          # Committed static assets; SVG icons under assets/svgs
├── theme/           # Theme extensions
├── main.tsx         # App entry
├── router.tsx       # Router instance (queryClient in context)
├── queryClient.ts   # QueryClient + QueryCache/MutationCache + global notifications
├── mantineTheme.ts  # Mantine theme
└── routeTree.gen.ts # GENERATED — never hand-edit
```

---

## Responsibility boundaries

### `src/routes/`

Owns only route adapter concerns:

- `createFileRoute(...)` definitions
- route-level layout, redirects, `beforeLoad`
- route params, search params, metadata
- `validateSearch` (Zod) and `loader` (prefetch via `ensureQueryData`)
- very simple static pages

Does **not** own complex page UI, business-flow orchestration, or large state. Every route file exports `Route`. See `patterns.md §11–§15` for the route-side protocol.

### `src/features/`

Owns feature-level pages, feature-local components/hooks/types/utils, and all page UI and business composition that does not belong in the route adapter.

Conventional shape — grow only as needed:

```
src/features/{feature}/
├── pages/                   # Page components mounted by routes
├── components/              # Feature-local reusable pieces
├── hooks/                   # Feature-local hooks
├── types/                   # Feature-local types
├── utils/                   # Feature-local utils
├── {feature}.query.ts       # queryKeys factory + queryOptions + custom hooks
├── {feature}.mutation.ts    # mutationOptions (when the feature writes)
└── {feature}.schema.ts      # Zod schemas + derived input/output types
```

**Composite features — nested sub-domains.** When a feature has a distinct sub-domain (e.g. members inside a project), nest it as its own sub-feature with the same shape:

```
src/features/projects/
├── projects.query.ts
├── projects.mutation.ts
├── pages/
├── components/
└── members/
    ├── members.query.ts
    ├── members.mutation.ts
    ├── components/
    └── pages/
```

Reference: `src/features/projects/members/`.

### `src/shared/`

Stable primitives reused across ≥2 features. Same folder shape as a feature (`components/`, `hooks/`, `types/`, `utils/`). Examples that already live here: `DataTable`, `SearchToolbar`, `Pagination`, `ModalWrapper`, `useRouteListState`, `useForm`, `usePayloadModal`.

**Shared table wrapper** goes under `src/shared/components/data-table/` (or the current shared location if it later moves). Centralise pagination, loading/empty states, row actions, selection behaviour, and shared styling inside the wrapper — feature tables stay focused on columns, cells, and feature-specific toolbar actions.

Only promote code to `src/shared/` once reuse is clear. Do not pre-emptively abstract.

### `src/i18n/` and `src/locales/`

- `src/i18n/` owns the runtime wiring (bootstrap, language detection, resource loading, dayjs locale). It is not a home for business copy.
- `src/locales/` owns translation resources organized by language. The path under `src/locales/{lang}/**/*.json` becomes the translation key prefix.

---

## File naming

- `{feature}.query.ts` — query key factory + queryOptions + (rare) custom hooks
- `{feature}.mutation.ts` — mutationOptions for create / update / delete
- `{feature}.schema.ts` — Zod schemas and derived types
- Page components live in `{feature}/pages/{Feature}Page.tsx`
- Form components live in `{feature}/components/{Feature}Form.tsx` (or under `pages/` when not reused)

---

## Feature page splitting

Split by responsibility and complexity, not by file length.

Split proactively when a single page file carries multiple distinct concerns: complex list rendering mixed with forms or dialogs; page composition mixed with heavy state or side effects; data loading/submission mixed with large UI blocks.

- Extract independently understandable complex sections into the current feature's `components/`.
- Extract complex state and side effects into the current feature's `hooks/`.
- Only move code into `shared/` when reuse across features is clear or a stable common pattern has emerged.
- Do not split for the sake of splitting. The goal is to reduce reading and modification cost.

---

## Route ↔ feature mapping example

For the projects list page:

```
src/routes/(auth)/(app)/projects/index.tsx     ← adapter (validateSearch + loader + mount)
src/features/projects/pages/ProjectsPage.tsx   ← page implementation
src/locales/en/projects.json                   ← en copy
src/locales/zh/projects.json                   ← zh copy
```

The route mounts the page. The page owns the UI.

---

## Incremental migration

- Simple placeholder UI inside existing route files may stay temporarily.
- For any new non-trivial page, put the implementation in `src/features/` and mount it from `src/routes/`.
- If a route file you are already touching has accumulated a large amount of UI, extract a feature page while you are there.
- Do not introduce new top-level architecture layers just to make the structure look more complete.

---

## When a shared change lands

If a change creates or modifies a shared wrapper, shared component convention, or other stable project pattern, update the relevant `ui/agents/` docs in the **same** PR. Shared primitives without docs drift fastest.
