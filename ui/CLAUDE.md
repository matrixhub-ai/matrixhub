# Claude Entry

Use `AGENTS.md` as the authoritative reference for this repository's collaboration rules and workflow.

Bootstrap requirements for this repository:

1. Read `AGENTS.md` first.
2. Then continue with the relevant docs under `ui/agents/` following the read order defined in `AGENTS.md`.
3. Do not create or rely on a parallel tool-specific ruleset when `AGENTS.md` or `ui/agents/*` already defines the rule.

Critical constraints to preserve even when context is short:

- Do not manually edit `src/routeTree.gen.ts`.
- Keep non-trivial page implementation out of `src/routes`; use `src/features` for complex page UI and business composition.
- Add new user-facing copy through locale files rather than new hardcoded strings.
- Before handoff, run at least the relevant `pnpm lint` and `pnpm typecheck` for the current change.
