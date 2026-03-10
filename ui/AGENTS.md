# MatrixHub UI Collaboration Entry

This file is the shared entry point for both human developers and AI collaborators working in `ui/`.

Default project rules, collaboration conventions, and example materials live in `ui/agents/`. Do not turn core project rules into tool-specific skills or configuration.

## Read Order

1. Read this file first.
2. Then read the relevant docs based on the scope of the task:
   - Structure and boundaries: `ui/agents/rules/structure.md`
   - Implementation constraints: `ui/agents/rules/implementation.md`
   - Page planning: `ui/agents/rules/page-planning.md`
   - API notes: `ui/agents/rules/api-layer.md`
   - Review checklist: `ui/agents/collaboration/review-checklist.md`
   - Page planning example: `ui/agents/examples/page-plan-example.md`

## Source Of Truth

- `ui/agents/rules/*`: core project rules
- `ui/agents/collaboration/*`: collaboration checklists and review conventions
- `ui/agents/examples/*`: real working examples
- If workflow skills are added later, put them under `ui/agents/skills/*`

Temporary task materials should live under `ui/.planning/<task-slug>/`.

Organize the folder by task, and only keep inputs and drafts that are actually needed for that task. Typical contents include:

- `task.md`: the working brief for the current task, including references, page planning, temporary notes, and open questions
- Other local attachments: screenshots, cropped images, exported docs, and similar working files

`ui/.planning/` is a local working directory and is already ignored by `.gitignore`. Use it for task inputs, comparisons, and drafts, but do not treat it as a long-term rules repository.

If directories such as `.claude/`, `.codex/`, or `.opencode/` appear later, they are adapters only. They must not become the source of project rules.

## Current Code Boundaries

- `src/routes`: route files, layouts, redirects, and route-level metadata
- `src/features`: non-trivial pages and feature-local implementation
- `src/i18n`: i18n bootstrap and resource loading
- `src/locales`: translation resource files
- `src/main.tsx`, `src/router.tsx`, `src/mantineTheme.ts`: app-level infrastructure
- `src/routeTree.gen.ts`: generated file, do not edit manually

## Incremental Adoption

- These rules apply first to new code and code currently being changed.
- Existing files may be migrated gradually. Do not force a large refactor just to satisfy the rules in one pass.
- When a route file becomes too complex, move UI and logic into `src/features` and keep the route file as a thin adapter.

## Do Not

- Manually edit `src/routeTree.gen.ts`
- Keep adding complex business logic directly in `src/routes`
- Add more hardcoded user-facing copy for new UI
- Build a parallel styling system when Mantine theme tokens already cover the use case
- Add new top-level architecture layers without agreement

## Common Commands

```bash
pnpm dev
pnpm build
pnpm lint
pnpm typecheck
```

Before submitting changes, at minimum make sure the relevant parts of the current change pass `pnpm lint` and `pnpm typecheck`.
