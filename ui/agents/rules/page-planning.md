# Page Planning Rules

For a new page, do the page plan before writing code.

The goal is not to produce a large document. The goal is to align inputs, page boundaries, states, and API needs early enough to avoid rework.

## Recommended Sequence

1. Read `ui/.planning/<task-slug>/task.md` first to confirm the task goal, references, and open questions
2. Review the requirement and route entry point
3. Review the `modao` prototype to understand flow and information structure
4. Review `figma` to understand visual hierarchy, layout, and component shape
5. Review adjacent pages, existing components, and current Mantine patterns
6. If the page depends on data, add the API notes as part of the plan
7. Start route and feature implementation only after the page plan is aligned

## How To Resolve Conflicting Inputs

- `modao` primarily answers what the page needs to do and how the flow should work
- `figma` primarily answers what the page should look like and how the hierarchy should read
- If `figma` is incomplete, fill the gap by following existing implemented pages and Mantine conventions first
- Do not invent a brand-new page pattern just because part of the design is missing

## A Page Plan Should At Least Answer

1. Which route and which feature the page belongs to
2. Whether the route file should stay as a thin adapter or whether a separate feature page is needed
3. What the main sections or components of the page are
4. Which user-facing copy is needed and which locale keys should be added
5. Which states the page has: loading, empty, error, and success
6. Which parts come from `figma`, and which gaps should be filled by existing code and Mantine
7. Which API reads and writes the page depends on

## Component Split Guidelines

- Keep route files focused on route wiring, params, redirects, and metadata
- Put full page implementations under `src/features/{feature}/pages`
- Only split into `components/` when a section is meaningfully independent and likely reusable
- Do not over-plan a deeply nested component tree at the planning stage

## Working Principles

- Learn from existing code patterns first, then add rules only when necessary
- For a new page, do the page plan first and implementation second
- When uncertain, choose the smaller change that is easier to roll back
- If a rule changes, write it down in the docs instead of leaving it in chat history only

## Recommended Artifact

Prefer maintaining one short, real, executable page plan. It can live as a section inside `task.md`.

Typical locations:

- the PR description
- a page-planning section in the requirement doc
- a page-planning section in `ui/.planning/<task-slug>/task.md`
- a document following the format of `ui/agents/examples/page-plan-example.md`
