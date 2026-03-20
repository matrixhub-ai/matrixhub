# MatrixHub UI Collaboration Entry

This file is the shared entry point for both human developers and AI collaborators working in `ui/`.

Default project rules, collaboration conventions, and example materials live in `ui/agents/`. Do not turn core project rules into tool-specific skills or configuration.

## Read Order

1. Read this file first.
2. Then read the relevant docs under `ui/agents/`:
   - `ui/agents/rules/structure.md` for boundaries and folder responsibility
   - `ui/agents/rules/implementation.md` for stack and implementation constraints
   - `ui/agents/rules/tanstack-query.md` for TanStack Query conventions
   - `ui/agents/rules/tanstack-form.md` for TanStack Form conventions
   - `ui/agents/rules/tanstack-router.md` for TanStack Router conventions
   - `ui/agents/rules/tanstack-integration.md` for Query+Form+Router integration and error notifications
   - `ui/agents/rules/page-planning.md` for new page work
   - `ui/agents/rules/api-layer.md` for generated SDK usage
   - `ui/agents/collaboration/review-checklist.md` when reviewing or before handoff
3. Use `ui/agents/examples/page-plan-example.md` only when a concrete example is useful.
4. If `ui/.planning/<task-slug>/task.md` exists, treat it as an optional local working note for that specific task. It does not override project rules.

## Workflow

- Humans usually provide concrete task inputs such as Figma links, screenshots, API references, and short remarks.
- The agent reads the project rules first, then infers route placement, feature structure, API usage, and implementation details from the rules and codebase unless the task explicitly says otherwise.
- Use `ui/.planning/<task-slug>/task.md` only when that task benefits from a short local planning note.
- If the agent creates, changes, or expands a shared wrapper, shared component convention, or other stable project pattern, the agent must update the relevant `ui/agents/` docs in the same change.

## Local Planning Notes

For non-trivial feature work, optional local planning notes may live under:

`ui/.planning/<task-slug>/`

Use this directory only for task-specific working materials such as screenshots, cropped comparisons, exported references, and short planning notes.

Create `task.md` only when the task needs brief recording of:
- concrete inputs
- task-specific constraints or exceptions
- a few implementation decisions
- open questions affecting implementation

Keep `task.md` short and implementation-oriented. Recommended minimal structure:

```md
# Task: <Short Task Name>

## 1. Inputs
- Route / requirement:
- Figma:
- Prototype / screenshot:
- API:
- Related existing pages or components:
- Other inputs:

## 2. Notes
- Task-specific constraints, exceptions, or supplemental context

## 3. Open Questions
- Questions to resolve before or during implementation
````

`ui/.planning/` is local-only, ignored by Git, and must not become a long-term rules repository.

## Rules And Inputs

- `ui/agents/rules/*`: core project rules
- `ui/agents/collaboration/*`: collaboration checklists and review conventions
- `ui/agents/examples/*`: real working examples
- Workflow skills live under `.agents/skills/*`

If directories such as `.claude/`, `.codex/`, or `.opencode/` appear later, they are adapters only. They must not become the source of project rules.

## Do Not

- Manually edit `src/routeTree.gen.ts`
- Keep adding complex business logic directly in `src/routes`
- Add more hardcoded user-facing copy for new UI
- Introduce a new form library, Mantine `useForm`, ad-hoc form state, or a different validation scheme for new forms when `TanStack Form` and `Zod` are the project standards
- Silence lint or type errors in hand-written code with broad suppression comments such as file-level `eslint-disable`, `eslint-disable-next-line`, `@ts-ignore`, or `@ts-expect-error` when a real fix is practical
- Use raw `mantine-react-table` components or page-local table wiring directly in feature pages instead of the project's wrapped table component or adapter
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
