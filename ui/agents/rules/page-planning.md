# Page Planning Rules

For a new page, do the page plan before writing code.

The goal is not to produce a large document. The goal is to align inputs, page boundaries, states, and API needs early enough to avoid rework.

## Recommended Sequence

1. Read `ui/AGENTS.md` and the relevant docs under `ui/agents/`
2. Review the route entry point and nearby pages
3. Review `figma`, screenshots, or the requirement description to understand the page structure, visual hierarchy, layout, and component shape
4. Inspect adjacent pages, existing components, and current Mantine patterns
5. Inspect the relevant SDK / API definition when the page depends on data
6. Start route and feature implementation only after the page plan is aligned

Typical inputs are enough if they are available: a Figma link, Figma MCP reference, CLI Dev Mode access, screenshots, a short requirement description, a relevant API file or SDK module, and short free-form notes. A separate requirement document is optional.

## A Page Plan Should At Least Answer

1. Which route and which feature the page belongs to
2. Whether the route file should stay as a thin adapter or whether a separate feature page is needed
3. What the main sections or components of the page are
4. Which user-facing copy is needed and which locale keys should be added
5. Which states the page has: loading, empty, error, and success
6. Which parts come from design inputs or requirements, and which gaps should be filled by existing code and Mantine
7. Which API reads and writes the page depends on, and which API surfaces are intentionally not needed yet
8. If the page includes forms, which form sections, `Zod` schemas, validation rules, submit flows, and success or error states should be modeled with `TanStack Form`
9. If the page includes a data table, which project table wrapper it should use or extend, and which shared styling or behavior must be centralized in that wrapper instead of left page-specific

## Component Split Guidelines

- Keep route files focused on route wiring, params, redirects, and metadata
- Put full page implementations under `src/features/{feature}/pages`
- Only split into `components/` when a section is meaningfully independent and likely reusable
- Do not over-plan a deeply nested component tree at the planning stage

## Working Principles

- Learn from existing code patterns first, then add rules only when necessary
- For a new page, do the page plan first and implementation second
- If the work introduces or changes a shared wrapper, shared component convention, or other stable project pattern, update the relevant `ui/agents/` docs in the same change
- When uncertain, choose the smaller change that is easier to roll back
- If a rule changes, write it down in the docs instead of leaving it in chat history only

## Recommended Artifact

Prefer maintaining one short, real, executable page plan.

Typical locations:

- the PR description
- a page-planning section in the requirement doc
- a document following the format of `ui/agents/examples/page-plan-example.md`
