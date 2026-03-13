# MatrixHub UI Collaboration Entry

This file is the shared entry point for both human developers and AI collaborators working in `ui/`.

Default project rules, collaboration conventions, and example materials live in `ui/agents/`. Do not turn core project rules into tool-specific skills or configuration.

## Read Order

1. Read this file first.
2. Then read the relevant docs under `ui/agents/`:
   - `ui/agents/rules/structure.md` for boundaries and folder responsibility
   - `ui/agents/rules/implementation.md` for stack and implementation constraints
   - `ui/agents/rules/page-planning.md` for new page work
   - `ui/agents/rules/api-layer.md` for generated SDK usage
   - `ui/agents/collaboration/review-checklist.md` when reviewing or before handoff
3. Use `ui/agents/examples/page-plan-example.md` only when a concrete example is useful.
4. If `ui/.planning/<task-slug>/task.md` exists, treat it as an optional working example for implementing a specific new feature. It does not override project rules.

## Workflow

- Humans usually provide task inputs such as Figma links, screenshots, API references, and short remarks.
- The agent reads the project rules first, then uses `task.md` only when a specific feature task chose that lightweight example format for local working notes.
- The agent should infer route placement, feature structure, API usage, and implementation details from the rules and codebase unless the task explicitly says otherwise.
- If the agent creates, changes, or expands a shared wrapper, shared component convention, or other stable project pattern, the agent must update the relevant `ui/agents/` docs in the same change.

## Rules And Inputs

- `ui/agents/rules/*`: core project rules
- `ui/agents/collaboration/*`: collaboration checklists and review conventions
- `ui/agents/examples/*`: real working examples
- Workflow skills live under `.agents/skills/*`

Temporary working materials for a specific feature task may live under `ui/.planning/<task-slug>/`.

Organize the folder by task, and only keep inputs and drafts that are actually needed for that task. Typical contents include:

- `task.md`: an optional lightweight example note for implementing a specific new feature, usually just inputs, special remarks, and open questions
- Other local attachments: screenshots, cropped images, exported docs, and similar working files

Typical inputs can be as small as a Figma link, Figma MCP reference, CLI Dev Mode access, screenshots, and a few short notes. The default expectation is that humans provide only inputs and special remarks; the agent should infer implementation details from the rules and codebase unless the task has unusual constraints.

`ui/.planning/` is a local working directory and is already ignored by `.gitignore`. Use it for feature-specific working inputs, comparisons, and drafts when helpful, but do not treat it as a required workflow step or a long-term rules repository.

If directories such as `.claude/`, `.codex/`, or `.opencode/` appear later, they are adapters only. They must not become the source of project rules.

## Do Not

- Manually edit `src/routeTree.gen.ts`
- Keep adding complex business logic directly in `src/routes`
- Add more hardcoded user-facing copy for new UI
- Introduce a new form library, Mantine `useForm`, ad-hoc form state, or a different validation scheme for new forms when `TanStack Form` and `Zod` are the project standards
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
