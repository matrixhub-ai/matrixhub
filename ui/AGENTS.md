# MatrixHub UI Collaboration Entry

Shared entry point for human developers and AI collaborators (Claude, Codex, OpenCode, GitHub PR review bots, …) working in `ui/`.

**This file is the single source of truth.** Tool-specific files (`CLAUDE.md`, `.codex/`, `.opencode/`, …) must only redirect here — never define parallel rules.

**PR review standard**: `ui/agents/workflows/review-checklist.md` (human + AI reviewers read the same file).

---

## Read Order

1. **Always** read this file first.
2. Before any code change, read the root references on demand:
   - `ui/agents/architecture.md` — where code lives
   - `ui/agents/stack.md` — libraries, versions, global wirings, commands
   - `ui/agents/visual.md` — Mantine, colors, Figma / SVG / icons
   - `ui/agents/patterns.md` — cross-library code protocols (Query / Form / Router / SDK / notifications)
3. Pick the **archetype** matching your task and read that one file — it points to real reference code in `src/` and lists the patterns you need.
4. Before handoff, run through `ui/agents/workflows/review-checklist.md`.

If `ui/.planning/<task-slug>/` exists, treat its contents as optional local working notes for that task only. Local notes never override project rules.

---

## Archetype Index

| Archetype | When to use | File |
|---|---|---|
| **List page** | Route-backed paginated list with search params, table, create/delete | `ui/agents/archetypes/list-page.md` |
| **Detail page** | Route-backed detail view keyed by a URL param, prefetched via loader | `ui/agents/archetypes/detail-page.md` |
| **Modal form** | Button opens a Mantine modal with a TanStack Form bound to a mutation | `ui/agents/archetypes/modal-form.md` |
| **Confirmation action** | Destructive action guarded by a confirm modal | `ui/agents/archetypes/confirmation-action.md` |
| **Route-only change** | Search params / loaders / guards without feature UI changes | `ui/agents/archetypes/route-only.md` |

If your task does not fit any archetype, flag it in the PR description — a missing archetype usually means new structural guidance is needed.

---

## Hard Rules (index)

The full definitions live in the files linked. This table is an index so every reviewer can scan the must-not list at a glance.

| Rule | Where it's defined |
|---|---|
| `src/routeTree.gen.ts` is tool-generated — never hand-edit | `stack.md`, `patterns.md §19` |
| Complex page UI lives in `src/features/`, not `src/routes/` | `architecture.md` |
| New forms use `@tanstack/react-form` + Zod — not Mantine `useForm`, not `useState` for field values | `patterns.md §7–§9`, `§19` |
| Data tables go through the project wrapper — `mantine-react-table` is not wired directly in a feature page | `patterns.md §17`, `§19` |
| API errors surface via global `MutationCache` / `QueryCache` — no per-component `try/catch` / `notifications.show()` | `patterns.md §4`, `§16`, `§19` |
| New user-facing copy goes into locale files (`en` + `zh` both) — not hardcoded strings | `stack.md` |
| SDK access via `@matrixhub/api-ts/*` alias only — generated `.pb.ts` files are read-only | `patterns.md §18`, `§19` |
| Do not create a second `QueryClient` — read from router context | `patterns.md §14`, `§19` |
| Do not silence lint/type errors with broad suppressions | `stack.md` |
| Colors: component-native → project semantic → Mantine semantic → palette-index (last resort) | `visual.md` |
| Shared-wrapper changes update `ui/agents/` docs in the **same** PR | `architecture.md`, below |

---

## Working Principles

- **Learn from existing code first.** When something is already expressed in `src/features/projects/`, `src/shared/`, or `src/queryClient.ts`, mimic it before inventing a new convention.
- **Plan before implementing** for non-trivial pages. The archetype file for your task lists what to answer before writing code.
- **Shared primitives and their docs ship together.** Creating or changing a shared wrapper, shared component convention, or other stable project pattern requires updating the relevant `ui/agents/` doc in the same PR. Primitives without docs drift fastest.
- **Prefer smaller, rollbackable changes.** When uncertain, choose the smaller change that is easier to revert.
- **Write rules down, not in chat.** If an agreement emerges, put it in the appropriate `ui/agents/` doc — don't leave it in PR comments or chat history.

---

## Local Planning Notes (optional)

For non-trivial feature work, task-specific working materials may live under:

```
ui/.planning/<task-slug>/
```

Use it for screenshots, cropped comparisons, exported references, and a short `task.md`. `ui/.planning/` is gitignored and must not become a long-term rules repository. Project rules always live under `ui/agents/`.

Minimal `task.md` structure when one is useful:

```md
# Task: <Short Task Name>

## Inputs
- Route / requirement:
- Figma / screenshots:
- API / SDK modules:
- Related existing pages or components:

## Notes
- Task-specific constraints, exceptions, or supplemental context

## Open Questions
- Questions to resolve before or during implementation
```

---

## Structure Reference

```
ui/
├── AGENTS.md                           ← this file
├── CLAUDE.md                           ← redirect to AGENTS.md
└── agents/
    ├── architecture.md                 ← where code lives
    ├── stack.md                        ← libraries, versions, wirings, commands
    ├── visual.md                       ← Mantine, colors, Figma assets
    ├── patterns.md                     ← cross-library code protocols (§1–§19)
    ├── archetypes/                     ← "I'm building X" playbooks
    │   ├── list-page.md
    │   ├── detail-page.md
    │   ├── modal-form.md
    │   ├── confirmation-action.md
    │   └── route-only.md
    ├── workflows/                      ← "I'm going through process X"
    │   └── review-checklist.md
    └── examples/                       ← concrete worked samples
        └── page-plan-example.md
```

Workflow skills, if any, live under `.agents/skills/*` and are separate from these project rules.
