# Directory Structure And Responsibility Boundaries

This document defines the target structure of `ui/` and the direction for incremental migration.

## Current Stable Boundaries

- `src/routes`: route adapter layer
- `src/features`: feature folders and non-trivial page implementation
- `src/i18n`: i18n bootstrap, language detection, and resource loading
- `src/locales`: translation resources
- `src/main.tsx`, `src/router.tsx`, `src/mantineTheme.ts`: app-level infrastructure

## Recommended Responsibilities

### `src/routes`

Owns:

- `createFileRoute(...)`
- route-level layout
- `redirect` and `beforeLoad`
- route params, search params, and metadata
- very simple static pages

Does not own:

- new complex page UI
- business-flow orchestration
- large blocks of state management

### `src/features`

Owns:

- feature-level pages
- feature-local components
- feature-local hooks, types, and utils
- page UI and business composition outside the route layer
- stable shared feature UI that is reused across multiple areas but does not justify a new top-level layer yet, such as common table adapters under `src/shared/*`

Suggested structure should grow only when needed. Do not force every folder to exist upfront:

```text
src/features/{feature}/
├── pages/
├── components/
├── hooks/
├── types/
└── utils/
src/shared/
├── components/
├── hooks/
├── types/
└── utils/
```

### `src/i18n`

Owns i18n runtime wiring. It should not become the home for business copy.

### `src/locales`

Owns translation resources, organized by language.

## Gradual Migration Rules

- Simple placeholder UI inside existing route files may remain temporarily.
- For any new non-trivial page, prefer putting the implementation in `src/features` and mounting it from `src/routes`.
- If a route file you are already touching has accumulated a large amount of UI, prefer extracting a feature page while you are there.
- Do not introduce new top-level architecture layers just to make the structure look more complete.

## Example Mapping

For a `projects` list page:

```text
src/routes/(auth)/(app)/projects/index.tsx
src/features/projects/pages/ProjectsPage.tsx
src/locales/en/projects.json
src/locales/zh/projects.json
```

The route file mounts the page. The actual page implementation lives in the feature.
